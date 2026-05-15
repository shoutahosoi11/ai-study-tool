package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shout/ai-study-tool/backend/internal/domain"
	"github.com/shout/ai-study-tool/backend/internal/usecase"
)

type mockHighlightImportQueueRepository struct {
	item               *domain.HighlightImportQueue
	enqueueID          uuid.UUID
	enqueueErr         error
	enqueued           bool
	enqueuedUserID     uuid.UUID
	enqueuedSource     string
	enqueuedPayload    []byte
	claimed            bool
	completed          bool
	failed             bool
	failedID           uuid.UUID
	failedError        string
	markedQueued       bool
	enqueueFailed      bool
	enqueueFailedID    uuid.UUID
	enqueueFailedError string
	requeued           bool
	lastRetryError     string
	recoverableItems   []*domain.HighlightImportQueue
}

func (m *mockHighlightImportQueueRepository) Enqueue(ctx context.Context, userID uuid.UUID, source string, payload []byte) (uuid.UUID, error) {
	m.enqueued = true
	m.enqueuedUserID = userID
	m.enqueuedSource = source
	m.enqueuedPayload = append([]byte(nil), payload...)
	if m.enqueueErr != nil {
		return uuid.Nil, m.enqueueErr
	}
	if m.enqueueID == uuid.Nil {
		return uuid.Nil, errors.New("unexpected enqueue call")
	}
	return m.enqueueID, nil
}

func (m *mockHighlightImportQueueRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.HighlightImportQueue, error) {
	return m.item, nil
}

func (m *mockHighlightImportQueueRepository) DequeueBatch(ctx context.Context, limit int) ([]*domain.HighlightImportQueue, error) {
	return nil, errors.New("unexpected dequeue call")
}

func (m *mockHighlightImportQueueRepository) ClaimProcessing(ctx context.Context, id uuid.UUID) (bool, error) {
	m.claimed = true
	return true, nil
}

func (m *mockHighlightImportQueueRepository) MarkCompleted(ctx context.Context, id uuid.UUID) error {
	m.completed = true
	return nil
}

func (m *mockHighlightImportQueueRepository) MarkQueued(ctx context.Context, id uuid.UUID) error {
	m.markedQueued = true
	return nil
}

func (m *mockHighlightImportQueueRepository) MarkFailed(ctx context.Context, id uuid.UUID, errMsg string) error {
	m.failed = true
	m.failedID = id
	m.failedError = errMsg
	return nil
}

func (m *mockHighlightImportQueueRepository) MarkEnqueueFailed(ctx context.Context, id uuid.UUID, errMsg string) error {
	m.enqueueFailed = true
	m.enqueueFailedID = id
	m.enqueueFailedError = errMsg
	return nil
}

func (m *mockHighlightImportQueueRepository) RequeueWithRetry(ctx context.Context, id uuid.UUID, errMsg string) error {
	m.requeued = true
	m.lastRetryError = errMsg
	return nil
}

func (m *mockHighlightImportQueueRepository) RequeueStale(ctx context.Context, cutoff time.Time) (int, error) {
	return 0, nil
}

func (m *mockHighlightImportQueueRepository) ListRecoverableEnqueuesByUserID(ctx context.Context, userID uuid.UUID, staleQueuedCutoff time.Time, limit int) ([]*domain.HighlightImportQueue, error) {
	return m.recoverableItems, nil
}

func TestHighlightImportProcessSingleRejectsUserMismatch(t *testing.T) {
	queueID := uuid.New()
	ownerID := uuid.New()
	requestUserID := uuid.New()
	queueRepo := &mockHighlightImportQueueRepository{
		item: &domain.HighlightImportQueue{
			ID:         queueID,
			UserID:     ownerID,
			RawPayload: []byte(`[{"content":"highlight"}]`),
			Status:     domain.ImportQueueStatusQueued,
		},
	}
	highlightRepo := &mockImportHighlightRepository{}
	uc := usecase.NewHighlightImportJobUsecase(queueRepo, highlightRepo)

	err := uc.ProcessSingle(context.Background(), queueID, requestUserID)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
	if queueRepo.claimed {
		t.Fatal("expected queue not to be claimed for mismatched user")
	}
	if highlightRepo.bulkUpsertCalled {
		t.Fatal("expected highlights not to be imported for mismatched user")
	}
}

func TestHighlightImportProcessSingleRequeuesBeforeMaxRetry(t *testing.T) {
	queueID := uuid.New()
	userID := uuid.New()
	queueRepo := &mockHighlightImportQueueRepository{
		item: &domain.HighlightImportQueue{
			ID:         queueID,
			UserID:     userID,
			RawPayload: []byte(`[{"content":"highlight"}]`),
			Status:     domain.ImportQueueStatusQueued,
			RetryCount: domain.ImportQueueMaxRetry - 2,
		},
	}
	highlightRepo := &mockImportHighlightRepository{bulkUpsertErr: errors.New("db down")}
	uc := usecase.NewHighlightImportJobUsecase(queueRepo, highlightRepo)

	if err := uc.ProcessSingle(context.Background(), queueID, userID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !queueRepo.requeued {
		t.Fatal("expected queue item to be requeued before max retry")
	}
	if queueRepo.failed {
		t.Fatal("expected queue item not to be failed before max retry")
	}
}

func TestHighlightImportProcessSingleResumesProcessingItem(t *testing.T) {
	queueID := uuid.New()
	userID := uuid.New()
	startedAt := time.Now().UTC().Add(-time.Minute)
	queueRepo := &mockHighlightImportQueueRepository{
		item: &domain.HighlightImportQueue{
			ID:                  queueID,
			UserID:              userID,
			RawPayload:          []byte(`[{"content":"highlight"}]`),
			Status:              domain.ImportQueueStatusProcessing,
			ProcessingStartedAt: &startedAt,
		},
	}
	highlightRepo := &mockImportHighlightRepository{}
	uc := usecase.NewHighlightImportJobUsecase(queueRepo, highlightRepo)

	if err := uc.ProcessSingle(context.Background(), queueID, userID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if queueRepo.claimed {
		t.Fatal("processing item should be resumed without queued claim")
	}
	if !highlightRepo.bulkUpsertCalled {
		t.Fatal("expected processing item payload to be imported")
	}
	if !queueRepo.completed {
		t.Fatal("expected processing item to be completed")
	}
}

func TestHighlightImportProcessSingleRecoversEnqueueFailedItem(t *testing.T) {
	queueID := uuid.New()
	userID := uuid.New()
	queueRepo := &mockHighlightImportQueueRepository{
		item: &domain.HighlightImportQueue{
			ID:         queueID,
			UserID:     userID,
			RawPayload: []byte(`[{"content":"highlight"}]`),
			Status:     domain.ImportQueueStatusEnqueueFailed,
		},
	}
	highlightRepo := &mockImportHighlightRepository{}
	uc := usecase.NewHighlightImportJobUsecase(queueRepo, highlightRepo)

	if err := uc.ProcessSingle(context.Background(), queueID, userID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !queueRepo.markedQueued {
		t.Fatal("expected enqueue_failed item to be marked queued")
	}
	if !queueRepo.claimed {
		t.Fatal("expected recovered item to be claimed")
	}
	if !highlightRepo.bulkUpsertCalled {
		t.Fatal("expected recovered item payload to be imported")
	}
}

func TestHighlightImportProcessSingleFailsAtMaxRetry(t *testing.T) {
	queueID := uuid.New()
	userID := uuid.New()
	queueRepo := &mockHighlightImportQueueRepository{
		item: &domain.HighlightImportQueue{
			ID:         queueID,
			UserID:     userID,
			RawPayload: []byte(`[{"content":"highlight"}]`),
			Status:     domain.ImportQueueStatusQueued,
			RetryCount: domain.ImportQueueMaxRetry - 1,
		},
	}
	highlightRepo := &mockImportHighlightRepository{bulkUpsertErr: errors.New("db down")}
	uc := usecase.NewHighlightImportJobUsecase(queueRepo, highlightRepo)

	err := uc.ProcessSingle(context.Background(), queueID, userID)
	if err == nil {
		t.Fatal("expected error")
	}
	if !queueRepo.failed {
		t.Fatal("expected queue item to be failed at max retry")
	}
	if queueRepo.requeued {
		t.Fatal("expected queue item not to be requeued at max retry")
	}
}
