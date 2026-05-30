package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

const (
	FreeDailyQuestionLimit = 10
	AdTokensPerView        = 3
	MaxAdViewsPerDay       = 3
)

type QuestionTokenBalance struct {
	AvailableTokens int
	FreeUsedToday   int
	FreeLimit       int
	AdViewsToday    int
	AdViewsLimit    int
	Plan            string
}

type AdRewardClaim struct {
	Provider   string
	Nonce      string
	RewardedAt time.Time
}

type QuestionBudgetRepository interface {
	GetBalance(ctx context.Context, userID uuid.UUID, plan string, now time.Time) (*QuestionTokenBalance, error)
	AwardAdTokens(ctx context.Context, userID uuid.UUID, claim AdRewardClaim, now time.Time) (*QuestionTokenBalance, error)
	AwardAdMobSSVTokens(ctx context.Context, event AdMobSSVEvent, now time.Time) (*QuestionTokenBalance, error)
	ReserveQuestions(ctx context.Context, userID uuid.UUID, plan string, questionCount int, now time.Time) (*QuestionTokenBalance, error)
}
