package persistence

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/shout/ai-study-tool/backend/internal/domain"
	"github.com/shout/ai-study-tool/backend/internal/repository/sqlcgen"
)

type answerRepository struct {
	db      *sql.DB
	queries *sqlcgen.Queries
}

func NewAnswerRepository(db *sql.DB) domain.AnswerRepository {
	return &answerRepository{
		db:      db,
		queries: sqlcgen.New(db),
	}
}

func (r *answerRepository) Upsert(ctx context.Context, input domain.AnswerUpsertInput) (*domain.Answer, error) {
	userID, err := parseAnswerUUID(input.UserID)
	if err != nil {
		return nil, fmt.Errorf("answer repo: parse user id: %w", err)
	}

	questionID, err := parseAnswerUUID(input.QuestionID)
	if err != nil {
		return nil, fmt.Errorf("answer repo: parse question id: %w", err)
	}

	answer, err := r.queries.UpsertAnswer(ctx, sqlcgen.UpsertAnswerParams{
		UserID:      userID,
		QuestionID:  questionID,
		UserAnswer:  input.UserAnswer,
		IsCorrect:   input.IsCorrect,
		Score:       toNullInt32(input.Score),
		Feedback:    toNullString(input.Feedback),
		GraderModel: toNullString(input.GraderModel),
	})
	if err != nil {
		return nil, fmt.Errorf("answer repo: upsert: %w", err)
	}

	return toDomainAnswer(answer), nil
}

func (r *answerRepository) UpsertAndUpdateStats(ctx context.Context, input domain.AnswerUpsertInput, questionID string, isCorrect bool) (*domain.Answer, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("answer repo: begin tx: %w", err)
	}
	defer tx.Rollback()

	userID, err := parseAnswerUUID(input.UserID)
	if err != nil {
		return nil, fmt.Errorf("answer repo: parse user id: %w", err)
	}

	inputQuestionID, err := parseAnswerUUID(input.QuestionID)
	if err != nil {
		return nil, fmt.Errorf("answer repo: parse question id: %w", err)
	}

	statsQuestionID, err := parseAnswerUUID(questionID)
	if err != nil {
		return nil, fmt.Errorf("answer repo: parse stats question id: %w", err)
	}

	txQueries := r.queries.WithTx(tx)

	answer, err := txQueries.UpsertAnswer(ctx, sqlcgen.UpsertAnswerParams{
		UserID:      userID,
		QuestionID:  inputQuestionID,
		UserAnswer:  input.UserAnswer,
		IsCorrect:   input.IsCorrect,
		Score:       toNullInt32(input.Score),
		Feedback:    toNullString(input.Feedback),
		GraderModel: toNullString(input.GraderModel),
	})
	if err != nil {
		return nil, fmt.Errorf("answer repo: upsert: %w", err)
	}

	if err := txQueries.UpdateQuestionStats(ctx, sqlcgen.UpdateQuestionStatsParams{
		ID:        statsQuestionID,
		IsCorrect: isCorrect,
	}); err != nil {
		return nil, fmt.Errorf("answer repo: update stats: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("answer repo: commit: %w", err)
	}

	return toDomainAnswer(answer), nil
}

func toDomainAnswer(answer sqlcgen.Answer) *domain.Answer {
	return &domain.Answer{
		ID:          answer.ID.String(),
		UserID:      answer.UserID.String(),
		QuestionID:  answer.QuestionID.String(),
		UserAnswer:  answer.UserAnswer,
		IsCorrect:   answer.IsCorrect,
		Score:       fromNullInt32(answer.Score),
		Feedback:    fromNullString(answer.Feedback),
		GraderModel: fromNullString(answer.GraderModel),
		CreatedAt:   answer.CreatedAt,
		UpdatedAt:   answer.UpdatedAt,
	}
}

func parseAnswerUUID(value string) (uuid.UUID, error) {
	return uuid.Parse(value)
}

func toNullInt32(value *int) sql.NullInt32 {
	if value == nil {
		return sql.NullInt32{}
	}

	return sql.NullInt32{Int32: int32(*value), Valid: true}
}

func fromNullInt32(value sql.NullInt32) *int {
	if !value.Valid {
		return nil
	}

	converted := int(value.Int32)
	return &converted
}
