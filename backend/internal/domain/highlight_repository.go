package domain

import (
	"context"

	"github.com/google/uuid"
)

type HighlightRepository interface {
	BulkUpsert(ctx context.Context, highlights []*Highlight) (saved int, err error)
	ListByUserIDAndASIN(ctx context.Context, userID uuid.UUID, asin string) ([]*Highlight, error)
	ListByUserIDAndBookMetadata(ctx context.Context, userID uuid.UUID, bookTitle, bookAuthor string) ([]*Highlight, error)
	ListBooksWithHighlightsByUserID(ctx context.Context, userID uuid.UUID) ([]*KindleBook, error)
	UpdateExplanation(ctx context.Context, id, userID uuid.UUID, explanation *string) (*Highlight, error)
}
