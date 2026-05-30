package admob

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shout/ai-study-tool/backend/internal/domain"
)

const (
	defaultPublicKeysURL = "https://www.gstatic.com/admob/reward/verifier-keys.json"
	defaultKeyCacheTTL   = 23 * time.Hour
	defaultStaleKeyTTL   = 24 * time.Hour
	defaultClockSkew     = 10 * time.Minute
)

type PublicKeyFetcher interface {
	FetchPublicKeys(ctx context.Context) (map[string]*ecdsa.PublicKey, error)
}

type HTTPPublicKeyFetcher struct {
	url        string
	httpClient *http.Client
}

func NewHTTPPublicKeyFetcherFromEnv() *HTTPPublicKeyFetcher {
	keysURL := strings.TrimSpace(os.Getenv("ADMOB_SSV_PUBLIC_KEYS_URL"))
	if keysURL == "" {
		keysURL = defaultPublicKeysURL
	}
	return &HTTPPublicKeyFetcher{
		url:        keysURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (f *HTTPPublicKeyFetcher) FetchPublicKeys(ctx context.Context) (map[string]*ecdsa.PublicKey, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.url, nil)
	if err != nil {
		return nil, fmt.Errorf("admob ssv: create public key request: %w", err)
	}
	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("admob ssv: fetch public keys: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("admob ssv: public key status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("admob ssv: read public keys: %w", err)
	}
	return parsePublicKeys(body)
}

type SSVVerifier struct {
	fetcher PublicKeyFetcher
	now     func() time.Time
	ttl     time.Duration
	skew    time.Duration

	mu        sync.Mutex
	cached    map[string]*ecdsa.PublicKey
	expiresAt time.Time
	staleAt   time.Time
}

func NewSSVVerifier(fetcher PublicKeyFetcher) *SSVVerifier {
	return &SSVVerifier{
		fetcher: fetcher,
		now:     time.Now,
		ttl:     defaultKeyCacheTTL,
		skew:    defaultClockSkew,
	}
}

func NewSSVVerifierFromEnv() *SSVVerifier {
	return NewSSVVerifier(NewHTTPPublicKeyFetcherFromEnv())
}

func (v *SSVVerifier) Verify(ctx context.Context, rawQuery string, now time.Time) (*domain.AdMobSSVCallback, error) {
	rawQuery = strings.TrimSpace(rawQuery)
	if rawQuery == "" {
		return nil, domain.ErrInvalidInput
	}
	if now.IsZero() {
		now = v.now().UTC()
	}

	content, signatureValue, keyID, err := splitSignedQuery(rawQuery)
	if err != nil {
		return nil, err
	}
	signature, err := decodeURLSafeBase64(signatureValue)
	if err != nil {
		return nil, domain.ErrForbidden
	}
	key, err := v.publicKey(ctx, keyID, now)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256([]byte(content))
	if !ecdsa.VerifyASN1(key, sum[:], signature) {
		return nil, domain.ErrForbidden
	}

	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		return nil, domain.ErrInvalidInput
	}
	callback, err := parseCallback(values)
	if err != nil {
		return nil, err
	}
	if callback.Timestamp.Before(now.Add(-v.skew)) || callback.Timestamp.After(now.Add(v.skew)) {
		return nil, domain.ErrForbidden
	}
	return callback, nil
}

func (v *SSVVerifier) publicKey(ctx context.Context, keyID string, now time.Time) (*ecdsa.PublicKey, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.cached == nil || now.After(v.expiresAt) {
		keys, err := v.fetcher.FetchPublicKeys(ctx)
		if err != nil {
			if v.cached == nil || now.After(v.staleAt) {
				return nil, err
			}
			slog.Warn("admob_ssv_public_key_fetch_failed_using_stale_cache", "error", err)
		} else {
			if len(keys) == 0 {
				return nil, domain.ErrForbidden
			}
			v.cached = keys
			v.expiresAt = now.Add(v.ttl)
			v.staleAt = v.expiresAt.Add(defaultStaleKeyTTL)
		}
	}
	key := v.cached[strings.TrimSpace(keyID)]
	if key == nil {
		return nil, domain.ErrForbidden
	}
	return key, nil
}

func splitSignedQuery(rawQuery string) (string, string, string, error) {
	const signatureMarker = "&signature="
	const keyIDMarker = "&key_id="

	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		return "", "", "", domain.ErrInvalidInput
	}
	if len(values["signature"]) != 1 || len(values["key_id"]) != 1 {
		return "", "", "", domain.ErrInvalidInput
	}

	sigIndex := strings.LastIndex(rawQuery, signatureMarker)
	if sigIndex <= 0 {
		return "", "", "", domain.ErrInvalidInput
	}
	content := rawQuery[:sigIndex]
	signedSuffix := rawQuery[sigIndex+len(signatureMarker):]
	keyIndex := strings.LastIndex(signedSuffix, keyIDMarker)
	if keyIndex <= 0 {
		return "", "", "", domain.ErrInvalidInput
	}
	signature := signedSuffix[:keyIndex]
	keyID := signedSuffix[keyIndex+len(keyIDMarker):]
	if content == "" || signature == "" || keyID == "" {
		return "", "", "", domain.ErrInvalidInput
	}
	if strings.Contains(keyID, "&") {
		return "", "", "", domain.ErrInvalidInput
	}
	return content, signature, keyID, nil
}

func parseCallback(values url.Values) (*domain.AdMobSSVCallback, error) {
	rewardAmount, err := strconv.Atoi(strings.TrimSpace(values.Get("reward_amount")))
	if err != nil || rewardAmount <= 0 {
		return nil, domain.ErrInvalidInput
	}
	timestampMS, err := strconv.ParseInt(strings.TrimSpace(values.Get("timestamp")), 10, 64)
	if err != nil || timestampMS <= 0 {
		return nil, domain.ErrInvalidInput
	}
	callback := &domain.AdMobSSVCallback{
		TransactionID: strings.TrimSpace(values.Get("transaction_id")),
		UserID:        strings.TrimSpace(values.Get("user_id")),
		CustomData:    strings.TrimSpace(values.Get("custom_data")),
		AdUnit:        strings.TrimSpace(values.Get("ad_unit")),
		RewardAmount:  rewardAmount,
		RewardItem:    strings.TrimSpace(values.Get("reward_item")),
		Timestamp:     time.UnixMilli(timestampMS).UTC(),
	}
	if callback.TransactionID == "" || callback.AdUnit == "" || callback.RewardItem == "" {
		return nil, domain.ErrInvalidInput
	}
	return callback, nil
}

func parsePublicKeys(data []byte) (map[string]*ecdsa.PublicKey, error) {
	var payload struct {
		Keys []struct {
			KeyID  int64  `json:"keyId"`
			Base64 string `json:"base64"`
			PEM    string `json:"pem"`
		} `json:"keys"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("admob ssv: parse public keys json: %w", err)
	}
	keys := make(map[string]*ecdsa.PublicKey, len(payload.Keys))
	for _, item := range payload.Keys {
		key, err := parsePublicKey(item.Base64, item.PEM)
		if err != nil {
			return nil, err
		}
		keys[strconv.FormatInt(item.KeyID, 10)] = key
	}
	if len(keys) == 0 {
		return nil, domain.ErrForbidden
	}
	return keys, nil
}

func parsePublicKey(rawBase64 string, rawPEM string) (*ecdsa.PublicKey, error) {
	var der []byte
	var err error
	if strings.TrimSpace(rawBase64) != "" {
		der, err = base64.StdEncoding.DecodeString(strings.TrimSpace(rawBase64))
		if err != nil {
			return nil, fmt.Errorf("admob ssv: decode public key: %w", err)
		}
	} else {
		return nil, domain.ErrForbidden
	}
	publicKey, err := x509.ParsePKIXPublicKey(der)
	if err != nil {
		return nil, fmt.Errorf("admob ssv: parse public key: %w", err)
	}
	ecKey, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, domain.ErrForbidden
	}
	return ecKey, nil
}

func decodeURLSafeBase64(value string) ([]byte, error) {
	value = strings.TrimSpace(value)
	if decoded, err := base64.RawURLEncoding.DecodeString(value); err == nil {
		return decoded, nil
	}
	return base64.URLEncoding.DecodeString(value)
}

var _ domain.AdMobSSVVerifier = (*SSVVerifier)(nil)
