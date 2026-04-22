package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

type questionSourceResolver struct {
	highlightRepo domain.HighlightRepository
}

func NewQuestionSourceResolver(highlightRepo domain.HighlightRepository) domain.QuestionSourceResolver {
	return &questionSourceResolver{
		highlightRepo: highlightRepo,
	}
}

func (r *questionSourceResolver) ResolveHighlights(ctx context.Context, userID string, sourceType domain.SourceType, sourceID string, bookTitle string, bookAuthor string) ([]*domain.Highlight, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, domain.ErrInvalidSourceType
	}

	switch sourceType {
	case domain.SourceTypeKindleBook:
		return r.resolveKindleBookHighlights(ctx, userUUID, sourceID, bookTitle, bookAuthor)
	default:
		return nil, domain.ErrInvalidSourceType
	}
}

func (r *questionSourceResolver) resolveKindleBookHighlights(ctx context.Context, userUUID uuid.UUID, sourceID string, bookTitle string, bookAuthor string) ([]*domain.Highlight, error) {
	asin := strings.TrimSpace(sourceID)
	if asin == "" {
		return nil, domain.ErrInvalidSourceType
	}

	highlights, err := r.highlightRepo.ListByUserIDAndASIN(ctx, userUUID, asin)
	if err != nil {
		return nil, fmt.Errorf("question source resolver: list highlights by asin: %w", err)
	}
	if len(highlights) > 0 {
		return highlights, nil
	}

	if strings.TrimSpace(bookTitle) == "" {
		return make([]*domain.Highlight, 0), nil
	}

	metadataHighlights, metadataErr := r.highlightRepo.ListByUserIDAndBookMetadata(ctx, userUUID, bookTitle, bookAuthor)
	if metadataErr != nil {
		return nil, fmt.Errorf("question source resolver: list highlights by metadata: %w", metadataErr)
	}
	return metadataHighlights, nil
}
