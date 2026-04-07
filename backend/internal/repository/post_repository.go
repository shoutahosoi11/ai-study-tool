package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

type PostRepository interface {
	GetTimeline(ctx context.Context, params domain.TimelineParams) ([]*domain.TimelinePost, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.TimelinePost, error)
	Create(ctx context.Context, input domain.CreatePostInput) (*domain.Post, error)
	IncrementLike(ctx context.Context, id uuid.UUID) error
	DecrementLike(ctx context.Context, id uuid.UUID) error
	IncrementRepost(ctx context.Context, id uuid.UUID) error
	IncrementComment(ctx context.Context, id uuid.UUID) error
}
