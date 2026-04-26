package usecase

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

const (
	defaultWorkerBatchSize           = 10
	defaultWorkerMaxRetry            = 3
	defaultWorkerMaxQuestionsPerCall = 8
	defaultWorkerRequestInterval     = time.Second
	defaultWorkerGlobalPendingMaxAge = 5 * time.Minute
	defaultWorkerMaxPromptTokens     = 6000
)

type QuestionWorkerUsecase struct {
	highlightRepo       domain.HighlightRepository
	questionRepo        domain.QuestionRepository
	llmClient           domain.LLMClient
	maxBatchSize        int
	maxRetry            int
	maxQuestionsPerCall int
	maxPromptTokens     int
	requestInterval     time.Duration
	globalPendingMaxAge time.Duration
	now                 func() time.Time
	lastRequestAt       time.Time
}

type highlightGenerationPlan struct {
	highlight     *domain.Highlight
	perspectives  []string
	versions      []int
	questionCount int
}

func NewQuestionWorkerUsecase(
	highlightRepo domain.HighlightRepository,
	questionRepo domain.QuestionRepository,
	llmClient domain.LLMClient,
) *QuestionWorkerUsecase {
	return &QuestionWorkerUsecase{
		highlightRepo:       highlightRepo,
		questionRepo:        questionRepo,
		llmClient:           llmClient,
		maxBatchSize:        readEnvIntOrDefault("QUESTION_WORKER_BATCH_SIZE", defaultWorkerBatchSize),
		maxRetry:            readEnvIntOrDefault("QUESTION_WORKER_MAX_RETRY", defaultWorkerMaxRetry),
		maxQuestionsPerCall: readEnvIntOrDefault("QUESTION_WORKER_MAX_QUESTIONS_PER_CALL", defaultWorkerMaxQuestionsPerCall),
		maxPromptTokens:     readEnvIntOrDefault("QUESTION_WORKER_MAX_PROMPT_TOKENS", defaultWorkerMaxPromptTokens),
		requestInterval:     readEnvDurationMSOrDefault("QUESTION_WORKER_REQUEST_INTERVAL_MS", defaultWorkerRequestInterval),
		globalPendingMaxAge: readEnvDurationSecondsOrDefault("QUESTION_WORKER_GLOBAL_PENDING_SECONDS", defaultWorkerGlobalPendingMaxAge),
		now:                 time.Now,
	}
}

func (u *QuestionWorkerUsecase) RunOnce(ctx context.Context) error {
	now := u.now()

	stats, err := u.highlightRepo.ListPendingUserStats(ctx)
	if err != nil {
		return fmt.Errorf("question worker: list pending user stats: %w", err)
	}

	for _, stat := range stats {
		if !shouldProcessPendingBatch(stat, now, u.globalPendingMaxAge) {
			continue
		}

		highlights, claimErr := u.highlightRepo.ClaimPendingByUserID(ctx, stat.UserID, u.maxBatchSize)
		if claimErr != nil {
			log.Printf("question worker: claim pending highlights error: %v", claimErr)
			continue
		}
		if len(highlights) == 0 {
			continue
		}

		if processErr := u.ProcessPendingHighlights(ctx, stat.UserID.String(), highlights); processErr != nil {
			log.Printf("question worker: process pending highlights error: %v", processErr)
		}
	}

	tasks, err := u.questionRepo.ClaimPendingRegenerationTasks(ctx, u.maxBatchSize)
	if err != nil {
		return fmt.Errorf("question worker: claim regeneration tasks: %w", err)
	}
	if len(tasks) == 0 {
		return nil
	}

	for _, task := range tasks {
		if task == nil || task.Highlight == nil {
			continue
		}
		if err := u.ProcessRegenerationTask(ctx, task); err != nil {
			log.Printf("question worker: process regeneration task error: %v", err)
		}
	}

	return nil
}

func (u *QuestionWorkerUsecase) TriggerQueuedHighlights(ctx context.Context, userID uuid.UUID, highlightIDs []uuid.UUID) error {
	claimedHighlights, err := u.highlightRepo.ClaimPendingByIDs(ctx, userID, highlightIDs)
	if err != nil {
		return fmt.Errorf("question worker: claim queued highlights: %w", err)
	}
	if len(claimedHighlights) == 0 {
		return nil
	}

	return u.ProcessPendingHighlights(ctx, userID.String(), claimedHighlights)
}

func (u *QuestionWorkerUsecase) ProcessPendingHighlights(ctx context.Context, userID string, highlights []*domain.Highlight) error {
	plans := make([]highlightGenerationPlan, 0, len(highlights))
	failedHighlightIDs := make([]uuid.UUID, 0)
	completedHighlightIDs := make([]uuid.UUID, 0)

	for _, highlight := range highlights {
		if highlight == nil || strings.TrimSpace(highlight.Content) == "" {
			if highlight != nil {
				failedHighlightIDs = append(failedHighlightIDs, highlight.ID)
			}
			continue
		}

		usedPerspectives, err := u.questionRepo.ListPerspectivesByHighlightID(ctx, userID, highlight.ID)
		if err != nil {
			return fmt.Errorf("question worker: list perspectives: %w", err)
		}

		questionCount := remainingQuestionCapacity(highlight.Content, len(usedPerspectives))
		if questionCount <= 0 {
			completedHighlightIDs = append(completedHighlightIDs, highlight.ID)
			continue
		}
		perspectives, versions := nextPerspectiveVersions(usedPerspectives, questionCount)
		if len(perspectives) == 0 {
			completedHighlightIDs = append(completedHighlightIDs, highlight.ID)
			continue
		}

		plans = append(plans, highlightGenerationPlan{
			highlight:     highlight,
			perspectives:  perspectives,
			versions:      versions,
			questionCount: len(perspectives),
		})
	}

	if len(failedHighlightIDs) > 0 {
		if err := u.highlightRepo.MarkGenerationFailed(ctx, failedHighlightIDs, domain.ErrSourceTextUnavailable.Error(), u.maxRetry); err != nil {
			log.Printf("question worker: mark invalid pending highlights failed: %v", err)
		}
	}
	if len(completedHighlightIDs) > 0 {
		if err := u.highlightRepo.MarkGenerationCompleted(ctx, completedHighlightIDs); err != nil {
			log.Printf("question worker: mark already-covered highlights completed: %v", err)
		}
	}
	if len(plans) == 0 {
		return nil
	}

	chunks := splitHighlightPlansByLimit(plans, u.maxPromptTokens, u.maxQuestionsPerCall)
	completedHighlights := make(map[uuid.UUID]struct{})
	failedHighlights := make(map[uuid.UUID]string)

	for _, chunk := range chunks {
		if len(chunk) == 0 {
			continue
		}

		if err := u.waitForRateLimit(ctx); err != nil {
			return err
		}

		customInstruction := buildPerspectiveInstruction(chunk)
		materials := buildPerspectiveGenerationMaterials(chunk)
		expectedCount := 0
		for _, plan := range chunk {
			expectedCount += len(plan.perspectives)
		}
		log.Printf("question worker: generating %d question(s) across %d highlight(s)", expectedCount, len(chunk))
		generationID, err := u.questionRepo.SaveGeneration(ctx, userID, "highlight_batch", "", customInstruction, modelForPlan("free"))
		if err != nil {
			for _, plan := range chunk {
				failedHighlights[plan.highlight.ID] = err.Error()
			}
			continue
		}

		generatedQuestions, err := u.llmClient.GenerateQuestions(
			ctx,
			materials,
			domain.QuestionTypeMultipleChoice,
			customInstruction,
			modelForPlan("free"),
		)
		if err != nil {
			for _, plan := range chunk {
				failedHighlights[plan.highlight.ID] = err.Error()
			}
			continue
		}

		if len(generatedQuestions) < expectedCount {
			errMessage := fmt.Sprintf("generated questions count mismatch: expected %d, got %d", expectedCount, len(generatedQuestions))
			for _, plan := range chunk {
				failedHighlights[plan.highlight.ID] = errMessage
			}
			continue
		}

		offset := 0
		for _, plan := range chunk {
			savedForHighlight := 0
			for index, perspective := range plan.perspectives {
				generated := generatedQuestions[offset]
				offset++

				question := &domain.Question{
					ID:            uuid.NewString(),
					QuestionType:  domain.QuestionTypeMultipleChoice,
					Content:       generated.Content,
					Options:       generated.Options,
					CorrectAnswer: generated.CorrectAnswer,
					Explanation:   generated.Explanation,
				}
				meta := &domain.QuestionMeta{
					QuestionID:    question.ID,
					CreatorID:     userID,
					SourceType:    domain.SourceTypeKindleBook,
					HighlightID:   plan.highlight.ID.String(),
					GenerationID:  generationID,
					Perspective:   perspective,
					Version:       plan.versions[index],
					IsAIGenerated: true,
				}

				if err := u.questionRepo.Save(ctx, question, meta); err != nil {
					log.Printf("question worker: save generated question error: %v", err)
					failedHighlights[plan.highlight.ID] = err.Error()
					continue
				}
				savedForHighlight++
			}

			if savedForHighlight > 0 {
				delete(failedHighlights, plan.highlight.ID)
				completedHighlights[plan.highlight.ID] = struct{}{}
			}
		}
	}

	if len(completedHighlights) > 0 {
		if err := u.highlightRepo.MarkGenerationCompleted(ctx, mapUUIDKeys(completedHighlights)); err != nil {
			log.Printf("question worker: mark generation completed error: %v", err)
		}
	}

	if len(failedHighlights) > 0 {
		ids := make([]uuid.UUID, 0, len(failedHighlights))
		message := ""
		for id, errMessage := range failedHighlights {
			ids = append(ids, id)
			if message == "" {
				message = errMessage
			}
		}
		if err := u.highlightRepo.MarkGenerationFailed(ctx, ids, message, u.maxRetry); err != nil {
			log.Printf("question worker: mark generation failed error: %v", err)
		}
	}

	return nil
}

func (u *QuestionWorkerUsecase) ProcessRegenerationTask(ctx context.Context, task *domain.RegenerationTask) error {
	if task == nil {
		return nil
	}
	if task.Highlight == nil || strings.TrimSpace(task.Highlight.Content) == "" {
		return u.questionRepo.MarkRegenerationTasksFailed(ctx, []uuid.UUID{task.ID}, domain.ErrSourceTextUnavailable.Error(), u.maxRetry)
	}

	usedPerspectives, err := u.questionRepo.ListPerspectivesByHighlightID(ctx, task.UserID.String(), task.Highlight.ID)
	if err != nil {
		return fmt.Errorf("question worker: list regeneration perspectives: %w", err)
	}

	perspectives, versions := nextPerspectiveVersions(usedPerspectives, 1)
	if len(perspectives) == 0 {
		return u.questionRepo.MarkRegenerationTasksFailed(ctx, []uuid.UUID{task.ID}, domain.ErrQuestionGenerationFailed.Error(), u.maxRetry)
	}

	if err := u.waitForRateLimit(ctx); err != nil {
		return err
	}

	customInstruction := fmt.Sprintf("このハイライトから %s 観点で1問だけ作成してください。", perspectives[0])
	generationID, err := u.questionRepo.SaveGeneration(ctx, task.UserID.String(), "regeneration", task.Highlight.ID.String(), customInstruction, modelForPlan("free"))
	if err != nil {
		return u.questionRepo.MarkRegenerationTasksFailed(ctx, []uuid.UUID{task.ID}, err.Error(), u.maxRetry)
	}

	generatedQuestions, err := u.llmClient.GenerateQuestions(
		ctx,
		[]domain.ExtractedPoint{
			{
				Point:   task.Highlight.Content,
				Context: buildPerspectiveContext(task.Highlight, perspectives[0]),
			},
		},
		domain.QuestionTypeMultipleChoice,
		customInstruction,
		modelForPlan("free"),
	)
	if err != nil || len(generatedQuestions) == 0 {
		if err == nil {
			err = domain.ErrQuestionGenerationFailed
		}
		return u.questionRepo.MarkRegenerationTasksFailed(ctx, []uuid.UUID{task.ID}, err.Error(), u.maxRetry)
	}

	question := &domain.Question{
		ID:            uuid.NewString(),
		QuestionType:  domain.QuestionTypeMultipleChoice,
		Content:       generatedQuestions[0].Content,
		Options:       generatedQuestions[0].Options,
		CorrectAnswer: generatedQuestions[0].CorrectAnswer,
		Explanation:   generatedQuestions[0].Explanation,
	}
	meta := &domain.QuestionMeta{
		QuestionID:    question.ID,
		CreatorID:     task.UserID.String(),
		SourceType:    domain.SourceTypeKindleBook,
		HighlightID:   task.Highlight.ID.String(),
		GenerationID:  generationID,
		Perspective:   perspectives[0],
		Version:       versions[0],
		IsAIGenerated: true,
	}

	if err := u.questionRepo.Save(ctx, question, meta); err != nil {
		return u.questionRepo.MarkRegenerationTasksFailed(ctx, []uuid.UUID{task.ID}, err.Error(), u.maxRetry)
	}

	return u.questionRepo.MarkRegenerationTasksCompleted(ctx, []uuid.UUID{task.ID})
}

func shouldProcessPendingBatch(stat domain.PendingHighlightUserStat, now time.Time, globalPendingMaxAge time.Duration) bool {
	if stat.PendingCount >= defaultWorkerBatchSize {
		return true
	}

	oldestAge := now.Sub(stat.OldestPendingAt)
	if oldestAge >= globalPendingMaxAge {
		return true
	}

	return oldestAge >= dynamicMaxWaitTime(stat.TotalCount)
}

func dynamicMaxWaitTime(totalHighlights int) time.Duration {
	switch {
	case totalHighlights <= 1:
		return 5 * time.Second
	case totalHighlights <= 5:
		seconds := 5 + ((totalHighlights-1)*10)/4
		return time.Duration(seconds) * time.Second
	case totalHighlights <= 20:
		seconds := 15 + ((totalHighlights-6)*15)/14
		return time.Duration(seconds) * time.Second
	default:
		capped := totalHighlights
		if capped > 41 {
			capped = 41
		}
		seconds := 30 + ((capped-21)*30)/20
		if seconds > 60 {
			seconds = 60
		}
		return time.Duration(seconds) * time.Second
	}
}

func questionCountForHighlight(content string) int {
	length := len([]rune(strings.TrimSpace(content)))
	switch {
	case length == 0:
		return 0
	case length < 120:
		return 1
	case length < 320:
		return 2
	default:
		return 3
	}
}

func remainingQuestionCapacity(content string, existingQuestionCount int) int {
	remaining := questionCountForHighlight(content) - existingQuestionCount
	if remaining < 0 {
		return 0
	}
	return remaining
}

func nextPerspectiveVersions(existingPerspectives []string, count int) ([]string, []int) {
	if count <= 0 {
		return []string{}, []int{}
	}

	usage := make(map[string]int, len(domain.QuestionPerspectiveOrder))
	for _, perspective := range existingPerspectives {
		usage[strings.TrimSpace(perspective)]++
	}

	selected := make([]string, 0, count)
	versions := make([]int, 0, count)
	for len(selected) < count {
		perspective := selectLeastUsedPerspective(usage)
		usage[perspective]++
		selected = append(selected, perspective)
		versions = append(versions, usage[perspective])
	}

	return selected, versions
}

func selectLeastUsedPerspective(usage map[string]int) string {
	selected := domain.QuestionPerspectiveOrder[0]
	selectedUsage := usage[selected]

	for _, perspective := range domain.QuestionPerspectiveOrder[1:] {
		if usage[perspective] < selectedUsage {
			selected = perspective
			selectedUsage = usage[perspective]
		}
	}

	return selected
}

func splitHighlightPlansByLimit(plans []highlightGenerationPlan, maxPromptTokens int, maxQuestionsPerCall int) [][]highlightGenerationPlan {
	if maxPromptTokens <= 0 && maxQuestionsPerCall <= 0 {
		return [][]highlightGenerationPlan{plans}
	}

	chunks := make([][]highlightGenerationPlan, 0)
	current := make([]highlightGenerationPlan, 0)
	currentTokens := 0
	currentQuestionCount := 0

	for _, plan := range plans {
		estimatedTokens := estimatePlanTokens(plan)
		nextQuestionCount := currentQuestionCount + plan.questionCount
		shouldSplitByTokens := maxPromptTokens > 0 && currentTokens+estimatedTokens > maxPromptTokens
		shouldSplitByQuestions := maxQuestionsPerCall > 0 && nextQuestionCount > maxQuestionsPerCall
		if len(current) > 0 && (shouldSplitByTokens || shouldSplitByQuestions) {
			chunks = append(chunks, current)
			current = make([]highlightGenerationPlan, 0)
			currentTokens = 0
			currentQuestionCount = 0
		}

		current = append(current, plan)
		currentTokens += estimatedTokens
		currentQuestionCount += plan.questionCount
	}

	if len(current) > 0 {
		chunks = append(chunks, current)
	}

	return chunks
}

func splitHighlightPlansByTokenLimit(plans []highlightGenerationPlan, maxPromptTokens int) [][]highlightGenerationPlan {
	return splitHighlightPlansByLimit(plans, maxPromptTokens, defaultWorkerMaxQuestionsPerCall)
}

func estimatePlanTokens(plan highlightGenerationPlan) int {
	if plan.highlight == nil {
		return 0
	}
	base := len([]rune(strings.TrimSpace(plan.highlight.Content)))/4 + 120
	return base * max(1, len(plan.perspectives))
}

func buildPerspectiveGenerationMaterials(plans []highlightGenerationPlan) []domain.ExtractedPoint {
	materials := make([]domain.ExtractedPoint, 0)
	for _, plan := range plans {
		for _, perspective := range plan.perspectives {
			materials = append(materials, domain.ExtractedPoint{
				Point:   strings.TrimSpace(plan.highlight.Content),
				Context: buildPerspectiveContext(plan.highlight, perspective),
			})
		}
	}
	return materials
}

func buildPerspectiveInstruction(plans []highlightGenerationPlan) string {
	return "各素材の「ユーザー解説」欄に `観点:` が含まれる場合は、その観点に必ず従って1問ずつ作成してください。同じハイライト本文が複数回出る場合は、別観点の別問題として扱ってください。"
}

func buildPerspectiveContext(highlight *domain.Highlight, perspective string) string {
	parts := make([]string, 0, 2)
	if highlight != nil && highlight.Explanation != nil && strings.TrimSpace(*highlight.Explanation) != "" {
		parts = append(parts, strings.TrimSpace(*highlight.Explanation))
	}
	parts = append(parts, fmt.Sprintf("観点: %s", perspective))
	return strings.Join(parts, "\n")
}

func (u *QuestionWorkerUsecase) waitForRateLimit(ctx context.Context) error {
	if u.requestInterval <= 0 {
		return nil
	}

	if !u.lastRequestAt.IsZero() {
		wait := u.lastRequestAt.Add(u.requestInterval).Sub(u.now())
		if wait > 0 {
			timer := time.NewTimer(wait)
			defer timer.Stop()

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
			}
		}
	}

	u.lastRequestAt = u.now()
	return nil
}

func mapUUIDKeys(values map[uuid.UUID]struct{}) []uuid.UUID {
	items := make([]uuid.UUID, 0, len(values))
	for value := range values {
		items = append(items, value)
	}
	return items
}

func readEnvIntOrDefault(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		return fallback
	}

	return parsed
}

func readEnvDurationMSOrDefault(key string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		return fallback
	}

	return time.Duration(parsed) * time.Millisecond
}

func readEnvDurationSecondsOrDefault(key string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		return fallback
	}

	return time.Duration(parsed) * time.Second
}

func max(left, right int) int {
	if left > right {
		return left
	}
	return right
}
