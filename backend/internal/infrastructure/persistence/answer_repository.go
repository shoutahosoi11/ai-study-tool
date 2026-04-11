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

func (r *answerRepository) UpsertAndUpdateStats(ctx context.Context, input domain.AnswerUpsertInput, questionID string, isCorrect bool) (*domain.Answer, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("answer repo: begin tx: %w", err)
	}
	defer tx.Rollback()

	upsertQuery := `
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
	qID, err := uuid.Parse(input.QuestionID)
	if err != nil {
		return nil, fmt.Errorf("answer repo: parse question id: %w", err)
	}

	row := tx.QueryRowContext(ctx, upsertQuery,
		userID, qID, input.UserAnswer, input.IsCorrect,
		input.Score, input.Feedback, input.GraderModel,
	)

	var (
		id          uuid.UUID
		uID         uuid.UUID
		rQID        uuid.UUID
		userAnswer  string
		rIsCorrect  bool
		score       *int
		feedback    *string
		graderModel *string
		createdAt   time.Time
		updatedAt   time.Time
	)
	if err := row.Scan(&id, &uID, &rQID, &userAnswer, &rIsCorrect, &score, &feedback, &graderModel, &createdAt, &updatedAt); err != nil {
		return nil, fmt.Errorf("answer repo: upsert scan: %w", err)
	}

	statsQuery := `
UPDATE questions
SET
    answer_count  = answer_count + 1,
    correct_count = correct_count + CASE WHEN $2 THEN 1 ELSE 0 END,
    updated_at    = NOW()
WHERE id = $1`
	if _, err := tx.ExecContext(ctx, statsQuery, questionID, isCorrect); err != nil {
		return nil, fmt.Errorf("answer repo: update stats: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("answer repo: commit: %w", err)
	}

	return &domain.Answer{
		ID:          id.String(),
		UserID:      uID.String(),
		QuestionID:  rQID.String(),
		UserAnswer:  userAnswer,
		IsCorrect:   rIsCorrect,
		Score:       score,
		Feedback:    feedback,
		GraderModel: graderModel,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}, nil
}
