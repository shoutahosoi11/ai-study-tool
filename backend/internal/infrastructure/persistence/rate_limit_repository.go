package persistence

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/shout/ai-study-tool/backend/internal/domain"
	"github.com/shout/ai-study-tool/backend/internal/repository/sqlcgen"
)

type rateLimitRepository struct {
	db      *sql.DB
	queries *sqlcgen.Queries
}

func NewRateLimitRepository(db *sql.DB) domain.RateLimitRepository {
	return &rateLimitRepository{
		db:      db,
		queries: sqlcgen.New(db),
	}
}

func (r *rateLimitRepository) IncrementAndCheck(ctx context.Context, userID, bucket string, limit int64) (int64, bool, error) {
	current, err := r.queries.IncrementRateLimitCounter(ctx, sqlcgen.IncrementRateLimitCounterParams{
		UserID: userID,
		Bucket: bucket,
	})
	if err != nil {
		return 0, false, fmt.Errorf("rate limit repo: increment counter: %w", err)
	}

	return current, current > limit, nil
}

func (r *rateLimitRepository) Count(ctx context.Context, userID, bucket string) (int64, error) {
	var count int64
	err := r.db.QueryRowContext(ctx, `
SELECT count
FROM rate_limit_counters
WHERE user_id = $1
  AND bucket = $2
  AND period = CURRENT_DATE
`, userID, bucket).Scan(&count)
	if err == nil {
		return count, nil
	}
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return 0, fmt.Errorf("rate limit repo: count counter: %w", err)
}
