package persistence

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

type highlightRepository struct {
	db *sql.DB
}

func NewHighlightRepository(db *sql.DB) domain.HighlightRepository {
	return &highlightRepository{db: db}
}

func (r *highlightRepository) Create(ctx context.Context, h *domain.Highlight) (*domain.Highlight, error) {
	query := `
INSERT INTO highlights (user_id, book_id, book_title, book_author, asin, content, location, highlighted_at, source)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id, user_id, book_id, book_title, book_author, asin, content, location, highlighted_at, source, created_at, updated_at`

	row := r.db.QueryRowContext(ctx, query,
		h.UserID, h.BookID, h.BookTitle, h.BookAuthor, h.ASIN,
		h.Content, h.Location, h.HighlightedAt, h.Source,
	)
	created, err := scanHighlight(row)
	if err != nil {
		return nil, fmt.Errorf("highlight repo: create: %w", err)
	}
	return created, nil
}

func (r *highlightRepository) GetByID(ctx context.Context, id, userID uuid.UUID) (*domain.Highlight, error) {
	query := `
SELECT id, user_id, book_id, book_title, book_author, asin, content, location, highlighted_at, source, created_at, updated_at
FROM highlights WHERE id = $1 AND user_id = $2 LIMIT 1`

	row := r.db.QueryRowContext(ctx, query, id, userID)
	h, err := scanHighlight(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("highlight repo: get by id: %w", err)
	}
	return h, nil
}

func (r *highlightRepository) ListByUserID(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]*domain.Highlight, error) {
	query := `
SELECT id, user_id, book_id, book_title, book_author, asin, content, location, highlighted_at, source, created_at, updated_at
FROM highlights WHERE user_id = $1
ORDER BY created_at DESC LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("highlight repo: list: %w", err)
	}
	defer rows.Close()

	highlights := make([]*domain.Highlight, 0)
	for rows.Next() {
		h, err := scanHighlightRow(rows)
		if err != nil {
			return nil, fmt.Errorf("highlight repo: scan: %w", err)
		}
		highlights = append(highlights, h)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("highlight repo: rows: %w", err)
	}

	return highlights, nil
}

func (r *highlightRepository) CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM highlights WHERE user_id = $1`, userID).Scan(&count); err != nil {
		return 0, fmt.Errorf("highlight repo: count: %w", err)
	}
	return count, nil
}

func (r *highlightRepository) Delete(ctx context.Context, id, userID uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM highlights WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("highlight repo: delete: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("highlight repo: delete rows affected: %w", err)
	}
	if rows == 0 {
		return domain.ErrNotFound
	}

	return nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanHighlight(row scanner) (*domain.Highlight, error) {
	var h domain.Highlight
	var (
		bookID        sql.NullString
		bookTitle     sql.NullString
		bookAuthor    sql.NullString
		asin          sql.NullString
		location      sql.NullString
		highlightedAt sql.NullTime
	)

	err := row.Scan(
		&h.ID, &h.UserID, &bookID, &bookTitle, &bookAuthor, &asin,
		&h.Content, &location, &highlightedAt, &h.Source,
		&h.CreatedAt, &h.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if bookID.Valid {
		parsedBookID, err := uuid.Parse(bookID.String)
		if err != nil {
			return nil, fmt.Errorf("highlight repo: parse book id: %w", err)
		}
		h.BookID = &parsedBookID
	}
	if bookTitle.Valid {
		h.BookTitle = &bookTitle.String
	}
	if bookAuthor.Valid {
		h.BookAuthor = &bookAuthor.String
	}
	if asin.Valid {
		h.ASIN = &asin.String
	}
	if location.Valid {
		h.Location = &location.String
	}
	if highlightedAt.Valid {
		h.HighlightedAt = &highlightedAt.Time
	}

	return &h, nil
}

func scanHighlightRow(rows *sql.Rows) (*domain.Highlight, error) {
	return scanHighlight(rows)
}
