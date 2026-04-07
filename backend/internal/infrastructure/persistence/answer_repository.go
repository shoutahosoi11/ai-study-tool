package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

type answerRepository struct {
	db *sql.DB
}

func NewAnswerRepository(db *sql.DB) domain.AnswerRepository {
	return &answerRepository{db: db}
}

func (r *answerRepository) Upsert(ctx context.Context, input domain.AnswerUpsertInput) (*domain.Answer, error) {
	query := `
INSERT INTO answers (user_id, question_id, user_answer, is_correct, score, feedback, grader_model)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (user_id, question_id) DO UPDATE SET
    user_answer  = EXCLUDED.user_answer,
    is_correct   = EXCLUDED.is_correct,
    score        = EXCLUDED.score,
    feedback     = EXCLUDED.feedback,
    grader_model = EXCLUDED.grader_model,
    updated_at   = NOW()
RETURNING id, user_id, question_id, user_answer, is_correct, score, feedback, grader_model, created_at, updated_at`

	userID, err := uuid.Parse(input.UserID)
	if err != nil {
		return nil, fmt.Errorf("answer repo: parse user id: %w", err)
	}

	questionID, err := uuid.Parse(input.QuestionID)
	if err != nil {
		return nil, fmt.Errorf("answer repo: parse question id: %w", err)
	}

	row := r.db.QueryRowContext(
		ctx,
		query,
		userID,
		questionID,
		input.UserAnswer,
		input.IsCorrect,
		input.Score,
		input.Feedback,
		input.GraderModel,
	)

	var (
		id          uuid.UUID
		uID         uuid.UUID
		qID         uuid.UUID
		userAnswer  string
		isCorrect   bool
		score       *int
		feedback    *string
		graderModel *string
		createdAt   time.Time
		updatedAt   time.Time
	)

	err = row.Scan(
		&id,
		&uID,
		&qID,
		&userAnswer,
		&isCorrect,
		&score,
		&feedback,
		&graderModel,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("answer repo: upsert scan: %w", err)
	}

	return &domain.Answer{
		ID:          id.String(),
		UserID:      uID.String(),
		QuestionID:  qID.String(),
		UserAnswer:  userAnswer,
		IsCorrect:   isCorrect,
		Score:       score,
		Feedback:    feedback,
		GraderModel: graderModel,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}, nil
}
