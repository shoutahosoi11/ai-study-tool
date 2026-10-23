package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type QuestionGenerationJobStatus string

const (
	JobStatusQueued        QuestionGenerationJobStatus = "queued"
	JobStatusProcessing    QuestionGenerationJobStatus = "processing"
	JobStatusCompleted     QuestionGenerationJobStatus = "completed"
	JobStatusFailed        QuestionGenerationJobStatus = "failed"
	JobStatusEnqueueFailed QuestionGenerationJobStatus = "enqueue_failed"
)

type QuestionGenerationJobReason string

const (
	JobReasonHighlightBatchThreshold QuestionGenerationJobReason = "highlight_batch_threshold"
	JobReasonAllUnansweredConsumed   QuestionGenerationJobReason = "all_unanswered_consumed"
	JobReasonManualSelection         QuestionGenerationJobReason = "manual_selection"
)

const (
	HighlightBatchThreshold = 10
	MinHighlightsForRefresh = 5
	MaxHighlightsPerJob     = 10
	JobMaxRetryCount        = 3
	// JobStaleProcessingTimeout is how long a job may sit in processing before
	// it is considered orphaned by a dead worker and becomes reclaimable.
	JobStaleProcessingTimeout = 15 * time.Minute
)

type QuestionGenerationJob struct {
	ID                  uuid.UUID
	UserID              uuid.UUID
	BookKey             string
	Status              QuestionGenerationJobStatus
	Reason              QuestionGenerationJobReason
	RetryCount          int
	LastError           string
	HighlightIDs        []uuid.UUID
	CreatedAt           time.Time
	ProcessingStartedAt *time.Time
	CompletedAt         *time.Time
	FailedAt            *time.Time
}

type CreateQuestionGenerationJobInput struct {
	UserID       uuid.UUID
	BookKey      string
	Reason       QuestionGenerationJobReason
	HighlightIDs []uuid.UUID
}

type QuestionGenerationJobRepository interface {
	Create(ctx context.Context, input CreateQuestionGenerationJobInput) (*QuestionGenerationJob, error)
	CountPendingByUserID(ctx context.Context, userID uuid.UUID) (int, error)
	CountPendingByBookKey(ctx context.Context, userID uuid.UUID, bookKey string) (int, error)
	CountPending(ctx context.Context) (int, error)
	ListQueuedByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*QuestionGenerationJob, error)
	ListEnqueueFailedByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*QuestionGenerationJob, error)
	ClaimQueued(ctx context.Context, jobID, userID uuid.UUID) (*QuestionGenerationJob, bool, error)
	RequeueStaleProcessing(ctx context.Context, userID uuid.UUID, limit int) ([]*QuestionGenerationJob, error)
	FailExhaustedStaleProcessing(ctx context.Context, userID uuid.UUID, limit int) ([]*QuestionGenerationJob, error)
	MarkQueued(ctx context.Context, jobID, userID uuid.UUID) error
	MarkCompleted(ctx context.Context, jobID, userID uuid.UUID) error
	MarkEnqueueFailed(ctx context.Context, jobID, userID uuid.UUID, lastError string) error
	RecordFailure(ctx context.Context, jobID, userID uuid.UUID, lastError string, maxRetry int) (*QuestionGenerationJob, error)
}

type QuestionGenerationTaskEnqueuer interface {
	// attempt is the job's retry_count at enqueue time. It becomes part of the
	// Cloud Tasks task name: reusing one fixed name per job made re-enqueues
	// after a stale requeue (or an admin retry) dedup against the completed
	// original task and silently no-op for up to ~1 hour.
	EnqueueQuestionGeneration(ctx context.Context, jobID uuid.UUID, userID uuid.UUID, attempt int) error
}
