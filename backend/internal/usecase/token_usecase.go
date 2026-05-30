package usecase

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	appconfig "github.com/shout/ai-study-tool/backend/internal/config"
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
	adMobVerifier   domain.AdMobSSVVerifier
	adRewardHMACKey string
	appEnv          string
	now             func() time.Time
}

func NewTokenUsecaseWithAdRewardSecret(budgetRepo domain.QuestionBudgetRepository, secret string) *TokenUsecase {
	return NewTokenUsecaseWithAdRewardSecretAndEnv(budgetRepo, nil, secret, "")
}

func NewTokenUsecaseWithAdRewardSecretAndEnv(budgetRepo domain.QuestionBudgetRepository, adMobVerifier domain.AdMobSSVVerifier, secret string, appEnv string) *TokenUsecase {
	return &TokenUsecase{
		budgetRepo:      budgetRepo,
		adMobVerifier:   adMobVerifier,
		adRewardHMACKey: strings.TrimSpace(secret),
		appEnv:          strings.TrimSpace(appEnv),
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
	if appconfig.NormalizeAppEnv(u.appEnv).IsProduction() {
		return nil, domain.ErrForbidden
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

func (u *TokenUsecase) AwardAdMobSSV(ctx context.Context, rawQuery string) error {
	if u.adMobVerifier == nil {
		return domain.ErrInvalidInput
	}
	callback, err := u.adMobVerifier.Verify(ctx, rawQuery, u.now().UTC())
	if err != nil {
		return err
	}
	userID, err := userIDFromAdMobCallback(callback)
	if err != nil {
		return err
	}
	verifiedAt := u.now().UTC()
	event := domain.AdMobSSVEvent{
		TransactionID: strings.TrimSpace(callback.TransactionID),
		UserID:        userID,
		AdUnit:        strings.TrimSpace(callback.AdUnit),
		RewardAmount:  callback.RewardAmount,
		RewardItem:    strings.TrimSpace(callback.RewardItem),
		RawQueryHash:  hashAdMobRawQuery(rawQuery),
		VerifiedAt:    verifiedAt,
	}
	_, err = u.budgetRepo.AwardAdMobSSVTokens(ctx, event, verifiedAt)
	return err
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

func userIDFromAdMobCallback(callback *domain.AdMobSSVCallback) (uuid.UUID, error) {
	if callback == nil {
		return uuid.Nil, domain.ErrInvalidInput
	}
	rawUserID := strings.TrimSpace(callback.UserID)
	rawCustomData := strings.TrimSpace(callback.CustomData)
	if rawUserID == "" && rawCustomData == "" {
		return uuid.Nil, domain.ErrInvalidInput
	}

	var parsed uuid.UUID
	if rawUserID != "" {
		userID, err := uuid.Parse(rawUserID)
		if err != nil {
			return uuid.Nil, domain.ErrForbidden
		}
		parsed = userID
	}
	if rawCustomData != "" {
		customUserID, err := uuid.Parse(rawCustomData)
		if err != nil {
			return uuid.Nil, domain.ErrForbidden
		}
		if parsed != uuid.Nil && parsed != customUserID {
			// AdMob can carry the app user through user_id or custom_data. If
			// both are present, they must refer to the same account.
			slog.Warn("admob_ssv_user_mismatch",
				"reason", "user_id_custom_data_mismatch",
				"user_id", parsed.String(),
				"custom_data_user_id", customUserID.String(),
			)
			return uuid.Nil, domain.ErrForbidden
		}
		parsed = customUserID
	}
	if parsed == uuid.Nil {
		return uuid.Nil, domain.ErrInvalidInput
	}
	return parsed, nil
}

func hashAdMobRawQuery(rawQuery string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(rawQuery)))
	return hex.EncodeToString(sum[:])
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
