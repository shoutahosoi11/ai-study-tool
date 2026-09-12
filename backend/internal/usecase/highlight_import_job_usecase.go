package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

type HighlightImportJobUsecase struct {
	queueRepo     domain.HighlightImportQueueRepository
	highlightRepo domain.HighlightImportRepository
}

func NewHighlightImportJobUsecase(
	queueRepo domain.HighlightImportQueueRepository,
	highlightRepo domain.HighlightImportRepository,
) *HighlightImportJobUsecase {
	return &HighlightImportJobUsecase{
		queueRepo:     queueRepo,
		highlightRepo: highlightRepo,
	}
}

const importJobBatchSize = 50

// ProcessAll は queued 状態のキューを全件処理する。
// Cloud Run Job のエントリーポイントから呼ばれる。
func (u *HighlightImportJobUsecase) ProcessAll(ctx context.Context) error {
	// stale な processing を queued に戻す
	cutoff := time.Now().UTC().Add(-domain.ImportQueueStaleTimeout)
	if requeued, err := u.queueRepo.RequeueStale(ctx, cutoff); err != nil {
		slog.Error("highlight_import_requeue_stale_failed", "error", err.Error())
	} else if requeued > 0 {
		slog.Info("highlight_import_requeued_stale", "count", requeued)
	}

	processed := 0
	for {
		batch, err := u.queueRepo.DequeueBatch(ctx, importJobBatchSize)
		if err != nil {
			return fmt.Errorf("highlight import job: dequeue batch: %w", err)
		}
		if len(batch) == 0 {
			break
		}

		for _, item := range batch {
			if err := u.processOne(ctx, item); err != nil {
				slog.Error("highlight_import_item_failed", "item_id", item.ID, "error", err.Error())
			}
		}
		processed += len(batch)
	}

	slog.Info("highlight_import_processed", "count", processed)
	return nil
}

func (u *HighlightImportJobUsecase) processOne(ctx context.Context, item *domain.HighlightImportQueue) error {
	// CAS: queued → processing（他の実行との競合防止）
	ok, err := u.queueRepo.ClaimProcessing(ctx, item.ID)
	if err != nil {
		return fmt.Errorf("claim processing: %w", err)
	}
	if !ok {
		return nil // 別の実行がすでに処理中
	}

	return u.processClaimed(ctx, item)
}

func (u *HighlightImportJobUsecase) processClaimed(ctx context.Context, item *domain.HighlightImportQueue) error {
	highlights, err := u.deserializePayload(item)
	if err != nil {
		return u.fail(ctx, item, fmt.Sprintf("deserialize payload: %v", err))
	}

	if _, err := u.highlightRepo.BulkUpsert(ctx, highlights); err != nil {
		if item.RetryCount+1 >= domain.ImportQueueMaxRetry {
			return u.fail(ctx, item, fmt.Sprintf("bulk upsert (max retry reached): %v", err))
		}
		if requeueErr := u.queueRepo.RequeueWithRetry(ctx, item.ID, err.Error()); requeueErr != nil {
			slog.Error("highlight_import_requeue_failed", "item_id", item.ID, "error", requeueErr.Error())
		}
		return nil
	}

	if err := u.queueRepo.MarkCompleted(ctx, item.ID); err != nil {
		slog.Error("highlight_import_mark_completed_failed", "item_id", item.ID, "error", err.Error())
	}
	return nil
}

func (u *HighlightImportJobUsecase) deserializePayload(item *domain.HighlightImportQueue) ([]*domain.Highlight, error) {
	highlights, err := unmarshalHighlightImportPayload(item.RawPayload, item.UserID)
	if err != nil {
		return nil, err
	}
	if len(highlights) == 0 {
		return nil, fmt.Errorf("empty highlights payload for queue_id=%s", item.ID)
	}
	return highlights, nil
}

func (u *HighlightImportJobUsecase) fail(ctx context.Context, item *domain.HighlightImportQueue, reason string) error {
	if err := u.queueRepo.MarkFailed(ctx, item.ID, reason); err != nil {
		slog.Error("highlight_import_mark_failed_failed", "item_id", item.ID, "error", err.Error())
	}
	return fmt.Errorf("queue_id=%s failed: %s", item.ID, reason)
}

// ProcessSingle は特定の queue_id を処理する（Cloud Tasks / テスト・デバッグ用）。
func (u *HighlightImportJobUsecase) ProcessSingle(ctx context.Context, queueID uuid.UUID, userID uuid.UUID) error {
	item, err := u.queueRepo.GetByID(ctx, queueID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil
		}
		return fmt.Errorf("highlight import job: get queue item: %w", err)
	}
	if item.UserID != userID {
		return domain.ErrForbidden
	}
	switch item.Status {
	case domain.ImportQueueStatusCompleted, domain.ImportQueueStatusFailed:
		return nil
	case domain.ImportQueueStatusProcessing:
		return u.processClaimed(ctx, item)
	case domain.ImportQueueStatusEnqueueFailed:
		if err := u.queueRepo.MarkQueued(ctx, item.ID); err != nil {
			return fmt.Errorf("mark enqueue failed item queued: %w", err)
		}
	}
	return u.processOne(ctx, item)
}
