package usecase

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/shout/ai-study-tool/backend/internal/domain"
	"github.com/shout/ai-study-tool/backend/internal/repository"
)

type UserUsecase struct {
	userRepo repository.UserRepository
}

func NewUserUsecase(userRepo repository.UserRepository) *UserUsecase {
	return &UserUsecase{userRepo: userRepo}
}

func (u *UserUsecase) SignUp(ctx context.Context, input domain.CreateUserInput) (*domain.User, error) {
	existing, err := u.userRepo.GetByFirebaseUID(ctx, input.FirebaseUID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}
	return u.userRepo.Create(ctx, input)
}

func (u *UserUsecase) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return u.userRepo.GetByID(ctx, id)
}

func (u *UserUsecase) GetByFirebaseUID(ctx context.Context, firebaseUID string) (*domain.User, error) {
	return u.userRepo.GetByFirebaseUID(ctx, firebaseUID)
}

func (u *UserUsecase) UpdateProfile(ctx context.Context, id uuid.UUID, input domain.UpdateUserInput) (*domain.User, error) {
	return u.userRepo.Update(ctx, id, input)
}

func (u *UserUsecase) GetPlanByFirebaseUID(ctx context.Context, firebaseUID string) string {
	user, err := u.userRepo.GetByFirebaseUID(ctx, firebaseUID)
	if err != nil {
		return "free"
	}
	return user.Plan
}
