package usecase

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

const (
	maxQuestionsPerSyncRun = 30
	maxQuestionsPerDay     = 100
)

var questionSyncDailyLocation = time.FixedZone("Asia/Tokyo", 9*60*60)

type QuestionSyncUsecase struct {
	highlightRepo domain.HighlightRepository
	questionRepo  domain.QuestionRepository
	worker        *QuestionWorkerUsecase
	now           func() time.Time
}

type QuestionStockBook struct {
	BookKey    string
	BookTitle  string
	BookAuthor string
	Stock      int
	Target     int
	Preparing  int
}

type SyncQuestionStockResult struct {
	Books                  []QuestionStockBook
	QueuedCount            int
	SkippedDueToDailyLimit bool
}

type questionSyncCandidate struct {
	highlight *domain.Highlight
	remaining int
}

type questionSyncSelection struct {
	highlightIDs  []uuid.UUID
	questionCount int
}

func NewQuestionSyncUsecase(
	highlightRepo domain.HighlightRepository,
	questionRepo domain.QuestionRepository,
	worker *QuestionWorkerUsecase,
) *QuestionSyncUsecase {
	return &QuestionSyncUsecase{
		highlightRepo: highlightRepo,
		questionRepo:  questionRepo,
		worker:        worker,
		now:           time.Now,
	}
}

func (u *QuestionSyncUsecase) SyncQuestionStock(ctx context.Context, user *domain.User) (*SyncQuestionStockResult, error) {
	if user == nil {
		return nil, domain.ErrNotFound
	}

	target := resolveQuestionStockTarget(user.DefaultQuestionCount)
	bookStocks, err := u.highlightRepo.ListBookStockByUserID(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("question sync usecase: list book stock: %w", err)
	}

	result := &SyncQuestionStockResult{
		Books: make([]QuestionStockBook, 0, len(bookStocks)),
	}
	bookStatusByKey := make(map[string]*QuestionStockBook, len(bookStocks))
	understockBooks := make([]domain.BookStock, 0, len(bookStocks))

	for _, book := range bookStocks {
		bookStatus := QuestionStockBook{
			BookKey:    strings.TrimSpace(book.BookKey),
			BookTitle:  strings.TrimSpace(book.BookTitle),
			BookAuthor: strings.TrimSpace(book.BookAuthor),
			Stock:      max(book.Stock, 0),
			Target:     target,
			Preparing:  max(book.Preparing, 0),
		}
		result.Books = append(result.Books, bookStatus)
		bookStatusByKey[bookStatus.BookKey] = &result.Books[len(result.Books)-1]

		if bookStatus.BookKey == "" {
			continue
		}
		if bookStatus.Stock+bookStatus.Preparing < target {
			understockBooks = append(understockBooks, book)
		}
	}

	if len(understockBooks) == 0 {
		return result, nil
	}

	currentDay := questionSyncDay(u.now())
	dailyCount, err := u.questionRepo.GetDailyGeneratedCount(ctx, user.ID, currentDay)
	if err != nil {
		return nil, fmt.Errorf("question sync usecase: get daily generated count: %w", err)
	}

	remainingDailyBudget := maxQuestionsPerDay - dailyCount
	if remainingDailyBudget <= 0 {
		result.SkippedDueToDailyLimit = true
		return result, nil
	}

	remainingBudget := minInt(maxQuestionsPerSyncRun, remainingDailyBudget)
	sortUnderstockBooks(understockBooks)

	queuedPerBook := make(map[string]int, len(understockBooks))
	queuedHighlightIDs := make([]uuid.UUID, 0)

	for _, book := range understockBooks {
		if remainingBudget <= 0 {
			break
		}

		bookKey := strings.TrimSpace(book.BookKey)
		if bookKey == "" {
			continue
		}

		currentPreparing := max(book.Preparing, 0) + queuedPerBook[bookKey]
		currentStock := max(book.Stock, 0)
		missingCount := target - (currentStock + currentPreparing)
		if missingCount <= 0 {
			continue
		}

		selection, err := u.selectHighlightsForBook(ctx, user, bookKey, missingCount, remainingBudget)
		if err != nil {
			return nil, fmt.Errorf("question sync usecase: select highlights for book %s: %w", bookKey, err)
		}
		if selection.questionCount == 0 || len(selection.highlightIDs) == 0 {
			continue
		}

		queuedPerBook[bookKey] += selection.questionCount
		queuedHighlightIDs = append(queuedHighlightIDs, selection.highlightIDs...)
		remainingBudget -= selection.questionCount
		result.QueuedCount += selection.questionCount
	}

	for bookKey, queuedCount := range queuedPerBook {
		if status := bookStatusByKey[bookKey]; status != nil {
			status.Preparing += queuedCount
		}
	}

	if result.QueuedCount == 0 {
		return result, nil
	}

	requestedAt := u.now().UTC()
	if err := u.highlightRepo.QueueHighlightsForGeneration(ctx, user.ID, queuedHighlightIDs, requestedAt); err != nil {
		return nil, fmt.Errorf("question sync usecase: queue highlights: %w", err)
	}

	if err := u.questionRepo.IncrementDailyGeneratedCount(ctx, user.ID, currentDay, result.QueuedCount); err != nil {
		return nil, fmt.Errorf("question sync usecase: increment daily generated count: %w", err)
	}

	log.Printf(
		"question sync: queued %d question(s) for user %s across %d highlight(s); expected gemini calls <= %d",
		result.QueuedCount,
		user.ID.String(),
		len(queuedHighlightIDs),
		(result.QueuedCount+defaultWorkerMaxQuestionsPerCall-1)/defaultWorkerMaxQuestionsPerCall,
	)

	if u.worker != nil {
		userID := user.ID
		highlightIDs := append([]uuid.UUID(nil), queuedHighlightIDs...)
		go func() {
			backgroundCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()

			if err := u.worker.TriggerQueuedHighlights(backgroundCtx, userID, highlightIDs); err != nil {
				log.Printf("question sync: trigger queued highlights error: %v", err)
			}
		}()
	}

	return result, nil
}

func (u *QuestionSyncUsecase) selectHighlightsForBook(
	ctx context.Context,
	user *domain.User,
	bookKey string,
	needed int,
	remainingBudget int,
) (questionSyncSelection, error) {
	selection := questionSyncSelection{
		highlightIDs: make([]uuid.UUID, 0),
	}
	if needed <= 0 || remainingBudget <= 0 {
		return selection, nil
	}

	softTarget := minInt(needed, remainingBudget)
	seen := make(map[uuid.UUID]struct{})

	unusedHighlights, err := u.highlightRepo.ListUnusedHighlightsByBook(ctx, user.ID, bookKey, remainingBudget)
	if err != nil {
		return selection, err
	}

	unusedCandidates, err := u.buildSyncCandidates(ctx, user.ID.String(), unusedHighlights)
	if err != nil {
		return selection, err
	}
	appendQuestionSyncCandidates(&selection, unusedCandidates, softTarget, remainingBudget, seen)
	if selection.questionCount >= softTarget {
		return selection, nil
	}

	usedHighlights, err := u.highlightRepo.ListUsedHighlightsWithUncoveredPerspectives(ctx, user.ID, bookKey, remainingBudget)
	if err != nil {
		return selection, err
	}

	usedCandidates, err := u.buildSyncCandidates(ctx, user.ID.String(), usedHighlights)
	if err != nil {
		return selection, err
	}
	appendQuestionSyncCandidates(&selection, usedCandidates, softTarget, remainingBudget, seen)

	return selection, nil
}

func (u *QuestionSyncUsecase) buildSyncCandidates(
	ctx context.Context,
	userID string,
	highlights []*domain.Highlight,
) ([]questionSyncCandidate, error) {
	candidates := make([]questionSyncCandidate, 0, len(highlights))
	for _, highlight := range highlights {
		if highlight == nil || strings.TrimSpace(highlight.Content) == "" {
			continue
		}

		usedPerspectives, err := u.questionRepo.ListPerspectivesByHighlightID(ctx, userID, highlight.ID)
		if err != nil {
			return nil, err
		}

		remaining := remainingQuestionCapacity(highlight.Content, len(usedPerspectives))
		if remaining <= 0 {
			continue
		}

		candidates = append(candidates, questionSyncCandidate{
			highlight: highlight,
			remaining: remaining,
		})
	}

	return candidates, nil
}

func appendQuestionSyncCandidates(
	selection *questionSyncSelection,
	candidates []questionSyncCandidate,
	softTarget int,
	hardBudget int,
	seen map[uuid.UUID]struct{},
) {
	if selection == nil || hardBudget <= selection.questionCount {
		return
	}

	var fallback *questionSyncCandidate

	for _, candidate := range candidates {
		if candidate.highlight == nil {
			continue
		}
		if _, ok := seen[candidate.highlight.ID]; ok {
			continue
		}
		if candidate.remaining <= 0 {
			continue
		}
		if selection.questionCount+candidate.remaining > hardBudget {
			continue
		}

		remainingNeed := softTarget - selection.questionCount
		if remainingNeed > 0 && candidate.remaining <= remainingNeed {
			selection.highlightIDs = append(selection.highlightIDs, candidate.highlight.ID)
			selection.questionCount += candidate.remaining
			seen[candidate.highlight.ID] = struct{}{}
			continue
		}

		if fallback == nil || candidate.remaining < fallback.remaining {
			candidateCopy := candidate
			fallback = &candidateCopy
		}
	}

	if fallback != nil && selection.questionCount < softTarget {
		if _, ok := seen[fallback.highlight.ID]; ok {
			return
		}
		selection.highlightIDs = append(selection.highlightIDs, fallback.highlight.ID)
		selection.questionCount += fallback.remaining
		seen[fallback.highlight.ID] = struct{}{}
	}
}

func sortUnderstockBooks(books []domain.BookStock) {
	sort.SliceStable(books, func(left, right int) bool {
		leftZero := max(books[left].Stock, 0) == 0
		rightZero := max(books[right].Stock, 0) == 0
		if leftZero != rightZero {
			return leftZero
		}
		return books[left].LatestHighlightAt.After(books[right].LatestHighlightAt)
	})
}

func resolveQuestionStockTarget(defaultQuestionCount int16) int {
	if defaultQuestionCount == domain.DefaultQuestionCountAll {
		return maxQuestionCountForAll
	}
	if defaultQuestionCount <= 0 {
		return int(domain.DefaultQuestionCountDefault)
	}
	return int(defaultQuestionCount)
}

func questionSyncDay(now time.Time) time.Time {
	localNow := now.In(questionSyncDailyLocation)
	return time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, questionSyncDailyLocation)
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}
