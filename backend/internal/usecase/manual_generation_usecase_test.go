package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

type stubManualGenerationHighlightReader struct {
	called bool
}

func (s *stubManualGenerationHighlightReader) ListByIDs(ctx context.Context, userID uuid.UUID, highlightIDs []uuid.UUID) ([]*domain.Highlight, error) {
	s.called = true
	return make([]*domain.Highlight, 0), nil
}

func TestManualGenerationRejectsTooManyHighlightsBeforeRepositoryAccess(t *testing.T) {
	highlightRepo := &stubManualGenerationHighlightReader{}
	uc := NewManualGenerationUsecase(nil, highlightRepo, nil, nil)

	highlightIDs := make([]uuid.UUID, 0, domain.MaxHighlightsPerJob+1)
	for range domain.MaxHighlightsPerJob + 1 {
		highlightIDs = append(highlightIDs, uuid.New())
	}

	_, err := uc.Generate(context.Background(), &domain.User{ID: uuid.New()}, "book-a", highlightIDs)
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
	if highlightRepo.called {
		t.Fatal("highlight repository should not be called for oversized manual generation")
	}
}
