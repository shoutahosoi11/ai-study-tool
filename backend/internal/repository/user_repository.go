package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

type UserRepository interface {
	GetByFirebaseUID(ctx context.Context, firebaseUID string) (*domain.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByUsername(ctx context.Context, username string) (*domain.User, error)
	Create(ctx context.Context, input domain.CreateUserInput) (*domain.User, error)
	Update(ctx context.Context, id uuid.UUID, input domain.UpdateUserInput) (*domain.User, error)
}
