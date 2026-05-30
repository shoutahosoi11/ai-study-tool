package usecase

import (
	"context"
	"encoding/hex"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

type stubQuestionBudgetRepository struct {
	claim      domain.AdRewardClaim
	adMobEvent domain.AdMobSSVEvent
	adMobErr   error
}

func (s *stubQuestionBudgetRepository) GetBalance(ctx context.Context, userID uuid.UUID, plan string, now time.Time) (*domain.QuestionTokenBalance, error) {
	return &domain.QuestionTokenBalance{Plan: plan}, nil
}

func (s *stubQuestionBudgetRepository) AwardAdTokens(ctx context.Context, userID uuid.UUID, claim domain.AdRewardClaim, now time.Time) (*domain.QuestionTokenBalance, error) {
	s.claim = claim
	return &domain.QuestionTokenBalance{AvailableTokens: domain.AdTokensPerView}, nil
}

func (s *stubQuestionBudgetRepository) AwardAdMobSSVTokens(ctx context.Context, event domain.AdMobSSVEvent, now time.Time) (*domain.QuestionTokenBalance, error) {
	if s.adMobErr != nil {
		return nil, s.adMobErr
	}
	s.adMobEvent = event
	return &domain.QuestionTokenBalance{AvailableTokens: domain.AdTokensPerView}, nil
}

func (s *stubQuestionBudgetRepository) ReserveQuestions(ctx context.Context, userID uuid.UUID, plan string, questionCount int, now time.Time) (*domain.QuestionTokenBalance, error) {
	return &domain.QuestionTokenBalance{Plan: plan}, nil
}

type stubAdMobSSVVerifier struct {
	callback *domain.AdMobSSVCallback
	err      error
}

func (s stubAdMobSSVVerifier) Verify(ctx context.Context, rawQuery string, now time.Time) (*domain.AdMobSSVCallback, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.callback, nil
}

func TestTokenUsecaseRejectsUnsignedAdReward(t *testing.T) {
	repo := &stubQuestionBudgetRepository{}
	uc := NewTokenUsecaseWithAdRewardSecret(repo, "secret")
	now := time.Date(2026, 5, 23, 12, 0, 0, 0, time.UTC)
	uc.now = func() time.Time { return now }

	user := &domain.User{ID: uuid.New(), Plan: "free"}
	_, err := uc.Award(context.Background(), user, AwardAdTokensInput{
		Provider:   "test",
		Nonce:      "nonce_1234567890",
		RewardedAt: now,
		Signature:  "bad-signature",
	})
	if err == nil {
		t.Fatal("expected invalid signature to be rejected")
	}
	if repo.claim.Nonce != "" {
		t.Fatalf("budget repository should not be called, got claim %#v", repo.claim)
	}
}

func TestTokenUsecaseAcceptsSignedAdReward(t *testing.T) {
	repo := &stubQuestionBudgetRepository{}
	uc := NewTokenUsecaseWithAdRewardSecret(repo, "secret")
	now := time.Date(2026, 5, 23, 12, 0, 0, 0, time.UTC)
	uc.now = func() time.Time { return now }

	user := &domain.User{ID: uuid.New(), Plan: "free"}
	claim := domain.AdRewardClaim{
		Provider:   "test",
		Nonce:      "nonce_1234567890",
		RewardedAt: now,
	}
	signature := signAdRewardClaim("secret", user.ID.String(), claim)

	balance, err := uc.Award(context.Background(), user, AwardAdTokensInput{
		Provider:   claim.Provider,
		Nonce:      claim.Nonce,
		RewardedAt: claim.RewardedAt,
		Signature:  hex.EncodeToString(signature),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if balance.AvailableTokens != domain.AdTokensPerView {
		t.Fatalf("unexpected balance: %#v", balance)
	}
	if repo.claim.Nonce != claim.Nonce {
		t.Fatalf("expected claim to be persisted, got %#v", repo.claim)
	}
}

func TestTokenUsecaseRejectsInvalidNonceCharacters(t *testing.T) {
	repo := &stubQuestionBudgetRepository{}
	uc := NewTokenUsecaseWithAdRewardSecret(repo, "secret")
	now := time.Date(2026, 5, 23, 12, 0, 0, 0, time.UTC)
	uc.now = func() time.Time { return now }

	user := &domain.User{ID: uuid.New(), Plan: "free"}
	claim := domain.AdRewardClaim{
		Provider:   "test",
		Nonce:      "nonce|with|pipe123",
		RewardedAt: now,
	}
	signature := signAdRewardClaim("secret", user.ID.String(), claim)

	_, err := uc.Award(context.Background(), user, AwardAdTokensInput{
		Provider:   claim.Provider,
		Nonce:      claim.Nonce,
		RewardedAt: claim.RewardedAt,
		Signature:  hex.EncodeToString(signature),
	})
	if err == nil {
		t.Fatal("expected invalid nonce to be rejected")
	}
}

func TestTokenUsecaseRejectsClientAdRewardInProduction(t *testing.T) {
	repo := &stubQuestionBudgetRepository{}
	uc := NewTokenUsecaseWithAdRewardSecretAndEnv(repo, nil, "secret", "production")
	now := time.Date(2026, 5, 23, 12, 0, 0, 0, time.UTC)
	uc.now = func() time.Time { return now }

	_, err := uc.Award(context.Background(), &domain.User{ID: uuid.New(), Plan: "free"}, AwardAdTokensInput{
		Provider:   "test",
		Nonce:      "nonce_1234567890",
		RewardedAt: now,
		Signature:  "client-notification",
	})
	if err == nil {
		t.Fatal("expected production client reward endpoint to be rejected")
	}
	if repo.claim.Nonce != "" {
		t.Fatal("client notification must not award tokens")
	}
}

func TestTokenUsecaseAwardsAdMobSSV(t *testing.T) {
	userID := uuid.New()
	repo := &stubQuestionBudgetRepository{}
	uc := NewTokenUsecaseWithAdRewardSecretAndEnv(repo, stubAdMobSSVVerifier{callback: &domain.AdMobSSVCallback{
		TransactionID: "txn_1",
		UserID:        userID.String(),
		AdUnit:        "ad-unit",
		RewardAmount:  1,
		RewardItem:    "token",
		Timestamp:     time.Date(2026, 5, 23, 12, 0, 0, 0, time.UTC),
	}}, "", "production")
	uc.now = func() time.Time { return time.Date(2026, 5, 23, 12, 0, 0, 0, time.UTC) }

	if err := uc.AwardAdMobSSV(context.Background(), "ad_unit=ad-unit&transaction_id=txn_1"); err != nil {
		t.Fatalf("AwardAdMobSSV failed: %v", err)
	}
	if repo.adMobEvent.TransactionID != "txn_1" || repo.adMobEvent.UserID != userID {
		t.Fatalf("unexpected admob event: %#v", repo.adMobEvent)
	}
}

func TestTokenUsecaseRejectsAdMobUserMismatch(t *testing.T) {
	userID := uuid.New()
	otherID := uuid.New()
	uc := NewTokenUsecaseWithAdRewardSecretAndEnv(&stubQuestionBudgetRepository{}, stubAdMobSSVVerifier{callback: &domain.AdMobSSVCallback{
		TransactionID: "txn_1",
		UserID:        userID.String(),
		CustomData:    otherID.String(),
		AdUnit:        "ad-unit",
		RewardAmount:  1,
		RewardItem:    "token",
		Timestamp:     time.Date(2026, 5, 23, 12, 0, 0, 0, time.UTC),
	}}, "", "production")

	if err := uc.AwardAdMobSSV(context.Background(), "raw-query"); err == nil {
		t.Fatal("expected user mismatch to be rejected")
	}
}

func TestTokenUsecaseDoesNotDoubleAwardDuplicateAdMobTransaction(t *testing.T) {
	userID := uuid.New()
	repo := &stubQuestionBudgetRepository{adMobErr: domain.ErrAlreadyExists}
	uc := NewTokenUsecaseWithAdRewardSecretAndEnv(repo, stubAdMobSSVVerifier{callback: &domain.AdMobSSVCallback{
		TransactionID: "txn_1",
		UserID:        userID.String(),
		AdUnit:        "ad-unit",
		RewardAmount:  1,
		RewardItem:    "token",
		Timestamp:     time.Date(2026, 5, 23, 12, 0, 0, 0, time.UTC),
	}}, "", "production")

	if err := uc.AwardAdMobSSV(context.Background(), "raw-query"); err == nil {
		t.Fatal("expected duplicate transaction to be rejected")
	}
	if repo.adMobEvent.TransactionID != "" {
		t.Fatal("duplicate transaction should not be recorded as awarded in stub")
	}
}
