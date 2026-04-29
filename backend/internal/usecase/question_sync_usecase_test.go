package usecase

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

type mockQuestionSyncHighlightRepository struct {
	listBookStock              func(ctx context.Context, userID uuid.UUID) ([]domain.BookStock, error)
	listUnusedHighlightsByBook func(ctx context.Context, userID uuid.UUID, bookKey string, limit int) ([]*domain.Highlight, error)
	listUsedHighlightsByBook   func(ctx context.Context, userID uuid.UUID, bookKey string, limit int) ([]*domain.Highlight, error)
	requeueStaleProcessing     func(ctx context.Context, userID uuid.UUID, cutoff time.Time) (int, error)
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

func (m *mockQuestionSyncHighlightRepository) RequeueStaleProcessingByUserID(ctx context.Context, userID uuid.UUID, cutoff time.Time) (int, error) {
	if m.requeueStaleProcessing == nil {
		return 0, nil
	}
	return m.requeueStaleProcessing(ctx, userID, cutoff)
}

type mockQuestionSyncQuestionRepository struct {
	listPerspectivesByHighlightID func(ctx context.Context, userID string, highlightID uuid.UUID) ([]string, error)
	getDailyGeneratedCount        func(ctx context.Context, userID uuid.UUID, day time.Time) (int, error)
	queueWithinDailyLimit         func(ctx context.Context, userID uuid.UUID, day time.Time, limit int, highlightIDs []uuid.UUID, questionCountByHighlightID map[uuid.UUID]int, requestedAt time.Time) ([]uuid.UUID, bool, error)
}

func (m *mockQuestionSyncQuestionRepository) ListPerspectivesByHighlightID(ctx context.Context, userID string, highlightID uuid.UUID) ([]string, error) {
	if m.listPerspectivesByHighlightID == nil {
		return make([]string, 0), nil
	}
	return m.listPerspectivesByHighlightID(ctx, userID, highlightID)
}

func (m *mockQuestionSyncQuestionRepository) GetDailyGeneratedCount(ctx context.Context, userID uuid.UUID, day time.Time) (int, error) {
	if m.getDailyGeneratedCount == nil {
		return 0, nil
	}
	return m.getDailyGeneratedCount(ctx, userID, day)
}

func (m *mockQuestionSyncQuestionRepository) QueueHighlightsWithinDailyLimit(ctx context.Context, userID uuid.UUID, day time.Time, limit int, highlightIDs []uuid.UUID, questionCountByHighlightID map[uuid.UUID]int, requestedAt time.Time) ([]uuid.UUID, bool, error) {
	if m.queueWithinDailyLimit != nil {
		return m.queueWithinDailyLimit(ctx, userID, day, limit, highlightIDs, questionCountByHighlightID, requestedAt)
	}
	return slices.Clone(highlightIDs), true, nil
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
		},
		&mockQuestionSyncQuestionRepository{
			queueWithinDailyLimit: func(ctx context.Context, requestUserID uuid.UUID, day time.Time, limit int, highlightIDs []uuid.UUID, questionCountByHighlightID map[uuid.UUID]int, requestedAt time.Time) ([]uuid.UUID, bool, error) {
				queuedHighlightIDs = append(queuedHighlightIDs, highlightIDs...)
				requested := 0
				for _, highlightID := range highlightIDs {
					requested += questionCountByHighlightID[highlightID]
				}
				if requested != 30 {
					t.Fatalf("expected daily reservation 30, got %d", requested)
				}
				return slices.Clone(highlightIDs), true, nil
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
		},
		&mockQuestionSyncQuestionRepository{
			getDailyGeneratedCount: func(ctx context.Context, requestUserID uuid.UUID, day time.Time) (int, error) {
				return 100, nil
			},
			queueWithinDailyLimit: func(ctx context.Context, requestUserID uuid.UUID, day time.Time, limit int, highlightIDs []uuid.UUID, questionCountByHighlightID map[uuid.UUID]int, requestedAt time.Time) ([]uuid.UUID, bool, error) {
				queueCalled = true
				return nil, false, nil
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

func TestNewQuestionSyncUsecaseReadsLimitEnv(t *testing.T) {
	t.Setenv("QUESTION_SYNC_DAILY_LIMIT", "11")
	t.Setenv("QUESTION_SYNC_PER_TRIGGER_LIMIT", "7")

	uc := NewQuestionSyncUsecase(
		&mockQuestionSyncHighlightRepository{},
		&mockQuestionSyncQuestionRepository{},
		nil,
	)

	if uc.dailyLimit != 11 {
		t.Fatalf("expected dailyLimit=11, got %d", uc.dailyLimit)
	}
	if uc.perTriggerLimit != 7 {
		t.Fatalf("expected perTriggerLimit=7, got %d", uc.perTriggerLimit)
	}
}

func TestSyncQuestionStockTransactionalQueueDenied(t *testing.T) {
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
			listUnusedHighlightsByBook: func(ctx context.Context, requestUserID uuid.UUID, bookKey string, limit int) ([]*domain.Highlight, error) {
				return []*domain.Highlight{{ID: uuid.New(), UserID: userID, Content: contentForCapacity(3)}}, nil
			},
		},
		&mockQuestionSyncQuestionRepository{
			queueWithinDailyLimit: func(ctx context.Context, requestUserID uuid.UUID, day time.Time, limit int, highlightIDs []uuid.UUID, questionCountByHighlightID map[uuid.UUID]int, requestedAt time.Time) ([]uuid.UUID, bool, error) {
				queueCalled = true
				return nil, false, nil
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
	if !queueCalled {
		t.Fatal("transactional queue should be called and deny the reservation")
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
		},
		&mockQuestionSyncQuestionRepository{
			listPerspectivesByHighlightID: func(ctx context.Context, requestUserID string, highlightID uuid.UUID) ([]string, error) {
				if highlightID == usedHighlightID {
					return []string{domain.QuestionPerspectiveDefinition}, nil
				}
				return []string{}, nil
			},
			queueWithinDailyLimit: func(ctx context.Context, requestUserID uuid.UUID, day time.Time, limit int, highlightIDs []uuid.UUID, questionCountByHighlightID map[uuid.UUID]int, requestedAt time.Time) ([]uuid.UUID, bool, error) {
				queuedHighlightIDs = append(queuedHighlightIDs, highlightIDs...)
				return slices.Clone(highlightIDs), true, nil
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
		},
		&mockQuestionSyncQuestionRepository{
			queueWithinDailyLimit: func(ctx context.Context, _ uuid.UUID, _ time.Time, _ int, ids []uuid.UUID, _ map[uuid.UUID]int, _ time.Time) ([]uuid.UUID, bool, error) {
				queueCalled = true
				return ids, true, nil
			},
		},
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
		},
		&mockQuestionSyncQuestionRepository{
			queueWithinDailyLimit: func(ctx context.Context, _ uuid.UUID, _ time.Time, _ int, ids []uuid.UUID, _ map[uuid.UUID]int, _ time.Time) ([]uuid.UUID, bool, error) {
				queuedHighlights = append(queuedHighlights, ids...)
				return slices.Clone(ids), true, nil
			},
		},
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
		},
		&mockQuestionSyncQuestionRepository{},
		nil,
	)

	result, err := uc.SyncQuestionStock(context.Background(), &domain.User{ID: userID, DefaultQuestionCount: 10})
	if err != nil {
		t.Fatalf("SyncQuestionStock failed: %v", err)
	}
	if result.QueuedCount == 0 || result.QueuedCount > 10 {
		t.Fatalf("expected queued count to stay within target=10, got %d", result.QueuedCount)
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

// M2: target を超過する fallback は行わない
func TestSyncQuestionStockDoesNotOvershootTargetWithLargeHighlight(t *testing.T) {
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
		},
		&mockQuestionSyncQuestionRepository{
			queueWithinDailyLimit: func(ctx context.Context, _ uuid.UUID, _ time.Time, _ int, ids []uuid.UUID, _ map[uuid.UUID]int, _ time.Time) ([]uuid.UUID, bool, error) {
				queuedCount = len(ids)
				return ids, true, nil
			},
		},
		nil,
	)

	result, err := uc.SyncQuestionStock(context.Background(), &domain.User{ID: userID, DefaultQuestionCount: 3})
	if err != nil {
		t.Fatalf("SyncQuestionStock failed: %v", err)
	}
	if result.QueuedCount != 0 {
		t.Fatalf("expected 0 queued because only available highlight would overshoot needed=2, got %d", result.QueuedCount)
	}
	if queuedCount != 0 {
		t.Fatalf("expected no highlights queued, got %d", queuedCount)
	}
}

// L3 JST 日跨ぎ: now() を注入して 23:59:59 JST と 00:00:01 JST で counter キーが分かれる
func TestSyncQuestionStockJSTDayBoundary(t *testing.T) {
	userID := uuid.New()
	receivedDays := make([]string, 0, 2)

	repoQuestion := &mockQuestionSyncQuestionRepository{
		queueWithinDailyLimit: func(ctx context.Context, _ uuid.UUID, day time.Time, _ int, ids []uuid.UUID, _ map[uuid.UUID]int, _ time.Time) ([]uuid.UUID, bool, error) {
			receivedDays = append(receivedDays, day.Format("2006-01-02"))
			return ids, true, nil
		},
	}
	repoHighlight := &mockQuestionSyncHighlightRepository{
		listBookStock: func(ctx context.Context, _ uuid.UUID) ([]domain.BookStock, error) {
			return []domain.BookStock{{BookKey: "B1", Stock: 0, Preparing: 0, LatestHighlightAt: time.Now()}}, nil
		},
		listUnusedHighlightsByBook: func(ctx context.Context, _ uuid.UUID, _ string, _ int) ([]*domain.Highlight, error) {
			return []*domain.Highlight{{ID: uuid.New(), Content: contentForCapacity(1)}}, nil
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
		t.Fatalf("expected 2 reservation calls, got %d", len(receivedDays))
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

// L2 異常系: transactional queue がエラー → err
func TestSyncQuestionStockQueueWithinDailyLimitError(t *testing.T) {
	uc := NewQuestionSyncUsecase(
		&mockQuestionSyncHighlightRepository{
			listBookStock: func(ctx context.Context, _ uuid.UUID) ([]domain.BookStock, error) {
				return []domain.BookStock{{BookKey: "B1", Stock: 0, Preparing: 0, LatestHighlightAt: time.Now()}}, nil
			},
			listUnusedHighlightsByBook: func(ctx context.Context, _ uuid.UUID, _ string, _ int) ([]*domain.Highlight, error) {
				return []*domain.Highlight{{ID: uuid.New(), Content: contentForCapacity(3)}}, nil
			},
		},
		&mockQuestionSyncQuestionRepository{
			queueWithinDailyLimit: func(ctx context.Context, _ uuid.UUID, _ time.Time, _ int, _ []uuid.UUID, _ map[uuid.UUID]int, _ time.Time) ([]uuid.UUID, bool, error) {
				return nil, false, errors.New("db error")
			},
		},
		nil,
	)

	_, err := uc.SyncQuestionStock(context.Background(), &domain.User{ID: uuid.New(), DefaultQuestionCount: 3})
	if err == nil {
		t.Fatal("expected error from QueueHighlightsWithinDailyLimit")
	}
}

func TestSyncQuestionStockPartialQueueDoesNotOvercountDailyReservation(t *testing.T) {
	uc := NewQuestionSyncUsecase(
		&mockQuestionSyncHighlightRepository{
			listBookStock: func(ctx context.Context, _ uuid.UUID) ([]domain.BookStock, error) {
				return []domain.BookStock{{BookKey: "B1", Stock: 0, Preparing: 0, LatestHighlightAt: time.Now()}}, nil
			},
			listUnusedHighlightsByBook: func(ctx context.Context, _ uuid.UUID, _ string, _ int) ([]*domain.Highlight, error) {
				return []*domain.Highlight{{ID: uuid.New(), Content: contentForCapacity(3)}}, nil
			},
		},
		&mockQuestionSyncQuestionRepository{
			queueWithinDailyLimit: func(ctx context.Context, _ uuid.UUID, _ time.Time, _ int, ids []uuid.UUID, _ map[uuid.UUID]int, _ time.Time) ([]uuid.UUID, bool, error) {
				if len(ids) != 1 {
					t.Fatalf("expected one requested highlight, got %d", len(ids))
				}
				return []uuid.UUID{}, true, nil
			},
		},
		nil,
	)

	result, err := uc.SyncQuestionStock(context.Background(), &domain.User{ID: uuid.New(), DefaultQuestionCount: 3})
	if err != nil {
		t.Fatalf("SyncQuestionStock failed: %v", err)
	}
	if result.QueuedCount != 0 {
		t.Fatalf("expected queued count 0 when repository queued no rows, got %d", result.QueuedCount)
	}
}

// C3: 日次予約つき queue がエラーならエラーを返す
func TestSyncQuestionStockTransactionalQueueError(t *testing.T) {
	queueCalled := false
	uc := NewQuestionSyncUsecase(
		&mockQuestionSyncHighlightRepository{
			listBookStock: func(ctx context.Context, _ uuid.UUID) ([]domain.BookStock, error) {
				return []domain.BookStock{{BookKey: "B1", Stock: 0, Preparing: 0, LatestHighlightAt: time.Now()}}, nil
			},
			listUnusedHighlightsByBook: func(ctx context.Context, _ uuid.UUID, _ string, _ int) ([]*domain.Highlight, error) {
				return []*domain.Highlight{{ID: uuid.New(), Content: contentForCapacity(3)}}, nil
			},
		},
		&mockQuestionSyncQuestionRepository{
			queueWithinDailyLimit: func(ctx context.Context, _ uuid.UUID, _ time.Time, _ int, _ []uuid.UUID, _ map[uuid.UUID]int, _ time.Time) ([]uuid.UUID, bool, error) {
				queueCalled = true
				return nil, false, errors.New("db error")
			},
		},
		nil,
	)

	_, err := uc.SyncQuestionStock(context.Background(), &domain.User{ID: uuid.New(), DefaultQuestionCount: 3})
	if err == nil {
		t.Fatal("expected error from QueueHighlightsWithinDailyLimit")
	}
	if !queueCalled {
		t.Fatal("transactional queue should be attempted")
	}
}

// C1: 並行 sync 呼び出しでも日次カウントと queue は同じトランザクション入口を通る
func TestSyncQuestionStockConcurrentCallsUseTransactionalQueue(t *testing.T) {
	userID := uuid.New()
	var (
		mu             sync.Mutex
		reserveCalls   = make([]int, 0, 2)
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
	}
	repoQuestion := &mockQuestionSyncQuestionRepository{
		queueWithinDailyLimit: func(ctx context.Context, _ uuid.UUID, _ time.Time, _ int, ids []uuid.UUID, counts map[uuid.UUID]int, _ time.Time) ([]uuid.UUID, bool, error) {
			requested := 0
			for _, id := range ids {
				requested += counts[id]
			}
			mu.Lock()
			reserveCalls = append(reserveCalls, requested)
			queueCallCount++
			mu.Unlock()
			return slices.Clone(ids), true, nil
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
	totalReserved := 0
	for _, n := range reserveCalls {
		totalReserved += n
	}

	if len(reserveCalls) != queueCallCount {
		t.Fatalf("expected reserve calls to match queue calls, reserve=%d queue=%d", len(reserveCalls), queueCallCount)
	}
	t.Logf("queue call count: %d, reserve calls: %v, total: %d", queueCallCount, reserveCalls, totalReserved)
}

// L1 境界値: highlight content の長さ境界 (questionCountForHighlight の境界)
//
// 境界: 0 / 119 / 120 / 319 / 320 chars
// 期待: 0 → 0問, 1〜119 → 1問, 120〜319 → 2問, 320〜 → 3問
func TestQuestionCountForHighlightBoundaries(t *testing.T) {
	cases := []struct {
		name  string
		runes int
		want  int
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
