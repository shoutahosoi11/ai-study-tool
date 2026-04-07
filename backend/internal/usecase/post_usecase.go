package usecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/shout/ai-study-tool/backend/internal/domain"
	"github.com/shout/ai-study-tool/backend/internal/repository"
)

type PostUsecase struct {
	postRepo repository.PostRepository
}

func NewPostUsecase(postRepo repository.PostRepository) *PostUsecase {
	return &PostUsecase{postRepo: postRepo}
}

func (u *PostUsecase) GetTimeline(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.TimelinePost, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	return u.postRepo.GetTimeline(ctx, domain.TimelineParams{
		UserID: userID,
		Limit:  limit,
		Offset: offset,
	})
}

func (u *PostUsecase) GetByID(ctx context.Context, id uuid.UUID) (*domain.TimelinePost, error) {
	return u.postRepo.GetByID(ctx, id)
}

func (u *PostUsecase) CreatePost(ctx context.Context, input domain.CreatePostInput) (*domain.Post, error) {
	return u.postRepo.Create(ctx, input)
}

func (u *PostUsecase) LikePost(ctx context.Context, postID uuid.UUID) error {
	return u.postRepo.IncrementLike(ctx, postID)
}

func (u *PostUsecase) UnlikePost(ctx context.Context, postID uuid.UUID) error {
	return u.postRepo.DecrementLike(ctx, postID)
}
