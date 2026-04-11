package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

type userRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) domain.UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) GetByFirebaseUID(ctx context.Context, firebaseUID string) (*domain.User, error) {
	query := `SELECT id, firebase_uid, username, display_name, avatar_url, bio, university, faculty, grade, country, plan, created_at, updated_at FROM users WHERE firebase_uid = $1 LIMIT 1`
	row := r.db.QueryRowContext(ctx, query, firebaseUID)
	return scanUser(row)
}

func (r *userRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := `SELECT id, firebase_uid, username, display_name, avatar_url, bio, university, faculty, grade, country, plan, created_at, updated_at FROM users WHERE id = $1 LIMIT 1`
	row := r.db.QueryRowContext(ctx, query, id)
	return scanUser(row)
}

func (r *userRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	query := `SELECT id, firebase_uid, username, display_name, avatar_url, bio, university, faculty, grade, country, plan, created_at, updated_at FROM users WHERE username = $1 LIMIT 1`
	row := r.db.QueryRowContext(ctx, query, username)
	return scanUser(row)
}

func (r *userRepository) Create(ctx context.Context, input domain.CreateUserInput) (*domain.User, error) {
	query := `
		INSERT INTO users (firebase_uid, username, display_name, avatar_url, bio, university, faculty, grade, country, plan)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'free')
		RETURNING id, firebase_uid, username, display_name, avatar_url, bio, university, faculty, grade, country, plan, created_at, updated_at`
	row := r.db.QueryRowContext(ctx, query,
		input.FirebaseUID,
		input.Username,
		input.DisplayName,
		input.AvatarURL,
		input.Bio,
		input.University,
		input.Faculty,
		input.Grade,
		input.Country,
	)
	return scanUser(row)
}

func (r *userRepository) Update(ctx context.Context, id uuid.UUID, input domain.UpdateUserInput) (*domain.User, error) {
	query := `
		UPDATE users SET username=$2, display_name=$3, avatar_url=$4, bio=$5, university=$6, faculty=$7, grade=$8, country=$9, updated_at=NOW()
		WHERE id=$1
		RETURNING id, firebase_uid, username, display_name, avatar_url, bio, university, faculty, grade, country, plan, created_at, updated_at`
	row := r.db.QueryRowContext(ctx, query,
		id,
		input.Username,
		input.DisplayName,
		input.AvatarURL,
		input.Bio,
		input.University,
		input.Faculty,
		input.Grade,
		input.Country,
	)
	return scanUser(row)
}

func scanUser(row *sql.Row) (*domain.User, error) {
	u := &domain.User{}
	err := row.Scan(
		&u.ID,
		&u.FirebaseUID,
		&u.Username,
		&u.DisplayName,
		&u.AvatarURL,
		&u.Bio,
		&u.University,
		&u.Faculty,
		&u.Grade,
		&u.Country,
		&u.Plan,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return u, nil
}
