package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

type mockQuestionWorkerRepository struct {
	save             func(ctx context.Context, question *domain.Question, meta *domain.QuestionMeta) error
	reserveDaily     func(ctx context.Context, userID uuid.UUID, day time.Time, delta int, limit int) (bool, error)
	releasedDelta    int
	savedHighlight   []uuid.UUID
	superseded       []uuid.UUID
	reservedDelta    int
	completedJob     uuid.UUID
	completedJobRefs []uuid.UUID
	saveGenerationID string
}

func (m *mockQuestionWorkerRepository) Save(ctx context.Context, question *domain.Question, meta *domain.QuestionMeta) error {
	if m.save == nil {
		if meta != nil {
			highlightID, _ := uuid.Parse(meta.HighlightID)
			m.savedHighlight = append(m.savedHighlight, highlightID)
		}
		return nil
	}
	return m.save(ctx, question, meta)
}

func (m *mockQuestionWorkerRepository) SupersedeActiveQuestionsForHighlight(ctx context.Context, userID uuid.UUID, highlightID uuid.UUID) error {
	m.superseded = append(m.superseded, highlightID)
	return nil
}

func (m *mockQuestionWorkerRepository) SaveGeneration(ctx context.Context, userID, sourceType, sourceID, promptUsed, modelUsed string) (string, error) {
	if m.saveGenerationID != "" {
		return m.saveGenerationID, nil
	}
	return "generation-id", nil
}

func (m *mockQuestionWorkerRepository) ListPerspectivesByHighlightID(ctx context.Context, userID string, highlightID uuid.UUID) ([]string, error) {
	return make([]string, 0), nil
}

func (m *mockQuestionWorkerRepository) GetDailyGeneratedCount(ctx context.Context, userID uuid.UUID, day time.Time) (int, error) {
	return 0, nil
}

func (m *mockQuestionWorkerRepository) ReserveDailyGeneratedCount(ctx context.Context, userID uuid.UUID, day time.Time, delta int, limit int) (bool, error) {
	m.reservedDelta = delta
	if m.reserveDaily != nil {
		return m.reserveDaily(ctx, userID, day, delta, limit)
	}
	return true, nil
}

func (m *mockQuestionWorkerRepository) ReleaseDailyGeneratedCount(ctx context.Context, userID uuid.UUID, day time.Time, delta int) error {
	m.releasedDelta += delta
	return nil
}

func (m *mockQuestionWorkerRepository) ReplaceActiveQuestionsForHighlights(ctx context.Context, userID uuid.UUID, replacements []domain.QuestionReplacement) error {
	for _, replacement := range replacements {
		if err := m.SupersedeActiveQuestionsForHighlight(ctx, userID, replacement.HighlightID); err != nil {
			return err
		}
		if err := m.Save(ctx, replacement.Question, replacement.Meta); err != nil {
			return err
		}
	}
	return nil
}

func (m *mockQuestionWorkerRepository) CompleteQuestionGenerationJob(ctx context.Context, userID uuid.UUID, jobID uuid.UUID, replacements []domain.QuestionReplacement, highlightIDs []uuid.UUID) error {
	if err := m.ReplaceActiveQuestionsForHighlights(ctx, userID, replacements); err != nil {
		return err
	}
	m.completedJob = jobID
	m.completedJobRefs = append(m.completedJobRefs, highlightIDs...)
	return nil
}

type mockWorkerHighlightLifecycle struct {
	highlights      []*domain.Highlight
	listByIDsCalled int
	completed       []uuid.UUID
	failed          []uuid.UUID
}

func (m *mockWorkerHighlightLifecycle) ListByIDs(ctx context.Context, userID uuid.UUID, highlightIDs []uuid.UUID) ([]*domain.Highlight, error) {
	m.listByIDsCalled++
	return append([]*domain.Highlight(nil), m.highlights...), nil
}

func (m *mockWorkerHighlightLifecycle) MarkGenerationCompleted(ctx context.Context, userID uuid.UUID, highlightIDs []uuid.UUID) error {
	m.completed = append(m.completed, highlightIDs...)
	return nil
}

func (m *mockWorkerHighlightLifecycle) MarkGenerationFailed(ctx context.Context, userID uuid.UUID, highlightIDs []uuid.UUID, lastError string, maxRetry int) error {
	m.failed = append(m.failed, highlightIDs...)
	return nil
}

type mockQuestionWorkerLLMClient struct {
	generateCalled int
	generated      []domain.GeneratedQuestion
	err            error
	usage          *domain.LLMUsage
}

func (m *mockQuestionWorkerLLMClient) ModelForPlan(plan string) string {
	return "gemini-test"
}

func (m *mockQuestionWorkerLLMClient) ProviderName() string {
	return "test-provider"
}

func (m *mockQuestionWorkerLLMClient) LastUsage() (domain.LLMUsage, bool) {
	if m.usage == nil {
		return domain.LLMUsage{}, false
	}
	return *m.usage, true
}

func (m *mockQuestionWorkerLLMClient) GenerateQuestions(ctx context.Context, points []domain.ExtractedPoint, questionType domain.QuestionType, customInstruction string, model string) ([]domain.GeneratedQuestion, error) {
	m.generateCalled++
	if m.err != nil {
		return nil, m.err
	}
	if m.generated != nil {
		return append([]domain.GeneratedQuestion(nil), m.generated...), nil
	}
	questions := make([]domain.GeneratedQuestion, 0, len(points))
	for range points {
		questions = append(questions, domain.GeneratedQuestion{
			Content:       "問題",
			Options:       []string{"A", "B", "C", "D"},
			CorrectAnswer: "A",
			Explanation:   "解説",
		})
	}
	return questions, nil
}

type mockGlobalLLMBudget struct {
	reserveErr       error
	reserveCalls     int
	recordUsageCalls int
	recordedUsage    []domain.LLMUsageLogInput
}

func (m *mockGlobalLLMBudget) EstimateRequestCostYen(requestCount int) int {
	return requestCount
}

func (m *mockGlobalLLMBudget) Reserve(ctx context.Context, requestCount int, estimatedCostYen int) error {
	m.reserveCalls++
	if m.reserveErr != nil {
		return m.reserveErr
	}
	return nil
}

func (m *mockGlobalLLMBudget) RecordUsage(ctx context.Context, input domain.LLMUsageLogInput) error {
	m.recordUsageCalls++
	m.recordedUsage = append(m.recordedUsage, input)
	return nil
}

func TestProcessQuestionGenerationJobNoOpsWhenJobAlreadyClaimed(t *testing.T) {
	jobID := uuid.New()
	userID := uuid.New()
	jobRepo := &mockQuestionGenerationJobRepository{claimOK: false}
	highlightRepo := &mockWorkerHighlightLifecycle{}
	llm := &mockQuestionWorkerLLMClient{}
	uc := NewQuestionWorkerUsecaseWithJobRepository(highlightRepo, &mockQuestionWorkerRepository{}, jobRepo, llm)

	if err := uc.ProcessQuestionGenerationJob(context.Background(), jobID, userID); err != nil {
		t.Fatalf("ProcessQuestionGenerationJob failed: %v", err)
	}

	if jobRepo.claimCalls != 1 {
		t.Fatalf("expected one claim attempt, got %d", jobRepo.claimCalls)
	}
	if highlightRepo.listByIDsCalled != 0 || llm.generateCalled != 0 {
		t.Fatalf("expected no work after unclaimed job, list=%d generate=%d", highlightRepo.listByIDsCalled, llm.generateCalled)
	}
}

func TestProcessQuestionGenerationJobReturnsQueuedWhenDailyLimitExceeded(t *testing.T) {
	jobID := uuid.New()
	userID := uuid.New()
	highlightID := uuid.New()
	jobRepo := &mockQuestionGenerationJobRepository{
		claimOK:  true,
		claimJob: &domain.QuestionGenerationJob{ID: jobID, UserID: userID, HighlightIDs: []uuid.UUID{highlightID}},
	}
	questionRepo := &mockQuestionWorkerRepository{
		reserveDaily: func(ctx context.Context, userID uuid.UUID, day time.Time, delta int, limit int) (bool, error) {
			return false, nil
		},
	}
	highlightRepo := &mockWorkerHighlightLifecycle{
		highlights: []*domain.Highlight{{ID: highlightID, UserID: userID, Content: "生成対象のハイライト"}},
	}
	uc := NewQuestionWorkerUsecaseWithJobRepository(highlightRepo, questionRepo, jobRepo, &mockQuestionWorkerLLMClient{})

	if err := uc.ProcessQuestionGenerationJob(context.Background(), jobID, userID); err != nil {
		t.Fatalf("ProcessQuestionGenerationJob failed: %v", err)
	}

	if len(jobRepo.markedQueued) != 1 || jobRepo.markedQueued[0] != jobID {
		t.Fatalf("expected quota-limited job returned to queued, got %#v", jobRepo.markedQueued)
	}
	if highlightRepo.listByIDsCalled != 1 {
		t.Fatalf("expected highlight load before quota reservation, got %d", highlightRepo.listByIDsCalled)
	}
}

func TestProcessQuestionGenerationJobCompletesWhenHighlightsWereDeleted(t *testing.T) {
	jobID := uuid.New()
	userID := uuid.New()
	jobRepo := &mockQuestionGenerationJobRepository{
		claimOK:  true,
		claimJob: &domain.QuestionGenerationJob{ID: jobID, UserID: userID},
	}
	uc := NewQuestionWorkerUsecaseWithJobRepository(&mockWorkerHighlightLifecycle{}, &mockQuestionWorkerRepository{}, jobRepo, &mockQuestionWorkerLLMClient{})

	if err := uc.ProcessQuestionGenerationJob(context.Background(), jobID, userID); err != nil {
		t.Fatalf("ProcessQuestionGenerationJob failed: %v", err)
	}

	if len(jobRepo.markedCompleted) != 1 || jobRepo.markedCompleted[0] != jobID {
		t.Fatalf("expected empty job completed, got %#v", jobRepo.markedCompleted)
	}
}

func TestProcessQuestionGenerationJobCreatesOneActiveQuestionPerHighlight(t *testing.T) {
	jobID := uuid.New()
	userID := uuid.New()
	highlightID := uuid.New()
	jobRepo := &mockQuestionGenerationJobRepository{
		claimOK:  true,
		claimJob: &domain.QuestionGenerationJob{ID: jobID, UserID: userID, HighlightIDs: []uuid.UUID{highlightID}},
	}
	highlightRepo := &mockWorkerHighlightLifecycle{
		highlights: []*domain.Highlight{{
			ID:      highlightID,
			UserID:  userID,
			Content: "ハイライト本文",
		}},
	}
	questionRepo := &mockQuestionWorkerRepository{}
	llm := &mockQuestionWorkerLLMClient{}
	uc := NewQuestionWorkerUsecaseWithJobRepository(highlightRepo, questionRepo, jobRepo, llm)

	if err := uc.ProcessQuestionGenerationJob(context.Background(), jobID, userID); err != nil {
		t.Fatalf("ProcessQuestionGenerationJob failed: %v", err)
	}

	if questionRepo.reservedDelta != 1 {
		t.Fatalf("expected one daily quota reservation, got %d", questionRepo.reservedDelta)
	}
	if len(questionRepo.superseded) != 1 || questionRepo.superseded[0] != highlightID {
		t.Fatalf("expected active question superseded, got %#v", questionRepo.superseded)
	}
	if len(questionRepo.savedHighlight) != 1 || questionRepo.savedHighlight[0] != highlightID {
		t.Fatalf("expected generated question saved for highlight, got %#v", questionRepo.savedHighlight)
	}
	if len(questionRepo.completedJobRefs) != 1 || questionRepo.completedJobRefs[0] != highlightID {
		t.Fatalf("expected highlight completed in job transaction, got %#v", questionRepo.completedJobRefs)
	}
	if questionRepo.completedJob != jobID {
		t.Fatalf("expected job completed in transaction, got %s", questionRepo.completedJob)
	}
	if llm.generateCalled != 1 {
		t.Fatalf("expected one Gemini call, got %d", llm.generateCalled)
	}
}

func TestProcessQuestionGenerationJobSkipsLLMWhenGlobalBudgetExceeded(t *testing.T) {
	jobID := uuid.New()
	userID := uuid.New()
	highlightID := uuid.New()
	jobRepo := &mockQuestionGenerationJobRepository{
		claimOK:  true,
		claimJob: &domain.QuestionGenerationJob{ID: jobID, UserID: userID, HighlightIDs: []uuid.UUID{highlightID}},
	}
	highlightRepo := &mockWorkerHighlightLifecycle{
		highlights: []*domain.Highlight{{
			ID:      highlightID,
			UserID:  userID,
			Content: "ハイライト本文",
		}},
	}
	questionRepo := &mockQuestionWorkerRepository{}
	llm := &mockQuestionWorkerLLMClient{}
	globalBudget := &mockGlobalLLMBudget{reserveErr: domain.ErrGlobalLLMBudgetExceeded}
	uc := NewQuestionWorkerUsecaseWithJobRepository(highlightRepo, questionRepo, jobRepo, llm).
		WithGlobalLLMBudget(globalBudget)

	err := uc.ProcessQuestionGenerationJob(context.Background(), jobID, userID)
	if !errors.Is(err, domain.ErrGlobalLLMBudgetExceeded) {
		t.Fatalf("expected global budget exceeded, got %v", err)
	}
	if llm.generateCalled != 0 {
		t.Fatalf("LLM must not be called when global budget is exceeded, got %d", llm.generateCalled)
	}
	if globalBudget.reserveCalls != 1 {
		t.Fatalf("expected one global reserve attempt, got %d", globalBudget.reserveCalls)
	}
	if questionRepo.releasedDelta != 1 {
		t.Fatalf("expected user quota release, got %d", questionRepo.releasedDelta)
	}
	if len(jobRepo.recordedFailures) != 1 || jobRepo.recordedFailures[0] != jobID {
		t.Fatalf("expected job failure recorded, got %#v", jobRepo.recordedFailures)
	}
}

func TestProcessQuestionGenerationJobRecordsUsageOnLLMSuccess(t *testing.T) {
	jobID := uuid.New()
	userID := uuid.New()
	highlightID := uuid.New()
	jobRepo := &mockQuestionGenerationJobRepository{
		claimOK:  true,
		claimJob: &domain.QuestionGenerationJob{ID: jobID, UserID: userID, HighlightIDs: []uuid.UUID{highlightID}},
	}
	highlightRepo := &mockWorkerHighlightLifecycle{
		highlights: []*domain.Highlight{{ID: highlightID, UserID: userID, Content: "ハイライト本文"}},
	}
	llm := &mockQuestionWorkerLLMClient{
		usage: &domain.LLMUsage{InputTokens: 12, OutputTokens: 34, EstimatedCostYen: 2.5},
	}
	globalBudget := &mockGlobalLLMBudget{}
	uc := NewQuestionWorkerUsecaseWithJobRepository(highlightRepo, &mockQuestionWorkerRepository{}, jobRepo, llm).
		WithGlobalLLMBudget(globalBudget)

	if err := uc.ProcessQuestionGenerationJob(context.Background(), jobID, userID); err != nil {
		t.Fatalf("ProcessQuestionGenerationJob failed: %v", err)
	}
	if globalBudget.recordUsageCalls != 1 {
		t.Fatalf("expected one usage log, got %d", globalBudget.recordUsageCalls)
	}
	usage := globalBudget.recordedUsage[0]
	if usage.UserID != userID || usage.JobID == nil || *usage.JobID != jobID {
		t.Fatalf("unexpected usage identity: %#v", usage)
	}
	if usage.Provider != "test-provider" || usage.Model != "gemini-test" {
		t.Fatalf("unexpected provider/model: %#v", usage)
	}
	if usage.InputTokens != 12 || usage.OutputTokens != 34 || usage.EstimatedCostYen != 2.5 {
		t.Fatalf("unexpected provider usage: %#v", usage)
	}
}

func TestProcessQuestionGenerationJobRecordsEstimatedUsageWhenProviderUsageMissing(t *testing.T) {
	jobID := uuid.New()
	userID := uuid.New()
	highlightID := uuid.New()
	jobRepo := &mockQuestionGenerationJobRepository{
		claimOK:  true,
		claimJob: &domain.QuestionGenerationJob{ID: jobID, UserID: userID, HighlightIDs: []uuid.UUID{highlightID}},
	}
	highlightRepo := &mockWorkerHighlightLifecycle{
		highlights: []*domain.Highlight{{ID: highlightID, UserID: userID, Content: "ハイライト本文"}},
	}
	globalBudget := &mockGlobalLLMBudget{}
	uc := NewQuestionWorkerUsecaseWithJobRepository(highlightRepo, &mockQuestionWorkerRepository{}, jobRepo, &mockQuestionWorkerLLMClient{}).
		WithGlobalLLMBudget(globalBudget)

	if err := uc.ProcessQuestionGenerationJob(context.Background(), jobID, userID); err != nil {
		t.Fatalf("ProcessQuestionGenerationJob failed: %v", err)
	}
	if globalBudget.recordUsageCalls != 1 {
		t.Fatalf("expected one usage log, got %d", globalBudget.recordUsageCalls)
	}
	usage := globalBudget.recordedUsage[0]
	if usage.InputTokens <= 0 || usage.OutputTokens <= 0 || usage.EstimatedCostYen <= 0 {
		t.Fatalf("expected estimated usage, got %#v", usage)
	}
}

func TestProcessQuestionGenerationJobRecordsUsageWhenLLMFailsAfterReserve(t *testing.T) {
	jobID := uuid.New()
	userID := uuid.New()
	highlightID := uuid.New()
	jobRepo := &mockQuestionGenerationJobRepository{
		claimOK:  true,
		claimJob: &domain.QuestionGenerationJob{ID: jobID, UserID: userID, HighlightIDs: []uuid.UUID{highlightID}},
	}
	highlightRepo := &mockWorkerHighlightLifecycle{
		highlights: []*domain.Highlight{{ID: highlightID, UserID: userID, Content: "ハイライト本文"}},
	}
	globalBudget := &mockGlobalLLMBudget{}
	uc := NewQuestionWorkerUsecaseWithJobRepository(
		highlightRepo,
		&mockQuestionWorkerRepository{},
		jobRepo,
		&mockQuestionWorkerLLMClient{err: errors.New("llm unavailable")},
	).WithGlobalLLMBudget(globalBudget)

	err := uc.ProcessQuestionGenerationJob(context.Background(), jobID, userID)
	if err == nil {
		t.Fatal("expected LLM failure")
	}
	if globalBudget.reserveCalls != 1 || globalBudget.recordUsageCalls != 1 {
		t.Fatalf("expected reserved failed LLM call to be logged, reserve=%d usage=%d", globalBudget.reserveCalls, globalBudget.recordUsageCalls)
	}
}

func TestProcessQuestionGenerationJobRecordsFailureForInvalidGeneratedQuestion(t *testing.T) {
	jobID := uuid.New()
	userID := uuid.New()
	highlightID := uuid.New()
	jobRepo := &mockQuestionGenerationJobRepository{
		claimOK:  true,
		claimJob: &domain.QuestionGenerationJob{ID: jobID, UserID: userID, HighlightIDs: []uuid.UUID{highlightID}},
	}
	highlightRepo := &mockWorkerHighlightLifecycle{
		highlights: []*domain.Highlight{{
			ID:      highlightID,
			UserID:  userID,
			Content: "ハイライト本文",
		}},
	}
	questionRepo := &mockQuestionWorkerRepository{}
	llm := &mockQuestionWorkerLLMClient{
		generated: []domain.GeneratedQuestion{{
			Content:       "",
			Options:       []string{"A", "B", "C", "D"},
			CorrectAnswer: "A",
			Explanation:   "解説",
		}},
	}
	uc := NewQuestionWorkerUsecaseWithJobRepository(highlightRepo, questionRepo, jobRepo, llm)

	if err := uc.ProcessQuestionGenerationJob(context.Background(), jobID, userID); err == nil {
		t.Fatal("expected invalid generated question to fail the job")
	}
	if len(jobRepo.recordedFailures) != 1 || jobRepo.recordedFailures[0] != jobID {
		t.Fatalf("expected job failure recorded, got %#v", jobRepo.recordedFailures)
	}
	if questionRepo.releasedDelta != 1 {
		t.Fatalf("expected reserved quota to be released, got %d", questionRepo.releasedDelta)
	}
	if questionRepo.completedJob != uuid.Nil {
		t.Fatalf("invalid generated question should not complete job, got %s", questionRepo.completedJob)
	}
}
