package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/shout/ai-study-tool/backend/internal/domain"
	"github.com/shout/ai-study-tool/backend/internal/repository/sqlcgen"
	"github.com/sqlc-dev/pqtype"
)

type questionRepository struct {
	db      *sql.DB
	queries *sqlcgen.Queries
}

func NewQuestionRepository(db *sql.DB) domain.QuestionRepository {
	return &questionRepository{
		db:      db,
		queries: sqlcgen.New(db),
	}
}

func (r *questionRepository) Save(ctx context.Context, q *domain.Question, meta *domain.QuestionMeta) error {
	optionsJSON, err := json.Marshal(q.Options)
	if err != nil {
		return fmt.Errorf("question repo: marshal options: %w", err)
	}

	questionID, err := uuid.Parse(q.ID)
	if err != nil {
		return fmt.Errorf("question repo: parse question id: %w", err)
	}

	creatorID, err := uuid.Parse(meta.CreatorID)
	if err != nil {
		return fmt.Errorf("question repo: parse creator id: %w", err)
	}

	if err := r.queries.CreateQuestion(ctx, sqlcgen.CreateQuestionParams{
		ID:            questionID,
		UserID:        creatorID,
		SourceType:    string(meta.SourceType),
		QuestionType:  string(q.QuestionType),
		Body:          q.Content,
		Options:       pqtype.NullRawMessage{RawMessage: optionsJSON, Valid: true},
		CorrectAnswer: q.CorrectAnswer,
		Explanation:   sql.NullString{String: q.Explanation, Valid: true},
		IsAiGenerated: meta.IsAIGenerated,
		GenerationID:  parseOptionalUUID(meta.GenerationID),
		HighlightID:   parseOptionalUUID(meta.HighlightID),
	}); err != nil {
		return fmt.Errorf("question repo: save: %w", err)
	}

	return nil
}

func (r *questionRepository) ListByCreatorID(ctx context.Context, creatorID string, limit int) ([]*domain.Question, error) {
	userID, err := uuid.Parse(creatorID)
	if err != nil {
		return nil, fmt.Errorf("question repo: parse creator id: %w", err)
	}

	rows, err := r.queries.ListQuestionsByCreatorID(ctx, sqlcgen.ListQuestionsByCreatorIDParams{
		UserID: userID,
		Limit:  normalizeQuestionLimit(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("question repo: list by creator id: %w", err)
	}

	questions := make([]*domain.Question, 0, len(rows))
	for _, row := range rows {
		questions = append(questions, &domain.Question{
			ID:            row.ID.String(),
			QuestionType:  domain.QuestionType(row.QuestionType),
			Content:       row.Body,
			Options:       decodeQuestionOptionsMessage(row.Options),
			CorrectAnswer: row.CorrectAnswer,
			Explanation:   nullStringOrEmpty(row.Explanation),
		})
	}

	return questions, nil
}

func (r *questionRepository) ListSavedByUserID(ctx context.Context, userID string, limit int) ([]*domain.SavedQuestion, error) {
	if limit <= 0 {
		limit = 50
	}

	uID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("question repo: parse user id for saved list: %w", err)
	}

	query := `
SELECT q.id, q.question_type, q.body, q.options, q.correct_answer, q.explanation, sq.note, sq.updated_at
FROM saved_questions sq
JOIN questions q ON q.id = sq.question_id
WHERE sq.user_id = $1
ORDER BY sq.updated_at DESC
LIMIT $2`

	rows, err := r.db.QueryContext(ctx, query, uID, limit)
	if err != nil {
		return nil, fmt.Errorf("question repo: list saved by user id: %w", err)
	}
	defer rows.Close()

	savedQuestions := make([]*domain.SavedQuestion, 0)
	for rows.Next() {
		var (
			qID           uuid.UUID
			questionType  string
			body          string
			optionsJSON   []byte
			correctAnswer string
			explanation   string
			note          sql.NullString
			savedAt       sql.NullTime
		)

		if err := rows.Scan(&qID, &questionType, &body, &optionsJSON, &correctAnswer, &explanation, &note, &savedAt); err != nil {
			return nil, fmt.Errorf("question repo: scan saved list: %w", err)
		}

		savedQuestions = append(savedQuestions, &domain.SavedQuestion{
			Question: domain.Question{
				ID:            qID.String(),
				QuestionType:  domain.QuestionType(questionType),
				Content:       body,
				Options:       decodeQuestionOptionsBytes(optionsJSON),
				CorrectAnswer: correctAnswer,
				Explanation:   explanation,
			},
			Note:    note.String,
			SavedAt: savedAt.Time,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("question repo: rows saved list: %w", err)
	}

	return savedQuestions, nil
}

func (r *questionRepository) ListIncorrectByUserID(ctx context.Context, userID string, limit int) ([]*domain.IncorrectQuestion, error) {
	if limit <= 0 {
		limit = 50
	}

	uID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("question repo: parse user id for incorrect list: %w", err)
	}

	query := `
SELECT q.id, q.question_type, q.body, q.options, q.correct_answer, q.explanation,
       sq.note, COALESCE(a.updated_at, a.created_at)
FROM answers a
JOIN questions q ON q.id = a.question_id
LEFT JOIN saved_questions sq ON sq.user_id = a.user_id AND sq.question_id = a.question_id
WHERE a.user_id = $1
  AND a.is_correct = FALSE
ORDER BY COALESCE(a.updated_at, a.created_at) DESC
LIMIT $2`

	rows, err := r.db.QueryContext(ctx, query, uID, limit)
	if err != nil {
		return nil, fmt.Errorf("question repo: list incorrect by user id: %w", err)
	}
	defer rows.Close()

	incorrectQuestions := make([]*domain.IncorrectQuestion, 0)
	for rows.Next() {
		var (
			qID           uuid.UUID
			questionType  string
			body          string
			optionsJSON   []byte
			correctAnswer string
			explanation   string
			note          sql.NullString
			answeredAt    sql.NullTime
		)

		if err := rows.Scan(&qID, &questionType, &body, &optionsJSON, &correctAnswer, &explanation, &note, &answeredAt); err != nil {
			return nil, fmt.Errorf("question repo: scan incorrect list: %w", err)
		}

		incorrectQuestions = append(incorrectQuestions, &domain.IncorrectQuestion{
			Question: domain.Question{
				ID:            qID.String(),
				QuestionType:  domain.QuestionType(questionType),
				Content:       body,
				Options:       decodeQuestionOptionsBytes(optionsJSON),
				CorrectAnswer: correctAnswer,
				Explanation:   explanation,
			},
			Note:       note.String,
			AnsweredAt: answeredAt.Time,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("question repo: rows incorrect list: %w", err)
	}

	return incorrectQuestions, nil
}

func (r *questionRepository) ListUsedHighlightIDsByUserID(ctx context.Context, userID string, highlightIDs []uuid.UUID) ([]uuid.UUID, error) {
	if len(highlightIDs) == 0 {
		return make([]uuid.UUID, 0), nil
	}

	uID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("question repo: parse user id for used highlights: %w", err)
	}

	highlightIDStrings := make([]string, 0, len(highlightIDs))
	for _, highlightID := range highlightIDs {
		highlightIDStrings = append(highlightIDStrings, highlightID.String())
	}

	query := `
SELECT DISTINCT highlight_id
FROM questions
WHERE user_id = $1
  AND highlight_id IS NOT NULL
  AND highlight_id::text = ANY($2)`

	rows, err := r.db.QueryContext(ctx, query, uID, pq.Array(highlightIDStrings))
	if err != nil {
		return nil, fmt.Errorf("question repo: list used highlight ids: %w", err)
	}
	defer rows.Close()

	usedHighlightIDs := make([]uuid.UUID, 0)
	for rows.Next() {
		var highlightID uuid.UUID
		if err := rows.Scan(&highlightID); err != nil {
			return nil, fmt.Errorf("question repo: scan used highlight id: %w", err)
		}
		usedHighlightIDs = append(usedHighlightIDs, highlightID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("question repo: rows used highlight ids: %w", err)
	}

	return usedHighlightIDs, nil
}

func (r *questionRepository) FindByID(ctx context.Context, id string) (*domain.Question, *domain.QuestionMeta, *domain.QuestionStats, error) {
	questionID, err := uuid.Parse(id)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("question repo: parse question id: %w", err)
	}

	row, err := r.queries.FindQuestionByID(ctx, questionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, nil, domain.ErrNotFound
		}
		return nil, nil, nil, fmt.Errorf("question repo: find by id: %w", err)
	}

	q := &domain.Question{
		ID:            row.ID.String(),
		QuestionType:  domain.QuestionType(row.QuestionType),
		Content:       row.Body,
		Options:       decodeQuestionOptionsMessage(row.Options),
		CorrectAnswer: row.CorrectAnswer,
		Explanation:   nullStringOrEmpty(row.Explanation),
	}
	meta := &domain.QuestionMeta{
		QuestionID:    row.ID.String(),
		CreatorID:     row.UserID.String(),
		SourceType:    domain.SourceType(row.SourceType),
		HighlightID:   nullUUIDString(row.HighlightID),
		GenerationID:  nullUUIDString(row.GenerationID),
		IsAIGenerated: row.IsAiGenerated,
	}
	stats := &domain.QuestionStats{
		QuestionID:   row.ID.String(),
		AnswerCount:  int(row.AnswerCount),
		CorrectCount: int(row.CorrectCount),
	}

	return q, meta, stats, nil
}

func (r *questionRepository) GetByID(ctx context.Context, id string) (*domain.Question, error) {
	questionID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("question repo: parse question id: %w", err)
	}

	row, err := r.queries.GetQuestionByID(ctx, questionID)
	if err != nil {
		return nil, wrapQuestionError("get by id", err)
	}

	return &domain.Question{
		ID:            row.ID.String(),
		QuestionType:  domain.QuestionType(row.QuestionType),
		Content:       row.Body,
		Options:       decodeQuestionOptionsMessage(row.Options),
		CorrectAnswer: row.CorrectAnswer,
		Explanation:   nullStringOrEmpty(row.Explanation),
	}, nil
}

func (r *questionRepository) UpdateStats(ctx context.Context, questionID string, isCorrect bool) error {
	id, err := uuid.Parse(questionID)
	if err != nil {
		return fmt.Errorf("question repo: parse question id for update stats: %w", err)
	}

	if err := r.queries.UpdateQuestionStats(ctx, sqlcgen.UpdateQuestionStatsParams{
		ID:        id,
		IsCorrect: isCorrect,
	}); err != nil {
		return fmt.Errorf("question repo: update stats: %w", err)
	}

	return nil
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

func (r *questionRepository) SaveForUser(ctx context.Context, userID, questionID, note string) error {
	uID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("question repo: parse user id for save: %w", err)
	}

	qID, err := uuid.Parse(questionID)
	if err != nil {
		return fmt.Errorf("question repo: parse question id for save: %w", err)
	}

	if err := r.queries.SaveQuestionForUser(ctx, sqlcgen.SaveQuestionForUserParams{
		UserID:     uID,
		QuestionID: qID,
		Note:       sql.NullString{String: note, Valid: true},
	}); err != nil {
		return fmt.Errorf("question repo: save for user: %w", err)
	}

	return nil
}

func wrapQuestionError(action string, err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("question repo: %s: %w", action, domain.ErrNotFound)
	}

	return fmt.Errorf("question repo: %s: %w", action, err)
}

func normalizeQuestionLimit(limit int) int32 {
	if limit <= 0 {
		return 50
	}

	return int32(limit)
}

func parseOptionalUUID(value string) uuid.NullUUID {
	if value == "" {
		return uuid.NullUUID{}
	}

	parsed, err := uuid.Parse(value)
	if err != nil {
		return uuid.NullUUID{}
	}

	return uuid.NullUUID{UUID: parsed, Valid: true}
}

func nullUUIDString(value uuid.NullUUID) string {
	if !value.Valid {
		return ""
	}

	return value.UUID.String()
}

func nullStringOrEmpty(value sql.NullString) string {
	if !value.Valid {
		return ""
	}

	return value.String
}

func decodeQuestionOptionsMessage(value pqtype.NullRawMessage) []string {
	if !value.Valid {
		return []string{}
	}

	return decodeQuestionOptionsBytes(value.RawMessage)
}

func decodeQuestionOptionsBytes(value []byte) []string {
	options := make([]string, 0)
	if len(value) == 0 {
		return options
	}

	if err := json.Unmarshal(value, &options); err != nil {
		return []string{}
	}

	return options
}
