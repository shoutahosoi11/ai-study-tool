package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shout/ai-study-tool/backend/internal/domain"
	"github.com/shout/ai-study-tool/backend/internal/usecase"
)

type mockHighlightRepository struct {
	listByUserIDAndASIN    func(ctx context.Context, userID uuid.UUID, asin string) ([]*domain.Highlight, error)
	listByUserIDAndBookMet func(ctx context.Context, userID uuid.UUID, bookTitle, bookAuthor string) ([]*domain.Highlight, error)
}

func (m *mockHighlightRepository) BulkUpsert(ctx context.Context, highlights []*domain.Highlight) (int, error) {
	return 0, errors.New("not implemented")
}

func (m *mockHighlightRepository) ListExistingContentHashesByUserID(ctx context.Context, userID uuid.UUID, hashes []string) ([]string, error) {
	return make([]string, 0), errors.New("not implemented")
}

func (m *mockHighlightRepository) ListByUserIDAndASIN(ctx context.Context, userID uuid.UUID, asin string) ([]*domain.Highlight, error) {
	if m.listByUserIDAndASIN == nil {
		return make([]*domain.Highlight, 0), nil
	}
	return m.listByUserIDAndASIN(ctx, userID, asin)
}

func (m *mockHighlightRepository) ListByUserIDAndBookMetadata(ctx context.Context, userID uuid.UUID, bookTitle, bookAuthor string) ([]*domain.Highlight, error) {
	if m.listByUserIDAndBookMet == nil {
		return make([]*domain.Highlight, 0), nil
	}
	return m.listByUserIDAndBookMet(ctx, userID, bookTitle, bookAuthor)
}

func (m *mockHighlightRepository) ListBooksWithHighlightsByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.KindleBook, error) {
	return make([]*domain.KindleBook, 0), errors.New("not implemented")
}

func (m *mockHighlightRepository) ListBookStockByUserID(ctx context.Context, userID uuid.UUID) ([]domain.BookStock, error) {
	return make([]domain.BookStock, 0), errors.New("not implemented")
}

func (m *mockHighlightRepository) ListUnusedHighlightsByBook(ctx context.Context, userID uuid.UUID, bookKey string, limit int) ([]*domain.Highlight, error) {
	return make([]*domain.Highlight, 0), errors.New("not implemented")
}

func (m *mockHighlightRepository) ListUsedHighlightsWithUncoveredPerspectives(ctx context.Context, userID uuid.UUID, bookKey string, limit int) ([]*domain.Highlight, error) {
	return make([]*domain.Highlight, 0), errors.New("not implemented")
}

func (m *mockHighlightRepository) ListPendingUserStats(ctx context.Context) ([]domain.PendingHighlightUserStat, error) {
	return make([]domain.PendingHighlightUserStat, 0), errors.New("not implemented")
}

func (m *mockHighlightRepository) ClaimPendingByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*domain.Highlight, error) {
	return make([]*domain.Highlight, 0), errors.New("not implemented")
}

func (m *mockHighlightRepository) ClaimPendingByIDs(ctx context.Context, userID uuid.UUID, highlightIDs []uuid.UUID) ([]*domain.Highlight, error) {
	return make([]*domain.Highlight, 0), errors.New("not implemented")
}

func (m *mockHighlightRepository) QueueHighlightsForGeneration(ctx context.Context, userID uuid.UUID, highlightIDs []uuid.UUID, requestedAt time.Time) error {
	return errors.New("not implemented")
}

func (m *mockHighlightRepository) MarkGenerationCompleted(ctx context.Context, highlightIDs []uuid.UUID) error {
	return errors.New("not implemented")
}

func (m *mockHighlightRepository) MarkGenerationFailed(ctx context.Context, highlightIDs []uuid.UUID, lastError string, maxRetry int) error {
	return errors.New("not implemented")
}

func (m *mockHighlightRepository) UpdateExplanation(ctx context.Context, id, userID uuid.UUID, explanation *string) (*domain.Highlight, error) {
	return nil, errors.New("not implemented")
}

func TestQuestionSourceResolver_ResolveHighlightsFromKindleBook(t *testing.T) {
	userID := uuid.New()
	asin := "B00TEST"

	resolver := usecase.NewQuestionSourceResolver(
		&mockHighlightRepository{
			listByUserIDAndASIN: func(ctx context.Context, requestUserID uuid.UUID, requestASIN string) ([]*domain.Highlight, error) {
				if requestUserID != userID {
					t.Fatalf("unexpected user id: %s", requestUserID)
				}
				if requestASIN != asin {
					t.Fatalf("unexpected asin: %s", requestASIN)
				}

				return []*domain.Highlight{
					{ID: uuid.New(), Content: "1つ目のハイライト"},
					{ID: uuid.New(), Content: "2つ目のハイライト"},
				}, nil
			},
		},
	)

	highlights, err := resolver.ResolveHighlights(context.Background(), userID.String(), domain.SourceTypeKindleBook, asin, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(highlights) != 2 {
		t.Fatalf("expected 2 highlights, got %d", len(highlights))
	}
}

func TestQuestionSourceResolver_ResolveHighlightsFallsBackToBookMetadata(t *testing.T) {
	userID := uuid.New()

	resolver := usecase.NewQuestionSourceResolver(
		&mockHighlightRepository{
			listByUserIDAndASIN: func(ctx context.Context, requestUserID uuid.UUID, requestASIN string) ([]*domain.Highlight, error) {
				return []*domain.Highlight{}, nil
			},
			listByUserIDAndBookMet: func(ctx context.Context, requestUserID uuid.UUID, bookTitle, bookAuthor string) ([]*domain.Highlight, error) {
				if requestUserID != userID {
					t.Fatalf("unexpected user id: %s", requestUserID)
				}
				if bookTitle != "テスト本" {
					t.Fatalf("unexpected title: %s", bookTitle)
				}
				if bookAuthor != "著者A" {
					t.Fatalf("unexpected author: %s", bookAuthor)
				}

				return []*domain.Highlight{
					{ID: uuid.New(), Content: "metadata 1"},
					{ID: uuid.New(), Content: "metadata 2"},
				}, nil
			},
		},
	)

	highlights, err := resolver.ResolveHighlights(context.Background(), userID.String(), domain.SourceTypeKindleBook, "B00FALLBACK", "テスト本", "著者A")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(highlights) != 2 {
		t.Fatalf("expected 2 highlights, got %d", len(highlights))
	}
}

func TestQuestionSourceResolver_ResolveHighlightsRejectsUnsupportedSourceType(t *testing.T) {
	userID := uuid.New()

	resolver := usecase.NewQuestionSourceResolver(&mockHighlightRepository{})

	_, err := resolver.ResolveHighlights(context.Background(), userID.String(), domain.SourceType("highlight"), "source", "", "")
	if !errors.Is(err, domain.ErrInvalidSourceType) {
		t.Fatalf("expected ErrInvalidSourceType, got %v", err)
	}
}
