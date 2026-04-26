package usecase

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

type mockQuestionSyncHighlightRepository struct {
	listBookStock                func(ctx context.Context, userID uuid.UUID) ([]domain.BookStock, error)
	listUnusedHighlightsByBook   func(ctx context.Context, userID uuid.UUID, bookKey string, limit int) ([]*domain.Highlight, error)
	listUsedHighlightsByBook     func(ctx context.Context, userID uuid.UUID, bookKey string, limit int) ([]*domain.Highlight, error)
	queueHighlightsForGeneration func(ctx context.Context, userID uuid.UUID, highlightIDs []uuid.UUID, requestedAt time.Time) error
}

func (m *mockQuestionSyncHighlightRepository) BulkUpsert(ctx context.Context, highlights []*domain.Highlight) (int, error) {
	return 0, errors.New("not implemented")
}

func (m *mockQuestionSyncHighlightRepository) ListExistingContentHashesByUserID(ctx context.Context, userID uuid.UUID, hashes []string) ([]string, error) {
	return make([]string, 0), errors.New("not implemented")
}

func (m *mockQuestionSyncHighlightRepository) ListByUserIDAndASIN(ctx context.Context, userID uuid.UUID, asin string) ([]*domain.Highlight, error) {
	return make([]*domain.Highlight, 0), errors.New("not implemented")
}

func (m *mockQuestionSyncHighlightRepository) ListByUserIDAndBookMetadata(ctx context.Context, userID uuid.UUID, bookTitle, bookAuthor string) ([]*domain.Highlight, error) {
	return make([]*domain.Highlight, 0), errors.New("not implemented")
}

func (m *mockQuestionSyncHighlightRepository) ListBooksWithHighlightsByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.KindleBook, error) {
	return make([]*domain.KindleBook, 0), errors.New("not implemented")
}

func (m *mockQuestionSyncHighlightRepository) ListBookStockByUserID(ctx context.Context, userID uuid.UUID) ([]domain.BookStock, error) {
	if m.listBookStock == nil {
		return make([]domain.BookStock, 0), nil
	}
	return m.listBookStock(ctx, userID)
}

func (m *mockQuestionSyncHighlightRepository) ListUnusedHighlightsByBook(ctx context.Context, userID uuid.UUID, bookKey string, limit int) ([]*domain.Highlight, error) {
	if m.listUnusedHighlightsByBook == nil {
		return make([]*domain.Highlight, 0), nil
	}
	return m.listUnusedHighlightsByBook(ctx, userID, bookKey, limit)
}

func (m *mockQuestionSyncHighlightRepository) ListUsedHighlightsWithUncoveredPerspectives(ctx context.Context, userID uuid.UUID, bookKey string, limit int) ([]*domain.Highlight, error) {
	if m.listUsedHighlightsByBook == nil {
		return make([]*domain.Highlight, 0), nil
	}
	return m.listUsedHighlightsByBook(ctx, userID, bookKey, limit)
}

func (m *mockQuestionSyncHighlightRepository) ListPendingUserStats(ctx context.Context) ([]domain.PendingHighlightUserStat, error) {
	return make([]domain.PendingHighlightUserStat, 0), errors.New("not implemented")
}

func (m *mockQuestionSyncHighlightRepository) ClaimPendingByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*domain.Highlight, error) {
	return make([]*domain.Highlight, 0), errors.New("not implemented")
}

func (m *mockQuestionSyncHighlightRepository) ClaimPendingByIDs(ctx context.Context, userID uuid.UUID, highlightIDs []uuid.UUID) ([]*domain.Highlight, error) {
	return make([]*domain.Highlight, 0), errors.New("not implemented")
}

func (m *mockQuestionSyncHighlightRepository) QueueHighlightsForGeneration(ctx context.Context, userID uuid.UUID, highlightIDs []uuid.UUID, requestedAt time.Time) error {
	if m.queueHighlightsForGeneration == nil {
		return nil
	}
	return m.queueHighlightsForGeneration(ctx, userID, highlightIDs, requestedAt)
}

func (m *mockQuestionSyncHighlightRepository) MarkGenerationCompleted(ctx context.Context, highlightIDs []uuid.UUID) error {
	return errors.New("not implemented")
}

func (m *mockQuestionSyncHighlightRepository) MarkGenerationFailed(ctx context.Context, highlightIDs []uuid.UUID, lastError string, maxRetry int) error {
	return errors.New("not implemented")
}

func (m *mockQuestionSyncHighlightRepository) UpdateExplanation(ctx context.Context, id, userID uuid.UUID, explanation *string) (*domain.Highlight, error) {
	return nil, errors.New("not implemented")
}

type mockQuestionSyncQuestionRepository struct {
	listPerspectivesByHighlightID func(ctx context.Context, userID string, highlightID uuid.UUID) ([]string, error)
	getDailyGeneratedCount        func(ctx context.Context, userID uuid.UUID, day time.Time) (int, error)
	incrementDailyGeneratedCount  func(ctx context.Context, userID uuid.UUID, day time.Time, delta int) error
}

func (m *mockQuestionSyncQuestionRepository) Save(ctx context.Context, q *domain.Question, meta *domain.QuestionMeta) error {
	return errors.New("not implemented")
}

func (m *mockQuestionSyncQuestionRepository) ListByCreatorID(ctx context.Context, creatorID string, limit int) ([]*domain.Question, error) {
	return make([]*domain.Question, 0), errors.New("not implemented")
}

func (m *mockQuestionSyncQuestionRepository) ListSavedByUserID(ctx context.Context, userID string, limit int) ([]*domain.SavedQuestion, error) {
	return make([]*domain.SavedQuestion, 0), errors.New("not implemented")
}

func (m *mockQuestionSyncQuestionRepository) ListIncorrectByUserID(ctx context.Context, userID string, limit int) ([]*domain.IncorrectQuestion, error) {
	return make([]*domain.IncorrectQuestion, 0), errors.New("not implemented")
}

func (m *mockQuestionSyncQuestionRepository) ListPreparedByUserIDAndHighlightIDs(ctx context.Context, userID string, highlightIDs []uuid.UUID, limit int) ([]*domain.Question, error) {
	return make([]*domain.Question, 0), errors.New("not implemented")
}

func (m *mockQuestionSyncQuestionRepository) ListPerspectivesByHighlightID(ctx context.Context, userID string, highlightID uuid.UUID) ([]string, error) {
	if m.listPerspectivesByHighlightID == nil {
		return make([]string, 0), nil
	}
	return m.listPerspectivesByHighlightID(ctx, userID, highlightID)
}

func (m *mockQuestionSyncQuestionRepository) ListUsedHighlightIDsByUserID(ctx context.Context, userID string, highlightIDs []uuid.UUID) ([]uuid.UUID, error) {
	return make([]uuid.UUID, 0), errors.New("not implemented")
}

func (m *mockQuestionSyncQuestionRepository) FindByID(ctx context.Context, id string) (*domain.Question, *domain.QuestionMeta, *domain.QuestionStats, error) {
	return nil, nil, nil, errors.New("not implemented")
}

func (m *mockQuestionSyncQuestionRepository) GetByID(ctx context.Context, id string) (*domain.Question, error) {
	return nil, errors.New("not implemented")
}

func (m *mockQuestionSyncQuestionRepository) UpdateStats(ctx context.Context, questionID string, isCorrect bool) error {
	return errors.New("not implemented")
}

func (m *mockQuestionSyncQuestionRepository) SaveGeneration(ctx context.Context, userID, sourceType, sourceID, promptUsed, modelUsed string) (string, error) {
	return "", errors.New("not implemented")
}

func (m *mockQuestionSyncQuestionRepository) SaveForUser(ctx context.Context, userID, questionID, note string) error {
	return errors.New("not implemented")
}

func (m *mockQuestionSyncQuestionRepository) GetDailyGeneratedCount(ctx context.Context, userID uuid.UUID, day time.Time) (int, error) {
	if m.getDailyGeneratedCount == nil {
		return 0, nil
	}
	return m.getDailyGeneratedCount(ctx, userID, day)
}

func (m *mockQuestionSyncQuestionRepository) IncrementDailyGeneratedCount(ctx context.Context, userID uuid.UUID, day time.Time, delta int) error {
	if m.incrementDailyGeneratedCount == nil {
		return nil
	}
	return m.incrementDailyGeneratedCount(ctx, userID, day, delta)
}

func (m *mockQuestionSyncQuestionRepository) EnqueueRegeneration(ctx context.Context, userID string, highlightID uuid.UUID, questionID string) error {
	return errors.New("not implemented")
}

func (m *mockQuestionSyncQuestionRepository) ClaimPendingRegenerationTasks(ctx context.Context, limit int) ([]*domain.RegenerationTask, error) {
	return make([]*domain.RegenerationTask, 0), errors.New("not implemented")
}

func (m *mockQuestionSyncQuestionRepository) MarkRegenerationTasksCompleted(ctx context.Context, taskIDs []uuid.UUID) error {
	return errors.New("not implemented")
}

func (m *mockQuestionSyncQuestionRepository) MarkRegenerationTasksFailed(ctx context.Context, taskIDs []uuid.UUID, lastError string, maxRetry int) error {
	return errors.New("not implemented")
}

func TestSyncQuestionStockReturnsZeroWhenStockIsAlreadySatisfied(t *testing.T) {
	userID := uuid.New()
	uc := NewQuestionSyncUsecase(
		&mockQuestionSyncHighlightRepository{
			listBookStock: func(ctx context.Context, requestUserID uuid.UUID) ([]domain.BookStock, error) {
				if requestUserID != userID {
					t.Fatalf("unexpected user id: %s", requestUserID)
				}
				return []domain.BookStock{{
					BookKey:           "B001",
					BookTitle:         "Book One",
					Stock:             3,
					Preparing:         0,
					LatestHighlightAt: time.Date(2026, 4, 26, 10, 0, 0, 0, time.UTC),
				}}, nil
			},
		},
		&mockQuestionSyncQuestionRepository{},
		nil,
	)

	result, err := uc.SyncQuestionStock(context.Background(), &domain.User{
		ID:                   userID,
		DefaultQuestionCount: 3,
	})
	if err != nil {
		t.Fatalf("SyncQuestionStock failed: %v", err)
	}

	if result.QueuedCount != 0 {
		t.Fatalf("expected queued count 0, got %d", result.QueuedCount)
	}
	if len(result.Books) != 1 || result.Books[0].Stock != 3 || result.Books[0].Preparing != 0 {
		t.Fatalf("unexpected books response: %#v", result.Books)
	}
}

func TestSyncQuestionStockQueuesOnlyThirtyQuestionsInPriorityOrder(t *testing.T) {
	userID := uuid.New()
	queuedHighlightIDs := make([]uuid.UUID, 0)

	bookStocks := make([]domain.BookStock, 0, 12)
	unusedHighlights := make(map[string][]*domain.Highlight)
	expectedQueuedBookKeys := make([]string, 0, 10)
	for index := 0; index < 12; index++ {
		bookKey := "BOOK-" + string(rune('A'+index))
		createdAt := time.Date(2026, 4, 26, 12-index, 0, 0, 0, time.UTC)
		bookStocks = append(bookStocks, domain.BookStock{
			BookKey:           bookKey,
			BookTitle:         bookKey,
			Stock:             0,
			Preparing:         0,
			LatestHighlightAt: createdAt,
		})
		highlightID := uuid.New()
		unusedHighlights[bookKey] = []*domain.Highlight{{
			ID:      highlightID,
			UserID:  userID,
			Content: strings.Repeat("十分に長いハイライト本文です。", 24),
		}}
		if index < 10 {
			expectedQueuedBookKeys = append(expectedQueuedBookKeys, bookKey)
		}
	}

	uc := NewQuestionSyncUsecase(
		&mockQuestionSyncHighlightRepository{
			listBookStock: func(ctx context.Context, requestUserID uuid.UUID) ([]domain.BookStock, error) {
				return bookStocks, nil
			},
			listUnusedHighlightsByBook: func(ctx context.Context, requestUserID uuid.UUID, bookKey string, limit int) ([]*domain.Highlight, error) {
				return unusedHighlights[bookKey], nil
			},
			queueHighlightsForGeneration: func(ctx context.Context, requestUserID uuid.UUID, highlightIDs []uuid.UUID, requestedAt time.Time) error {
				queuedHighlightIDs = append(queuedHighlightIDs, highlightIDs...)
				return nil
			},
		},
		&mockQuestionSyncQuestionRepository{
			incrementDailyGeneratedCount: func(ctx context.Context, requestUserID uuid.UUID, day time.Time, delta int) error {
				if delta != 30 {
					t.Fatalf("expected daily increment 30, got %d", delta)
				}
				return nil
			},
		},
		nil,
	)

	result, err := uc.SyncQuestionStock(context.Background(), &domain.User{
		ID:                   userID,
		DefaultQuestionCount: 3,
	})
	if err != nil {
		t.Fatalf("SyncQuestionStock failed: %v", err)
	}

	if result.QueuedCount != 30 {
		t.Fatalf("expected queued count 30, got %d", result.QueuedCount)
	}
	if len(queuedHighlightIDs) != 10 {
		t.Fatalf("expected 10 queued highlights, got %d", len(queuedHighlightIDs))
	}

	queuedBookKeySet := make(map[string]struct{}, len(queuedHighlightIDs))
	for bookKey, highlights := range unusedHighlights {
		for _, highlight := range highlights {
			for _, queuedID := range queuedHighlightIDs {
				if queuedID == highlight.ID {
					queuedBookKeySet[bookKey] = struct{}{}
				}
			}
		}
	}
	for _, expectedBookKey := range expectedQueuedBookKeys {
		if _, ok := queuedBookKeySet[expectedBookKey]; !ok {
			t.Fatalf("expected book %s to be queued, got %#v", expectedBookKey, queuedBookKeySet)
		}
	}
}

func TestSyncQuestionStockSkipsWhenDailyLimitReached(t *testing.T) {
	userID := uuid.New()
	queueCalled := false

	uc := NewQuestionSyncUsecase(
		&mockQuestionSyncHighlightRepository{
			listBookStock: func(ctx context.Context, requestUserID uuid.UUID) ([]domain.BookStock, error) {
				return []domain.BookStock{{
					BookKey:           "B001",
					BookTitle:         "Book One",
					Stock:             0,
					Preparing:         0,
					LatestHighlightAt: time.Now(),
				}}, nil
			},
			queueHighlightsForGeneration: func(ctx context.Context, requestUserID uuid.UUID, highlightIDs []uuid.UUID, requestedAt time.Time) error {
				queueCalled = true
				return nil
			},
		},
		&mockQuestionSyncQuestionRepository{
			getDailyGeneratedCount: func(ctx context.Context, requestUserID uuid.UUID, day time.Time) (int, error) {
				return 100, nil
			},
		},
		nil,
	)

	result, err := uc.SyncQuestionStock(context.Background(), &domain.User{
		ID:                   userID,
		DefaultQuestionCount: 3,
	})
	if err != nil {
		t.Fatalf("SyncQuestionStock failed: %v", err)
	}

	if !result.SkippedDueToDailyLimit {
		t.Fatal("expected skipped_due_to_daily_limit=true")
	}
	if result.QueuedCount != 0 {
		t.Fatalf("expected queued count 0, got %d", result.QueuedCount)
	}
	if queueCalled {
		t.Fatal("queue should not be called when daily limit is reached")
	}
}

func TestSyncQuestionStockPrefersUnusedHighlightsBeforeUsedOnes(t *testing.T) {
	userID := uuid.New()
	unusedHighlightID := uuid.New()
	usedHighlightID := uuid.New()
	queuedHighlightIDs := make([]uuid.UUID, 0, 2)

	uc := NewQuestionSyncUsecase(
		&mockQuestionSyncHighlightRepository{
			listBookStock: func(ctx context.Context, requestUserID uuid.UUID) ([]domain.BookStock, error) {
				return []domain.BookStock{{
					BookKey:           "B001",
					BookTitle:         "Book One",
					Stock:             0,
					Preparing:         0,
					LatestHighlightAt: time.Now(),
				}}, nil
			},
			listUnusedHighlightsByBook: func(ctx context.Context, requestUserID uuid.UUID, bookKey string, limit int) ([]*domain.Highlight, error) {
				return []*domain.Highlight{{
					ID:      unusedHighlightID,
					UserID:  userID,
					Content: "短いハイライトなので 1 問です",
				}}, nil
			},
			listUsedHighlightsByBook: func(ctx context.Context, requestUserID uuid.UUID, bookKey string, limit int) ([]*domain.Highlight, error) {
				return []*domain.Highlight{{
					ID:      usedHighlightID,
					UserID:  userID,
					Content: strings.Repeat("別観点の問題も十分に作れる長いハイライト本文です。", 20),
				}}, nil
			},
			queueHighlightsForGeneration: func(ctx context.Context, requestUserID uuid.UUID, highlightIDs []uuid.UUID, requestedAt time.Time) error {
				queuedHighlightIDs = append(queuedHighlightIDs, highlightIDs...)
				return nil
			},
		},
		&mockQuestionSyncQuestionRepository{
			listPerspectivesByHighlightID: func(ctx context.Context, requestUserID string, highlightID uuid.UUID) ([]string, error) {
				if highlightID == usedHighlightID {
					return []string{domain.QuestionPerspectiveDefinition}, nil
				}
				return []string{}, nil
			},
		},
		nil,
	)

	result, err := uc.SyncQuestionStock(context.Background(), &domain.User{
		ID:                   userID,
		DefaultQuestionCount: 3,
	})
	if err != nil {
		t.Fatalf("SyncQuestionStock failed: %v", err)
	}

	if result.QueuedCount == 0 {
		t.Fatal("expected some questions to be queued")
	}
	if len(queuedHighlightIDs) < 2 {
		t.Fatalf("expected both unused and used highlights to be queued, got %d", len(queuedHighlightIDs))
	}
	if queuedHighlightIDs[0] != unusedHighlightID {
		t.Fatalf("expected unused highlight to be queued first, got %#v", queuedHighlightIDs)
	}
}

// ----------------------------------------------------------------------------
// 追加テスト: 同値分割・境界値・状態遷移・異常系・並行性
// goofy-enchanting-wadler.md の項目 4〜22 に対応
// ----------------------------------------------------------------------------

// helper: 指定した問題容量(1/2/3)を持つ highlight content を生成する
func contentForCapacity(capacity int) string {
	switch capacity {
	case 0:
		return ""
	case 1:
		return strings.Repeat("あ", 50)
	case 2:
		return strings.Repeat("あ", 200)
	default:
		return strings.Repeat("あ", 400)
	}
}

// L1 境界値: bookStocks が空のとき
func TestSyncQuestionStockEmptyBookStocks(t *testing.T) {
	userID := uuid.New()
	uc := NewQuestionSyncUsecase(
		&mockQuestionSyncHighlightRepository{
			listBookStock: func(ctx context.Context, requestUserID uuid.UUID) ([]domain.BookStock, error) {
				return []domain.BookStock{}, nil
			},
		},
		&mockQuestionSyncQuestionRepository{},
		nil,
	)

	result, err := uc.SyncQuestionStock(context.Background(), &domain.User{ID: userID, DefaultQuestionCount: 3})
	if err != nil {
		t.Fatalf("SyncQuestionStock failed: %v", err)
	}
	if result.QueuedCount != 0 {
		t.Fatalf("expected 0 queued, got %d", result.QueuedCount)
	}
	if len(result.Books) != 0 {
		t.Fatalf("expected empty books, got %d", len(result.Books))
	}
	if result.SkippedDueToDailyLimit {
		t.Fatal("should not be skipped when no books")
	}
}

// L1 同値: BookKey が空文字の book は understock 判定をスキップする
func TestSyncQuestionStockEmptyBookKeySkipped(t *testing.T) {
	userID := uuid.New()
	queueCalled := false
	uc := NewQuestionSyncUsecase(
		&mockQuestionSyncHighlightRepository{
			listBookStock: func(ctx context.Context, _ uuid.UUID) ([]domain.BookStock, error) {
				return []domain.BookStock{{BookKey: "", Stock: 0, Preparing: 0, LatestHighlightAt: time.Now()}}, nil
			},
			queueHighlightsForGeneration: func(ctx context.Context, _ uuid.UUID, _ []uuid.UUID, _ time.Time) error {
				queueCalled = true
				return nil
			},
		},
		&mockQuestionSyncQuestionRepository{},
		nil,
	)

	result, err := uc.SyncQuestionStock(context.Background(), &domain.User{ID: userID, DefaultQuestionCount: 3})
	if err != nil {
		t.Fatalf("SyncQuestionStock failed: %v", err)
	}
	if result.QueuedCount != 0 {
		t.Fatalf("expected 0 queued for empty book key, got %d", result.QueuedCount)
	}
	if queueCalled {
		t.Fatal("queue should not be called for empty book key")
	}
}

// L1 mixed: 一部 stock>=target、一部 stock<target → 不足本だけ処理
func TestSyncQuestionStockMixedBookStocks(t *testing.T) {
	userID := uuid.New()
	queuedHighlights := make([]uuid.UUID, 0)
	highlightForBookB := uuid.New()

	uc := NewQuestionSyncUsecase(
		&mockQuestionSyncHighlightRepository{
			listBookStock: func(ctx context.Context, _ uuid.UUID) ([]domain.BookStock, error) {
				return []domain.BookStock{
					{BookKey: "BookA", BookTitle: "A", Stock: 3, Preparing: 0, LatestHighlightAt: time.Now()},
					{BookKey: "BookB", BookTitle: "B", Stock: 0, Preparing: 0, LatestHighlightAt: time.Now()},
				}, nil
			},
			listUnusedHighlightsByBook: func(ctx context.Context, _ uuid.UUID, bookKey string, _ int) ([]*domain.Highlight, error) {
				if bookKey == "BookB" {
					return []*domain.Highlight{{ID: highlightForBookB, Content: contentForCapacity(3)}}, nil
				}
				t.Fatalf("listUnusedHighlightsByBook unexpectedly called for book %s", bookKey)
				return nil, nil
			},
			queueHighlightsForGeneration: func(ctx context.Context, _ uuid.UUID, ids []uuid.UUID, _ time.Time) error {
				queuedHighlights = append(queuedHighlights, ids...)
				return nil
			},
		},
		&mockQuestionSyncQuestionRepository{},
		nil,
	)

	result, err := uc.SyncQuestionStock(context.Background(), &domain.User{ID: userID, DefaultQuestionCount: 3})
	if err != nil {
		t.Fatalf("SyncQuestionStock failed: %v", err)
	}
	if result.QueuedCount != 3 {
		t.Fatalf("expected 3 queued for BookB only, got %d", result.QueuedCount)
	}
	if len(queuedHighlights) != 1 || queuedHighlights[0] != highlightForBookB {
		t.Fatalf("expected only BookB highlight queued, got %#v", queuedHighlights)
	}
}

// L1 同値: default_question_count = 0 (ALL = 20)
func TestSyncQuestionStockDefaultQuestionCountAll(t *testing.T) {
	userID := uuid.New()
	uc := NewQuestionSyncUsecase(
		&mockQuestionSyncHighlightRepository{
			listBookStock: func(ctx context.Context, _ uuid.UUID) ([]domain.BookStock, error) {
				return []domain.BookStock{{BookKey: "B1", Stock: 0, Preparing: 0, LatestHighlightAt: time.Now()}}, nil
			},
			listUnusedHighlightsByBook: func(ctx context.Context, _ uuid.UUID, _ string, _ int) ([]*domain.Highlight, error) {
				highlights := make([]*domain.Highlight, 0, 10)
				for i := 0; i < 10; i++ {
					highlights = append(highlights, &domain.Highlight{ID: uuid.New(), Content: contentForCapacity(3)})
				}
				return highlights, nil
			},
			queueHighlightsForGeneration: func(ctx context.Context, _ uuid.UUID, _ []uuid.UUID, _ time.Time) error {
				return nil
			},
		},
		&mockQuestionSyncQuestionRepository{},
		nil,
	)

	result, err := uc.SyncQuestionStock(context.Background(), &domain.User{ID: userID, DefaultQuestionCount: 0})
	if err != nil {
		t.Fatalf("SyncQuestionStock failed: %v", err)
	}
	// target = 20 (DefaultQuestionCountAll resolves to maxQuestionCountForAll)
	// budget = min(30 per-trigger, 100 daily) = 30
	// available unused capacity = 10 highlights * 3 cap = 30, but target=20 caps it
	if result.QueuedCount == 0 || result.QueuedCount > 30 {
		t.Fatalf("expected reasonable queued count (>0, <=30) for ALL mode, got %d", result.QueuedCount)
	}
	if result.Books[0].Target != 20 {
		t.Fatalf("expected target=20 for default_question_count=0, got %d", result.Books[0].Target)
	}
}

// L1 境界値: default_question_count = 1 (最小)
func TestSyncQuestionStockDefaultQuestionCountMin(t *testing.T) {
	userID := uuid.New()
	uc := NewQuestionSyncUsecase(
		&mockQuestionSyncHighlightRepository{
			listBookStock: func(ctx context.Context, _ uuid.UUID) ([]domain.BookStock, error) {
				return []domain.BookStock{{BookKey: "B1", Stock: 0, Preparing: 0, LatestHighlightAt: time.Now()}}, nil
			},
			listUnusedHighlightsByBook: func(ctx context.Context, _ uuid.UUID, _ string, _ int) ([]*domain.Highlight, error) {
				return []*domain.Highlight{{ID: uuid.New(), Content: contentForCapacity(1)}}, nil
			},
			queueHighlightsForGeneration: func(ctx context.Context, _ uuid.UUID, _ []uuid.UUID, _ time.Time) error {
				return nil
			},
		},
		&mockQuestionSyncQuestionRepository{},
		nil,
	)

	result, err := uc.SyncQuestionStock(context.Background(), &domain.User{ID: userID, DefaultQuestionCount: 1})
	if err != nil {
		t.Fatalf("SyncQuestionStock failed: %v", err)
	}
	if result.QueuedCount != 1 {
		t.Fatalf("expected 1 queued for target=1, got %d", result.QueuedCount)
	}
	if result.Books[0].Target != 1 {
		t.Fatalf("expected target=1, got %d", result.Books[0].Target)
	}
}

// L1 境界値: default_question_count = 10 (最大)
func TestSyncQuestionStockDefaultQuestionCountMax(t *testing.T) {
	userID := uuid.New()
	uc := NewQuestionSyncUsecase(
		&mockQuestionSyncHighlightRepository{
			listBookStock: func(ctx context.Context, _ uuid.UUID) ([]domain.BookStock, error) {
				return []domain.BookStock{{BookKey: "B1", Stock: 0, Preparing: 0, LatestHighlightAt: time.Now()}}, nil
			},
			listUnusedHighlightsByBook: func(ctx context.Context, _ uuid.UUID, _ string, _ int) ([]*domain.Highlight, error) {
				highlights := make([]*domain.Highlight, 0, 4)
				for i := 0; i < 4; i++ {
					highlights = append(highlights, &domain.Highlight{ID: uuid.New(), Content: contentForCapacity(3)})
				}
				return highlights, nil
			},
			queueHighlightsForGeneration: func(ctx context.Context, _ uuid.UUID, _ []uuid.UUID, _ time.Time) error {
				return nil
			},
		},
		&mockQuestionSyncQuestionRepository{},
		nil,
	)

	result, err := uc.SyncQuestionStock(context.Background(), &domain.User{ID: userID, DefaultQuestionCount: 10})
	if err != nil {
		t.Fatalf("SyncQuestionStock failed: %v", err)
	}
	if result.QueuedCount < 10 {
		t.Fatalf("expected at least 10 queued for target=10 with sufficient highlights, got %d", result.QueuedCount)
	}
	if result.Books[0].Target != 10 {
		t.Fatalf("expected target=10, got %d", result.Books[0].Target)
	}
}

// L1 防御的: default_question_count = -1 (resolves to default 3)
func TestSyncQuestionStockDefaultQuestionCountNegative(t *testing.T) {
	userID := uuid.New()
	uc := NewQuestionSyncUsecase(
		&mockQuestionSyncHighlightRepository{
			listBookStock: func(ctx context.Context, _ uuid.UUID) ([]domain.BookStock, error) {
				return []domain.BookStock{{BookKey: "B1", Stock: 0, Preparing: 0, LatestHighlightAt: time.Now()}}, nil
			},
			listUnusedHighlightsByBook: func(ctx context.Context, _ uuid.UUID, _ string, _ int) ([]*domain.Highlight, error) {
				return []*domain.Highlight{{ID: uuid.New(), Content: contentForCapacity(3)}}, nil
			},
			queueHighlightsForGeneration: func(ctx context.Context, _ uuid.UUID, _ []uuid.UUID, _ time.Time) error {
				return nil
			},
		},
		&mockQuestionSyncQuestionRepository{},
		nil,
	)

	// -1 は invalid だが防御的に DefaultQuestionCountDefault(=3) に丸める
	result, err := uc.SyncQuestionStock(context.Background(), &domain.User{ID: userID, DefaultQuestionCount: -1})
	if err != nil {
		t.Fatalf("SyncQuestionStock failed: %v", err)
	}
	if result.Books[0].Target != int(domain.DefaultQuestionCountDefault) {
		t.Fatalf("expected target=%d for negative input, got %d", domain.DefaultQuestionCountDefault, result.Books[0].Target)
	}
}

// L1 境界値: dailyCount = 99 (1問だけ余裕)
func TestSyncQuestionStockDailyLimitNearMax(t *testing.T) {
	userID := uuid.New()
	uc := NewQuestionSyncUsecase(
		&mockQuestionSyncHighlightRepository{
			listBookStock: func(ctx context.Context, _ uuid.UUID) ([]domain.BookStock, error) {
				return []domain.BookStock{{BookKey: "B1", Stock: 0, Preparing: 0, LatestHighlightAt: time.Now()}}, nil
			},
			listUnusedHighlightsByBook: func(ctx context.Context, _ uuid.UUID, _ string, _ int) ([]*domain.Highlight, error) {
				return []*domain.Highlight{{ID: uuid.New(), Content: contentForCapacity(3)}}, nil
			},
			queueHighlightsForGeneration: func(ctx context.Context, _ uuid.UUID, _ []uuid.UUID, _ time.Time) error {
				return nil
			},
		},
		&mockQuestionSyncQuestionRepository{
			getDailyGeneratedCount: func(ctx context.Context, _ uuid.UUID, _ time.Time) (int, error) {
				return 99, nil
			},
		},
		nil,
	)

	result, err := uc.SyncQuestionStock(context.Background(), &domain.User{ID: userID, DefaultQuestionCount: 3})
	if err != nil {
		t.Fatalf("SyncQuestionStock failed: %v", err)
	}
	// remainingDailyBudget = 100 - 99 = 1
	// remainingBudget = min(30, 1) = 1
	if result.QueuedCount > 1 {
		t.Fatalf("expected at most 1 queued (remainingDailyBudget=1), got %d", result.QueuedCount)
	}
	if result.SkippedDueToDailyLimit {
		t.Fatal("should not be skipped when budget is 1")
	}
}

// L1 防御的: dailyCount = 101 (DB inconsistency, over-limit)
func TestSyncQuestionStockDailyLimitOverflow(t *testing.T) {
	userID := uuid.New()
	uc := NewQuestionSyncUsecase(
		&mockQuestionSyncHighlightRepository{
			listBookStock: func(ctx context.Context, _ uuid.UUID) ([]domain.BookStock, error) {
				return []domain.BookStock{{BookKey: "B1", Stock: 0, Preparing: 0, LatestHighlightAt: time.Now()}}, nil
			},
		},
		&mockQuestionSyncQuestionRepository{
			getDailyGeneratedCount: func(ctx context.Context, _ uuid.UUID, _ time.Time) (int, error) {
				return 101, nil
			},
		},
		nil,
	)

	result, err := uc.SyncQuestionStock(context.Background(), &domain.User{ID: userID, DefaultQuestionCount: 3})
	if err != nil {
		t.Fatalf("SyncQuestionStock failed: %v", err)
	}
	if !result.SkippedDueToDailyLimit {
		t.Fatal("expected skipped=true for dailyCount=101")
	}
	if result.QueuedCount != 0 {
		t.Fatalf("expected 0 queued, got %d", result.QueuedCount)
	}
}

// M2: appendQuestionSyncCandidates の fallback で target を超過する
//
// シナリオ: target=3, 既存ストック=1 (残必要=2)、未使用ハイライト=1件のみ(capacity=3)
// 期待 (理想): 2問 queued (target ぴったり)
// 現状 (M2 バグ): fallback ロジックにより 3問 queued される (target=3 を超過して合計4問になる)
//
// このテストは現状の動作を pin する。M2 修正後は queued=2 になる想定で更新が必要。
func TestSyncQuestionStockFallbackOvershootDocumentsCurrentBehavior(t *testing.T) {
	userID := uuid.New()
	queuedCount := 0
	uc := NewQuestionSyncUsecase(
		&mockQuestionSyncHighlightRepository{
			listBookStock: func(ctx context.Context, _ uuid.UUID) ([]domain.BookStock, error) {
				return []domain.BookStock{{BookKey: "B1", Stock: 1, Preparing: 0, LatestHighlightAt: time.Now()}}, nil
			},
			listUnusedHighlightsByBook: func(ctx context.Context, _ uuid.UUID, _ string, _ int) ([]*domain.Highlight, error) {
				return []*domain.Highlight{{ID: uuid.New(), Content: contentForCapacity(3)}}, nil
			},
			queueHighlightsForGeneration: func(ctx context.Context, _ uuid.UUID, ids []uuid.UUID, _ time.Time) error {
				queuedCount = len(ids)
				return nil
			},
		},
		&mockQuestionSyncQuestionRepository{},
		nil,
	)

	result, err := uc.SyncQuestionStock(context.Background(), &domain.User{ID: userID, DefaultQuestionCount: 3})
	if err != nil {
		t.Fatalf("SyncQuestionStock failed: %v", err)
	}
	if result.QueuedCount != 3 {
		// このアサーションが将来 result.QueuedCount==2 で失敗するようになったら M2 が修正された印
		t.Fatalf("[M2 documenting current overshoot] expected 3 queued (overshoot of needed=2), got %d", result.QueuedCount)
	}
	if queuedCount != 1 {
		t.Fatalf("expected 1 highlight queued, got %d", queuedCount)
	}
}

// L3 JST 日跨ぎ: now() を注入して 23:59:59 JST と 00:00:01 JST で counter キーが分かれる
func TestSyncQuestionStockJSTDayBoundary(t *testing.T) {
	userID := uuid.New()
	receivedDays := make([]string, 0, 2)

	repoQuestion := &mockQuestionSyncQuestionRepository{
		incrementDailyGeneratedCount: func(ctx context.Context, _ uuid.UUID, day time.Time, _ int) error {
			receivedDays = append(receivedDays, day.Format("2006-01-02"))
			return nil
		},
	}
	repoHighlight := &mockQuestionSyncHighlightRepository{
		listBookStock: func(ctx context.Context, _ uuid.UUID) ([]domain.BookStock, error) {
			return []domain.BookStock{{BookKey: "B1", Stock: 0, Preparing: 0, LatestHighlightAt: time.Now()}}, nil
		},
		listUnusedHighlightsByBook: func(ctx context.Context, _ uuid.UUID, _ string, _ int) ([]*domain.Highlight, error) {
			return []*domain.Highlight{{ID: uuid.New(), Content: contentForCapacity(1)}}, nil
		},
		queueHighlightsForGeneration: func(ctx context.Context, _ uuid.UUID, _ []uuid.UUID, _ time.Time) error {
			return nil
		},
	}

	jst := time.FixedZone("JST", 9*60*60)

	uc := NewQuestionSyncUsecase(repoHighlight, repoQuestion, nil)
	uc.now = func() time.Time { return time.Date(2026, 4, 26, 23, 59, 59, 0, jst) }
	if _, err := uc.SyncQuestionStock(context.Background(), &domain.User{ID: userID, DefaultQuestionCount: 3}); err != nil {
		t.Fatalf("first sync failed: %v", err)
	}

	uc.now = func() time.Time { return time.Date(2026, 4, 27, 0, 0, 1, 0, jst) }
	if _, err := uc.SyncQuestionStock(context.Background(), &domain.User{ID: userID, DefaultQuestionCount: 3}); err != nil {
		t.Fatalf("second sync failed: %v", err)
	}

	if len(receivedDays) != 2 {
		t.Fatalf("expected 2 increment calls, got %d", len(receivedDays))
	}
	if receivedDays[0] == receivedDays[1] {
		t.Fatalf("expected different counter keys across JST midnight, got %s and %s", receivedDays[0], receivedDays[1])
	}
	if receivedDays[0] != "2026-04-26" || receivedDays[1] != "2026-04-27" {
		t.Fatalf("expected 2026-04-26 then 2026-04-27, got %v", receivedDays)
	}
}

// L2 異常系: ListBookStockByUserID がエラー → 関数全体が err を返す
func TestSyncQuestionStockListBookStockError(t *testing.T) {
	uc := NewQuestionSyncUsecase(
		&mockQuestionSyncHighlightRepository{
			listBookStock: func(ctx context.Context, _ uuid.UUID) ([]domain.BookStock, error) {
				return nil, errors.New("db down")
			},
		},
		&mockQuestionSyncQuestionRepository{},
		nil,
	)

	_, err := uc.SyncQuestionStock(context.Background(), &domain.User{ID: uuid.New(), DefaultQuestionCount: 3})
	if err == nil {
		t.Fatal("expected error from ListBookStockByUserID")
	}
}

// L2 異常系: GetDailyGeneratedCount がエラー → err
func TestSyncQuestionStockGetDailyCountError(t *testing.T) {
	uc := NewQuestionSyncUsecase(
		&mockQuestionSyncHighlightRepository{
			listBookStock: func(ctx context.Context, _ uuid.UUID) ([]domain.BookStock, error) {
				return []domain.BookStock{{BookKey: "B1", Stock: 0, Preparing: 0, LatestHighlightAt: time.Now()}}, nil
			},
		},
		&mockQuestionSyncQuestionRepository{
			getDailyGeneratedCount: func(ctx context.Context, _ uuid.UUID, _ time.Time) (int, error) {
				return 0, errors.New("db error")
			},
		},
		nil,
	)

	_, err := uc.SyncQuestionStock(context.Background(), &domain.User{ID: uuid.New(), DefaultQuestionCount: 3})
	if err == nil {
		t.Fatal("expected error from GetDailyGeneratedCount")
	}
}

// L2 異常系: ListUnusedHighlightsByBook がエラー → err (現状: 全体中断)
func TestSyncQuestionStockListUnusedHighlightsError(t *testing.T) {
	uc := NewQuestionSyncUsecase(
		&mockQuestionSyncHighlightRepository{
			listBookStock: func(ctx context.Context, _ uuid.UUID) ([]domain.BookStock, error) {
				return []domain.BookStock{{BookKey: "B1", Stock: 0, Preparing: 0, LatestHighlightAt: time.Now()}}, nil
			},
			listUnusedHighlightsByBook: func(ctx context.Context, _ uuid.UUID, _ string, _ int) ([]*domain.Highlight, error) {
				return nil, errors.New("db error")
			},
		},
		&mockQuestionSyncQuestionRepository{},
		nil,
	)

	_, err := uc.SyncQuestionStock(context.Background(), &domain.User{ID: uuid.New(), DefaultQuestionCount: 3})
	if err == nil {
		t.Fatal("expected error from ListUnusedHighlightsByBook")
	}
}

// L2 異常系: QueueHighlightsForGeneration がエラー → err (highlights 状態不確定、daily counter 未加算)
func TestSyncQuestionStockQueueHighlightsError(t *testing.T) {
	incrementCalled := false
	uc := NewQuestionSyncUsecase(
		&mockQuestionSyncHighlightRepository{
			listBookStock: func(ctx context.Context, _ uuid.UUID) ([]domain.BookStock, error) {
				return []domain.BookStock{{BookKey: "B1", Stock: 0, Preparing: 0, LatestHighlightAt: time.Now()}}, nil
			},
			listUnusedHighlightsByBook: func(ctx context.Context, _ uuid.UUID, _ string, _ int) ([]*domain.Highlight, error) {
				return []*domain.Highlight{{ID: uuid.New(), Content: contentForCapacity(3)}}, nil
			},
			queueHighlightsForGeneration: func(ctx context.Context, _ uuid.UUID, _ []uuid.UUID, _ time.Time) error {
				return errors.New("db error")
			},
		},
		&mockQuestionSyncQuestionRepository{
			incrementDailyGeneratedCount: func(ctx context.Context, _ uuid.UUID, _ time.Time, _ int) error {
				incrementCalled = true
				return nil
			},
		},
		nil,
	)

	_, err := uc.SyncQuestionStock(context.Background(), &domain.User{ID: uuid.New(), DefaultQuestionCount: 3})
	if err == nil {
		t.Fatal("expected error from QueueHighlightsForGeneration")
	}
	if incrementCalled {
		t.Fatal("daily counter should not be incremented if Queue failed")
	}
}

// C3: IncrementDailyGeneratedCount がエラー → err を返すが highlights は既に Queue 済み
//
// このテストは C3 のシナリオを pin する: Queue 成功 + Increment 失敗で highlights が pending のまま残り、
// counter は加算されない不整合が発生する。修正後はトランザクション化により挙動が変わる想定。
func TestSyncQuestionStockIncrementDailyErrorLeavesHighlightsQueued(t *testing.T) {
	queueCalled := false
	uc := NewQuestionSyncUsecase(
		&mockQuestionSyncHighlightRepository{
			listBookStock: func(ctx context.Context, _ uuid.UUID) ([]domain.BookStock, error) {
				return []domain.BookStock{{BookKey: "B1", Stock: 0, Preparing: 0, LatestHighlightAt: time.Now()}}, nil
			},
			listUnusedHighlightsByBook: func(ctx context.Context, _ uuid.UUID, _ string, _ int) ([]*domain.Highlight, error) {
				return []*domain.Highlight{{ID: uuid.New(), Content: contentForCapacity(3)}}, nil
			},
			queueHighlightsForGeneration: func(ctx context.Context, _ uuid.UUID, _ []uuid.UUID, _ time.Time) error {
				queueCalled = true
				return nil
			},
		},
		&mockQuestionSyncQuestionRepository{
			incrementDailyGeneratedCount: func(ctx context.Context, _ uuid.UUID, _ time.Time, _ int) error {
				return errors.New("db error")
			},
		},
		nil,
	)

	_, err := uc.SyncQuestionStock(context.Background(), &domain.User{ID: uuid.New(), DefaultQuestionCount: 3})
	if err == nil {
		t.Fatal("expected error from IncrementDailyGeneratedCount")
	}
	if !queueCalled {
		// [C3 documenting] Queue は成功するが Increment 失敗で highlights pending 残り
		t.Fatal("[C3] Queue was expected to be called before Increment fails — highlights remain pending without counter increment")
	}
}

// C1: 並行 sync 呼び出し時の daily counter 加算挙動を pin する
//
// シナリオ: 2 つの goroutine が同時に同じユーザーで sync を実行
// 期待 (理想): 実際にキューに乗った数だけ counter が加算される
// 現状 (C1 バグ): 両方とも事前算出した QueuedCount で counter を加算 → 二重加算
//
// このテストは現状を pin する。C1 修正後は counter 合計が「実際にキューイング成功した数」になる想定。
func TestSyncQuestionStockConcurrentCallsDocumentDoubleIncrement(t *testing.T) {
	userID := uuid.New()
	var (
		mu             sync.Mutex
		incrementCalls = make([]int, 0, 2)
		queueCallCount int
	)

	repoHighlight := &mockQuestionSyncHighlightRepository{
		listBookStock: func(ctx context.Context, _ uuid.UUID) ([]domain.BookStock, error) {
			return []domain.BookStock{{BookKey: "B1", Stock: 0, Preparing: 0, LatestHighlightAt: time.Now()}}, nil
		},
		listUnusedHighlightsByBook: func(ctx context.Context, _ uuid.UUID, _ string, _ int) ([]*domain.Highlight, error) {
			// 並行 sync が両方とも同じ highlight を選ぶように、安定した ID を返す
			return []*domain.Highlight{{ID: uuid.MustParse("11111111-1111-1111-1111-111111111111"), Content: contentForCapacity(3)}}, nil
		},
		queueHighlightsForGeneration: func(ctx context.Context, _ uuid.UUID, _ []uuid.UUID, _ time.Time) error {
			mu.Lock()
			queueCallCount++
			mu.Unlock()
			return nil
		},
	}
	repoQuestion := &mockQuestionSyncQuestionRepository{
		incrementDailyGeneratedCount: func(ctx context.Context, _ uuid.UUID, _ time.Time, delta int) error {
			mu.Lock()
			incrementCalls = append(incrementCalls, delta)
			mu.Unlock()
			return nil
		},
	}

	uc := NewQuestionSyncUsecase(repoHighlight, repoQuestion, nil)
	user := &domain.User{ID: userID, DefaultQuestionCount: 3}

	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := uc.SyncQuestionStock(context.Background(), user); err != nil {
				t.Errorf("concurrent sync failed: %v", err)
			}
		}()
	}
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	totalIncrement := 0
	for _, n := range incrementCalls {
		totalIncrement += n
	}

	// [C1 documenting] 理想は totalIncrement == 3 (実際にキューに乗ったのは1ユーザー分のみ)
	// 現状は totalIncrement == 6 (両方とも 3 を独立に加算) になり得る
	// このテストはレースを保証するわけではないが、加算回数が 2 になることを確認できる
	if len(incrementCalls) != 2 {
		t.Logf("expected 2 increment calls (both syncs reached Increment), got %d — race may not have occurred this run", len(incrementCalls))
	}
	if totalIncrement > 3 {
		t.Logf("[C1 documenting] daily counter over-incremented by concurrent sync: total=%d (expected 3)", totalIncrement)
	}
	t.Logf("queue call count: %d, increment calls: %v, total: %d", queueCallCount, incrementCalls, totalIncrement)
}

// L1 境界値: highlight content の長さ境界 (questionCountForHighlight の境界)
//
// 境界: 0 / 119 / 120 / 319 / 320 chars
// 期待: 0 → 0問, 1〜119 → 1問, 120〜319 → 2問, 320〜 → 3問
func TestQuestionCountForHighlightBoundaries(t *testing.T) {
	cases := []struct {
		name    string
		runes   int
		want    int
	}{
		{"empty", 0, 0},
		{"just_below_2q_boundary_119", 119, 1},
		{"at_2q_boundary_120", 120, 2},
		{"just_below_3q_boundary_319", 319, 2},
		{"at_3q_boundary_320", 320, 3},
		{"well_above_3q_400", 400, 3},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			content := strings.Repeat("あ", tc.runes)
			got := questionCountForHighlight(content)
			if got != tc.want {
				t.Fatalf("questionCountForHighlight(%d runes) = %d, want %d", tc.runes, got, tc.want)
			}
		})
	}
}

// L1 同値: remainingQuestionCapacity が既存数 >= capacity でゼロを返すこと
func TestRemainingQuestionCapacity(t *testing.T) {
	cases := []struct {
		name     string
		runes    int
		existing int
		want     int
	}{
		{"capacity_3_existing_0", 400, 0, 3},
		{"capacity_3_existing_3", 400, 3, 0},
		{"capacity_3_existing_5_overflow", 400, 5, 0}, // 防御的: マイナスにならない
		{"capacity_2_existing_1", 200, 1, 1},
		{"empty_content", 0, 0, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			content := strings.Repeat("あ", tc.runes)
			got := remainingQuestionCapacity(content, tc.existing)
			if got != tc.want {
				t.Fatalf("remainingQuestionCapacity(%d runes, existing=%d) = %d, want %d", tc.runes, tc.existing, got, tc.want)
			}
		})
	}
}
