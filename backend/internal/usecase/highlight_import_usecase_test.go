package usecase_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shout/ai-study-tool/backend/internal/domain"
	"github.com/shout/ai-study-tool/backend/internal/usecase"
)

type mockImportHighlightRepository struct {
	bulkUpsertSaved         int
	bulkUpsertErr           error
	bulkUpsertCalled        bool
	bulkUpsertInput         []*domain.Highlight
	bulkUpsertTime          time.Time
	existingHashes          []string
	checkedHashes           []string
	updateExplanationCalled bool
	updateExplanationID     uuid.UUID
	updateExplanationUserID uuid.UUID
	updateExplanationValue  *string
	updateExplanationResult *domain.Highlight
	updateExplanationErr    error
}

type mockHighlightImportJobTrigger struct {
	err     error
	called  bool
	queueID uuid.UUID
	userID  uuid.UUID
}

func (m *mockHighlightImportJobTrigger) TriggerHighlightImportJob(ctx context.Context, queueID uuid.UUID, userID uuid.UUID) error {
	m.called = true
	m.queueID = queueID
	m.userID = userID
	return m.err
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

func (m *mockImportHighlightRepository) ListExistingContentHashesByUserID(ctx context.Context, userID uuid.UUID, hashes []string) ([]string, error) {
	m.checkedHashes = append([]string(nil), hashes...)
	return m.existingHashes, nil
}

func (m *mockImportHighlightRepository) FindByUserIDAndContentHash(ctx context.Context, userID uuid.UUID, contentHash string) (*domain.Highlight, error) {
	return &domain.Highlight{
		ID:          uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		UserID:      userID,
		Content:     "existing",
		ContentHash: &contentHash,
		Source:      domain.HighlightSourcePaste,
	}, nil
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

func (m *mockImportHighlightRepository) UpdateExplanation(ctx context.Context, id, userID uuid.UUID, explanation *string) (*domain.Highlight, error) {
	m.updateExplanationCalled = true
	m.updateExplanationID = id
	m.updateExplanationUserID = userID
	m.updateExplanationValue = explanation
	if m.updateExplanationErr != nil {
		return nil, m.updateExplanationErr
	}
	if m.updateExplanationResult != nil {
		return m.updateExplanationResult, nil
	}
	return &domain.Highlight{ID: id, UserID: userID, Explanation: explanation}, nil
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
	uc := usecase.NewHighlightImportUsecase(repo)

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
	if repo.bulkUpsertInput[0].BookKey != "B001" {
		t.Fatalf("expected ASIN book key, got %q", repo.bulkUpsertInput[0].BookKey)
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
	if first.Source != domain.HighlightSourceExtension {
		t.Fatalf("expected extension source, got %q", first.Source)
	}
	expectedHash := computeExpectedContentHash("First highlight")
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
	uc := usecase.NewHighlightImportUsecase(repo)

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

func TestImportKindleHighlightsSkipsInvalidItems(t *testing.T) {
	userID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	repo := &mockImportHighlightRepository{
		bulkUpsertSaved: 1,
		bulkUpsertTime:  time.Date(2026, 4, 18, 13, 15, 0, 0, time.UTC),
	}
	uc := usecase.NewHighlightImportUsecase(repo)

	result, err := uc.ImportKindleHighlights(context.Background(), userID, []usecase.ImportHighlightItem{
		{
			ASIN:    "B004",
			Content: strings.Repeat("a", 301),
		},
		{
			ASIN:    "B005",
			Content: "Valid highlight",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Saved != 1 {
		t.Fatalf("expected Saved=1, got %d", result.Saved)
	}
	if result.InvalidItemCount != 1 {
		t.Fatalf("expected InvalidItemCount=1, got %d", result.InvalidItemCount)
	}
	if len(repo.bulkUpsertInput) != 1 || repo.bulkUpsertInput[0].Content != "Valid highlight" {
		t.Fatalf("expected only valid highlight to be imported, got %#v", repo.bulkUpsertInput)
	}
	if result.Warning == nil || !strings.Contains(*result.Warning, "入力不備") {
		t.Fatalf("expected invalid item warning, got %#v", result.Warning)
	}
}

func TestImportKindleHighlightsAllCopyProtectedReturnsError(t *testing.T) {
	userID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	repo := &mockImportHighlightRepository{}
	uc := usecase.NewHighlightImportUsecase(repo)

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

func TestImportKindleHighlightsRejectsEmptyItems(t *testing.T) {
	userID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	repo := &mockImportHighlightRepository{}
	uc := usecase.NewHighlightImportUsecase(repo)

	_, err := uc.ImportKindleHighlights(context.Background(), userID, nil)
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
	if errors.Is(err, domain.ErrAllCopyProtected) {
		t.Fatalf("empty import should not be copy protected: %v", err)
	}
	if repo.bulkUpsertCalled {
		t.Fatal("expected BulkUpsert not to be called")
	}
}

func TestImportKindleHighlightsRejectsTooManyItems(t *testing.T) {
	userID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	repo := &mockImportHighlightRepository{}
	uc := usecase.NewHighlightImportUsecase(repo)
	items := make([]usecase.ImportHighlightItem, 1001)
	for i := range items {
		items[i] = usecase.ImportHighlightItem{Content: "highlight"}
	}

	_, err := uc.ImportKindleHighlights(context.Background(), userID, items)
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
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
	uc := usecase.NewHighlightImportUsecase(repo)

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
	uc := usecase.NewHighlightImportUsecase(repo)

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

func TestImportKindleHighlightsExtremePastHighlightedAtIsSanitized(t *testing.T) {
	userID := uuid.MustParse("55555555-5555-5555-5555-555555555555")
	oldTime := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	repo := &mockImportHighlightRepository{
		bulkUpsertSaved: 1,
		bulkUpsertTime:  time.Date(2026, 4, 18, 15, 30, 0, 0, time.UTC),
	}
	uc := usecase.NewHighlightImportUsecase(repo)

	_, err := uc.ImportKindleHighlights(context.Background(), userID, []usecase.ImportHighlightItem{
		{
			ASIN:          "B009",
			Content:       "Old highlight",
			HighlightedAt: &oldTime,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.bulkUpsertInput[0].HighlightedAt != nil {
		t.Fatalf("expected extreme past highlighted_at to be nil, got %#v", repo.bulkUpsertInput[0].HighlightedAt)
	}
}

func TestImportKindleHighlightsQueuesValidatedPayload(t *testing.T) {
	userID := uuid.MustParse("55555555-5555-5555-5555-555555555555")
	queueID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	repo := &mockImportHighlightRepository{}
	queueRepo := &mockHighlightImportQueueRepository{enqueueID: queueID}
	trigger := &mockHighlightImportJobTrigger{}
	uc := usecase.NewHighlightImportUsecaseWithQueue(repo, queueRepo, trigger)

	result, err := uc.ImportKindleHighlights(context.Background(), userID, []usecase.ImportHighlightItem{
		{
			ASIN:    "B010",
			Content: "  Queued highlight  ",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.bulkUpsertCalled {
		t.Fatal("expected BulkUpsert not to be called in queue mode")
	}
	if !queueRepo.enqueued {
		t.Fatal("expected import to be enqueued")
	}
	if queueRepo.enqueuedUserID != userID {
		t.Fatalf("unexpected enqueued user id: %s", queueRepo.enqueuedUserID)
	}
	if queueRepo.enqueuedSource != domain.ImportQueueSourceKindle {
		t.Fatalf("unexpected queue source: %q", queueRepo.enqueuedSource)
	}
	if len(queueRepo.enqueuedPayload) == 0 {
		t.Fatal("expected queue payload")
	}
	var payload struct {
		Version    int `json:"version"`
		Highlights []struct {
			Content string `json:"content"`
			UserID  string `json:"user_id"`
		} `json:"highlights"`
	}
	if err := json.Unmarshal(queueRepo.enqueuedPayload, &payload); err != nil {
		t.Fatalf("expected versioned queue payload: %v", err)
	}
	if payload.Version != 1 {
		t.Fatalf("expected payload version 1, got %d", payload.Version)
	}
	if len(payload.Highlights) != 1 || payload.Highlights[0].Content != "Queued highlight" {
		t.Fatalf("unexpected payload highlights: %#v", payload.Highlights)
	}
	if payload.Highlights[0].UserID != "" {
		t.Fatalf("queue payload should not duplicate user id, got %q", payload.Highlights[0].UserID)
	}
	if !trigger.called {
		t.Fatal("expected job trigger to be called")
	}
	if trigger.queueID != queueID || trigger.userID != userID {
		t.Fatalf("unexpected trigger args: queue=%s user=%s", trigger.queueID, trigger.userID)
	}
	if result.QueueID != queueID {
		t.Fatalf("unexpected queue id: %s", result.QueueID)
	}
	if result.QueuedCount != 1 {
		t.Fatalf("expected queued_count=1, got %d", result.QueuedCount)
	}
}

func TestImportKindleHighlightsFailsQueueWhenTriggerFails(t *testing.T) {
	userID := uuid.MustParse("55555555-5555-5555-5555-555555555555")
	queueID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	repo := &mockImportHighlightRepository{}
	queueRepo := &mockHighlightImportQueueRepository{enqueueID: queueID}
	trigger := &mockHighlightImportJobTrigger{err: errors.New("cloud tasks unavailable")}
	uc := usecase.NewHighlightImportUsecaseWithQueue(repo, queueRepo, trigger)

	result, err := uc.ImportKindleHighlights(context.Background(), userID, []usecase.ImportHighlightItem{
		{
			ASIN:    "B011",
			Content: "Queued highlight",
		},
	})
	if err == nil {
		t.Fatal("expected trigger error")
	}
	if result != nil {
		t.Fatalf("expected nil result, got %#v", result)
	}
	if repo.bulkUpsertCalled {
		t.Fatal("expected BulkUpsert not to be called in queue mode")
	}
	if !queueRepo.enqueueFailed {
		t.Fatal("expected queue item to be marked enqueue_failed")
	}
	if queueRepo.enqueueFailedID != queueID {
		t.Fatalf("unexpected enqueue_failed queue id: %s", queueRepo.enqueueFailedID)
	}
	if !strings.Contains(queueRepo.enqueueFailedError, "cloud tasks unavailable") {
		t.Fatalf("expected enqueue_failed error to include trigger error, got %q", queueRepo.enqueueFailedError)
	}
}

func TestImportSharedHighlightSavesMobileShareMetadata(t *testing.T) {
	userID := uuid.MustParse("66666666-6666-6666-6666-666666666666")
	sharedAt := time.Now().Add(-30 * time.Minute).UTC()
	repo := &mockImportHighlightRepository{
		bulkUpsertSaved: 1,
		bulkUpsertTime:  time.Date(2026, 4, 24, 9, 30, 0, 0, time.UTC),
	}
	uc := usecase.NewHighlightImportUsecase(repo)

	result, err := uc.ImportSharedHighlight(context.Background(), userID, usecase.ImportSharedHighlightInput{
		BookTitle:  " Deep Work ",
		BookAuthor: " Cal Newport ",
		Content:    "  Focus is a superpower.  ",
		SourceApp:  " Kindle ",
		SourceURL:  "https://read.amazon.com/notebook",
		SharedAt:   &sharedAt,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Saved {
		t.Fatal("expected shared highlight to be saved")
	}
	if result.Duplicate {
		t.Fatal("expected duplicate=false")
	}
	if result.Highlight == nil {
		t.Fatal("expected persisted highlight in result")
	}
	if !repo.bulkUpsertCalled {
		t.Fatal("expected BulkUpsert to be called")
	}
	if len(repo.bulkUpsertInput) != 1 {
		t.Fatalf("expected one highlight to be upserted, got %d", len(repo.bulkUpsertInput))
	}

	highlight := repo.bulkUpsertInput[0]
	if highlight.Source != domain.HighlightSourceShare {
		t.Fatalf("expected share source, got %q", highlight.Source)
	}
	if highlight.SourceApp == nil || *highlight.SourceApp != "kindle" {
		t.Fatalf("unexpected source app: %#v", highlight.SourceApp)
	}
	if highlight.SourceURL == nil || *highlight.SourceURL != "https://read.amazon.com/notebook" {
		t.Fatalf("unexpected source url: %#v", highlight.SourceURL)
	}
	if highlight.BookTitle == nil || *highlight.BookTitle != "Deep Work" {
		t.Fatalf("unexpected book title: %#v", highlight.BookTitle)
	}
	if highlight.BookAuthor == nil || *highlight.BookAuthor != "Cal Newport" {
		t.Fatalf("unexpected book author: %#v", highlight.BookAuthor)
	}
	if highlight.BookKey != "metadata:Deep Work:Cal Newport" {
		t.Fatalf("unexpected book key: %q", highlight.BookKey)
	}
	if highlight.Content != "Focus is a superpower." {
		t.Fatalf("expected trimmed content, got %q", highlight.Content)
	}
	expectedHash := computeExpectedContentHash("Focus is a superpower.")
	if highlight.ContentHash == nil || *highlight.ContentHash != expectedHash {
		t.Fatalf("expected shared content hash %q, got %#v", expectedHash, highlight.ContentHash)
	}
	if highlight.HighlightedAt == nil || !highlight.HighlightedAt.Equal(sharedAt) {
		t.Fatalf("expected shared_at to be preserved, got %#v", highlight.HighlightedAt)
	}
}

func TestImportSharedHighlightDuplicateReturnsNoHighlight(t *testing.T) {
	userID := uuid.MustParse("77777777-7777-7777-7777-777777777777")
	repo := &mockImportHighlightRepository{
		bulkUpsertSaved: 0,
		bulkUpsertTime:  time.Date(2026, 4, 24, 10, 0, 0, 0, time.UTC),
	}
	uc := usecase.NewHighlightImportUsecase(repo)

	result, err := uc.ImportSharedHighlight(context.Background(), userID, usecase.ImportSharedHighlightInput{
		Content:   "Duplicate share",
		SourceApp: "kindle",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Saved {
		t.Fatal("expected saved=false")
	}
	if !result.Duplicate {
		t.Fatal("expected duplicate=true")
	}
	if result.Highlight != nil {
		t.Fatalf("expected nil highlight when duplicate, got %#v", result.Highlight)
	}
}

func TestImportSharedHighlightRejectsEmptyContent(t *testing.T) {
	userID := uuid.MustParse("88888888-8888-8888-8888-888888888888")
	repo := &mockImportHighlightRepository{}
	uc := usecase.NewHighlightImportUsecase(repo)

	_, err := uc.ImportSharedHighlight(context.Background(), userID, usecase.ImportSharedHighlightInput{
		Content: "   ",
	})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
	if repo.bulkUpsertCalled {
		t.Fatal("expected BulkUpsert not to be called")
	}
}

func TestImportSharedHighlightRejectsLongSourceApp(t *testing.T) {
	userID := uuid.MustParse("88888888-8888-8888-8888-888888888888")
	repo := &mockImportHighlightRepository{}
	uc := usecase.NewHighlightImportUsecase(repo)

	_, err := uc.ImportSharedHighlight(context.Background(), userID, usecase.ImportSharedHighlightInput{
		Content:   "Valid highlight",
		SourceApp: strings.Repeat("a", 101),
	})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
	if repo.bulkUpsertCalled {
		t.Fatal("expected BulkUpsert not to be called")
	}
}

func TestImportPastedHighlightSavesPasteSource(t *testing.T) {
	userID := uuid.MustParse("99999999-9999-9999-9999-999999999999")
	repo := &mockImportHighlightRepository{
		bulkUpsertSaved: 1,
		bulkUpsertTime:  time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC),
	}
	uc := usecase.NewHighlightImportUsecase(repo)

	result, err := uc.ImportPastedHighlight(context.Background(), userID, usecase.ImportPastedHighlightInput{
		BookTitle:  " Notes ",
		BookAuthor: " Me ",
		Content:    "  Paste this idea.  ",
		SourceApp:  "web",
		SourceURL:  "https://example.com/article",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Duplicate {
		t.Fatal("expected duplicate=false")
	}
	if result.ID == uuid.Nil {
		t.Fatal("expected result id")
	}
	if len(repo.bulkUpsertInput) != 1 {
		t.Fatalf("expected one highlight to be upserted, got %d", len(repo.bulkUpsertInput))
	}

	highlight := repo.bulkUpsertInput[0]
	if highlight.Source != domain.HighlightSourcePaste {
		t.Fatalf("expected paste source, got %q", highlight.Source)
	}
	if highlight.BookKey != "metadata:Notes:Me" {
		t.Fatalf("unexpected book key: %q", highlight.BookKey)
	}
	if highlight.Content != "Paste this idea." {
		t.Fatalf("expected normalized content, got %q", highlight.Content)
	}
	if highlight.ContentHash == nil || *highlight.ContentHash != computeExpectedContentHash("Paste this idea.") {
		t.Fatalf("unexpected content hash: %#v", highlight.ContentHash)
	}
}

func TestImportPastedHighlightRejectsInvalidSourceApp(t *testing.T) {
	userID := uuid.MustParse("99999999-9999-9999-9999-999999999999")
	repo := &mockImportHighlightRepository{}
	uc := usecase.NewHighlightImportUsecase(repo)

	_, err := uc.ImportPastedHighlight(context.Background(), userID, usecase.ImportPastedHighlightInput{
		Content:   "Paste this idea.",
		SourceApp: "mail",
	})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
	if repo.bulkUpsertCalled {
		t.Fatal("expected BulkUpsert not to be called")
	}
}

func TestListExistingContentHashesNormalizesAndDeduplicates(t *testing.T) {
	userID := uuid.MustParse("99999999-9999-9999-9999-999999999999")
	hash := strings.Repeat("a", 64)
	upperHash := strings.ToUpper(hash)
	repo := &mockImportHighlightRepository{
		existingHashes: []string{hash},
	}
	uc := usecase.NewHighlightQueryUsecase(repo)

	existing, err := uc.ListExistingContentHashes(context.Background(), userID, []string{" " + upperHash + " ", hash, ""})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.checkedHashes) != 1 || repo.checkedHashes[0] != hash {
		t.Fatalf("unexpected checked hashes: %#v", repo.checkedHashes)
	}
	if len(existing) != 1 || existing[0] != hash {
		t.Fatalf("unexpected existing hashes: %#v", existing)
	}
}

func TestListExistingContentHashesRejectsInvalidHash(t *testing.T) {
	userID := uuid.MustParse("99999999-9999-9999-9999-999999999999")
	repo := &mockImportHighlightRepository{}
	uc := usecase.NewHighlightQueryUsecase(repo)

	_, err := uc.ListExistingContentHashes(context.Background(), userID, []string{"not-a-sha256"})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestUpdateExplanationPassesAuthenticatedUserToRepository(t *testing.T) {
	userID := uuid.MustParse("99999999-9999-9999-9999-999999999999")
	highlightID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	repo := &mockImportHighlightRepository{}
	uc := usecase.NewHighlightQueryUsecase(repo)

	_, err := uc.UpdateExplanation(context.Background(), highlightID, userID, " safe explanation ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.updateExplanationCalled {
		t.Fatal("expected repository update to be called")
	}
	if repo.updateExplanationID != highlightID {
		t.Fatalf("unexpected highlight id: %s", repo.updateExplanationID)
	}
	if repo.updateExplanationUserID != userID {
		t.Fatalf("unexpected user id: %s", repo.updateExplanationUserID)
	}
	if repo.updateExplanationValue == nil || *repo.updateExplanationValue != "safe explanation" {
		t.Fatalf("unexpected explanation value: %#v", repo.updateExplanationValue)
	}
}

func computeExpectedContentHash(content string) string {
	sum := sha256.Sum256([]byte(domain.NormalizeText(content)))
	return hex.EncodeToString(sum[:])
}
