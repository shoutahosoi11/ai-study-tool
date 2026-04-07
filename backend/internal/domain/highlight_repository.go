package domain

import (
	"context"

	"github.com/google/uuid"
)

type HighlightRepository interface {
	Create(ctx context.Context, h *Highlight) (*Highlight, error)
	GetByID(ctx context.Context, id, userID uuid.UUID) (*Highlight, error)
	ListByUserID(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]*Highlight, error)
	CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error)
	Delete(ctx context.Context, id, userID uuid.UUID) error
}
