package usecase

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

const (
	defaultExtensionPairingTTL = 10 * time.Minute
	defaultExtensionTokenTTL   = 30 * 24 * time.Hour
	extensionTokenRandomBytes  = 32
	extensionUserCodeBytes     = 10
	extensionUserCodeRetries   = 3
	extensionPairingLimit      = 5
)

type ExtensionUsecase struct {
	repo       domain.ExtensionPairingRepository
	rateLimit  domain.RateLimitRepository
	now        func() time.Time
	random     io.Reader
	pairingTTL time.Duration
	tokenTTL   time.Duration
}

type ExtensionTokenIssueResult struct {
	RawToken  string
	Scopes    []string
	ExpiresAt time.Time
}

func NewExtensionUsecase(repo domain.ExtensionPairingRepository, rateLimit domain.RateLimitRepository) (*ExtensionUsecase, error) {
	if repo == nil {
		return nil, errors.New("extension usecase: repository is nil")
	}
	if rateLimit == nil {
		return nil, errors.New("extension usecase: rate limit repository is nil")
	}
	return &ExtensionUsecase{
		repo:       repo,
		rateLimit:  rateLimit,
		now:        time.Now,
		random:     rand.Reader,
		pairingTTL: defaultExtensionPairingTTL,
		tokenTTL:   defaultExtensionTokenTTL,
	}, nil
}

func (u *ExtensionUsecase) StartPairing(ctx context.Context) (*domain.ExtensionPairing, error) {
	now := u.now().UTC()
	var lastErr error
	for range extensionUserCodeRetries {
		userCode, err := u.generateUserCode()
		if err != nil {
			return nil, fmt.Errorf("extension usecase: generate user code: %w", err)
		}
		pairing, err := u.repo.CreateExtensionPairing(ctx, userCode, now.Add(u.pairingTTL))
		if err == nil {
			return pairing, nil
		}
		lastErr = err
		if err != nil && !errors.Is(err, domain.ErrAlreadyExists) {
			return nil, fmt.Errorf("extension usecase: start pairing: %w", err)
		}
	}
	return nil, fmt.Errorf("extension usecase: start pairing: %w", lastErr)
}

func (u *ExtensionUsecase) ApprovePairing(ctx context.Context, userID uuid.UUID, userCode string, clientIdentifier string) error {
	normalizedUserCode, err := normalizeUserCode(userCode)
	if err != nil {
		return err
	}
	clientIdentifier = strings.TrimSpace(clientIdentifier)
	if userID == uuid.Nil || clientIdentifier == "" {
		return domain.ErrInvalidInput
	}
	if err := u.checkPairingRateLimit(ctx, "extension_pairing_approve", "code:"+normalizedUserCode); err != nil {
		return err
	}
	if err := u.checkPairingRateLimit(ctx, "extension_pairing_approve", "client:"+clientIdentifier); err != nil {
		return err
	}
	scopes := domain.DefaultExtensionTokenScopes()
	if err := u.repo.ApproveExtensionPairing(ctx, normalizedUserCode, userID, scopes, u.now().UTC()); err != nil {
		return fmt.Errorf("extension usecase: approve pairing: %w", err)
	}
	return nil
}

func (u *ExtensionUsecase) PairingStatus(ctx context.Context, pairingID uuid.UUID) (*domain.ExtensionPairing, error) {
	if pairingID == uuid.Nil {
		return nil, domain.ErrInvalidInput
	}
	pairing, err := u.repo.GetExtensionPairingStatus(ctx, pairingID, u.now().UTC())
	if err != nil {
		return nil, fmt.Errorf("extension usecase: pairing status: %w", err)
	}
	return pairing, nil
}

func (u *ExtensionUsecase) ClaimPairing(ctx context.Context, pairingID uuid.UUID, clientIdentifier string) (*ExtensionTokenIssueResult, error) {
	clientIdentifier = strings.TrimSpace(clientIdentifier)
	if pairingID == uuid.Nil || clientIdentifier == "" {
		return nil, domain.ErrInvalidInput
	}
	if err := u.checkPairingRateLimit(ctx, "extension_pairing_claim", "pairing:"+pairingID.String()); err != nil {
		return nil, err
	}
	if err := u.checkPairingRateLimit(ctx, "extension_pairing_claim", "client:"+clientIdentifier); err != nil {
		return nil, err
	}

	rawToken, err := u.generateRawToken()
	if err != nil {
		return nil, fmt.Errorf("extension usecase: generate token: %w", err)
	}

	now := u.now().UTC()
	scopes := domain.DefaultExtensionTokenScopes()
	expiresAt := now.Add(u.tokenTTL)
	token, err := u.repo.CreateExtensionTokenForApprovedPairing(ctx, domain.CreateExtensionTokenForPairingInput{
		PairingID: pairingID,
		TokenID:   uuid.New(),
		TokenHash: domain.HashExtensionToken(rawToken),
		Scopes:    scopes,
		ExpiresAt: expiresAt,
		Now:       now,
	})
	if err != nil {
		return nil, fmt.Errorf("extension usecase: consume pairing: %w", err)
	}

	return &ExtensionTokenIssueResult{
		RawToken:  rawToken,
		Scopes:    append([]string(nil), token.Scopes...),
		ExpiresAt: expiresAt,
	}, nil
}

func (u *ExtensionUsecase) RevokeSelf(ctx context.Context, userID uuid.UUID, tokenID uuid.UUID) error {
	if userID == uuid.Nil || tokenID == uuid.Nil {
		return domain.ErrInvalidInput
	}
	if err := u.repo.RevokeExtensionToken(ctx, tokenID, userID, u.now().UTC()); err != nil {
		return fmt.Errorf("extension usecase: revoke self: %w", err)
	}
	return nil
}

func (u *ExtensionUsecase) generateRawToken() (string, error) {
	buf := make([]byte, extensionTokenRandomBytes)
	if _, err := io.ReadFull(u.random, buf); err != nil {
		return "", err
	}
	return domain.ExtensionTokenPrefix + base64.RawURLEncoding.EncodeToString(buf), nil
}

func (u *ExtensionUsecase) generateUserCode() (string, error) {
	buf := make([]byte, extensionUserCodeBytes)
	if _, err := io.ReadFull(u.random, buf); err != nil {
		return "", err
	}
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	chars := make([]byte, 10)
	for i := range chars {
		chars[i] = alphabet[int(buf[i])%len(alphabet)]
	}
	return string(chars[:5]) + "-" + string(chars[5:]), nil
}

func normalizeUserCode(value string) (string, error) {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	normalized = strings.ReplaceAll(normalized, " ", "")
	normalized = strings.ReplaceAll(normalized, "-", "")
	if len(normalized) != 10 {
		return "", domain.ErrInvalidInput
	}
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	for _, r := range normalized {
		if !strings.ContainsRune(alphabet, r) {
			return "", domain.ErrInvalidInput
		}
	}
	return normalized[:5] + "-" + normalized[5:], nil
}

func (u *ExtensionUsecase) checkPairingRateLimit(ctx context.Context, bucketPrefix string, sensitiveParts ...string) error {
	identifier := hashedRateLimitIdentifier(sensitiveParts...)
	if identifier == "" {
		return domain.ErrInvalidInput
	}
	now := u.now().UTC()
	bucket := bucketPrefix + ":" + now.Format("200601021504")
	current, exceeded, err := u.rateLimit.IncrementAndCheck(ctx, identifier, bucket, extensionPairingLimit)
	if err != nil {
		return fmt.Errorf("extension usecase: pairing rate limit: %w", err)
	}
	if exceeded {
		return domain.ErrRateLimitExceeded
	}
	previousBucket := bucketPrefix + ":" + now.Add(-time.Minute).Format("200601021504")
	previous, err := u.rateLimit.Count(ctx, identifier, previousBucket)
	if err != nil {
		return fmt.Errorf("extension usecase: pairing rate limit previous bucket: %w", err)
	}
	// The counter increment is intentionally not rolled back if a later
	// validation step fails; repeated invalid attempts should still consume
	// the short-window allowance.
	if current+previous > extensionPairingLimit {
		return domain.ErrRateLimitExceeded
	}
	return nil
}

func hashedRateLimitIdentifier(parts ...string) string {
	normalized := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			normalized = append(normalized, part)
		}
	}
	if len(normalized) == 0 {
		return ""
	}
	sum := sha256.Sum256([]byte(strings.Join(normalized, "\x00")))
	return "hash:" + hex.EncodeToString(sum[:])
}
