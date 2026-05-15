package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type HighlightImportQueueStatus string

const (
	ImportQueueStatusQueued        HighlightImportQueueStatus = "queued"
	ImportQueueStatusProcessing    HighlightImportQueueStatus = "processing"
	ImportQueueStatusCompleted     HighlightImportQueueStatus = "completed"
	ImportQueueStatusFailed        HighlightImportQueueStatus = "failed"
	ImportQueueStatusEnqueueFailed HighlightImportQueueStatus = "enqueue_failed"
)

const (
	ImportQueueSourceKindle       = "kindle"
	ImportQueueMaxRetry           = 3
	ImportQueueStaleTimeout       = 10 * time.Minute
	ImportQueueStaleQueuedTimeout = 5 * time.Minute
	ImportQueueRecoverLimit       = 20
)

type HighlightImportQueue struct {
	ID                  uuid.UUID
	UserID              uuid.UUID
	Source              string
	RawPayload          []byte
	Status              HighlightImportQueueStatus
	RetryCount          int
	LastError           string
	CreatedAt           time.Time
	ProcessingStartedAt *time.Time
	CompletedAt         *time.Time
	FailedAt            *time.Time
}

type HighlightImportQueueRepository interface {
	Enqueue(ctx context.Context, userID uuid.UUID, source string, payload []byte) (uuid.UUID, error)
	GetByID(ctx context.Context, id uuid.UUID) (*HighlightImportQueue, error)
	DequeueBatch(ctx context.Context, limit int) ([]*HighlightImportQueue, error)
	ClaimProcessing(ctx context.Context, id uuid.UUID) (bool, error)
	MarkQueued(ctx context.Context, id uuid.UUID) error
	MarkCompleted(ctx context.Context, id uuid.UUID) error
	MarkFailed(ctx context.Context, id uuid.UUID, errMsg string) error
	MarkEnqueueFailed(ctx context.Context, id uuid.UUID, errMsg string) error
	RequeueWithRetry(ctx context.Context, id uuid.UUID, errMsg string) error
	RequeueStale(ctx context.Context, cutoff time.Time) (int, error)
	ListRecoverableEnqueuesByUserID(ctx context.Context, userID uuid.UUID, staleQueuedCutoff time.Time, limit int) ([]*HighlightImportQueue, error)
}

// HighlightImportJobTrigger abstracts async import task enqueueing.
// usecase 層が infrastructure に依存しないよう抽象化。
type HighlightImportJobTrigger interface {
	TriggerHighlightImportJob(ctx context.Context, queueID uuid.UUID, userID uuid.UUID) error
}
