package usecase_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shout/ai-study-tool/backend/internal/domain"
	"github.com/shout/ai-study-tool/backend/internal/usecase"
)

type mockImportHighlightRepository struct {
	bulkUpsertSaved  int
	bulkUpsertErr    error
	bulkUpsertCalled bool
	bulkUpsertInput  []*domain.Highlight
	bulkUpsertTime   time.Time
}

func (m *mockImportHighlightRepository) Create(ctx context.Context, h *domain.Highlight) (*domain.Highlight, error) {
	return nil, errors.New("not implemented")
}

func (m *mockImportHighlightRepository) BulkUpsert(ctx context.Context, highlights []*domain.Highlight) (int, error) {
	m.bulkUpsertCalled = true
	m.bulkUpsertInput = highlights

	if m.bulkUpsertErr != nil {
		return 0, m.bulkUpsertErr
	}

	for i := 0; i < m.bulkUpsertSaved && i < len(highlights); i++ {
		m.markHighlightPersisted(highlights[i], i)
	}

	return m.bulkUpsertSaved, nil
}

func (m *mockImportHighlightRepository) GetByID(ctx context.Context, id, userID uuid.UUID) (*domain.Highlight, error) {
	return nil, errors.New("not implemented")
}

func (m *mockImportHighlightRepository) ListByUserID(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]*domain.Highlight, error) {
	return nil, errors.New("not implemented")
}

func (m *mockImportHighlightRepository) ListByUserIDAndASIN(ctx context.Context, userID uuid.UUID, asin string) ([]*domain.Highlight, error) {
	return make([]*domain.Highlight, 0), errors.New("not implemented")
}

func (m *mockImportHighlightRepository) ListByUserIDAndBookMetadata(ctx context.Context, userID uuid.UUID, bookTitle, bookAuthor string) ([]*domain.Highlight, error) {
	return make([]*domain.Highlight, 0), errors.New("not implemented")
}

func (m *mockImportHighlightRepository) ListBooksWithHighlightsByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.KindleBook, error) {
	return make([]*domain.KindleBook, 0), errors.New("not implemented")
}

func (m *mockImportHighlightRepository) CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	return 0, errors.New("not implemented")
}

func (m *mockImportHighlightRepository) UpdateExplanation(ctx context.Context, id, userID uuid.UUID, explanation *string) (*domain.Highlight, error) {
	return nil, errors.New("not implemented")
}

func (m *mockImportHighlightRepository) Delete(ctx context.Context, id, userID uuid.UUID) error {
	return errors.New("not implemented")
}

func (m *mockImportHighlightRepository) markHighlightPersisted(highlight *domain.Highlight, offset int) {
	if highlight == nil {
		return
	}

	highlight.ID = uuid.MustParse(buildPersistedHighlightID(offset))
	highlight.CreatedAt = m.bulkUpsertTime
	highlight.UpdatedAt = m.bulkUpsertTime
}

func buildPersistedHighlightID(offset int) string {
	base := "00000000-0000-0000-0000-00000000000"
	return base + string(rune('1'+offset))
}

func TestImportKindleHighlightsAllItemsSaved(t *testing.T) {
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	highlightedAt := time.Now().Add(-1 * time.Hour).UTC()
	repo := &mockImportHighlightRepository{
		bulkUpsertSaved: 2,
		bulkUpsertTime:  time.Date(2026, 4, 18, 12, 30, 0, 0, time.UTC),
	}
	uc := usecase.NewHighlightUsecase(repo)

	result, err := uc.ImportKindleHighlights(context.Background(), userID, []usecase.ImportHighlightItem{
		{
			ASIN:          "B001",
			BookTitle:     "Book One",
			BookAuthor:    "Author One",
			Content:       " First highlight ",
			Location:      "123-124",
			HighlightedAt: &highlightedAt,
		},
		{
			ASIN:       "B002",
			BookTitle:  "Book Two",
			BookAuthor: "Author Two",
			Content:    "Second highlight",
			Location:   "200-201",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !repo.bulkUpsertCalled {
		t.Fatal("expected BulkUpsert to be called")
	}
	if result.Saved != 2 {
		t.Fatalf("expected Saved=2, got %d", result.Saved)
	}
	if result.DuplicateCount != 0 {
		t.Fatalf("expected DuplicateCount=0, got %d", result.DuplicateCount)
	}
	if result.CopyProtectedCount != 0 {
		t.Fatalf("expected CopyProtectedCount=0, got %d", result.CopyProtectedCount)
	}
	if result.Warning != nil {
		t.Fatalf("expected nil warning, got %q", *result.Warning)
	}
	if len(result.Highlights) != 2 {
		t.Fatalf("expected 2 persisted highlights, got %d", len(result.Highlights))
	}

	first := repo.bulkUpsertInput[0]
	if first.Content != "First highlight" {
		t.Fatalf("expected trimmed content, got %q", first.Content)
	}
	if first.Source != domain.HighlightSourceKindle {
		t.Fatalf("expected kindle source, got %q", first.Source)
	}
	expectedHash := computeExpectedContentHash("B001", "123-124", "First highlight")
	if first.ContentHash == nil || *first.ContentHash != expectedHash {
		t.Fatalf("expected content hash %q, got %#v", expectedHash, first.ContentHash)
	}
	if first.HighlightedAt == nil || !first.HighlightedAt.Equal(highlightedAt) {
		t.Fatalf("expected highlighted_at to be preserved, got %#v", first.HighlightedAt)
	}
}

func TestImportKindleHighlightsPartialCopyProtectedSetsWarning(t *testing.T) {
	userID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	repo := &mockImportHighlightRepository{
		bulkUpsertSaved: 1,
		bulkUpsertTime:  time.Date(2026, 4, 18, 13, 0, 0, 0, time.UTC),
	}
	uc := usecase.NewHighlightUsecase(repo)

	result, err := uc.ImportKindleHighlights(context.Background(), userID, []usecase.ImportHighlightItem{
		{
			ASIN:    "B003",
			Content: "   ",
		},
		{
			ASIN:       "B004",
			BookTitle:  "Book Four",
			BookAuthor: "Author Four",
			Content:    "Valid highlight",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Saved != 1 {
		t.Fatalf("expected Saved=1, got %d", result.Saved)
	}
	if result.DuplicateCount != 0 {
		t.Fatalf("expected DuplicateCount=0, got %d", result.DuplicateCount)
	}
	if result.CopyProtectedCount != 1 {
		t.Fatalf("expected CopyProtectedCount=1, got %d", result.CopyProtectedCount)
	}
	if result.Warning == nil {
		t.Fatal("expected warning to be set")
	}
	if *result.Warning != "コピー制限により一部のハイライトが読み込めませんでした" {
		t.Fatalf("unexpected warning: %q", *result.Warning)
	}
	if len(repo.bulkUpsertInput) != 1 {
		t.Fatalf("expected 1 highlight to be sent to BulkUpsert, got %d", len(repo.bulkUpsertInput))
	}
	if len(result.Highlights) != 1 {
		t.Fatalf("expected 1 persisted highlight, got %d", len(result.Highlights))
	}
}

func TestImportKindleHighlightsAllCopyProtectedReturnsError(t *testing.T) {
	userID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	repo := &mockImportHighlightRepository{}
	uc := usecase.NewHighlightUsecase(repo)

	_, err := uc.ImportKindleHighlights(context.Background(), userID, []usecase.ImportHighlightItem{
		{
			ASIN:    "B005",
			Content: "   ",
		},
		{
			ASIN:    "B006",
			Content: "\n\t",
		},
	})
	if !errors.Is(err, domain.ErrAllCopyProtected) {
		t.Fatalf("expected ErrAllCopyProtected, got %v", err)
	}
	if repo.bulkUpsertCalled {
		t.Fatal("expected BulkUpsert not to be called")
	}
}

func TestImportKindleHighlightsAllDuplicatesReturnsSuccess(t *testing.T) {
	userID := uuid.MustParse("44444444-4444-4444-4444-444444444444")
	repo := &mockImportHighlightRepository{
		bulkUpsertSaved: 0,
		bulkUpsertTime:  time.Date(2026, 4, 18, 14, 0, 0, 0, time.UTC),
	}
	uc := usecase.NewHighlightUsecase(repo)

	result, err := uc.ImportKindleHighlights(context.Background(), userID, []usecase.ImportHighlightItem{
		{
			ASIN:     "B007",
			Content:  "Duplicate one",
			Location: "10-11",
		},
		{
			ASIN:     "B008",
			Content:  "Duplicate two",
			Location: "20-21",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Saved != 0 {
		t.Fatalf("expected Saved=0, got %d", result.Saved)
	}
	if result.DuplicateCount != 2 {
		t.Fatalf("expected DuplicateCount=2, got %d", result.DuplicateCount)
	}
	if result.CopyProtectedCount != 0 {
		t.Fatalf("expected CopyProtectedCount=0, got %d", result.CopyProtectedCount)
	}
	if result.Warning != nil {
		t.Fatalf("expected nil warning, got %q", *result.Warning)
	}
	if len(result.Highlights) != 0 {
		t.Fatalf("expected 0 persisted highlights, got %d", len(result.Highlights))
	}
}

func TestImportKindleHighlightsFutureHighlightedAtIsSanitized(t *testing.T) {
	userID := uuid.MustParse("55555555-5555-5555-5555-555555555555")
	futureTime := time.Now().Add(1 * time.Hour)
	repo := &mockImportHighlightRepository{
		bulkUpsertSaved: 1,
		bulkUpsertTime:  time.Date(2026, 4, 18, 15, 0, 0, 0, time.UTC),
	}
	uc := usecase.NewHighlightUsecase(repo)

	result, err := uc.ImportKindleHighlights(context.Background(), userID, []usecase.ImportHighlightItem{
		{
			ASIN:          "B009",
			Content:       "Future highlight",
			Location:      "30-31",
			HighlightedAt: &futureTime,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Saved != 1 {
		t.Fatalf("expected Saved=1, got %d", result.Saved)
	}
	if repo.bulkUpsertInput[0].HighlightedAt != nil {
		t.Fatalf("expected future highlighted_at to be nil, got %#v", repo.bulkUpsertInput[0].HighlightedAt)
	}
}

func computeExpectedContentHash(asin, location, content string) string {
	key := fmt.Sprintf(
		"asin:%s:loc:%s:content:%s",
		strings.TrimSpace(asin),
		strings.TrimSpace(location),
		normalizeExpectedContent(content),
	)
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

func normalizeExpectedContent(content string) string {
	return strings.Join(strings.Fields(strings.ToLower(content)), " ")
}
