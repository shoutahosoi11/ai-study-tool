package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

type ManualGenerationUsecase struct {
	jobRepo       domain.QuestionGenerationJobRepository
	highlightRepo manualGenerationHighlightReader
	budgetRepo    domain.QuestionBudgetRepository
	taskEnqueuer  domain.QuestionGenerationTaskEnqueuer
	now           func() time.Time
	queueLimits   questionGenerationQueueLimits
}

type manualGenerationHighlightReader interface {
	ListByIDs(ctx context.Context, userID uuid.UUID, highlightIDs []uuid.UUID) ([]*domain.Highlight, error)
}

func NewManualGenerationUsecase(
	jobRepo domain.QuestionGenerationJobRepository,
	highlightRepo manualGenerationHighlightReader,
	budgetRepo domain.QuestionBudgetRepository,
	taskEnqueuer domain.QuestionGenerationTaskEnqueuer,
) *ManualGenerationUsecase {
	return &ManualGenerationUsecase{
		jobRepo:       jobRepo,
		highlightRepo: highlightRepo,
		budgetRepo:    budgetRepo,
		taskEnqueuer:  taskEnqueuer,
		now:           time.Now,
		queueLimits:   questionGenerationQueueLimitsFromEnv(""),
	}
}

func (u *ManualGenerationUsecase) Generate(ctx context.Context, user *domain.User, bookKey string, highlightIDs []uuid.UUID) (*domain.QuestionGenerationJob, error) {
	uniqueHighlightIDs := uniqueManualHighlightIDs(highlightIDs)
	if user == nil || u.highlightRepo == nil || strings.TrimSpace(bookKey) == "" || len(uniqueHighlightIDs) < 5 {
		return nil, domain.ErrInvalidInput
	}
	if len(uniqueHighlightIDs) > domain.MaxHighlightsPerJob {
		return nil, domain.ErrInvalidInput
	}

	ownedHighlights, err := u.highlightRepo.ListByIDs(ctx, user.ID, uniqueHighlightIDs)
	if err != nil {
		return nil, fmt.Errorf("manual generation usecase: list selected highlights: %w", err)
	}
	if len(ownedHighlights) != len(uniqueHighlightIDs) {
		return nil, domain.ErrInvalidInput
	}
	for _, highlight := range ownedHighlights {
		if highlight == nil || strings.TrimSpace(highlight.Content) == "" {
			return nil, domain.ErrInvalidInput
		}
	}

	if err := ensureQuestionJobQueueDepth(ctx, u.jobRepo, u.queueLimits, user.ID, bookKey); err != nil {
		return nil, err
	}

	job, err := u.jobRepo.Create(ctx, domain.CreateQuestionGenerationJobInput{
		UserID:       user.ID,
		BookKey:      bookKey,
		Reason:       domain.JobReasonManualSelection,
		HighlightIDs: uniqueHighlightIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("manual generation usecase: create job: %w", err)
	}

	if _, err := u.budgetRepo.ReserveQuestions(ctx, user.ID, user.Plan, len(uniqueHighlightIDs), u.now()); err != nil {
		if _, markErr := u.jobRepo.RecordFailure(ctx, job.ID, user.ID, err.Error(), 1); markErr != nil {
			return nil, fmt.Errorf("manual generation usecase: mark budget failure: %w", markErr)
		}
		return nil, err
	}

	if u.taskEnqueuer == nil {
		return job, nil
	}
	if err := u.taskEnqueuer.EnqueueQuestionGeneration(ctx, job.ID, user.ID); err != nil {
		slog.Error("question_generation_event=enqueue_failed",
			"source", "manual",
			"job_id", job.ID.String(),
			"user_id", user.ID.String(),
			"error", err.Error(),
		)
		if markErr := u.jobRepo.MarkEnqueueFailed(ctx, job.ID, user.ID, err.Error()); markErr != nil {
			return nil, fmt.Errorf("manual generation usecase: mark enqueue failed: %w", markErr)
		}
	}

	return job, nil
}

func uniqueManualHighlightIDs(highlightIDs []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(highlightIDs))
	uniqueIDs := make([]uuid.UUID, 0, len(highlightIDs))
	for _, highlightID := range highlightIDs {
		if highlightID == uuid.Nil {
			continue
		}
		if _, ok := seen[highlightID]; ok {
			continue
		}
		seen[highlightID] = struct{}{}
		uniqueIDs = append(uniqueIDs, highlightID)
	}
	return uniqueIDs
}
