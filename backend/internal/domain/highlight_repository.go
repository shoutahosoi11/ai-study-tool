package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type HighlightRepository interface {
	BulkUpsert(ctx context.Context, highlights []*Highlight) (saved int, err error)
	ListExistingContentHashesByUserID(ctx context.Context, userID uuid.UUID, hashes []string) ([]string, error)
	ListByUserIDAndASIN(ctx context.Context, userID uuid.UUID, asin string) ([]*Highlight, error)
	ListByUserIDAndBookMetadata(ctx context.Context, userID uuid.UUID, bookTitle, bookAuthor string) ([]*Highlight, error)
	ListBooksWithHighlightsByUserID(ctx context.Context, userID uuid.UUID) ([]*KindleBook, error)
	ListBookStockByUserID(ctx context.Context, userID uuid.UUID) ([]BookStock, error)
	ListUnusedHighlightsByBook(ctx context.Context, userID uuid.UUID, bookKey string, limit int) ([]*Highlight, error)
	ListUsedHighlightsWithUncoveredPerspectives(ctx context.Context, userID uuid.UUID, bookKey string, limit int) ([]*Highlight, error)
	ListPendingUserStats(ctx context.Context) ([]PendingHighlightUserStat, error)
	ClaimPendingByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*Highlight, error)
	ClaimPendingByIDs(ctx context.Context, userID uuid.UUID, highlightIDs []uuid.UUID) ([]*Highlight, error)
	QueueHighlightsForGeneration(ctx context.Context, userID uuid.UUID, highlightIDs []uuid.UUID, requestedAt time.Time) error
	MarkGenerationCompleted(ctx context.Context, highlightIDs []uuid.UUID) error
	MarkGenerationFailed(ctx context.Context, highlightIDs []uuid.UUID, lastError string, maxRetry int) error
	UpdateExplanation(ctx context.Context, id, userID uuid.UUID, explanation *string) (*Highlight, error)
}
