package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type AdMobSSVCallback struct {
	TransactionID string
	UserID        string
	CustomData    string
	AdUnit        string
	RewardAmount  int
	RewardItem    string
	Timestamp     time.Time
}

type AdMobSSVEvent struct {
	TransactionID string
	UserID        uuid.UUID
	AdUnit        string
	RewardAmount  int
	RewardItem    string
	RawQueryHash  string
	VerifiedAt    time.Time
}

type AdMobSSVVerifier interface {
	Verify(ctx context.Context, rawQuery string, now time.Time) (*AdMobSSVCallback, error)
}
