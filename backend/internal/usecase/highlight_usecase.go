package usecase

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

type HighlightUsecase struct {
	repo domain.HighlightRepository
}

func NewHighlightUsecase(repo domain.HighlightRepository) *HighlightUsecase {
	return &HighlightUsecase{repo: repo}
}

type CreateHighlightInput struct {
	BookID        *uuid.UUID
	BookTitle     *string
	BookAuthor    *string
	ASIN          *string
	Content       string
	Location      *string
	HighlightedAt *time.Time
	Source        string
}

func (u *HighlightUsecase) Create(ctx context.Context, userID uuid.UUID, req CreateHighlightInput) (*domain.Highlight, error) {
	source := req.Source
	if source == "" {
		source = "manual"
	}

	content := strings.TrimSpace(req.Content)

	h := &domain.Highlight{
		ID:            uuid.New(),
		UserID:        userID,
		BookID:        req.BookID,
		BookTitle:     req.BookTitle,
		BookAuthor:    req.BookAuthor,
		ASIN:          req.ASIN,
		Content:       content,
		Location:      req.Location,
		HighlightedAt: req.HighlightedAt,
		Source:        source,
	}

	return u.repo.Create(ctx, h)
}

func (u *HighlightUsecase) GetByID(ctx context.Context, id, userID uuid.UUID) (*domain.Highlight, error) {
	return u.repo.GetByID(ctx, id, userID)
}

func (u *HighlightUsecase) List(ctx context.Context, userID uuid.UUID, page, limit int) ([]*domain.Highlight, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}
	if page <= 0 {
		page = 1
	}

	offset := int32((page - 1) * limit)

	highlights, err := u.repo.ListByUserID(ctx, userID, int32(limit), offset)
	if err != nil {
		return nil, 0, err
	}

	total, err := u.repo.CountByUserID(ctx, userID)
	if err != nil {
		return nil, 0, err
	}

	return highlights, total, nil
}

func (u *HighlightUsecase) Delete(ctx context.Context, id, userID uuid.UUID) error {
	return u.repo.Delete(ctx, id, userID)
}
