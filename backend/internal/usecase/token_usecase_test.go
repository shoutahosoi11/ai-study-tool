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
	claim domain.AdRewardClaim
}

func (s *stubQuestionBudgetRepository) GetBalance(ctx context.Context, userID uuid.UUID, plan string, now time.Time) (*domain.QuestionTokenBalance, error) {
	return &domain.QuestionTokenBalance{Plan: plan}, nil
}

func (s *stubQuestionBudgetRepository) AwardAdTokens(ctx context.Context, userID uuid.UUID, claim domain.AdRewardClaim, now time.Time) (*domain.QuestionTokenBalance, error) {
	s.claim = claim
	return &domain.QuestionTokenBalance{AvailableTokens: domain.AdTokensPerView}, nil
}

func (s *stubQuestionBudgetRepository) ReserveQuestions(ctx context.Context, userID uuid.UUID, plan string, questionCount int, now time.Time) (*domain.QuestionTokenBalance, error) {
	return &domain.QuestionTokenBalance{Plan: plan}, nil
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
