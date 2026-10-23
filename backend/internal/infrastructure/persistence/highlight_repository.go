package persistence

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/shout/ai-study-tool/backend/internal/domain"
	"github.com/shout/ai-study-tool/backend/internal/repository/sqlcgen"
)

type highlightRepository struct {
	db      *sql.DB
	queries *sqlcgen.Queries
}

const bookKeyExpressionSQL = `
COALESCE(
    NULLIF(trim(coalesce(h.book_key, '')), ''),
    NULLIF(trim(coalesce(h.asin, '')), ''),
    concat('metadata:', trim(coalesce(h.book_title, '')), ':', trim(coalesce(h.book_author, '')))
)
`

const questionTargetExpressionSQL = `
CASE
    WHEN char_length(trim(coalesce(h.content, ''))) = 0 THEN 0
    WHEN char_length(trim(coalesce(h.content, ''))) < 120 THEN 1
    WHEN char_length(trim(coalesce(h.content, ''))) < 320 THEN 2
    ELSE 3
END
`

func NewHighlightRepository(db *sql.DB) domain.HighlightRepository {
	return &highlightRepository{
		db:      db,
		queries: sqlcgen.New(db),
	}
}

func (r *highlightRepository) BulkUpsert(ctx context.Context, highlights []*domain.Highlight) (saved int, err error) {
	if len(highlights) == 0 {
		return 0, nil
	}

	query, args, hashIndex, err := buildHighlightBulkUpsert(highlights)
	if err != nil {
		return 0, fmt.Errorf("highlight repo: build bulk upsert: %w", err)
	}

	saved, err = r.executeBulkUpsertQuery(ctx, query, args, hashIndex)
	if err != nil {
		return 0, err
	}

	return saved, nil
}

func (r *highlightRepository) executeBulkUpsertQuery(
	ctx context.Context,
	query string,
	args []any,
	hashIndex map[string][]*domain.Highlight,
) (int, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("highlight repo: bulk upsert query: %w", err)
	}

	saved := 0
	savedIDs := make([]uuid.UUID, 0, len(hashIndex))
	for rows.Next() {
		var (
			id          uuid.UUID
			contentHash sql.NullString
			createdAt   time.Time
			updatedAt   time.Time
		)

		if err := rows.Scan(&id, &contentHash, &createdAt, &updatedAt); err != nil {
			rows.Close()
			return 0, fmt.Errorf("highlight repo: bulk upsert scan: %w", err)
		}

		saved++
		savedIDs = append(savedIDs, id)
		if contentHash.Valid {
			assignBulkUpsertResult(hashIndex, contentHash.String, id, createdAt, updatedAt)
		}
	}

	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, fmt.Errorf("highlight repo: bulk upsert rows: %w", err)
	}
	if err := rows.Close(); err != nil {
		return 0, fmt.Errorf("highlight repo: bulk upsert close rows: %w", err)
	}
	if saved > 0 {
		if err := r.fillMissingBookOrderIndexes(ctx, savedIDs); err != nil {
			return 0, err
		}
	}

	return saved, nil
}

func (r *highlightRepository) fillMissingBookOrderIndexes(ctx context.Context, highlightIDs []uuid.UUID) error {
	if len(highlightIDs) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("highlight repo: begin fill missing book order indexes: %w", err)
	}
	defer tx.Rollback()

	highlightIDArgs := pq.Array(uuidStrings(highlightIDs))
	if _, err := tx.ExecContext(ctx, `
	UPDATE highlights h
	SET book_key = `+bookKeyExpressionSQL+`
	WHERE NULLIF(trim(coalesce(h.book_key, '')), '') IS NULL
	  AND h.id = ANY($1::uuid[])
	`, highlightIDArgs); err != nil {
		return fmt.Errorf("highlight repo: fill missing book keys: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
	WITH missing AS (
	    SELECT id, user_id, book_key,
           ROW_NUMBER() OVER (
               PARTITION BY user_id, book_key
	               ORDER BY COALESCE(highlighted_at, created_at), created_at, id
	           ) AS rn
	    FROM highlights h
	    WHERE h.book_order_index IS NULL
	      AND h.id = ANY($1::uuid[])
	      AND NULLIF(trim(coalesce(h.book_key, '')), '') IS NOT NULL
	),
	offsets AS (
	    SELECT user_id, book_key, COALESCE(MAX(book_order_index), 0) AS offset_value
	    FROM highlights h
	    WHERE h.book_order_index IS NOT NULL
	      AND NULLIF(trim(coalesce(h.book_key, '')), '') IS NOT NULL
	      AND EXISTS (
	          SELECT 1
	          FROM missing
	          WHERE missing.user_id = h.user_id
	            AND missing.book_key = h.book_key
	      )
	    GROUP BY user_id, book_key
	)
UPDATE highlights h
SET book_order_index = missing.rn + COALESCE(offsets.offset_value, 0)
FROM missing
LEFT JOIN offsets
  ON offsets.user_id = missing.user_id
 AND offsets.book_key = missing.book_key
	WHERE h.id = missing.id
	`, highlightIDArgs)
	if err != nil {
		return fmt.Errorf("highlight repo: fill missing book order indexes: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("highlight repo: commit fill missing book order indexes: %w", err)
	}
	return nil
}

func (r *highlightRepository) ListExistingContentHashesByUserID(ctx context.Context, userID uuid.UUID, hashes []string) ([]string, error) {
	if len(hashes) == 0 {
		return make([]string, 0), nil
	}

	rows, err := r.db.QueryContext(ctx, `
SELECT content_hash
FROM highlights
WHERE user_id = $1
  AND content_hash = ANY($2)
`, userID, pq.Array(hashes))
	if err != nil {
		return nil, fmt.Errorf("highlight repo: list existing content hashes: %w", err)
	}
	defer rows.Close()

	items := make([]string, 0, len(hashes))
	for rows.Next() {
		var hash string
		if err := rows.Scan(&hash); err != nil {
			return nil, fmt.Errorf("highlight repo: scan existing content hash: %w", err)
		}
		items = append(items, hash)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("highlight repo: rows existing content hash: %w", err)
	}

	return items, nil
}

func (r *highlightRepository) FindByUserIDAndContentHash(ctx context.Context, userID uuid.UUID, contentHash string) (*domain.Highlight, error) {
	normalizedHash := strings.TrimSpace(contentHash)
	if normalizedHash == "" {
		return nil, domain.ErrInvalidInput
	}

	highlight, err := r.listOneHighlight(ctx, `
SELECT id, user_id, book_id, book_title, book_author, asin, content, explanation, content_hash, location,
       highlighted_at, source, source_app, source_url, status, retry_count, last_error,
       generation_requested_at, processing_started_at, completed_at, failed_at, book_order_index, created_at, updated_at
FROM highlights
WHERE user_id = $1
  AND content_hash = $2
ORDER BY created_at ASC, id ASC
LIMIT 1
`, userID, normalizedHash)
	if err != nil {
		return nil, fmt.Errorf("highlight repo: find by content hash: %w", err)
	}

	return highlight, nil
}

func (r *highlightRepository) ListByUserIDAndASIN(ctx context.Context, userID uuid.UUID, asin string) ([]*domain.Highlight, error) {
	return r.listHighlights(ctx, `
SELECT id, user_id, book_id, book_title, book_author, asin, content, explanation, content_hash, location,
       highlighted_at, source, source_app, source_url, status, retry_count, last_error,
       generation_requested_at, processing_started_at, completed_at, failed_at, book_order_index, created_at, updated_at
FROM highlights
WHERE user_id = $1
  AND asin = $2
  AND source = $3
ORDER BY highlighted_at ASC NULLS LAST, created_at ASC
`, userID, strings.TrimSpace(asin), domain.HighlightSourceExtension)
}

func (r *highlightRepository) ListByUserIDAndBookMetadata(ctx context.Context, userID uuid.UUID, bookTitle, bookAuthor string) ([]*domain.Highlight, error) {
	normalizedTitle := strings.TrimSpace(bookTitle)
	if normalizedTitle == "" {
		return make([]*domain.Highlight, 0), nil
	}

	normalizedAuthor := strings.TrimSpace(bookAuthor)
	if normalizedAuthor != "" {
		highlights, err := r.listHighlights(ctx, `
SELECT id, user_id, book_id, book_title, book_author, asin, content, explanation, content_hash, location,
       highlighted_at, source, source_app, source_url, status, retry_count, last_error,
       generation_requested_at, processing_started_at, completed_at, failed_at, book_order_index, created_at, updated_at
FROM highlights
WHERE user_id = $1
  AND lower(trim(coalesce(book_title, ''))) = lower(trim($2))
  AND lower(trim(coalesce(book_author, ''))) = lower(trim($3))
ORDER BY highlighted_at ASC NULLS LAST, created_at ASC
`, userID, normalizedTitle, normalizedAuthor)
		if err != nil {
			return nil, fmt.Errorf("highlight repo: list by book metadata: %w", err)
		}
		if len(highlights) > 0 {
			return highlights, nil
		}
	}

	// Kindle metadata can be incomplete or drift across clients. If exact
	// title+author matching misses, fall back to title-only matching.
	highlights, err := r.listHighlights(ctx, `
SELECT id, user_id, book_id, book_title, book_author, asin, content, explanation, content_hash, location,
       highlighted_at, source, source_app, source_url, status, retry_count, last_error,
       generation_requested_at, processing_started_at, completed_at, failed_at, book_order_index, created_at, updated_at
FROM highlights
WHERE user_id = $1
  AND lower(trim(coalesce(book_title, ''))) = lower(trim($2))
ORDER BY highlighted_at ASC NULLS LAST, created_at ASC
`, userID, normalizedTitle)
	if err != nil {
		return nil, fmt.Errorf("highlight repo: list by book title: %w", err)
	}

	return highlights, nil
}

func (r *highlightRepository) ListBooksWithHighlightsByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.KindleBook, error) {
	rows, err := r.queries.ListHighlightBooksByUserID(ctx, sqlcgen.ListHighlightBooksByUserIDParams{
		UserID: userID,
		Source: sql.NullString{String: domain.HighlightSourceExtension, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("highlight repo: list books: %w", err)
	}

	books := make([]*domain.KindleBook, 0, len(rows))
	for _, row := range rows {
		if !row.Asin.Valid || strings.TrimSpace(row.Asin.String) == "" {
			continue
		}

		books = append(books, &domain.KindleBook{
			ASIN:           row.Asin.String,
			BookTitle:      strings.TrimSpace(row.BookTitle),
			BookAuthor:     strings.TrimSpace(row.BookAuthor),
			HighlightCount: int(row.HighlightCount),
			Source:         fromNullStringValue(row.Source),
		})
	}

	return books, nil
}

func (r *highlightRepository) ListBookStockByUserID(ctx context.Context, userID uuid.UUID) ([]domain.BookStock, error) {
	query := `
WITH question_counts AS (
    SELECT highlight_id, COUNT(*)::int AS question_count
    FROM questions
    WHERE user_id = $1
      AND highlight_id IS NOT NULL
    GROUP BY highlight_id
),
book_highlights AS (
    SELECT
        ` + bookKeyExpressionSQL + ` AS book_key,
        trim(coalesce(h.book_title, '')) AS book_title,
        trim(coalesce(h.book_author, '')) AS book_author,
        h.created_at,
        h.status,
        ` + questionTargetExpressionSQL + ` AS question_target,
        COALESCE(qc.question_count, 0) AS question_count
    FROM highlights h
    LEFT JOIN question_counts qc
      ON qc.highlight_id = h.id
    WHERE h.user_id = $1
      AND (
        NULLIF(trim(coalesce(h.asin, '')), '') IS NOT NULL
        OR NULLIF(trim(coalesce(h.book_title, '')), '') IS NOT NULL
      )
)
SELECT
    book_key,
    MAX(book_title) AS book_title,
    MAX(book_author) AS book_author,
    COALESCE(SUM(question_count), 0)::int AS stock,
    COALESCE(SUM(
        CASE
            WHEN status IN ('pending', 'processing') THEN GREATEST(question_target - question_count, 0)
            ELSE 0
        END
    ), 0)::int AS preparing,
    MAX(created_at) AS latest_highlight_at
FROM book_highlights
GROUP BY book_key
ORDER BY CASE WHEN COALESCE(SUM(question_count), 0) = 0 THEN 0 ELSE 1 END ASC, MAX(created_at) DESC`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("highlight repo: list book stock: %w", err)
	}
	defer rows.Close()

	books := make([]domain.BookStock, 0)
	for rows.Next() {
		var book domain.BookStock
		if err := rows.Scan(
			&book.BookKey,
			&book.BookTitle,
			&book.BookAuthor,
			&book.Stock,
			&book.Preparing,
			&book.LatestHighlightAt,
		); err != nil {
			return nil, fmt.Errorf("highlight repo: scan book stock: %w", err)
		}
		book.BookKey = strings.TrimSpace(book.BookKey)
		book.BookTitle = strings.TrimSpace(book.BookTitle)
		book.BookAuthor = strings.TrimSpace(book.BookAuthor)
		books = append(books, book)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("highlight repo: rows book stock: %w", err)
	}

	return books, nil
}

func (r *highlightRepository) ListUnusedHighlightsByBook(ctx context.Context, userID uuid.UUID, bookKey string, limit int) ([]*domain.Highlight, error) {
	return r.listQuestionSourceHighlights(ctx, userID, bookKey, limit, true)
}

func (r *highlightRepository) ListUsedHighlightsWithUncoveredPerspectives(ctx context.Context, userID uuid.UUID, bookKey string, limit int) ([]*domain.Highlight, error) {
	return r.listQuestionSourceHighlights(ctx, userID, bookKey, limit, false)
}

func (r *highlightRepository) ListQuestionGenerationCandidates(ctx context.Context, userID uuid.UUID, changedSince *time.Time) ([]domain.QuestionGenerationBookCandidate, error) {
	changedFilter := ""
	args := []any{userID}
	if changedSince != nil {
		changedFilter = "AND h.updated_at > $2"
		args = append(args, changedSince.UTC())
	}

	query := `
WITH changed_books AS (
    SELECT DISTINCT ` + bookKeyExpressionSQL + ` AS book_key
    FROM highlights h
    WHERE h.user_id = $1
      ` + changedFilter + `
      AND (
        NULLIF(trim(coalesce(h.book_key, '')), '') IS NOT NULL
        OR NULLIF(trim(coalesce(h.asin, '')), '') IS NOT NULL
        OR NULLIF(trim(coalesce(h.book_title, '')), '') IS NOT NULL
      )
),
book_highlights AS (
    SELECT
        h.id,
        ` + bookKeyExpressionSQL + ` AS book_key,
        trim(coalesce(h.book_title, '')) AS book_title,
        trim(coalesce(h.book_author, '')) AS book_author,
        h.status,
        h.created_at,
        h.highlighted_at
    FROM highlights h
    JOIN changed_books cb
      ON cb.book_key = ` + bookKeyExpressionSQL + `
    WHERE h.user_id = $1
),
active_questions AS (
    SELECT q.id, q.highlight_id
    FROM questions q
    WHERE q.user_id = $1
      AND q.highlight_id IS NOT NULL
      AND q.superseded_at IS NULL
),
unanswered_questions AS (
    SELECT aq.highlight_id, COUNT(*)::int AS unanswered_count
    FROM active_questions aq
    LEFT JOIN answers a
      ON a.question_id = aq.id
     AND a.user_id = $1
    WHERE a.question_id IS NULL
    GROUP BY aq.highlight_id
)
SELECT
    bh.book_key,
    MAX(bh.book_title) AS book_title,
    MAX(bh.book_author) AS book_author,
    COUNT(*) FILTER (WHERE bh.status = 'pending')::int AS pending_highlight_count,
    COALESCE(SUM(uq.unanswered_count), 0)::int AS unanswered_question_count,
    MAX(COALESCE(bh.highlighted_at, bh.created_at)) AS latest_highlight_at
FROM book_highlights bh
LEFT JOIN unanswered_questions uq
  ON uq.highlight_id = bh.id
GROUP BY bh.book_key
ORDER BY MAX(COALESCE(bh.highlighted_at, bh.created_at)) DESC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("highlight repo: list question generation candidates: %w", err)
	}
	defer rows.Close()

	candidates := make([]domain.QuestionGenerationBookCandidate, 0)
	for rows.Next() {
		var candidate domain.QuestionGenerationBookCandidate
		if err := rows.Scan(
			&candidate.BookKey,
			&candidate.BookTitle,
			&candidate.BookAuthor,
			&candidate.PendingHighlightCount,
			&candidate.UnansweredQuestionCount,
			&candidate.LatestHighlightAt,
		); err != nil {
			return nil, fmt.Errorf("highlight repo: scan question generation candidate: %w", err)
		}
		candidate.BookKey = strings.TrimSpace(candidate.BookKey)
		candidate.BookTitle = strings.TrimSpace(candidate.BookTitle)
		candidate.BookAuthor = strings.TrimSpace(candidate.BookAuthor)
		candidates = append(candidates, candidate)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("highlight repo: rows question generation candidates: %w", err)
	}

	return candidates, nil
}

func (r *highlightRepository) ListQuestionGenerationCandidateByBookKey(ctx context.Context, userID uuid.UUID, bookKey string) (*domain.QuestionGenerationBookCandidate, error) {
	normalizedBookKey := strings.TrimSpace(bookKey)
	if normalizedBookKey == "" {
		return nil, domain.ErrInvalidInput
	}

	query := `
WITH book_highlights AS (
    SELECT
        h.id,
        ` + bookKeyExpressionSQL + ` AS book_key,
        trim(coalesce(h.book_title, '')) AS book_title,
        trim(coalesce(h.book_author, '')) AS book_author,
        h.status,
        h.created_at,
        h.highlighted_at
    FROM highlights h
    WHERE h.user_id = $1
      AND ` + bookKeyExpressionSQL + ` = $2
),
active_questions AS (
    SELECT q.id, q.highlight_id
    FROM questions q
    WHERE q.user_id = $1
      AND q.highlight_id IS NOT NULL
      AND q.superseded_at IS NULL
),
unanswered_questions AS (
    SELECT aq.highlight_id, COUNT(*)::int AS unanswered_count
    FROM active_questions aq
    LEFT JOIN answers a
      ON a.question_id = aq.id
     AND a.user_id = $1
    WHERE a.question_id IS NULL
    GROUP BY aq.highlight_id
)
SELECT
    bh.book_key,
    MAX(bh.book_title) AS book_title,
    MAX(bh.book_author) AS book_author,
    COUNT(*) FILTER (WHERE bh.status = 'pending')::int AS pending_highlight_count,
    COALESCE(SUM(uq.unanswered_count), 0)::int AS unanswered_question_count,
    MAX(COALESCE(bh.highlighted_at, bh.created_at)) AS latest_highlight_at
FROM book_highlights bh
LEFT JOIN unanswered_questions uq
  ON uq.highlight_id = bh.id
GROUP BY bh.book_key`

	rows, err := r.db.QueryContext(ctx, query, userID, normalizedBookKey)
	if err != nil {
		return nil, fmt.Errorf("highlight repo: list question generation candidate by book key: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("highlight repo: rows question generation candidate by book key: %w", err)
		}
		return nil, nil
	}

	var candidate domain.QuestionGenerationBookCandidate
	if err := rows.Scan(
		&candidate.BookKey,
		&candidate.BookTitle,
		&candidate.BookAuthor,
		&candidate.PendingHighlightCount,
		&candidate.UnansweredQuestionCount,
		&candidate.LatestHighlightAt,
	); err != nil {
		return nil, fmt.Errorf("highlight repo: scan question generation candidate by book key: %w", err)
	}
	candidate.BookKey = strings.TrimSpace(candidate.BookKey)
	candidate.BookTitle = strings.TrimSpace(candidate.BookTitle)
	candidate.BookAuthor = strings.TrimSpace(candidate.BookAuthor)

	if rows.Next() {
		return nil, fmt.Errorf("highlight repo: multiple candidates for book key %q", normalizedBookKey)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("highlight repo: rows question generation candidate by book key: %w", err)
	}
	return &candidate, nil
}

func (r *highlightRepository) ListPendingHighlightsByBook(ctx context.Context, userID uuid.UUID, bookKey string, limit int) ([]*domain.Highlight, error) {
	if limit <= 0 {
		limit = domain.MaxHighlightsPerJob
	}

	query := `
SELECT
    h.id, h.user_id, h.book_id, h.book_title, h.book_author, h.asin, h.content, h.explanation,
    h.content_hash, h.location, h.highlighted_at, h.source, h.source_app, h.source_url, h.status,
    h.retry_count, h.last_error, h.generation_requested_at, h.processing_started_at, h.completed_at,
    h.failed_at, h.book_order_index, h.created_at, h.updated_at
FROM highlights h
WHERE h.user_id = $1
  AND ` + bookKeyExpressionSQL + ` = $2
  AND h.status = 'pending'
  AND char_length(trim(coalesce(h.content, ''))) > 0
ORDER BY COALESCE(h.highlighted_at, h.created_at) ASC, h.created_at ASC
LIMIT $3`

	highlights, err := r.listHighlights(ctx, query, userID, strings.TrimSpace(bookKey), limit)
	if err != nil {
		return nil, fmt.Errorf("highlight repo: list pending highlights by book: %w", err)
	}
	return highlights, nil
}

func (r *highlightRepository) ListByIDs(ctx context.Context, userID uuid.UUID, highlightIDs []uuid.UUID) ([]*domain.Highlight, error) {
	if len(highlightIDs) == 0 {
		return make([]*domain.Highlight, 0), nil
	}

	query := `
SELECT
    h.id, h.user_id, h.book_id, h.book_title, h.book_author, h.asin, h.content, h.explanation,
    h.content_hash, h.location, h.highlighted_at, h.source, h.source_app, h.source_url, h.status,
    h.retry_count, h.last_error, h.generation_requested_at, h.processing_started_at, h.completed_at,
    h.failed_at, h.book_order_index, h.created_at, h.updated_at
FROM highlights h
WHERE h.user_id = $1
  AND h.id = ANY($2::uuid[])
ORDER BY h.created_at ASC`

	highlights, err := r.listHighlights(ctx, query, userID, pq.Array(uuidStrings(highlightIDs)))
	if err != nil {
		return nil, fmt.Errorf("highlight repo: list by ids: %w", err)
	}
	return highlights, nil
}

func (r *highlightRepository) MarkHighlightsProcessing(ctx context.Context, userID uuid.UUID, highlightIDs []uuid.UUID) error {
	if len(highlightIDs) == 0 {
		return nil
	}

	_, err := r.db.ExecContext(ctx, `
UPDATE highlights
SET status = 'processing',
    processing_started_at = NOW(),
    updated_at = NOW()
WHERE user_id = $1
  AND id = ANY($2::uuid[])
  AND status = 'pending'
`, userID, pq.Array(uuidStrings(highlightIDs)))
	if err != nil {
		return fmt.Errorf("highlight repo: mark highlights processing: %w", err)
	}
	return nil
}

func (r *highlightRepository) MarkHighlightsPending(ctx context.Context, userID uuid.UUID, highlightIDs []uuid.UUID) error {
	if len(highlightIDs) == 0 {
		return nil
	}

	_, err := r.db.ExecContext(ctx, `
UPDATE highlights
SET status = 'pending',
    generation_requested_at = NOW(),
    processing_started_at = NULL,
    last_error = NULL,
    updated_at = NOW()
WHERE user_id = $1
  AND id = ANY($2::uuid[])
  AND status = 'processing'
`, userID, pq.Array(uuidStrings(highlightIDs)))
	if err != nil {
		return fmt.Errorf("highlight repo: mark highlights pending: %w", err)
	}
	return nil
}

func (r *highlightRepository) MarkHighlightPendingForQuestion(ctx context.Context, userID uuid.UUID, questionID uuid.UUID) (string, error) {
	var bookKey string
	err := r.db.QueryRowContext(ctx, `
UPDATE highlights h
SET status = 'pending',
    retry_count = 0,
    generation_requested_at = NOW(),
    processing_started_at = NULL,
    completed_at = NULL,
    failed_at = NULL,
    last_error = NULL,
    updated_at = NOW()
FROM questions q
WHERE q.id = $2
  AND q.user_id = $1
  AND q.highlight_id = h.id
  AND h.user_id = $1
RETURNING `+bookKeyExpressionSQL, userID, questionID).Scan(&bookKey)
	if err != nil {
		return "", wrapHighlightError("mark highlight pending for question", err)
	}
	return strings.TrimSpace(bookKey), nil
}

func (r *highlightRepository) MarkGenerationCompleted(ctx context.Context, userID uuid.UUID, highlightIDs []uuid.UUID) error {
	if len(highlightIDs) == 0 {
		return nil
	}

	query := `
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
  AND id = ANY($2::uuid[])`

	if _, err := r.db.ExecContext(ctx, query, userID, pq.Array(uuidStrings(highlightIDs))); err != nil {
		return fmt.Errorf("highlight repo: mark generation completed: %w", err)
	}

	return nil
}

func (r *highlightRepository) MarkGenerationFailed(ctx context.Context, userID uuid.UUID, highlightIDs []uuid.UUID, lastError string, maxRetry int) error {
	if len(highlightIDs) == 0 {
		return nil
	}
	if maxRetry <= 0 {
		maxRetry = 3
	}

	query := `
UPDATE highlights
SET
    retry_count = retry_count + 1,
    status = CASE WHEN retry_count + 1 >= $4 THEN 'failed' ELSE 'pending' END,
    generation_requested_at = NOW(),
    processing_started_at = NULL,
	    failed_at = CASE WHEN retry_count + 1 >= $4 THEN NOW() ELSE failed_at END,
	    last_error = LEFT($3, 500),
	    updated_at = NOW()
WHERE user_id = $1
  AND id = ANY($2::uuid[])`

	if _, err := r.db.ExecContext(ctx, query, userID, pq.Array(uuidStrings(highlightIDs)), strings.TrimSpace(lastError), maxRetry); err != nil {
		return fmt.Errorf("highlight repo: mark generation failed: %w", err)
	}

	return nil
}

func (r *highlightRepository) UpdateExplanation(ctx context.Context, id, userID uuid.UUID, explanation *string) (*domain.Highlight, error) {
	highlight, err := r.queries.UpdateHighlightExplanation(ctx, sqlcgen.UpdateHighlightExplanationParams{
		ID:          id,
		UserID:      userID,
		Explanation: toNullString(explanation),
	})
	if err != nil {
		return nil, wrapHighlightError("update explanation", err)
	}

	return toDomainHighlight(highlight), nil
}

func (r *highlightRepository) listQuestionSourceHighlights(
	ctx context.Context,
	userID uuid.UUID,
	bookKey string,
	limit int,
	unusedOnly bool,
) ([]*domain.Highlight, error) {
	normalizedBookKey := strings.TrimSpace(bookKey)
	if normalizedBookKey == "" {
		return make([]*domain.Highlight, 0), nil
	}
	if limit <= 0 {
		limit = 10
	}

	comparisonOperator := ">"
	if unusedOnly {
		comparisonOperator = "="
	}

	query := `
WITH question_counts AS (
    SELECT highlight_id, COUNT(*)::int AS question_count
    FROM questions
    WHERE user_id = $1
      AND highlight_id IS NOT NULL
    GROUP BY highlight_id
)
SELECT
    h.id, h.user_id, h.book_id, h.book_title, h.book_author, h.asin, h.content, h.explanation,
    h.content_hash, h.location, h.highlighted_at, h.source, h.source_app, h.source_url, h.status,
    h.retry_count, h.last_error, h.generation_requested_at, h.processing_started_at, h.completed_at,
    h.failed_at, h.book_order_index, h.created_at, h.updated_at
FROM highlights h
LEFT JOIN question_counts qc
  ON qc.highlight_id = h.id
WHERE h.user_id = $1
  AND ` + bookKeyExpressionSQL + ` = $2
  AND char_length(trim(coalesce(h.content, ''))) > 0
  AND h.status NOT IN ('pending', 'processing')
  AND COALESCE(qc.question_count, 0) ` + comparisonOperator + ` 0
  AND COALESCE(qc.question_count, 0) < (` + questionTargetExpressionSQL + `)
ORDER BY COALESCE(h.highlighted_at, h.created_at) DESC, h.created_at DESC
LIMIT $3`

	highlights, err := r.listHighlights(ctx, query, userID, normalizedBookKey, limit)
	if err != nil {
		if unusedOnly {
			return nil, fmt.Errorf("highlight repo: list unused highlights by book: %w", err)
		}
		return nil, fmt.Errorf("highlight repo: list used highlights by book: %w", err)
	}

	return highlights, nil
}

func wrapHighlightError(action string, err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("highlight repo: %s: %w", action, domain.ErrNotFound)
	}

	return fmt.Errorf("highlight repo: %s: %w", action, err)
}

func (r *highlightRepository) listHighlights(ctx context.Context, query string, args ...any) ([]*domain.Highlight, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	highlights := make([]*domain.Highlight, 0)
	for rows.Next() {
		highlight, scanErr := scanHighlight(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		highlights = append(highlights, highlight)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return highlights, nil
}

func (r *highlightRepository) listOneHighlight(ctx context.Context, query string, args ...any) (*domain.Highlight, error) {
	row := r.db.QueryRowContext(ctx, query, args...)
	highlight, err := scanHighlight(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	return highlight, nil
}

type highlightScanner interface {
	Scan(dest ...any) error
}

func scanHighlight(scanner highlightScanner) (*domain.Highlight, error) {
	var (
		highlight      domain.Highlight
		bookID         uuid.NullUUID
		bookTitle      sql.NullString
		bookAuthor     sql.NullString
		asin           sql.NullString
		explanation    sql.NullString
		contentHash    sql.NullString
		location       sql.NullString
		highlightedAt  sql.NullTime
		sourceApp      sql.NullString
		sourceURL      sql.NullString
		source         sql.NullString
		status         sql.NullString
		lastError      sql.NullString
		processingAt   sql.NullTime
		completedAt    sql.NullTime
		failedAt       sql.NullTime
		requestedAt    sql.NullTime
		bookOrderIndex sql.NullInt32
	)

	if err := scanner.Scan(
		&highlight.ID,
		&highlight.UserID,
		&bookID,
		&bookTitle,
		&bookAuthor,
		&asin,
		&highlight.Content,
		&explanation,
		&contentHash,
		&location,
		&highlightedAt,
		&source,
		&sourceApp,
		&sourceURL,
		&status,
		&highlight.RetryCount,
		&lastError,
		&requestedAt,
		&processingAt,
		&completedAt,
		&failedAt,
		&bookOrderIndex,
		&highlight.CreatedAt,
		&highlight.UpdatedAt,
	); err != nil {
		return nil, err
	}

	highlight.BookID = fromNullUUID(bookID)
	highlight.BookTitle = fromNullString(bookTitle)
	highlight.BookAuthor = fromNullString(bookAuthor)
	highlight.ASIN = fromNullString(asin)
	highlight.Explanation = fromNullString(explanation)
	highlight.ContentHash = fromNullString(contentHash)
	highlight.Location = fromNullString(location)
	highlight.HighlightedAt = fromNullTime(highlightedAt)
	highlight.Source = fromNullStringValue(source)
	highlight.SourceApp = fromNullString(sourceApp)
	highlight.SourceURL = fromNullString(sourceURL)
	highlight.Status = domain.HighlightStatus(strings.TrimSpace(status.String))
	highlight.LastError = fromNullString(lastError)
	highlight.ProcessingAt = fromNullTime(processingAt)
	highlight.CompletedAt = fromNullTime(completedAt)
	highlight.FailedAt = fromNullTime(failedAt)
	highlight.BookOrderIndex = fromNullInt32AsInt(bookOrderIndex)
	if requestedAt.Valid {
		highlight.RequestedAt = requestedAt.Time
	} else {
		highlight.RequestedAt = highlight.CreatedAt
	}

	return &highlight, nil
}

func toDomainHighlight(item sqlcgen.Highlight) *domain.Highlight {
	return &domain.Highlight{
		ID:             item.ID,
		UserID:         item.UserID,
		BookID:         fromNullUUID(item.BookID),
		BookTitle:      fromNullString(item.BookTitle),
		BookAuthor:     fromNullString(item.BookAuthor),
		ASIN:           fromNullString(item.Asin),
		Content:        item.Content,
		Explanation:    fromNullString(item.Explanation),
		ContentHash:    fromNullString(item.ContentHash),
		Location:       fromNullString(item.Location),
		HighlightedAt:  fromNullTime(item.HighlightedAt),
		Source:         fromNullStringValue(item.Source),
		SourceApp:      fromNullString(item.SourceApp),
		SourceURL:      fromNullString(item.SourceUrl),
		BookOrderIndex: fromNullInt32AsInt(item.BookOrderIndex),
		Status:         domain.HighlightStatus(strings.TrimSpace(item.Status)),
		RetryCount:     int(item.RetryCount),
		LastError:      fromNullString(item.LastError),
		RequestedAt:    item.GenerationRequestedAt,
		ProcessingAt:   fromNullTime(item.ProcessingStartedAt),
		CompletedAt:    fromNullTime(item.CompletedAt),
		FailedAt:       fromNullTime(item.FailedAt),
		CreatedAt:      item.CreatedAt,
		UpdatedAt:      item.UpdatedAt,
	}
}

func toNullString(value *string) sql.NullString {
	if value == nil {
		return sql.NullString{}
	}

	return sql.NullString{String: *value, Valid: true}
}

func fromNullString(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}

	return &value.String
}

func fromNullStringValue(value sql.NullString) string {
	if !value.Valid {
		return ""
	}

	return value.String
}

func fromNullInt32AsInt(value sql.NullInt32) *int {
	if !value.Valid {
		return nil
	}
	converted := int(value.Int32)
	return &converted
}

func fromNullTime(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}

	return &value.Time
}

func fromNullUUID(value uuid.NullUUID) *uuid.UUID {
	if !value.Valid {
		return nil
	}

	return &value.UUID
}

func uuidStrings(values []uuid.UUID) []string {
	items := make([]string, 0, len(values))
	for _, value := range values {
		items = append(items, value.String())
	}
	return items
}

func buildHighlightBulkUpsert(highlights []*domain.Highlight) (string, []any, map[string][]*domain.Highlight, error) {
	var builder strings.Builder
	builder.WriteString(`
INSERT INTO highlights (user_id, book_id, book_title, book_author, asin, book_key, content, explanation, location, highlighted_at, source, content_hash, source_app, source_url)
VALUES `)

	args := make([]any, 0, len(highlights)*14)
	hashIndex := make(map[string][]*domain.Highlight)

	for i, highlight := range highlights {
		if highlight == nil {
			return "", nil, nil, fmt.Errorf("highlight is nil")
		}

		if i > 0 {
			builder.WriteString(", ")
		}

		writeBulkUpsertValueGroup(&builder, i*14+1)
		args = appendBulkUpsertArgs(args, highlight)
		indexHighlightByContentHash(hashIndex, highlight)
	}

	builder.WriteString(`
ON CONFLICT (user_id, content_hash) WHERE content_hash IS NOT NULL
DO NOTHING
RETURNING id, content_hash, created_at, updated_at`)

	return builder.String(), args, hashIndex, nil
}

func writeBulkUpsertValueGroup(builder *strings.Builder, start int) {
	builder.WriteString("(")
	for i := 0; i < 14; i++ {
		if i > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(fmt.Sprintf("$%d", start+i))
	}
	builder.WriteString(")")
}

func appendBulkUpsertArgs(args []any, highlight *domain.Highlight) []any {
	return append(
		args,
		highlight.UserID,
		highlight.BookID,
		highlight.BookTitle,
		highlight.BookAuthor,
		highlight.ASIN,
		toNullStringValue(strings.TrimSpace(highlight.BookKey)),
		highlight.Content,
		highlight.Explanation,
		highlight.Location,
		highlight.HighlightedAt,
		highlight.Source,
		highlight.ContentHash,
		highlight.SourceApp,
		highlight.SourceURL,
	)
}

func toNullStringValue(value string) sql.NullString {
	normalized := strings.TrimSpace(value)
	return sql.NullString{String: normalized, Valid: normalized != ""}
}

func indexHighlightByContentHash(hashIndex map[string][]*domain.Highlight, highlight *domain.Highlight) {
	if highlight.ContentHash == nil {
		return
	}

	hashIndex[*highlight.ContentHash] = append(hashIndex[*highlight.ContentHash], highlight)
}

func assignBulkUpsertResult(hashIndex map[string][]*domain.Highlight, contentHash string, id uuid.UUID, createdAt, updatedAt time.Time) {
	queuedHighlights := hashIndex[contentHash]
	if len(queuedHighlights) == 0 {
		return
	}

	highlight := queuedHighlights[0]
	highlight.ID = id
	highlight.CreatedAt = createdAt
	highlight.UpdatedAt = updatedAt
	hashIndex[contentHash] = queuedHighlights[1:]
}
