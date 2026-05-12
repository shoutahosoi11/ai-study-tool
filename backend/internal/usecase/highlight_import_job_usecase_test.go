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
	item           *domain.HighlightImportQueue
	claimed        bool
	completed      bool
	failed         bool
	requeued       bool
	lastRetryError string
}

func (m *mockHighlightImportQueueRepository) Enqueue(ctx context.Context, userID uuid.UUID, source string, payload []byte) (uuid.UUID, error) {
	return uuid.Nil, errors.New("unexpected enqueue call")
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

func (m *mockHighlightImportQueueRepository) MarkFailed(ctx context.Context, id uuid.UUID, errMsg string) error {
	m.failed = true
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
