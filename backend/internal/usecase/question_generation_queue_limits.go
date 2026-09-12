package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/google/uuid"
	appconfig "github.com/shout/ai-study-tool/backend/internal/config"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

const (
	defaultQuestionJobMaxPendingPerUser = 20
	defaultQuestionJobMaxPendingPerBook = 3
	defaultQuestionJobMaxPendingGlobal  = 1000
)

type questionGenerationQueueLimits struct {
	MaxPendingPerUser int
	MaxPendingPerBook int
	MaxPendingGlobal  int
}

func questionGenerationQueueLimitsFromEnv(appEnv string) questionGenerationQueueLimits {
	if strings.TrimSpace(appEnv) == "" {
		appEnv = os.Getenv("APP_ENV")
	}
	limits := questionGenerationQueueLimits{
		MaxPendingPerUser: readEnvIntOrDefault("QUESTION_JOB_MAX_PENDING_PER_USER", defaultQuestionJobMaxPendingPerUser),
		MaxPendingPerBook: readEnvIntOrDefault("QUESTION_JOB_MAX_PENDING_PER_BOOK", defaultQuestionJobMaxPendingPerBook),
		MaxPendingGlobal:  readEnvIntOrDefault("QUESTION_JOB_MAX_PENDING_GLOBAL", defaultQuestionJobMaxPendingGlobal),
	}
	if appconfig.NormalizeAppEnv(appEnv).IsProduction() {
		warnDangerousQuestionJobQueueLimits(limits)
	}
	return limits
}

func warnDangerousQuestionJobQueueLimits(limits questionGenerationQueueLimits) {
	if limits.MaxPendingPerUser > defaultQuestionJobMaxPendingPerUser*100 {
		slog.Warn("question_job_queue_limit_high", "key", "QUESTION_JOB_MAX_PENDING_PER_USER", "value", limits.MaxPendingPerUser)
	}
	if limits.MaxPendingPerBook > defaultQuestionJobMaxPendingPerBook*100 {
		slog.Warn("question_job_queue_limit_high", "key", "QUESTION_JOB_MAX_PENDING_PER_BOOK", "value", limits.MaxPendingPerBook)
	}
	if limits.MaxPendingGlobal > defaultQuestionJobMaxPendingGlobal*100 {
		slog.Warn("question_job_queue_limit_high", "key", "QUESTION_JOB_MAX_PENDING_GLOBAL", "value", limits.MaxPendingGlobal)
	}
}

// questionJobQueueDepthCounters carries the user/global pending counts that do
// not vary per book, so a sync pass over N candidate books issues them once
// instead of N times. Jobs created during the pass are added via record().
type questionJobQueueDepthCounters struct {
	userPending   int
	globalPending int
	loaded        bool
}

func (c *questionJobQueueDepthCounters) load(ctx context.Context, repo domain.QuestionGenerationJobRepository, userID uuid.UUID) error {
	if c.loaded {
		return nil
	}
	userPending, err := repo.CountPendingByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("question job queue depth: count user pending: %w", err)
	}
	globalPending, err := repo.CountPending(ctx)
	if err != nil {
		return fmt.Errorf("question job queue depth: count global pending: %w", err)
	}
	c.userPending = userPending
	c.globalPending = globalPending
	c.loaded = true
	return nil
}

func (c *questionJobQueueDepthCounters) record() {
	c.userPending++
	c.globalPending++
}

func ensureQuestionJobQueueDepth(ctx context.Context, repo domain.QuestionGenerationJobRepository, limits questionGenerationQueueLimits, counters *questionJobQueueDepthCounters, userID uuid.UUID, bookKey string) error {
	if repo == nil {
		return nil
	}

	if counters == nil {
		counters = &questionJobQueueDepthCounters{}
	}
	if err := counters.load(ctx, repo, userID); err != nil {
		return err
	}
	if counters.userPending >= limits.MaxPendingPerUser {
		return domain.ErrQuestionQueueDepthExceeded
	}
	if counters.globalPending >= limits.MaxPendingGlobal {
		return domain.ErrQuestionQueueDepthExceeded
	}

	bookPending, err := repo.CountPendingByBookKey(ctx, userID, bookKey)
	if err != nil {
		return fmt.Errorf("question job queue depth: count book pending: %w", err)
	}
	if bookPending >= limits.MaxPendingPerBook {
		return domain.ErrQuestionQueueDepthExceeded
	}

	return nil
}
