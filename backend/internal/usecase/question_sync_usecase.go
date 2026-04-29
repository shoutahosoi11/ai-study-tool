package usecase

import (
	"context"
	"fmt"
	"log"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

const (
	defaultQuestionSyncPerTriggerLimit = 30
	defaultQuestionSyncDailyLimit      = 100
	defaultQuestionSyncStaleTimeout    = 10 * time.Minute
	defaultQuestionSyncWorkerTimeout   = 2 * time.Minute
)

var questionSyncDailyLocation = time.FixedZone("Asia/Tokyo", 9*60*60)

type QuestionSyncUsecase struct {
	highlightRepo          domain.QuestionSyncHighlightRepository
	questionRepo           domain.QuestionSyncQuestionRepository
	worker                 QueuedQuestionWorker
	now                    func() time.Time
	perTriggerLimit        int
	dailyLimit             int
	staleProcessingTimeout time.Duration
	workerTimeout          time.Duration
}

type QueuedQuestionWorker interface {
	TriggerQueuedHighlights(ctx context.Context, userID uuid.UUID, highlightIDs []uuid.UUID) error
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
	countByID     map[uuid.UUID]int
}

type questionSyncPlan struct {
	highlightIDs               []uuid.UUID
	questionCount              int
	queuedPerBook              map[string]int
	bookKeyByHighlightID       map[uuid.UUID]string
	questionCountByHighlightID map[uuid.UUID]int
}

func NewQuestionSyncUsecase(
	highlightRepo domain.QuestionSyncHighlightRepository,
	questionRepo domain.QuestionSyncQuestionRepository,
	worker QueuedQuestionWorker,
) *QuestionSyncUsecase {
	return &QuestionSyncUsecase{
		highlightRepo:          highlightRepo,
		questionRepo:           questionRepo,
		worker:                 worker,
		now:                    time.Now,
		perTriggerLimit:        readEnvIntOrDefault("QUESTION_SYNC_PER_TRIGGER_LIMIT", defaultQuestionSyncPerTriggerLimit),
		dailyLimit:             readEnvIntOrDefault("QUESTION_SYNC_DAILY_LIMIT", defaultQuestionSyncDailyLimit),
		staleProcessingTimeout: readEnvDurationSecondsOrDefault("QUESTION_SYNC_STALE_PROCESSING_SECONDS", defaultQuestionSyncStaleTimeout),
		workerTimeout:          readEnvDurationSecondsOrDefault("QUESTION_SYNC_WORKER_TIMEOUT_SECONDS", defaultQuestionSyncWorkerTimeout),
	}
}

func (u *QuestionSyncUsecase) SyncQuestionStock(ctx context.Context, user *domain.User) (*SyncQuestionStockResult, error) {
	if user == nil {
		return nil, domain.ErrNotFound
	}

	if err := u.requeueStaleProcessing(ctx, user.ID); err != nil {
		return nil, err
	}

	target := resolveQuestionStockTarget(user.DefaultQuestionCount)
	result, bookStatusByKey, understockBooks, err := u.buildQuestionStockResult(ctx, user.ID, target)
	if err != nil {
		return nil, err
	}
	if len(understockBooks) == 0 {
		return result, nil
	}

	currentDay, remainingBudget, skipped, err := u.remainingQuestionSyncBudget(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	if skipped {
		result.SkippedDueToDailyLimit = true
		return result, nil
	}

	sortUnderstockBooks(understockBooks)
	plan, err := u.planQuestionSync(ctx, user, understockBooks, target, remainingBudget)
	if err != nil {
		return nil, err
	}
	if plan.questionCount == 0 {
		return result, nil
	}

	actualQueuedCount, actualQueuedPerBook, actualQueuedHighlightIDs, skipped, err := u.reserveAndQueueQuestionSync(
		ctx,
		user.ID,
		currentDay,
		plan,
	)
	if err != nil {
		return nil, err
	}
	if skipped {
		result.SkippedDueToDailyLimit = true
		return result, nil
	}

	result.QueuedCount = actualQueuedCount
	if result.QueuedCount == 0 {
		return result, nil
	}

	for bookKey, queuedCount := range actualQueuedPerBook {
		if status := bookStatusByKey[bookKey]; status != nil {
			status.Preparing += queuedCount
		}
	}

	log.Printf(
		"question_sync_event=queued user_id=%s queued_count=%d highlight_count=%d expected_gemini_calls_le=%d daily_limit=%d per_trigger_limit=%d",
		user.ID.String(),
		result.QueuedCount,
		len(actualQueuedHighlightIDs),
		(result.QueuedCount+defaultWorkerMaxQuestionsPerCall-1)/defaultWorkerMaxQuestionsPerCall,
		u.dailyLimit,
		u.perTriggerLimit,
	)

	u.triggerQueuedWorker(user.ID, actualQueuedHighlightIDs)

	return result, nil
}

func (u *QuestionSyncUsecase) requeueStaleProcessing(ctx context.Context, userID uuid.UUID) error {
	if u.staleProcessingTimeout <= 0 {
		return nil
	}

	requeued, err := u.highlightRepo.RequeueStaleProcessingByUserID(ctx, userID, u.now().UTC().Add(-u.staleProcessingTimeout))
	if err != nil {
		return fmt.Errorf("question sync usecase: requeue stale processing: %w", err)
	}
	if requeued > 0 {
		log.Printf("question_sync_event=stale_processing_requeued user_id=%s count=%d", userID.String(), requeued)
	}
	return nil
}

func (u *QuestionSyncUsecase) buildQuestionStockResult(
	ctx context.Context,
	userID uuid.UUID,
	target int,
) (*SyncQuestionStockResult, map[string]*QuestionStockBook, []domain.BookStock, error) {
	bookStocks, err := u.highlightRepo.ListBookStockByUserID(ctx, userID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("question sync usecase: list book stock: %w", err)
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
			understockBooks = append(understockBooks, domain.BookStock{
				BookKey:           bookStatus.BookKey,
				BookTitle:         bookStatus.BookTitle,
				BookAuthor:        bookStatus.BookAuthor,
				Stock:             bookStatus.Stock,
				Preparing:         bookStatus.Preparing,
				LatestHighlightAt: book.LatestHighlightAt,
			})
		}
	}

	return result, bookStatusByKey, understockBooks, nil
}

func (u *QuestionSyncUsecase) remainingQuestionSyncBudget(ctx context.Context, userID uuid.UUID) (time.Time, int, bool, error) {
	currentDay := questionSyncDay(u.now())
	dailyCount, err := u.questionRepo.GetDailyGeneratedCount(ctx, userID, currentDay)
	if err != nil {
		return currentDay, 0, false, fmt.Errorf("question sync usecase: get daily generated count: %w", err)
	}

	remainingDailyBudget := u.dailyLimit - dailyCount
	if remainingDailyBudget <= 0 {
		return currentDay, 0, true, nil
	}

	return currentDay, min(u.perTriggerLimit, remainingDailyBudget), false, nil
}

func (u *QuestionSyncUsecase) planQuestionSync(
	ctx context.Context,
	user *domain.User,
	understockBooks []domain.BookStock,
	target int,
	remainingBudget int,
) (questionSyncPlan, error) {
	plan := questionSyncPlan{
		highlightIDs:               make([]uuid.UUID, 0),
		queuedPerBook:              make(map[string]int, len(understockBooks)),
		bookKeyByHighlightID:       make(map[uuid.UUID]string),
		questionCountByHighlightID: make(map[uuid.UUID]int),
	}

	for _, book := range understockBooks {
		if remainingBudget <= 0 {
			break
		}

		bookKey := book.BookKey
		if bookKey == "" {
			continue
		}

		currentPreparing := max(book.Preparing, 0) + plan.queuedPerBook[bookKey]
		currentStock := max(book.Stock, 0)
		missingCount := target - (currentStock + currentPreparing)
		if missingCount <= 0 {
			continue
		}

		selection, err := u.selectHighlightsForBook(ctx, user, bookKey, missingCount, remainingBudget)
		if err != nil {
			return questionSyncPlan{}, fmt.Errorf("question sync usecase: select highlights for book %s: %w", bookKey, err)
		}
		if selection.questionCount == 0 || len(selection.highlightIDs) == 0 {
			continue
		}

		plan.queuedPerBook[bookKey] += selection.questionCount
		plan.highlightIDs = append(plan.highlightIDs, selection.highlightIDs...)
		for highlightID, questionCount := range selection.countByID {
			plan.bookKeyByHighlightID[highlightID] = bookKey
			plan.questionCountByHighlightID[highlightID] = questionCount
		}
		remainingBudget -= selection.questionCount
		plan.questionCount += selection.questionCount
	}

	return plan, nil
}

func (u *QuestionSyncUsecase) reserveAndQueueQuestionSync(
	ctx context.Context,
	userID uuid.UUID,
	currentDay time.Time,
	plan questionSyncPlan,
) (int, map[string]int, []uuid.UUID, bool, error) {
	requestedAt := u.now().UTC()
	actualQueuedHighlightIDs, reserved, err := u.questionRepo.QueueHighlightsWithinDailyLimit(
		ctx,
		userID,
		currentDay,
		u.dailyLimit,
		plan.highlightIDs,
		plan.questionCountByHighlightID,
		requestedAt,
	)
	if err != nil {
		return 0, nil, nil, false, fmt.Errorf("question sync usecase: queue highlights within daily limit: %w", err)
	}
	if !reserved {
		return 0, nil, nil, true, nil
	}

	actualQueuedPerBook := make(map[string]int, len(plan.queuedPerBook))
	actualQueuedCount := 0
	for _, highlightID := range actualQueuedHighlightIDs {
		questionCount := plan.questionCountByHighlightID[highlightID]
		if questionCount <= 0 {
			continue
		}
		actualQueuedCount += questionCount
		if bookKey := plan.bookKeyByHighlightID[highlightID]; bookKey != "" {
			actualQueuedPerBook[bookKey] += questionCount
		}
	}

	return actualQueuedCount, actualQueuedPerBook, actualQueuedHighlightIDs, false, nil
}

func (u *QuestionSyncUsecase) triggerQueuedWorker(userID uuid.UUID, actualQueuedHighlightIDs []uuid.UUID) {
	if u.worker != nil {
		highlightIDs := slices.Clone(actualQueuedHighlightIDs)
		workerTimeout := u.workerTimeout
		if workerTimeout <= 0 {
			workerTimeout = defaultQuestionSyncWorkerTimeout
		}
		go func() {
			backgroundCtx, cancel := context.WithTimeout(context.Background(), workerTimeout)
			defer cancel()

			if err := u.worker.TriggerQueuedHighlights(backgroundCtx, userID, highlightIDs); err != nil {
				log.Printf("question sync: trigger queued highlights error: %v", err)
			}
		}()
	}
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
		countByID:    make(map[uuid.UUID]int),
	}
	softTarget := min(needed, remainingBudget)
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
	if hardBudget <= selection.questionCount {
		return
	}

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
		if remainingNeed <= 0 || candidate.remaining > remainingNeed {
			continue
		}

		selection.highlightIDs = append(selection.highlightIDs, candidate.highlight.ID)
		selection.questionCount += candidate.remaining
		selection.countByID[candidate.highlight.ID] = candidate.remaining
		seen[candidate.highlight.ID] = struct{}{}
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
