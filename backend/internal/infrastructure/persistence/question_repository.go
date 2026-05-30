package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

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

type questionExecer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type questionInsertRow struct {
	QuestionID    uuid.UUID
	UserID        uuid.UUID
	SourceType    string
	QuestionType  string
	Body          string
	Options       pqtype.NullRawMessage
	CorrectAnswer string
	Explanation   sql.NullString
	IsAIGenerated bool
	GenerationID  uuid.NullUUID
	HighlightID   uuid.NullUUID
	Perspective   string
	Version       int
}

func NewQuestionRepository(db *sql.DB) domain.QuestionRepository {
	return &questionRepository{
		db:      db,
		queries: sqlcgen.New(db),
	}
}

func (r *questionRepository) Save(ctx context.Context, q *domain.Question, meta *domain.QuestionMeta) error {
	if err := insertQuestion(ctx, r.db, q, meta); err != nil {
		return fmt.Errorf("question repo: save: %w", err)
	}
	return nil
}

func insertQuestion(ctx context.Context, execer questionExecer, q *domain.Question, meta *domain.QuestionMeta) error {
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

	query := `
INSERT INTO questions (
    id,
    user_id,
    source_type,
    question_type,
    body,
    options,
    correct_answer,
    explanation,
    is_ai_generated,
    generation_id,
    highlight_id,
    perspective,
    version
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
)`

	if _, err := execer.ExecContext(ctx, query,
		questionID,
		creatorID,
		string(meta.SourceType),
		string(q.QuestionType),
		q.Content,
		pqtype.NullRawMessage{RawMessage: optionsJSON, Valid: true},
		q.CorrectAnswer,
		sql.NullString{String: q.Explanation, Valid: strings.TrimSpace(q.Explanation) != ""},
		meta.IsAIGenerated,
		parseOptionalUUID(meta.GenerationID),
		parseOptionalUUID(meta.HighlightID),
		normalizePerspective(meta.Perspective),
		normalizeQuestionVersion(meta.Version),
	); err != nil {
		return err
	}

	return nil
}

func (r *questionRepository) SupersedeActiveQuestionsForHighlight(ctx context.Context, userID uuid.UUID, highlightID uuid.UUID) error {
	if err := supersedeActiveQuestionsForHighlight(ctx, r.db, userID, highlightID); err != nil {
		return fmt.Errorf("question repo: supersede active questions for highlight: %w", err)
	}
	return nil
}

func supersedeActiveQuestionsForHighlight(ctx context.Context, execer questionExecer, userID uuid.UUID, highlightID uuid.UUID) error {
	_, err := execer.ExecContext(ctx, `
UPDATE questions
SET superseded_at = NOW(),
    updated_at = NOW()
WHERE user_id = $1
  AND highlight_id = $2
  AND superseded_at IS NULL
`, userID, highlightID)
	if err != nil {
		return err
	}
	return nil
}

func supersedeActiveQuestionsForHighlights(ctx context.Context, execer questionExecer, userID uuid.UUID, highlightIDs []uuid.UUID) error {
	if len(highlightIDs) == 0 {
		return nil
	}

	_, err := execer.ExecContext(ctx, `
UPDATE questions
SET superseded_at = NOW(),
    updated_at = NOW()
WHERE user_id = $1
  AND highlight_id::text = ANY($2)
  AND superseded_at IS NULL
`, userID, pq.Array(uuidTextSlice(uniqueUUIDs(highlightIDs))))
	return err
}

func buildQuestionInsertRows(replacements []domain.QuestionReplacement) ([]questionInsertRow, []uuid.UUID, error) {
	rows := make([]questionInsertRow, 0, len(replacements))
	highlightIDs := make([]uuid.UUID, 0, len(replacements))

	for _, replacement := range replacements {
		if replacement.Question == nil || replacement.Meta == nil || replacement.HighlightID == uuid.Nil {
			return nil, nil, domain.ErrInvalidInput
		}

		row, err := buildQuestionInsertRow(replacement.Question, replacement.Meta)
		if err != nil {
			return nil, nil, err
		}
		rows = append(rows, row)
		highlightIDs = append(highlightIDs, replacement.HighlightID)
	}

	return rows, highlightIDs, nil
}

func buildQuestionInsertRow(q *domain.Question, meta *domain.QuestionMeta) (questionInsertRow, error) {
	optionsJSON, err := json.Marshal(q.Options)
	if err != nil {
		return questionInsertRow{}, fmt.Errorf("question repo: marshal options: %w", err)
	}

	questionID, err := uuid.Parse(q.ID)
	if err != nil {
		return questionInsertRow{}, fmt.Errorf("question repo: parse question id: %w", err)
	}

	creatorID, err := uuid.Parse(meta.CreatorID)
	if err != nil {
		return questionInsertRow{}, fmt.Errorf("question repo: parse creator id: %w", err)
	}

	return questionInsertRow{
		QuestionID:    questionID,
		UserID:        creatorID,
		SourceType:    string(meta.SourceType),
		QuestionType:  string(q.QuestionType),
		Body:          q.Content,
		Options:       pqtype.NullRawMessage{RawMessage: optionsJSON, Valid: true},
		CorrectAnswer: q.CorrectAnswer,
		Explanation:   sql.NullString{String: q.Explanation, Valid: strings.TrimSpace(q.Explanation) != ""},
		IsAIGenerated: meta.IsAIGenerated,
		GenerationID:  parseOptionalUUID(meta.GenerationID),
		HighlightID:   parseOptionalUUID(meta.HighlightID),
		Perspective:   normalizePerspective(meta.Perspective),
		Version:       normalizeQuestionVersion(meta.Version),
	}, nil
}

func insertQuestions(ctx context.Context, execer questionExecer, rows []questionInsertRow) error {
	if len(rows) == 0 {
		return nil
	}

	var builder strings.Builder
	builder.WriteString(`
INSERT INTO questions (
    id,
    user_id,
    source_type,
    question_type,
    body,
    options,
    correct_answer,
    explanation,
    is_ai_generated,
    generation_id,
    highlight_id,
    perspective,
    version
) VALUES `)

	args := make([]any, 0, len(rows)*13)
	for index, row := range rows {
		if index > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString("(")
		start := index*13 + 1
		for offset := 0; offset < 13; offset++ {
			if offset > 0 {
				builder.WriteString(", ")
			}
			builder.WriteString(fmt.Sprintf("$%d", start+offset))
		}
		builder.WriteString(")")
		args = append(args,
			row.QuestionID,
			row.UserID,
			row.SourceType,
			row.QuestionType,
			row.Body,
			row.Options,
			row.CorrectAnswer,
			row.Explanation,
			row.IsAIGenerated,
			row.GenerationID,
			row.HighlightID,
			row.Perspective,
			row.Version,
		)
	}

	if _, err := execer.ExecContext(ctx, builder.String(), args...); err != nil {
		return err
	}
	return nil
}

func rollbackTx(tx *sql.Tx) {
	if tx != nil {
		_ = tx.Rollback()
	}
}

func (r *questionRepository) ReplaceActiveQuestionsForHighlights(ctx context.Context, userID uuid.UUID, replacements []domain.QuestionReplacement) error {
	if len(replacements) == 0 {
		return nil
	}
	rows, highlightIDs, err := buildQuestionInsertRows(replacements)
	if err != nil {
		return fmt.Errorf("question repo: build replacement questions: %w", err)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("question repo: begin replace active questions transaction: %w", err)
	}
	defer rollbackTx(tx)

	if err := supersedeActiveQuestionsForHighlights(ctx, tx, userID, highlightIDs); err != nil {
		return fmt.Errorf("question repo: supersede replacement questions: %w", err)
	}
	if err := insertQuestions(ctx, tx, rows); err != nil {
		return fmt.Errorf("question repo: insert replacement questions: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("question repo: commit replace active questions transaction: %w", err)
	}
	return nil
}

func (r *questionRepository) CompleteQuestionGenerationJob(
	ctx context.Context,
	userID uuid.UUID,
	jobID uuid.UUID,
	replacements []domain.QuestionReplacement,
	highlightIDs []uuid.UUID,
) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("question repo: begin complete generation job transaction: %w", err)
	}
	defer rollbackTx(tx)

	rows, replacementHighlightIDs, err := buildQuestionInsertRows(replacements)
	if err != nil {
		return fmt.Errorf("question repo: build job questions: %w", err)
	}
	if err := supersedeActiveQuestionsForHighlights(ctx, tx, userID, replacementHighlightIDs); err != nil {
		return fmt.Errorf("question repo: supersede job questions: %w", err)
	}
	if err := insertQuestions(ctx, tx, rows); err != nil {
		return fmt.Errorf("question repo: insert job questions: %w", err)
	}

	if len(highlightIDs) > 0 {
		if _, err := tx.ExecContext(ctx, `
UPDATE highlights
SET
    status = 'completed',
    retry_count = 0,
    processing_started_at = NULL,
    completed_at = NOW(),
    failed_at = NULL,
    last_error = NULL,
    updated_at = NOW()
WHERE user_id = $1
  AND id::text = ANY($2)
`, userID, pq.Array(uuidStrings(highlightIDs))); err != nil {
			return fmt.Errorf("question repo: mark job highlights completed: %w", err)
		}
	}

	if _, err := tx.ExecContext(ctx, `
UPDATE question_generation_jobs
SET status = $3,
    last_error = NULL,
    processing_started_at = NULL,
    completed_at = NOW(),
    failed_at = NULL,
    updated_at = NOW()
WHERE id = $1
  AND user_id = $2
`, jobID, userID, string(domain.JobStatusCompleted)); err != nil {
		return fmt.Errorf("question repo: mark generation job completed: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("question repo: commit complete generation job transaction: %w", err)
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

func (r *questionRepository) ListPreparedByUserIDAndHighlightIDs(ctx context.Context, userID string, highlightIDs []uuid.UUID, limit int) ([]*domain.Question, error) {
	if len(highlightIDs) == 0 {
		return make([]*domain.Question, 0), nil
	}
	if limit <= 0 {
		limit = 20
	}

	uID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("question repo: parse user id for prepared questions: %w", err)
	}

	query := `
SELECT q.id,
       q.question_type,
       q.body,
       q.options,
       q.correct_answer,
       q.explanation
FROM questions q
LEFT JOIN answers a
  ON a.question_id = q.id
 AND a.user_id = $1
WHERE q.user_id = $1
  AND q.superseded_at IS NULL
  AND q.highlight_id::text = ANY($2)
ORDER BY CASE WHEN a.question_id IS NULL THEN 0 ELSE 1 END ASC, q.created_at DESC
LIMIT $3`

	rows, err := r.db.QueryContext(ctx, query, uID, pq.Array(uuidTextSlice(highlightIDs)), limit)
	if err != nil {
		return nil, fmt.Errorf("question repo: list prepared by highlight ids: %w", err)
	}
	defer rows.Close()

	questions := make([]*domain.Question, 0)
	for rows.Next() {
		var (
			qID           uuid.UUID
			questionType  string
			body          string
			optionsJSON   []byte
			correctAnswer string
			explanation   sql.NullString
		)

		if err := rows.Scan(&qID, &questionType, &body, &optionsJSON, &correctAnswer, &explanation); err != nil {
			return nil, fmt.Errorf("question repo: scan prepared question: %w", err)
		}

		questions = append(questions, &domain.Question{
			ID:            qID.String(),
			QuestionType:  domain.QuestionType(questionType),
			Content:       body,
			Options:       decodeQuestionOptionsBytes(optionsJSON),
			CorrectAnswer: correctAnswer,
			Explanation:   nullStringOrEmpty(explanation),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("question repo: rows prepared question: %w", err)
	}

	return questions, nil
}

func (r *questionRepository) ListPerspectivesByHighlightID(ctx context.Context, userID string, highlightID uuid.UUID) ([]string, error) {
	uID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("question repo: parse user id for perspectives: %w", err)
	}

	query := `
SELECT perspective
FROM questions
WHERE user_id = $1
  AND highlight_id = $2
  AND perspective <> ''
ORDER BY created_at ASC`

	rows, err := r.db.QueryContext(ctx, query, uID, highlightID)
	if err != nil {
		return nil, fmt.Errorf("question repo: list perspectives: %w", err)
	}
	defer rows.Close()

	perspectives := make([]string, 0)
	for rows.Next() {
		var perspective sql.NullString
		if err := rows.Scan(&perspective); err != nil {
			return nil, fmt.Errorf("question repo: scan perspective: %w", err)
		}
		if perspective.Valid && strings.TrimSpace(perspective.String) != "" {
			perspectives = append(perspectives, strings.TrimSpace(perspective.String))
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("question repo: rows perspective: %w", err)
	}

	return perspectives, nil
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
	err = r.db.QueryRowContext(ctx, query, uID, sourceType, sID, promptUsedForStorage(promptUsed), modelUsed).Scan(&genID)
	if err != nil {
		return "", fmt.Errorf("question repo: save generation: %w", err)
	}
	return genID.String(), nil
}

func promptUsedForStorage(promptUsed string) string {
	if promptUsed == "" {
		return ""
	}
	return "[redacted]"
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

func (r *questionRepository) GetDailyGeneratedCount(ctx context.Context, userID uuid.UUID, day time.Time) (int, error) {
	query := `
SELECT count
FROM user_daily_generation_counts
WHERE user_id = $1
  AND date = $2`

	var count int
	err := r.db.QueryRowContext(ctx, query, userID, day.Format("2006-01-02")).Scan(&count)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, fmt.Errorf("question repo: get daily generated count: %w", err)
	}

	return count, nil
}

func (r *questionRepository) ReserveDailyGeneratedCount(ctx context.Context, userID uuid.UUID, day time.Time, delta int, limit int) (bool, error) {
	if delta <= 0 {
		return true, nil
	}
	if limit <= 0 {
		return false, nil
	}

	query := `
INSERT INTO user_daily_generation_counts (user_id, date, count)
VALUES ($1, $2, $3)
ON CONFLICT (user_id, date)
DO UPDATE SET
    count = user_daily_generation_counts.count + EXCLUDED.count
WHERE user_daily_generation_counts.count + EXCLUDED.count <= $4
RETURNING count`

	var count int
	err := r.db.QueryRowContext(ctx, query, userID, day.Format("2006-01-02"), delta, limit).Scan(&count)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("question repo: reserve daily generated count: %w", err)
	}

	return true, nil
}

func (r *questionRepository) ReleaseDailyGeneratedCount(ctx context.Context, userID uuid.UUID, day time.Time, delta int) error {
	if delta <= 0 {
		return nil
	}

	if _, err := r.db.ExecContext(ctx, `
UPDATE user_daily_generation_counts
SET count = GREATEST(count - $3, 0)
WHERE user_id = $1
  AND date = $2
`, userID, day.Format("2006-01-02"), delta); err != nil {
		return fmt.Errorf("question repo: release daily generated count: %w", err)
	}
	return nil
}

func (r *questionRepository) GetUserLastQuestionSyncAt(ctx context.Context, userID uuid.UUID) (*time.Time, error) {
	var lastSync sql.NullTime
	if err := r.db.QueryRowContext(ctx, `
SELECT last_sync_at
FROM users
WHERE id = $1
`, userID).Scan(&lastSync); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("question repo: get user last question sync at: %w", err)
	}
	if !lastSync.Valid {
		return nil, nil
	}
	return &lastSync.Time, nil
}

func (r *questionRepository) UpdateUserLastQuestionSyncAt(ctx context.Context, userID uuid.UUID, syncedAt time.Time) error {
	result, err := r.db.ExecContext(ctx, `
UPDATE users
SET last_sync_at = $2,
    updated_at = NOW()
WHERE id = $1
`, userID, syncedAt.UTC())
	if err != nil {
		return fmt.Errorf("question repo: update user last question sync at: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("question repo: update user last question sync rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return domain.ErrNotFound
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

func normalizePerspective(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return domain.QuestionPerspectiveDefinition
	}
	return trimmed
}

func normalizeQuestionVersion(value int) int {
	if value <= 0 {
		return 1
	}
	return value
}

func uuidTextSlice(values []uuid.UUID) []string {
	items := make([]string, 0, len(values))
	for _, value := range values {
		items = append(items, value.String())
	}
	return items
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
