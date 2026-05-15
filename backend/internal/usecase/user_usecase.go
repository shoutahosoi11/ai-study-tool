package usecase

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

type UserUsecaseInterface interface {
	SignUp(ctx context.Context, input domain.CreateUserInput) (*domain.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByFirebaseUID(ctx context.Context, firebaseUID string) (*domain.User, error)
	UpdateProfile(ctx context.Context, id uuid.UUID, input domain.UpdateUserInput) (*domain.User, error)
	UpdateQuestionSettings(ctx context.Context, id uuid.UUID, input domain.UpdateQuestionSettingsInput) (*domain.User, error)
	GetPlanByFirebaseUID(ctx context.Context, firebaseUID string) (string, error)
}

type UserUsecase struct {
	userRepo domain.UserRepository
}

func NewUserUsecase(userRepo domain.UserRepository) *UserUsecase {
	return &UserUsecase{userRepo: userRepo}
}

func (u *UserUsecase) SignUp(ctx context.Context, input domain.CreateUserInput) (*domain.User, error) {
	input.FirebaseUID = strings.TrimSpace(input.FirebaseUID)
	input.Username = domain.NormalizeUsername(input.Username)
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	if input.FirebaseUID == "" || !domain.IsValidUsername(input.Username) || strings.TrimSpace(input.DisplayName) == "" {
		return nil, domain.ErrInvalidInput
	}

	// SignUp は Firebase ユーザー作成後の再試行に備えて、同じ Firebase UID では冪等に既存ユーザーを返す。
	// 既存ユーザーの username/display_name 変更は UpdateProfile に集約する。
	existing, err := u.userRepo.GetByFirebaseUID(ctx, input.FirebaseUID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}
	// 他のユーザーが使っていないか
	existingByUsername, err := u.userRepo.GetByUsername(ctx, input.Username)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}
	if existingByUsername != nil {
		return nil, domain.ErrAlreadyExists
	}

	// DBの一意制約が最終防衛ラインなので、ここは事前チェックとの race を許容する。
	created, err := u.userRepo.Create(ctx, input)
	if err == nil {
		return created, nil
	}
	if errors.Is(err, domain.ErrAlreadyExists) {
		existing, lookupErr := u.userRepo.GetByFirebaseUID(ctx, input.FirebaseUID)
		if lookupErr == nil && existing != nil {
			return existing, nil
		}
	}
	return nil, err
}

func (u *UserUsecase) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return u.userRepo.GetByID(ctx, id)
}

func (u *UserUsecase) GetByFirebaseUID(ctx context.Context, firebaseUID string) (*domain.User, error) {
	return u.userRepo.GetByFirebaseUID(ctx, firebaseUID)
}

func (u *UserUsecase) UpdateProfile(ctx context.Context, id uuid.UUID, input domain.UpdateUserInput) (*domain.User, error) {
	if !input.HasChanges() {
		return nil, domain.ErrInvalidInput
	}

	if input.Username.Set {
		if input.Username.Value == nil {
			return nil, domain.ErrInvalidInput
		}
		username := domain.NormalizeUsername(*input.Username.Value)
		if !domain.IsValidUsername(username) {
			return nil, domain.ErrInvalidInput
		}
		input.Username.Value = &username

		//　同じ username を他のユーザーが使っていないか
		existingByUsername, err := u.userRepo.GetByUsername(ctx, username)
		if err != nil && !errors.Is(err, domain.ErrNotFound) {
			return nil, err
		}
		if existingByUsername != nil && existingByUsername.ID != id {
			return nil, domain.ErrAlreadyExists
		}
	}

	if input.DisplayName.Set {
		if input.DisplayName.Value == nil {
			return nil, domain.ErrInvalidInput
		}
		displayName := strings.TrimSpace(*input.DisplayName.Value)
		if displayName == "" {
			return nil, domain.ErrInvalidInput
		}
		input.DisplayName.Value = &displayName
	}

	// DBの一意制約が最終防衛ラインなので、ここは事前チェックとの race を許容する。
	return u.userRepo.Update(ctx, id, input)
}

func (u *UserUsecase) UpdateQuestionSettings(ctx context.Context, id uuid.UUID, input domain.UpdateQuestionSettingsInput) (*domain.User, error) {
	if !domain.IsValidDefaultQuestionCount(input.DefaultQuestionCount) {
		return nil, domain.ErrInvalidInput
	}

	return u.userRepo.UpdateQuestionSettings(ctx, id, input)
}

func (u *UserUsecase) GetPlanByFirebaseUID(ctx context.Context, firebaseUID string) (string, error) {
	user, err := u.userRepo.GetByFirebaseUID(ctx, firebaseUID)
	if err != nil {
		// FirebaseUIDに対応するユーザーが未登録の場合はfreeプランとして扱う。
		// これは初回ログイン直後など、ユーザーレコード作成前にプラン判定が走るケースへの対応。
		if errors.Is(err, domain.ErrNotFound) {
			return "free", nil
		}
		return "", err
	}
	return user.Plan, nil
}
