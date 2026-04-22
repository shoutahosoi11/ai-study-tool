package persistence

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shout/ai-study-tool/backend/internal/domain"
	"github.com/shout/ai-study-tool/backend/internal/repository/sqlcgen"
)

type highlightRepository struct {
	db      *sql.DB
	queries *sqlcgen.Queries
}

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

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("highlight repo: bulk upsert query: %w", err)
	}
	defer rows.Close()

	saved = 0
	for rows.Next() {
		var (
			id          uuid.UUID
			contentHash sql.NullString
			createdAt   time.Time
			updatedAt   time.Time
		)

		if err := rows.Scan(&id, &contentHash, &createdAt, &updatedAt); err != nil {
			return 0, fmt.Errorf("highlight repo: bulk upsert scan: %w", err)
		}

		saved++
		if contentHash.Valid {
			assignBulkUpsertResult(hashIndex, contentHash.String, id, createdAt, updatedAt)
		}
	}

	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("highlight repo: bulk upsert rows: %w", err)
	}

	return saved, nil
}

func (r *highlightRepository) ListByUserIDAndASIN(ctx context.Context, userID uuid.UUID, asin string) ([]*domain.Highlight, error) {
	highlights, err := r.queries.ListHighlightsByUserIDAndASIN(ctx, sqlcgen.ListHighlightsByUserIDAndASINParams{
		UserID: userID,
		Asin: sql.NullString{
			String: strings.TrimSpace(asin),
			Valid:  true,
		},
		Source: domain.HighlightSourceKindle,
	})
	if err != nil {
		return nil, fmt.Errorf("highlight repo: list by asin: %w", err)
	}

	return toDomainHighlights(highlights), nil
}

func (r *highlightRepository) ListByUserIDAndBookMetadata(ctx context.Context, userID uuid.UUID, bookTitle, bookAuthor string) ([]*domain.Highlight, error) {
	normalizedTitle := strings.TrimSpace(bookTitle)
	if normalizedTitle == "" {
		return make([]*domain.Highlight, 0), nil
	}

	normalizedAuthor := strings.TrimSpace(bookAuthor)
	if normalizedAuthor != "" {
		highlights, err := r.queries.ListHighlightsByUserIDAndBookMetadata(ctx, sqlcgen.ListHighlightsByUserIDAndBookMetadataParams{
			UserID:     userID,
			Source:     domain.HighlightSourceKindle,
			BookTitle:  normalizedTitle,
			BookAuthor: normalizedAuthor,
		})
		if err != nil {
			return nil, fmt.Errorf("highlight repo: list by book metadata: %w", err)
		}
		if len(highlights) > 0 {
			return toDomainHighlights(highlights), nil
		}
	}

	highlights, err := r.queries.ListHighlightsByUserIDAndBookTitle(ctx, sqlcgen.ListHighlightsByUserIDAndBookTitleParams{
		UserID:    userID,
		Source:    domain.HighlightSourceKindle,
		BookTitle: normalizedTitle,
	})
	if err != nil {
		return nil, fmt.Errorf("highlight repo: list by book title: %w", err)
	}

	return toDomainHighlights(highlights), nil
}

func (r *highlightRepository) ListBooksWithHighlightsByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.KindleBook, error) {
	rows, err := r.queries.ListHighlightBooksByUserID(ctx, sqlcgen.ListHighlightBooksByUserIDParams{
		UserID: userID,
		Source: domain.HighlightSourceKindle,
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
			Source:         row.Source,
		})
	}

	return books, nil
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

func wrapHighlightError(action string, err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("highlight repo: %s: %w", action, domain.ErrNotFound)
	}

	return fmt.Errorf("highlight repo: %s: %w", action, err)
}

func toDomainHighlights(items []sqlcgen.Highlight) []*domain.Highlight {
	highlights := make([]*domain.Highlight, 0, len(items))
	for _, item := range items {
		highlights = append(highlights, toDomainHighlight(item))
	}

	return highlights
}

func toDomainHighlight(item sqlcgen.Highlight) *domain.Highlight {
	return &domain.Highlight{
		ID:            item.ID,
		UserID:        item.UserID,
		BookID:        fromNullUUID(item.BookID),
		BookTitle:     fromNullString(item.BookTitle),
		BookAuthor:    fromNullString(item.BookAuthor),
		ASIN:          fromNullString(item.Asin),
		Content:       item.Content,
		Explanation:   fromNullString(item.Explanation),
		ContentHash:   fromNullString(item.ContentHash),
		Location:      fromNullString(item.Location),
		HighlightedAt: fromNullTime(item.HighlightedAt),
		Source:        item.Source,
		CreatedAt:     item.CreatedAt,
		UpdatedAt:     item.UpdatedAt,
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

func toNullTime(value *time.Time) sql.NullTime {
	if value == nil {
		return sql.NullTime{}
	}

	return sql.NullTime{Time: *value, Valid: true}
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

func buildHighlightBulkUpsert(highlights []*domain.Highlight) (string, []any, map[string][]*domain.Highlight, error) {
	var builder strings.Builder
	builder.WriteString(`
INSERT INTO highlights (user_id, book_id, book_title, book_author, asin, content, explanation, location, highlighted_at, source, content_hash)
VALUES `)

	args := make([]any, 0, len(highlights)*11)
	hashIndex := make(map[string][]*domain.Highlight)

	for i, highlight := range highlights {
		if highlight == nil {
			return "", nil, nil, fmt.Errorf("highlight is nil")
		}

		if i > 0 {
			builder.WriteString(", ")
		}

		writeBulkUpsertValueGroup(&builder, i*11+1)
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
	for i := 0; i < 11; i++ {
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
		highlight.Content,
		highlight.Explanation,
		highlight.Location,
		highlight.HighlightedAt,
		highlight.Source,
		highlight.ContentHash,
	)
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
