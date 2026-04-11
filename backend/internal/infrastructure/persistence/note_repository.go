package persistence

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/shout/ai-study-tool/backend/internal/domain"
)

type noteRepository struct {
	db *sql.DB
}

func NewNoteRepository(db *sql.DB) domain.NoteRepository {
	return &noteRepository{db: db}
}

func (r *noteRepository) Save(ctx context.Context, note *domain.Note) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO notes (id, user_id, title, file_url, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $5)`,
		note.ID, note.UserID, note.Title, note.FileURL, note.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("note repo: save: %w", err)
	}
	return nil
}
