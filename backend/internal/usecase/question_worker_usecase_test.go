package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

type mockQuestionWorkerRepository struct {
	save func(ctx context.Context, question *domain.Question, meta *domain.QuestionMeta) error
}

func (m *mockQuestionWorkerRepository) Save(ctx context.Context, question *domain.Question, meta *domain.QuestionMeta) error {
	if m.save == nil {
		return nil
	}
	return m.save(ctx, question, meta)
}

func (m *mockQuestionWorkerRepository) SaveGeneration(ctx context.Context, userID, sourceType, sourceID, promptUsed, modelUsed string) (string, error) {
	return "generation-id", nil
}

func (m *mockQuestionWorkerRepository) ListPerspectivesByHighlightID(ctx context.Context, userID string, highlightID uuid.UUID) ([]string, error) {
	return make([]string, 0), nil
}

func (m *mockQuestionWorkerRepository) EnqueueRegeneration(ctx context.Context, userID string, highlightID uuid.UUID, questionID string) error {
	return nil
}

func (m *mockQuestionWorkerRepository) ClaimPendingRegenerationTasks(ctx context.Context, limit int) ([]*domain.RegenerationTask, error) {
	return make([]*domain.RegenerationTask, 0), nil
}

func (m *mockQuestionWorkerRepository) MarkRegenerationTasksCompleted(ctx context.Context, taskIDs []uuid.UUID) error {
	return nil
}

func (m *mockQuestionWorkerRepository) MarkRegenerationTasksFailed(ctx context.Context, taskIDs []uuid.UUID, lastError string, maxRetry int) error {
	return nil
}

func TestDynamicMaxWaitTime(t *testing.T) {
	cases := []struct {
		total int
		want  time.Duration
	}{
		{total: 1, want: 5 * time.Second},
		{total: 5, want: 15 * time.Second},
		{total: 6, want: 15 * time.Second},
		{total: 20, want: 30 * time.Second},
		{total: 21, want: 30 * time.Second},
		{total: 41, want: 60 * time.Second},
		{total: 100, want: 60 * time.Second},
	}

	for _, tc := range cases {
		got := dynamicMaxWaitTime(tc.total)
		if got != tc.want {
			t.Fatalf("dynamicMaxWaitTime(%d) = %s, want %s", tc.total, got, tc.want)
		}
	}
}

func TestNextPerspectiveVersionsPrefersUnused(t *testing.T) {
	perspectives, versions := nextPerspectiveVersions([]string{
		domain.QuestionPerspectiveDefinition,
		domain.QuestionPerspectiveComparison,
	}, 3)

	wantPerspectives := []string{
		domain.QuestionPerspectivePractical,
		domain.QuestionPerspectiveUnderstanding,
		domain.QuestionPerspectivePitfall,
	}
	wantVersions := []int{1, 1, 1}

	for index, want := range wantPerspectives {
		if perspectives[index] != want {
			t.Fatalf("perspectives[%d] = %s, want %s", index, perspectives[index], want)
		}
	}
	for index, want := range wantVersions {
		if versions[index] != want {
			t.Fatalf("versions[%d] = %d, want %d", index, versions[index], want)
		}
	}
}

func TestShouldProcessPendingBatch(t *testing.T) {
	now := time.Date(2026, 4, 25, 6, 0, 0, 0, time.UTC)
	stat := domain.PendingHighlightUserStat{
		UserID:          uuid.New(),
		PendingCount:    3,
		TotalCount:      3,
		OldestPendingAt: now.Add(-11 * time.Second),
	}

	if !shouldProcessPendingBatch(stat, now, 5*time.Minute, defaultWorkerBatchSize) {
		t.Fatal("expected batch to be due once dynamic wait time is exceeded")
	}

	stat.PendingCount = 10
	stat.OldestPendingAt = now
	if !shouldProcessPendingBatch(stat, now, 5*time.Minute, defaultWorkerBatchSize) {
		t.Fatal("expected batch to be due immediately when pending count reaches 10")
	}

	stat.PendingCount = 5
	if !shouldProcessPendingBatch(stat, now, 5*time.Minute, 5) {
		t.Fatal("expected batch to respect configured worker batch size")
	}
}

func TestSaveGeneratedQuestionsForChunkKeepsPartialSaveFailureFailed(t *testing.T) {
	highlightID := uuid.New()
	saveCalls := 0
	uc := &QuestionWorkerUsecase{
		questionRepo: &mockQuestionWorkerRepository{
			save: func(ctx context.Context, question *domain.Question, meta *domain.QuestionMeta) error {
				saveCalls++
				if saveCalls == 2 {
					return errors.New("save failed")
				}
				return nil
			},
		},
	}

	completed := make(map[uuid.UUID]struct{})
	failed := make(map[uuid.UUID]string)
	uc.saveGeneratedQuestionsForChunk(
		context.Background(),
		uuid.NewString(),
		"generation-id",
		[]highlightGenerationPlan{{
			highlight: &domain.Highlight{
				ID: highlightID,
			},
			perspectives: []string{
				domain.QuestionPerspectiveDefinition,
				domain.QuestionPerspectiveUnderstanding,
			},
			versions: []int{1, 1},
		}},
		[]domain.GeneratedQuestion{
			{
				Content:       "question 1",
				Options:       []string{"a", "b", "c", "d"},
				CorrectAnswer: "a",
				Explanation:   "explanation 1",
			},
			{
				Content:       "question 2",
				Options:       []string{"a", "b", "c", "d"},
				CorrectAnswer: "b",
				Explanation:   "explanation 2",
			},
		},
		completed,
		failed,
	)

	if _, ok := completed[highlightID]; ok {
		t.Fatal("expected partially saved highlight not to be marked completed")
	}
	if failed[highlightID] == "" {
		t.Fatal("expected partially saved highlight to remain failed for retry")
	}
}

func TestSplitHighlightPlansByLimitRespectsQuestionBudget(t *testing.T) {
	plans := []highlightGenerationPlan{
		{highlight: &domain.Highlight{Content: strings.Repeat("a", 400)}, questionCount: 3},
		{highlight: &domain.Highlight{Content: strings.Repeat("b", 400)}, questionCount: 3},
		{highlight: &domain.Highlight{Content: strings.Repeat("c", 400)}, questionCount: 3},
	}

	chunks := splitHighlightPlansByLimit(plans, 100000, 8)
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}

	firstChunkQuestions := 0
	for _, plan := range chunks[0] {
		firstChunkQuestions += plan.questionCount
	}
	if firstChunkQuestions > 8 {
		t.Fatalf("expected first chunk to have at most 8 questions, got %d", firstChunkQuestions)
	}
}
