package admob

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/shout/ai-study-tool/backend/internal/domain"
)

type fakeKeyFetcher struct {
	key       *ecdsa.PublicKey
	err       error
	fetches   int
	lastKeyID string
}

func (f *fakeKeyFetcher) FetchPublicKeys(ctx context.Context) (map[string]*ecdsa.PublicKey, error) {
	f.fetches++
	if f.err != nil {
		return nil, f.err
	}
	return map[string]*ecdsa.PublicKey{"123": f.key}, nil
}

func TestSSVVerifierAcceptsValidSignature(t *testing.T) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	now := time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC)
	content := fmt.Sprintf("ad_unit=ad-unit&reward_amount=1&reward_item=token&timestamp=%d&transaction_id=txn-1&user_id=11111111-1111-1111-1111-111111111111", now.UnixMilli())
	rawQuery := signAdMobTestQuery(t, privateKey, content)
	verifier := NewSSVVerifier(&fakeKeyFetcher{key: &privateKey.PublicKey})

	callback, err := verifier.Verify(context.Background(), rawQuery, now)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if callback.TransactionID != "txn-1" || callback.AdUnit != "ad-unit" {
		t.Fatalf("unexpected callback: %#v", callback)
	}
}

func TestSSVVerifierRejectsInvalidSignature(t *testing.T) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	otherKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate other key: %v", err)
	}
	now := time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC)
	content := fmt.Sprintf("ad_unit=ad-unit&reward_amount=1&reward_item=token&timestamp=%d&transaction_id=txn-1&user_id=11111111-1111-1111-1111-111111111111", now.UnixMilli())
	rawQuery := signAdMobTestQuery(t, otherKey, content)
	verifier := NewSSVVerifier(&fakeKeyFetcher{key: &privateKey.PublicKey})

	if _, err := verifier.Verify(context.Background(), rawQuery, now); err == nil {
		t.Fatal("expected invalid signature to be rejected")
	}
}

func TestSSVVerifierRejectsStaleTimestamp(t *testing.T) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	now := time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC)
	old := now.Add(-defaultClockSkew - time.Second)
	content := fmt.Sprintf("ad_unit=ad-unit&reward_amount=1&reward_item=token&timestamp=%d&transaction_id=txn-1&user_id=11111111-1111-1111-1111-111111111111", old.UnixMilli())
	rawQuery := signAdMobTestQuery(t, privateKey, content)
	verifier := NewSSVVerifier(&fakeKeyFetcher{key: &privateKey.PublicKey})

	if _, err := verifier.Verify(context.Background(), rawQuery, now); err == nil {
		t.Fatal("expected stale timestamp to be rejected")
	}
}

func TestSSVVerifierCacheExpiresAndRefetches(t *testing.T) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	now := time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC)
	content := fmt.Sprintf("ad_unit=ad-unit&reward_amount=1&reward_item=token&timestamp=%d&transaction_id=txn-1&user_id=11111111-1111-1111-1111-111111111111", now.UnixMilli())
	rawQuery := signAdMobTestQuery(t, privateKey, content)
	fetcher := &fakeKeyFetcher{key: &privateKey.PublicKey}
	verifier := NewSSVVerifier(fetcher)
	verifier.ttl = time.Second

	if _, err := verifier.Verify(context.Background(), rawQuery, now); err != nil {
		t.Fatalf("first verify: %v", err)
	}
	if _, err := verifier.Verify(context.Background(), rawQuery, now.Add(2*time.Second)); err != nil {
		t.Fatalf("second verify: %v", err)
	}
	if fetcher.fetches != 2 {
		t.Fatalf("expected refetch after cache expiry, got %d", fetcher.fetches)
	}
}

func TestSSVVerifierUsesStaleCacheWhenRefetchFails(t *testing.T) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	now := time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC)
	content := fmt.Sprintf("ad_unit=ad-unit&reward_amount=1&reward_item=token&timestamp=%d&transaction_id=txn-1&user_id=11111111-1111-1111-1111-111111111111", now.UnixMilli())
	rawQuery := signAdMobTestQuery(t, privateKey, content)
	fetcher := &fakeKeyFetcher{key: &privateKey.PublicKey}
	verifier := NewSSVVerifier(fetcher)
	verifier.ttl = time.Second

	if _, err := verifier.Verify(context.Background(), rawQuery, now); err != nil {
		t.Fatalf("first verify: %v", err)
	}
	fetcher.err = domain.ErrForbidden
	if _, err := verifier.Verify(context.Background(), rawQuery, now.Add(2*time.Second)); err != nil {
		t.Fatalf("expected stale cache fallback, got %v", err)
	}
	if fetcher.fetches != 2 {
		t.Fatalf("expected failed refetch after cache expiry, got %d fetches", fetcher.fetches)
	}
}

func TestSSVVerifierFailsClosedWhenKeyFetchFails(t *testing.T) {
	verifier := NewSSVVerifier(&fakeKeyFetcher{err: domain.ErrForbidden})
	_, err := verifier.Verify(context.Background(), "ad_unit=a&reward_amount=1&reward_item=t&timestamp=1&transaction_id=x&signature=s&key_id=123", time.Now())
	if err == nil {
		t.Fatal("expected fetch failure to reject callback")
	}
}

func TestSSVVerifierRejectsStaleCacheAfterFallbackWindow(t *testing.T) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	now := time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC)
	content := fmt.Sprintf("ad_unit=ad-unit&reward_amount=1&reward_item=token&timestamp=%d&transaction_id=txn-1&user_id=11111111-1111-1111-1111-111111111111", now.UnixMilli())
	rawQuery := signAdMobTestQuery(t, privateKey, content)
	fetcher := &fakeKeyFetcher{key: &privateKey.PublicKey}
	verifier := NewSSVVerifier(fetcher)
	verifier.ttl = time.Second

	if _, err := verifier.Verify(context.Background(), rawQuery, now); err != nil {
		t.Fatalf("first verify: %v", err)
	}
	fetcher.err = domain.ErrForbidden
	if _, err := verifier.Verify(context.Background(), rawQuery, now.Add(defaultStaleKeyTTL+2*time.Second)); err == nil {
		t.Fatal("expected stale cache outside fallback window to fail closed")
	}
}

func TestSplitSignedQueryRequiresGoogleSignatureSuffix(t *testing.T) {
	content := "ad_unit=ad%20unit&reward_amount=1&reward_item=token&timestamp=1&transaction_id=txn%2B1&user_id=11111111-1111-1111-1111-111111111111"
	rawQuery := content + "&signature=abc_DEF-123&key_id=123"

	gotContent, signature, keyID, err := splitSignedQuery(rawQuery)
	if err != nil {
		t.Fatalf("splitSignedQuery failed: %v", err)
	}
	if gotContent != content {
		t.Fatalf("content was modified: %q", gotContent)
	}
	if signature != "abc_DEF-123" || keyID != "123" {
		t.Fatalf("unexpected signature/key: %q %q", signature, keyID)
	}
}

func TestSplitSignedQueryRejectsMalformedSignatureParams(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{name: "xsignature is not signature", raw: "ad_unit=a&xsignature=s&key_id=123"},
		{name: "multiple signatures", raw: "ad_unit=a&signature=first&signature=second&key_id=123"},
		{name: "key before signature", raw: "ad_unit=a&key_id=123&signature=s"},
		{name: "key missing", raw: "ad_unit=a&signature=s"},
		{name: "signature missing", raw: "ad_unit=a&key_id=123"},
		{name: "signature first", raw: "signature=s&key_id=123"},
		{name: "key not last", raw: "ad_unit=a&signature=s&key_id=123&extra=1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, _, _, err := splitSignedQuery(tt.raw); err == nil {
				t.Fatal("expected malformed query to be rejected")
			}
		})
	}
}

func TestParsePublicKeys(t *testing.T) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	der, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		t.Fatalf("marshal key: %v", err)
	}
	keys, err := parsePublicKeys([]byte(fmt.Sprintf(`{"keys":[{"keyId":123,"base64":%q}]}`, base64.StdEncoding.EncodeToString(der))))
	if err != nil {
		t.Fatalf("parsePublicKeys failed: %v", err)
	}
	if keys["123"] == nil {
		t.Fatal("expected key 123")
	}
}

func signAdMobTestQuery(t *testing.T, privateKey *ecdsa.PrivateKey, content string) string {
	t.Helper()
	sum := sha256.Sum256([]byte(content))
	signature, err := ecdsa.SignASN1(rand.Reader, privateKey, sum[:])
	if err != nil {
		t.Fatalf("sign query: %v", err)
	}
	return content + "&signature=" + base64.RawURLEncoding.EncodeToString(signature) + "&key_id=123"
}
