package usecase

import (
	"context"
	"time"

	"github.com/shout/ai-study-tool/backend/internal/domain"
)

type TokenUsecase struct {
	budgetRepo domain.QuestionBudgetRepository
	now        func() time.Time
}

func NewTokenUsecase(budgetRepo domain.QuestionBudgetRepository) *TokenUsecase {
	return &TokenUsecase{budgetRepo: budgetRepo, now: time.Now}
}

func (u *TokenUsecase) Award(ctx context.Context, user *domain.User) (*domain.QuestionTokenBalance, error) {
	balance, err := u.budgetRepo.AwardAdTokens(ctx, user.ID, u.now())
	if err != nil {
		return nil, err
	}
	balance.Plan = user.Plan
	return balance, nil
}

func (u *TokenUsecase) Balance(ctx context.Context, user *domain.User) (*domain.QuestionTokenBalance, error) {
	return u.budgetRepo.GetBalance(ctx, user.ID, user.Plan, u.now())
}
