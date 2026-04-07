package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

type questionRepository struct {
	db *sql.DB
}

func NewQuestionRepository(db *sql.DB) domain.QuestionRepository {
	return &questionRepository{db: db}
}

func (r *questionRepository) Save(ctx context.Context, q *domain.Question, meta *domain.QuestionMeta) error {
	optionsJSON, err := json.Marshal(q.Options)
	if err != nil {
		return fmt.Errorf("question repo: marshal options: %w", err)
	}

	var generationID *uuid.UUID
	if meta.GenerationID != "" {
		id, err := uuid.Parse(meta.GenerationID)
		if err == nil {
			generationID = &id
		}
	}

	creatorID, err := uuid.Parse(meta.CreatorID)
	if err != nil {
		return fmt.Errorf("question repo: parse creator id: %w", err)
	}

	query := `
INSERT INTO questions (
    id, user_id, source_type, question_type,
    body, options, correct_answer, explanation,
    is_ai_generated, generation_id
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	qID := uuid.MustParse(q.ID)
	_, err = r.db.ExecContext(ctx, query,
		qID, creatorID, string(meta.SourceType), string(q.QuestionType),
		q.Content, optionsJSON, q.CorrectAnswer, q.Explanation,
		meta.IsAIGenerated, generationID,
	)
	return err
}

func (r *questionRepository) FindByID(ctx context.Context, id string) (*domain.Question, *domain.QuestionMeta, *domain.QuestionStats, error) {
	query := `
SELECT id, user_id, source_type, question_type,
       body, options, correct_answer, explanation,
       is_ai_generated, generation_id,
       answer_count, correct_count
FROM questions WHERE id = $1 LIMIT 1`

	row := r.db.QueryRowContext(ctx, query, id)

	var (
		qID, userID                       uuid.UUID
		sourceType, questionType         string
		body, correctAnswer, explanation string
		optionsJSON                      []byte
		isAIGenerated                    bool
		generationID                     *uuid.UUID
		answerCount, correctCount        int
	)

	err := row.Scan(
		&qID, &userID, &sourceType, &questionType,
		&body, &optionsJSON, &correctAnswer, &explanation,
		&isAIGenerated, &generationID,
		&answerCount, &correctCount,
	)
	if err != nil {
		return nil, nil, nil, err
	}

	var options []string
	if err := json.Unmarshal(optionsJSON, &options); err != nil {
		options = []string{}
	}

	genID := ""
	if generationID != nil {
		genID = generationID.String()
	}

	q := &domain.Question{
		ID:            qID.String(),
		QuestionType:  domain.QuestionType(questionType),
		Content:       body,
		Options:       options,
		CorrectAnswer: correctAnswer,
		Explanation:   explanation,
	}
	meta := &domain.QuestionMeta{
		QuestionID:    qID.String(),
		CreatorID:     userID.String(),
		SourceType:    domain.SourceType(sourceType),
		GenerationID:  genID,
		IsAIGenerated: isAIGenerated,
	}
	stats := &domain.QuestionStats{
		QuestionID:   qID.String(),
		AnswerCount:  answerCount,
		CorrectCount: correctCount,
	}

	return q, meta, stats, nil
}

func (r *questionRepository) UpdateStats(ctx context.Context, questionID string, isCorrect bool) error {
	query := `
UPDATE questions
SET
    answer_count  = answer_count + 1,
    correct_count = correct_count + CASE WHEN $2 THEN 1 ELSE 0 END,
    updated_at    = NOW()
WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, questionID, isCorrect)
	return err
}

func (r *questionRepository) SaveGeneration(ctx context.Context, userID, sourceType, sourceID, promptUsed, modelUsed string) (string, error) {
	query := `
INSERT INTO question_generations (user_id, source_type, source_id, prompt_used, model_used)
VALUES ($1, $2, $3, $4, $5)
RETURNING id`

	uID, err := uuid.Parse(userID)
	if err != nil {
		return "", fmt.Errorf("question repo: parse user id: %w", err)
	}

	var sID *uuid.UUID
	if sourceID != "" {
		id, err := uuid.Parse(sourceID)
		if err == nil {
			sID = &id
		}
	}

	var genID uuid.UUID
	err = r.db.QueryRowContext(ctx, query, uID, sourceType, sID, promptUsed, modelUsed).Scan(&genID)
	if err != nil {
		return "", fmt.Errorf("question repo: save generation: %w", err)
	}
	return genID.String(), nil
}
