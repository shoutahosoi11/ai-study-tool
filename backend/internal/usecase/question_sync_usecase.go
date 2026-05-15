package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

const (
	defaultQuestionSyncDailyLimit   = 100
	defaultQuestionSyncStaleTimeout = 10 * time.Minute
	defaultQuestionSyncJobListLimit = 20
)

type QuestionSyncUsecase struct {
	highlightRepo domain.QuestionSyncHighlightRepository
	questionRepo  domain.QuestionSyncQuestionRepository
	jobRepo       domain.QuestionGenerationJobRepository
	taskEnqueuer  domain.QuestionGenerationTaskEnqueuer
	now           func() time.Time
	dailyLimit    int
}

type QuestionStockBook struct {
	BookKey    string
	BookTitle  string
	BookAuthor string
	Stock      int
	Target     int
	Preparing  int
}

type SyncQuestionStockResult struct {
	Books                  []QuestionStockBook
	QueuedCount            int
	SkippedDueToDailyLimit bool
}

func NewQuestionSyncUsecase(
	highlightRepo domain.QuestionSyncHighlightRepository,
	questionRepo domain.QuestionSyncQuestionRepository,
	jobRepo domain.QuestionGenerationJobRepository,
	taskEnqueuer domain.QuestionGenerationTaskEnqueuer,
) *QuestionSyncUsecase {
	return &QuestionSyncUsecase{
		highlightRepo: highlightRepo,
		questionRepo:  questionRepo,
		jobRepo:       jobRepo,
		taskEnqueuer:  taskEnqueuer,
		now:           time.Now,
		dailyLimit:    readEnvIntOrDefault("QUESTION_SYNC_DAILY_LIMIT", defaultQuestionSyncDailyLimit),
	}
}

func (u *QuestionSyncUsecase) SyncQuestionStock(ctx context.Context, user *domain.User) (*SyncQuestionStockResult, error) {
	if user == nil {
		return nil, domain.ErrNotFound
	}

	result := &SyncQuestionStockResult{}
	if skipped, err := u.skipIfDailyLimitReached(ctx, user.ID); err != nil {
		return nil, err
	} else if skipped {
		result.SkippedDueToDailyLimit = true
		return result, nil
	}

	if err := u.reenqueueQueuedJobs(ctx, user.ID, result); err != nil {
		return nil, err
	}

	if err := u.reenqueueFailedJobs(ctx, user.ID, result); err != nil {
		return nil, err
	}

	lastSyncAt, err := u.questionRepo.GetUserLastQuestionSyncAt(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("question sync usecase: get last sync at: %w", err)
	}

	candidates, err := u.highlightRepo.ListQuestionGenerationCandidates(ctx, user.ID, lastSyncAt)
	if err != nil {
		return nil, fmt.Errorf("question sync usecase: list generation candidates: %w", err)
	}
	sortQuestionGenerationCandidates(candidates)

	result.Books = buildQuestionSyncBookResponse(candidates)
	for _, candidate := range candidates {
		created, err := u.createJobIfNeeded(ctx, user.ID, candidate)
		if err != nil {
			return nil, err
		}
		if created {
			result.QueuedCount++
		}
	}

	if err := u.questionRepo.UpdateUserLastQuestionSyncAt(ctx, user.ID, u.now().UTC()); err != nil {
		return nil, fmt.Errorf("question sync usecase: update last sync at: %w", err)
	}

	return result, nil
}

func (u *QuestionSyncUsecase) EvaluateBookAfterAnswer(ctx context.Context, user *domain.User, questionID string) error {
	if user == nil {
		return domain.ErrNotFound
	}

	parsedQuestionID, err := uuid.Parse(strings.TrimSpace(questionID))
	if err != nil {
		return fmt.Errorf("question sync usecase: parse answered question id: %w", err)
	}

	question, meta, _, err := u.questionRepo.FindByID(ctx, parsedQuestionID.String())
	if err != nil {
		return fmt.Errorf("question sync usecase: find answered question: %w", err)
	}
	if question == nil || meta == nil || strings.TrimSpace(meta.HighlightID) == "" {
		return nil
	}

	highlightID, err := uuid.Parse(strings.TrimSpace(meta.HighlightID))
	if err != nil {
		return fmt.Errorf("question sync usecase: parse answered highlight id: %w", err)
	}

	if err := u.questionRepo.SupersedeActiveQuestionsForHighlight(ctx, user.ID, highlightID); err != nil {
		return fmt.Errorf("question sync usecase: supersede active question: %w", err)
	}

	bookKey, err := u.highlightRepo.MarkHighlightPendingForQuestion(ctx, user.ID, parsedQuestionID)
	if err != nil {
		return fmt.Errorf("question sync usecase: mark answered highlight pending: %w", err)
	}
	if strings.TrimSpace(bookKey) == "" {
		slog.Warn("question_sync_event=empty_book_key_after_answer", "user_id", user.ID.String(), "question_id", parsedQuestionID.String(), "highlight_id", highlightID.String())
		return nil
	}

	candidate, err := u.highlightRepo.ListQuestionGenerationCandidateByBookKey(ctx, user.ID, bookKey)
	if err != nil {
		return fmt.Errorf("question sync usecase: list candidate after answer: %w", err)
	}
	if candidate == nil {
		return nil
	}
	_, err = u.createJobIfNeeded(ctx, user.ID, *candidate)
	return err
}

func (u *QuestionSyncUsecase) skipIfDailyLimitReached(ctx context.Context, userID uuid.UUID) (bool, error) {
	dailyCount, err := u.questionRepo.GetDailyGeneratedCount(ctx, userID, questionSyncDay(u.now()))
	if err != nil {
		return false, fmt.Errorf("question sync usecase: get daily generated count: %w", err)
	}
	return dailyCount >= u.dailyLimit, nil
}

func (u *QuestionSyncUsecase) reenqueueFailedJobs(ctx context.Context, userID uuid.UUID, result *SyncQuestionStockResult) error {
	if u.jobRepo == nil {
		return nil
	}

	jobs, err := u.jobRepo.ListEnqueueFailedByUserID(ctx, userID, defaultQuestionSyncJobListLimit)
	if err != nil {
		return fmt.Errorf("question sync usecase: list enqueue failed jobs: %w", err)
	}

	for _, job := range jobs {
		if job == nil {
			continue
		}
		if err := u.jobRepo.MarkQueued(ctx, job.ID, job.UserID); err != nil {
			return fmt.Errorf("question sync usecase: mark recovered job queued: %w", err)
		}
		if err := u.enqueueJob(ctx, job); err != nil {
			if markErr := u.jobRepo.MarkEnqueueFailed(ctx, job.ID, job.UserID, err.Error()); markErr != nil {
				return fmt.Errorf("question sync usecase: mark enqueue failed retry: %w", markErr)
			}
			continue
		}
		result.QueuedCount++
	}

	return nil
}

func (u *QuestionSyncUsecase) reenqueueQueuedJobs(ctx context.Context, userID uuid.UUID, result *SyncQuestionStockResult) error {
	if u.jobRepo == nil {
		return nil
	}

	jobs, err := u.jobRepo.ListQueuedByUserID(ctx, userID, defaultQuestionSyncJobListLimit)
	if err != nil {
		return fmt.Errorf("question sync usecase: list queued jobs: %w", err)
	}
	for _, job := range jobs {
		if job == nil {
			continue
		}
		if err := u.enqueueJob(ctx, job); err != nil {
			if markErr := u.jobRepo.MarkEnqueueFailed(ctx, job.ID, job.UserID, err.Error()); markErr != nil {
				return fmt.Errorf("question sync usecase: mark queued job enqueue failed: %w", markErr)
			}
			continue
		}
		result.QueuedCount++
	}
	return nil
}

func (u *QuestionSyncUsecase) createJobIfNeeded(ctx context.Context, userID uuid.UUID, candidate domain.QuestionGenerationBookCandidate) (bool, error) {
	reason, ok := questionGenerationReason(candidate)
	if !ok {
		return false, nil
	}

	pendingHighlights, err := u.highlightRepo.ListPendingHighlightsByBook(ctx, userID, candidate.BookKey, domain.MaxHighlightsPerJob)
	if err != nil {
		return false, fmt.Errorf("question sync usecase: list pending highlights by book: %w", err)
	}
	if len(pendingHighlights) == 0 {
		return false, nil
	}

	highlightIDs := make([]uuid.UUID, 0, len(pendingHighlights))
	for _, highlight := range pendingHighlights {
		if highlight != nil {
			highlightIDs = append(highlightIDs, highlight.ID)
		}
	}
	if len(highlightIDs) == 0 {
		return false, nil
	}

	job, err := u.jobRepo.Create(ctx, domain.CreateQuestionGenerationJobInput{
		UserID:       userID,
		BookKey:      candidate.BookKey,
		Reason:       reason,
		HighlightIDs: highlightIDs,
	})
	if err != nil {
		if errors.Is(err, domain.ErrAlreadyExists) {
			return false, nil
		}
		return false, fmt.Errorf("question sync usecase: create generation job: %w", err)
	}

	if err := u.highlightRepo.MarkHighlightsProcessing(ctx, userID, highlightIDs); err != nil {
		return false, fmt.Errorf("question sync usecase: mark highlights processing: %w", err)
	}

	if err := u.enqueueJob(ctx, job); err != nil {
		markErr := u.jobRepo.MarkEnqueueFailed(ctx, job.ID, job.UserID, err.Error())
		pendingErr := u.highlightRepo.MarkHighlightsPending(ctx, userID, highlightIDs)
		if markErr != nil {
			return false, fmt.Errorf("question sync usecase: mark enqueue failed: %w", markErr)
		}
		if pendingErr != nil {
			return false, fmt.Errorf("question sync usecase: mark highlights pending after enqueue failure: %w", pendingErr)
		}
		slog.Error("question_generation_event=enqueue_failed",
			"user_id", userID.String(),
			"job_id", job.ID.String(),
			"book_key", job.BookKey,
			"error", err.Error(),
		)
		return true, nil
	}

	slog.Info("question_generation_event=job_created",
		"user_id", userID.String(),
		"job_id", job.ID.String(),
		"book_key", job.BookKey,
		"highlight_count", len(highlightIDs),
		"reason", job.Reason,
	)
	return true, nil
}

func (u *QuestionSyncUsecase) enqueueJob(ctx context.Context, job *domain.QuestionGenerationJob) error {
	if job == nil {
		return domain.ErrInvalidInput
	}
	if u.taskEnqueuer == nil {
		return nil
	}
	return u.taskEnqueuer.EnqueueQuestionGeneration(ctx, job.ID, job.UserID)
}

func questionGenerationReason(candidate domain.QuestionGenerationBookCandidate) (domain.QuestionGenerationJobReason, bool) {
	pending := max(candidate.PendingHighlightCount, 0)
	unanswered := max(candidate.UnansweredQuestionCount, 0)
	switch {
	case pending >= domain.HighlightBatchThreshold:
		return domain.JobReasonHighlightBatchThreshold, true
	case pending >= domain.MinHighlightsForRefresh && pending < domain.HighlightBatchThreshold && unanswered == 0:
		return domain.JobReasonAllUnansweredConsumed, true
	default:
		return "", false
	}
}

func sortQuestionGenerationCandidates(candidates []domain.QuestionGenerationBookCandidate) {
	sort.SliceStable(candidates, func(left, right int) bool {
		if candidates[left].LatestHighlightAt.IsZero() {
			return false
		}
		if candidates[right].LatestHighlightAt.IsZero() {
			return true
		}
		return candidates[left].LatestHighlightAt.After(candidates[right].LatestHighlightAt)
	})
}

func buildQuestionSyncBookResponse(candidates []domain.QuestionGenerationBookCandidate) []QuestionStockBook {
	books := make([]QuestionStockBook, 0, len(candidates))
	for _, candidate := range candidates {
		books = append(books, QuestionStockBook{
			BookKey:    strings.TrimSpace(candidate.BookKey),
			BookTitle:  strings.TrimSpace(candidate.BookTitle),
			BookAuthor: strings.TrimSpace(candidate.BookAuthor),
			Stock:      max(candidate.UnansweredQuestionCount, 0),
			Target:     domain.HighlightBatchThreshold,
			Preparing:  max(candidate.PendingHighlightCount, 0),
		})
	}
	return books
}

func questionSyncDay(now time.Time) time.Time {
	utcNow := now.UTC()
	return time.Date(utcNow.Year(), utcNow.Month(), utcNow.Day(), 0, 0, 0, 0, time.UTC)
}
