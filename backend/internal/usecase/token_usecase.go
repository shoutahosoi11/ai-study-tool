package usecase

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"github.com/shout/ai-study-tool/backend/internal/domain"
)

const maxAdRewardClockSkew = 10 * time.Minute

const (
	minAdRewardProviderLength = 2
	maxAdRewardProviderLength = 40
	minAdRewardNonceLength    = 16
	maxAdRewardNonceLength    = 128
)

type TokenUsecase struct {
	budgetRepo      domain.QuestionBudgetRepository
	adRewardHMACKey string
	now             func() time.Time
}

func NewTokenUsecaseWithAdRewardSecret(budgetRepo domain.QuestionBudgetRepository, secret string) *TokenUsecase {
	return &TokenUsecase{
		budgetRepo:      budgetRepo,
		adRewardHMACKey: strings.TrimSpace(secret),
		now:             time.Now,
	}
}

type AwardAdTokensInput struct {
	Provider   string
	Nonce      string
	RewardedAt time.Time
	Signature  string
}

func (u *TokenUsecase) Award(ctx context.Context, user *domain.User, input AwardAdTokensInput) (*domain.QuestionTokenBalance, error) {
	if user == nil {
		return nil, domain.ErrInvalidInput
	}

	claim, err := u.validateAdRewardClaim(user, input)
	if err != nil {
		return nil, err
	}

	balance, err := u.budgetRepo.AwardAdTokens(ctx, user.ID, claim, u.now())
	if err != nil {
		return nil, err
	}
	balance.Plan = user.Plan
	return balance, nil
}

func (u *TokenUsecase) Balance(ctx context.Context, user *domain.User) (*domain.QuestionTokenBalance, error) {
	return u.budgetRepo.GetBalance(ctx, user.ID, user.Plan, u.now())
}

func (u *TokenUsecase) validateAdRewardClaim(user *domain.User, input AwardAdTokensInput) (domain.AdRewardClaim, error) {
	if strings.TrimSpace(u.adRewardHMACKey) == "" {
		return domain.AdRewardClaim{}, domain.ErrInvalidInput
	}

	claim := domain.AdRewardClaim{
		Provider:   strings.ToLower(strings.TrimSpace(input.Provider)),
		Nonce:      strings.TrimSpace(input.Nonce),
		RewardedAt: input.RewardedAt.UTC(),
	}
	if !validAdRewardToken(claim.Provider, minAdRewardProviderLength, maxAdRewardProviderLength) ||
		!validAdRewardToken(claim.Nonce, minAdRewardNonceLength, maxAdRewardNonceLength) {
		return domain.AdRewardClaim{}, domain.ErrInvalidInput
	}
	if claim.RewardedAt.IsZero() {
		return domain.AdRewardClaim{}, domain.ErrInvalidInput
	}

	now := u.now()
	if claim.RewardedAt.Before(now.Add(-maxAdRewardClockSkew)) || claim.RewardedAt.After(now.Add(maxAdRewardClockSkew)) {
		return domain.AdRewardClaim{}, domain.ErrInvalidInput
	}

	expected := signAdRewardClaim(u.adRewardHMACKey, user.ID.String(), claim)
	actual, err := decodeAdRewardSignature(input.Signature)
	if err != nil {
		return domain.AdRewardClaim{}, domain.ErrInvalidInput
	}
	if hmac.Equal(actual, expected) {
		return claim, nil
	}
	return domain.AdRewardClaim{}, domain.ErrForbidden
}

func signAdRewardClaim(secret string, userID string, claim domain.AdRewardClaim) []byte {
	payload, err := json.Marshal(struct {
		UserID     string `json:"user_id"`
		Provider   string `json:"provider"`
		Nonce      string `json:"nonce"`
		RewardedAt string `json:"rewarded_at"`
	}{
		UserID:     userID,
		Provider:   claim.Provider,
		Nonce:      claim.Nonce,
		RewardedAt: claim.RewardedAt.UTC().Format(time.RFC3339),
	})
	if err != nil {
		payload = []byte{}
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(payload)
	return mac.Sum(nil)
}

func validAdRewardToken(value string, minLen int, maxLen int) bool {
	if len(value) < minLen || len(value) > maxLen {
		return false
	}
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-' || r == '_':
		default:
			return false
		}
	}
	return true
}

func decodeAdRewardSignature(signature string) ([]byte, error) {
	normalized := strings.TrimSpace(signature)
	if normalized == "" {
		return nil, domain.ErrInvalidInput
	}
	if decoded, err := hex.DecodeString(normalized); err == nil {
		return decoded, nil
	}
	if decoded, err := base64.StdEncoding.DecodeString(normalized); err == nil {
		return decoded, nil
	}
	return nil, domain.ErrInvalidInput
}
