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

func ensureQuestionJobQueueDepth(ctx context.Context, repo domain.QuestionGenerationJobRepository, limits questionGenerationQueueLimits, userID uuid.UUID, bookKey string) error {
	if repo == nil {
		return nil
	}

	userPending, err := repo.CountPendingByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("question job queue depth: count user pending: %w", err)
	}
	if userPending >= limits.MaxPendingPerUser {
		return domain.ErrQuestionQueueDepthExceeded
	}

	bookPending, err := repo.CountPendingByBookKey(ctx, userID, bookKey)
	if err != nil {
		return fmt.Errorf("question job queue depth: count book pending: %w", err)
	}
	if bookPending >= limits.MaxPendingPerBook {
		return domain.ErrQuestionQueueDepthExceeded
	}

	globalPending, err := repo.CountPending(ctx)
	if err != nil {
		return fmt.Errorf("question job queue depth: count global pending: %w", err)
	}
	if globalPending >= limits.MaxPendingGlobal {
		return domain.ErrQuestionQueueDepthExceeded
	}

	return nil
}
