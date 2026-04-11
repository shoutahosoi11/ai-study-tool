package domain

import (
	"context"

	"github.com/google/uuid"
)

type PostRepository interface {
	GetTimeline(ctx context.Context, params TimelineParams) ([]*TimelinePost, error)
	GetByID(ctx context.Context, id uuid.UUID) (*TimelinePost, error)
	Create(ctx context.Context, input CreatePostInput) (*Post, error)
}
