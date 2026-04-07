package usecase

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

type NoteUsecase struct {
	db              *sql.DB
	storageClient   domain.StorageClient
	ocrClient       domain.OCRClient
	questionUsecase *QuestionUsecase
}

func NewNoteUsecase(db *sql.DB, storageClient domain.StorageClient, ocrClient domain.OCRClient, questionUsecase *QuestionUsecase) *NoteUsecase {
	return &NoteUsecase{
		db:              db,
		storageClient:   storageClient,
		ocrClient:       ocrClient,
		questionUsecase: questionUsecase,
	}
}

type NoteResult struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	FileURL   string    `json:"file_url"`
	CreatedAt time.Time `json:"created_at"`
}

func (u *NoteUsecase) UploadNote(ctx context.Context, userID, title string, file io.Reader, contentType string) (*NoteResult, error) {
	noteID := uuid.New().String()
	key := fmt.Sprintf("notes/%s/%s", userID, noteID)

	fileURL, err := u.storageClient.Upload(ctx, key, file, contentType)
	if err != nil {
		return nil, fmt.Errorf("note usecase: upload file: %w", err)
	}

	now := time.Now()
	_, err = u.db.ExecContext(ctx,
		`INSERT INTO notes (id, user_id, title, file_url, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $5)`,
		noteID, userID, title, fileURL, now,
	)
	if err != nil {
		return nil, fmt.Errorf("note usecase: save note: %w", err)
	}

	return &NoteResult{
		ID:        noteID,
		Title:     title,
		FileURL:   fileURL,
		CreatedAt: now,
	}, nil
}
