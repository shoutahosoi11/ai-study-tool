package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

const (
	defaultWorkerRequestInterval = time.Second
)

type QuestionWorkerUsecase struct {
	highlightRepo   domain.HighlightGenerationLifecycle
	questionRepo    domain.QuestionWorkerRepository
	jobRepo         domain.QuestionGenerationJobRepository
	llmClient       domain.LLMClient
	requestInterval time.Duration
	now             func() time.Time
	rateLimitMu     sync.Mutex
	lastRequestAt   time.Time
}

func NewQuestionWorkerUsecaseWithJobRepository(
	highlightRepo domain.HighlightGenerationLifecycle,
	questionRepo domain.QuestionWorkerRepository,
	jobRepo domain.QuestionGenerationJobRepository,
	llmClient domain.LLMClient,
) *QuestionWorkerUsecase {
	return &QuestionWorkerUsecase{
		highlightRepo:   highlightRepo,
		questionRepo:    questionRepo,
		jobRepo:         jobRepo,
		llmClient:       llmClient,
		requestInterval: readEnvDurationMSOrDefault("QUESTION_WORKER_REQUEST_INTERVAL_MS", defaultWorkerRequestInterval),
		now:             time.Now,
	}
}

func (u *QuestionWorkerUsecase) ProcessQuestionGenerationJob(ctx context.Context, jobID uuid.UUID, userID uuid.UUID) error {
	if u.jobRepo == nil {
		return fmt.Errorf("question worker: job repository is not configured")
	}

	job, claimed, err := u.jobRepo.ClaimQueued(ctx, jobID, userID)
	if err != nil {
		return fmt.Errorf("question worker: claim generation job: %w", err)
	}
	if !claimed || job == nil {
		return nil
	}

	logQuestionWorkerEvent("job_processing_started", map[string]any{
		"user_id": userID.String(),
		"job_id":  jobID.String(),
	})

	highlightIDs := slices.Clone(job.HighlightIDs)
	if len(highlightIDs) == 0 {
		return u.jobRepo.MarkCompleted(ctx, job.ID, job.UserID)
	}

	highlights, err := u.highlightRepo.ListByIDs(ctx, userID, highlightIDs)
	if err != nil {
		return u.recordJobFailure(ctx, job, fmt.Errorf("question worker: list job highlights: %w", err))
	}
	if len(highlights) == 0 {
		return u.jobRepo.MarkCompleted(ctx, job.ID, job.UserID)
	}

	model := u.llmClient.ModelForPlan("free")
	customInstruction := "各ハイライトから選択式問題を1問ずつ作成してください。"
	materials := make([]domain.ExtractedPoint, 0, len(highlights))
	materialHighlights := make([]*domain.Highlight, 0, len(highlights))
	for _, highlight := range highlights {
		if highlight == nil || strings.TrimSpace(highlight.Content) == "" {
			continue
		}
		materialHighlights = append(materialHighlights, highlight)
		materials = append(materials, domain.ExtractedPoint{
			Point:   highlight.Content,
			Context: buildPerspectiveContext(highlight, "definition"),
		})
	}
	if len(materials) == 0 {
		return u.recordJobFailure(ctx, job, domain.ErrSourceTextUnavailable)
	}

	quotaDay := questionSyncDay(u.now())
	reserved, err := u.questionRepo.ReserveDailyGeneratedCount(ctx, userID, quotaDay, len(materials), readEnvIntOrDefault("QUESTION_SYNC_DAILY_LIMIT", defaultQuestionSyncDailyLimit))
	if err != nil {
		return fmt.Errorf("question worker: reserve generation quota: %w", err)
	}
	if !reserved {
		if err := u.jobRepo.MarkQueued(ctx, job.ID, job.UserID); err != nil {
			return fmt.Errorf("question worker: return quota-limited job to queued: %w", err)
		}
		logQuestionWorkerEvent("job_quota_skipped", map[string]any{
			"user_id": userID.String(),
			"job_id":  jobID.String(),
		})
		return nil
	}
	releaseReservedQuota := func(cause error) error {
		if releaseErr := u.questionRepo.ReleaseDailyGeneratedCount(ctx, userID, quotaDay, len(materials)); releaseErr != nil {
			if cause == nil {
				return fmt.Errorf("question worker: release generation quota: %w", releaseErr)
			}
			return errors.Join(cause, fmt.Errorf("question worker: release generation quota: %w", releaseErr))
		}
		return cause
	}

	if err := u.waitForRateLimit(ctx); err != nil {
		return releaseReservedQuota(u.recordJobFailure(ctx, job, err))
	}

	generationID, err := u.questionRepo.SaveGeneration(ctx, userID.String(), "question_generation_job", job.ID.String(), customInstruction, model)
	if err != nil {
		return releaseReservedQuota(u.recordJobFailure(ctx, job, fmt.Errorf("question worker: save generation: %w", err)))
	}

	generatedQuestions, err := u.llmClient.GenerateQuestions(ctx, materials, domain.QuestionTypeMultipleChoice, customInstruction, model)
	if err != nil {
		return releaseReservedQuota(u.recordJobFailure(ctx, job, fmt.Errorf("question worker: generate questions: %w", err)))
	}
	if len(generatedQuestions) < len(materials) {
		return releaseReservedQuota(u.recordJobFailure(ctx, job, fmt.Errorf("question worker: generated questions count mismatch: expected %d, got %d", len(materials), len(generatedQuestions))))
	}

	replacements := make([]domain.QuestionReplacement, 0, len(materialHighlights))
	for index, highlight := range materialHighlights {
		generated, err := normalizeGeneratedQuestion(generatedQuestions[index], domain.QuestionTypeMultipleChoice)
		if err != nil {
			return releaseReservedQuota(u.recordJobFailure(ctx, job, fmt.Errorf("question worker: invalid generated question: %w", err)))
		}
		question := &domain.Question{
			ID:            uuid.NewString(),
			QuestionType:  domain.QuestionTypeMultipleChoice,
			Content:       generated.Content,
			Options:       generated.Options,
			CorrectAnswer: generated.CorrectAnswer,
			Explanation:   generated.Explanation,
		}
		meta := &domain.QuestionMeta{
			QuestionID:    question.ID,
			CreatorID:     userID.String(),
			SourceType:    domain.SourceTypeKindleBook,
			HighlightID:   highlight.ID.String(),
			GenerationID:  generationID,
			Perspective:   "definition",
			Version:       1,
			IsAIGenerated: true,
		}
		replacements = append(replacements, domain.QuestionReplacement{
			HighlightID: highlight.ID,
			Question:    question,
			Meta:        meta,
		})
	}

	if err := u.questionRepo.CompleteQuestionGenerationJob(ctx, userID, job.ID, replacements, highlightIDs); err != nil {
		return releaseReservedQuota(u.recordJobFailure(ctx, job, fmt.Errorf("question worker: complete generation job: %w", err)))
	}

	logQuestionWorkerEvent("job_completed", map[string]any{
		"user_id":         userID.String(),
		"job_id":          jobID.String(),
		"highlight_count": len(highlights),
	})
	return nil
}

func (u *QuestionWorkerUsecase) recordJobFailure(ctx context.Context, job *domain.QuestionGenerationJob, err error) error {
	if job == nil || err == nil {
		return err
	}
	updated, recordErr := u.jobRepo.RecordFailure(ctx, job.ID, job.UserID, err.Error(), domain.JobMaxRetryCount)
	if recordErr != nil {
		return errors.Join(err, fmt.Errorf("question worker: record job failure: %w", recordErr))
	}
	if updated != nil && updated.Status == domain.JobStatusFailed {
		if markErr := u.highlightRepo.MarkGenerationFailed(ctx, job.UserID, job.HighlightIDs, err.Error(), domain.JobMaxRetryCount); markErr != nil {
			slog.Error("question_worker_event=mark_failed_job_highlights_failed", "job_id", job.ID.String(), "user_id", job.UserID.String(), "error", markErr)
		}
	}
	logQuestionWorkerEvent("job_failed", map[string]any{
		"user_id": job.UserID.String(),
		"job_id":  job.ID.String(),
		"error":   err.Error(),
	})
	return err
}

func buildPerspectiveContext(highlight *domain.Highlight, perspective string) string {
	parts := make([]string, 0, 2)
	if highlight != nil && highlight.Explanation != nil && strings.TrimSpace(*highlight.Explanation) != "" {
		parts = append(parts, strings.TrimSpace(*highlight.Explanation))
	}
	parts = append(parts, fmt.Sprintf("観点: %s", perspective))
	return strings.Join(parts, "\n")
}

func (u *QuestionWorkerUsecase) waitForRateLimit(ctx context.Context) error {
	if u.requestInterval <= 0 {
		return nil
	}

	u.rateLimitMu.Lock()
	defer u.rateLimitMu.Unlock()

	if !u.lastRequestAt.IsZero() {
		wait := u.lastRequestAt.Add(u.requestInterval).Sub(u.now())
		if wait > 0 {
			timer := time.NewTimer(wait)
			defer timer.Stop()

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
			}
		}
	}

	u.lastRequestAt = u.now()
	return nil
}

func logQuestionWorkerEvent(event string, fields map[string]any) {
	if fields == nil {
		fields = make(map[string]any)
	}
	args := make([]any, 0, (len(fields)+1)*2)
	args = append(args, "event", event)
	for key, value := range fields {
		args = append(args, key, value)
	}
	slog.Info("question_worker_event", args...)
}

func readEnvIntOrDefault(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		return fallback
	}

	return parsed
}

func readEnvDurationMSOrDefault(key string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		return fallback
	}

	return time.Duration(parsed) * time.Millisecond
}
