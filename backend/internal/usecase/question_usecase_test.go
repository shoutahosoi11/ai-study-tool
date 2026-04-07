package usecase_test

import (
	"context"
	"testing"

	"github.com/shout/ai-study-tool/backend/internal/domain"
	"github.com/shout/ai-study-tool/backend/internal/usecase"
)

type mockLLMClient struct {
	extractPoints    func(ctx context.Context, text string, model string) ([]domain.ExtractedPoint, error)
	generateQuestion func(ctx context.Context, point domain.ExtractedPoint, questionType domain.QuestionType, customInstruction string, model string) (*domain.GeneratedQuestion, error)
	gradeAnswer      func(ctx context.Context, question *domain.Question, userAnswer string, model string) (*domain.GradeResult, error)
}

func (m *mockLLMClient) ExtractPoints(ctx context.Context, text string, model string) ([]domain.ExtractedPoint, error) {
	return m.extractPoints(ctx, text, model)
}

func (m *mockLLMClient) GenerateQuestion(ctx context.Context, point domain.ExtractedPoint, questionType domain.QuestionType, customInstruction string, model string) (*domain.GeneratedQuestion, error) {
	return m.generateQuestion(ctx, point, questionType, customInstruction, model)
}

func (m *mockLLMClient) GradeAnswer(ctx context.Context, question *domain.Question, userAnswer string, model string) (*domain.GradeResult, error) {
	return m.gradeAnswer(ctx, question, userAnswer, model)
}

type mockQuestionRepository struct {
	save           func(ctx context.Context, q *domain.Question, meta *domain.QuestionMeta) error
	findByID       func(ctx context.Context, id string) (*domain.Question, *domain.QuestionMeta, *domain.QuestionStats, error)
	updateStats    func(ctx context.Context, questionID string, isCorrect bool) error
	saveGeneration func(ctx context.Context, userID, sourceType, sourceID, promptUsed, modelUsed string) (string, error)
}

func (m *mockQuestionRepository) Save(ctx context.Context, q *domain.Question, meta *domain.QuestionMeta) error {
	return m.save(ctx, q, meta)
}

func (m *mockQuestionRepository) FindByID(ctx context.Context, id string) (*domain.Question, *domain.QuestionMeta, *domain.QuestionStats, error) {
	return m.findByID(ctx, id)
}

func (m *mockQuestionRepository) UpdateStats(ctx context.Context, questionID string, isCorrect bool) error {
	return m.updateStats(ctx, questionID, isCorrect)
}

func (m *mockQuestionRepository) SaveGeneration(ctx context.Context, userID, sourceType, sourceID, promptUsed, modelUsed string) (string, error) {
	return m.saveGeneration(ctx, userID, sourceType, sourceID, promptUsed, modelUsed)
}

func TestGenerateQuestions_TwoStepFlow(t *testing.T) {
	ctx := context.Background()

	llm := &mockLLMClient{
		extractPoints: func(ctx context.Context, text string, model string) ([]domain.ExtractedPoint, error) {
			return []domain.ExtractedPoint{
				{Point: "テストポイント1", Context: "コンテキスト1"},
				{Point: "テストポイント2", Context: "コンテキスト2"},
			}, nil
		},
		generateQuestion: func(ctx context.Context, point domain.ExtractedPoint, questionType domain.QuestionType, customInstruction string, model string) (*domain.GeneratedQuestion, error) {
			return &domain.GeneratedQuestion{
				Content:       "問題: " + point.Point,
				Options:       []string{"A", "B", "C", "D"},
				CorrectAnswer: "A",
				Explanation:   "解説",
			}, nil
		},
	}

	repo := &mockQuestionRepository{
		saveGeneration: func(ctx context.Context, userID, sourceType, sourceID, promptUsed, modelUsed string) (string, error) {
			return "gen-id-123", nil
		},
		save: func(ctx context.Context, q *domain.Question, meta *domain.QuestionMeta) error {
			return nil
		},
	}

	uc := usecase.NewQuestionUsecase(repo, llm)

	input := domain.GenerateQuestionsInput{
		CreatorID:    "user-123",
		SourceType:   domain.SourceTypeNote,
		SourceText:   "テストテキスト",
		QuestionType: domain.QuestionTypeMultipleChoice,
		UserPlan:     "free",
	}

	questions, err := uc.GenerateQuestions(ctx, input)
	if err != nil {
		t.Fatalf("GenerateQuestions failed: %v", err)
	}

	if len(questions) != 2 {
		t.Errorf("expected 2 questions, got %d", len(questions))
	}
}

func TestGenerateQuestions_UsesFlashForFreePlan(t *testing.T) {
	ctx := context.Background()
	var usedModel string

	llm := &mockLLMClient{
		extractPoints: func(ctx context.Context, text string, model string) ([]domain.ExtractedPoint, error) {
			usedModel = model
			return []domain.ExtractedPoint{
				{Point: "ポイント", Context: "文脈"},
			}, nil
		},
		generateQuestion: func(ctx context.Context, point domain.ExtractedPoint, questionType domain.QuestionType, customInstruction string, model string) (*domain.GeneratedQuestion, error) {
			return &domain.GeneratedQuestion{
				Content:       "問題",
				Options:       []string{"A", "B", "C", "D"},
				CorrectAnswer: "A",
				Explanation:   "解説",
			}, nil
		},
	}

	repo := &mockQuestionRepository{
		saveGeneration: func(_ context.Context, _, _, _, _, _ string) (string, error) { return "gen-id", nil },
		save:           func(_ context.Context, _ *domain.Question, _ *domain.QuestionMeta) error { return nil },
	}

	uc := usecase.NewQuestionUsecase(repo, llm)

	input := domain.GenerateQuestionsInput{
		CreatorID:    "user-123",
		SourceType:   domain.SourceTypeNote,
		SourceText:   "テキスト",
		QuestionType: domain.QuestionTypeMultipleChoice,
		UserPlan:     "free",
	}

	_, err := uc.GenerateQuestions(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if usedModel != "gemini-1.5-flash" {
		t.Errorf("expected gemini-1.5-flash, got %s", usedModel)
	}
}
