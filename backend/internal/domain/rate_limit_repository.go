package domain

import "context"

type RateLimitRepository interface {
	IncrementAndCheck(ctx context.Context, userID, bucket string, limit int64) (current int64, exceeded bool, err error)
	Count(ctx context.Context, userID, bucket string) (int64, error)
}
