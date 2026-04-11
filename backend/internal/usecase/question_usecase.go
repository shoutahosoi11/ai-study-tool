package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/google/uuid"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

type QuestionUsecase struct {
	repo      domain.QuestionRepository
	llmClient domain.LLMClient
}

func NewQuestionUsecase(repo domain.QuestionRepository, llmClient domain.LLMClient) *QuestionUsecase {
	return &QuestionUsecase{
		repo:      repo,
		llmClient: llmClient,
	}
}

func (u *QuestionUsecase) GenerateQuestions(ctx context.Context, input domain.GenerateQuestionsInput) ([]*domain.Question, error) {
	model := modelForPlan(input.UserPlan)

	points, err := u.llmClient.ExtractPoints(ctx, input.SourceText, model)
	if err != nil {
		return nil, fmt.Errorf("question usecase: extract points: %w", err)
	}

	genID, err := u.repo.SaveGeneration(ctx,
		input.CreatorID,
		string(input.SourceType),
		input.SourceID,
		"2-step prompt",
		model,
	)
	if err != nil {
		return nil, fmt.Errorf("question usecase: save generation: %w", err)
	}

	type result struct {
		q   *domain.GeneratedQuestion
		err error
	}

	results := make([]result, len(points))
	var wg sync.WaitGroup

	for i, point := range points {
		wg.Add(1)
		go func(idx int, p domain.ExtractedPoint) {
			defer wg.Done()
			gq, err := u.llmClient.GenerateQuestion(ctx, p, input.QuestionType, input.CustomInstruction, model)
			results[idx] = result{q: gq, err: err}
		}(i, point)
	}
	wg.Wait()

	questions := make([]*domain.Question, 0, len(results))
	for _, r := range results {
		if r.err != nil {
			log.Printf("question usecase: generate question error: %v", r.err)
			continue
		}

		q := &domain.Question{
			ID:            uuid.New().String(),
			QuestionType:  input.QuestionType,
			Content:       r.q.Content,
			Options:       r.q.Options,
			CorrectAnswer: r.q.CorrectAnswer,
			Explanation:   r.q.Explanation,
		}
		meta := &domain.QuestionMeta{
			QuestionID:    q.ID,
			CreatorID:     input.CreatorID,
			SourceType:    input.SourceType,
			SourceID:      input.SourceID,
			GenerationID:  genID,
			IsAIGenerated: true,
		}

		if err := u.repo.Save(ctx, q, meta); err != nil {
			log.Printf("question usecase: save question error: %v", err)
			continue
		}
		questions = append(questions, q)
	}

	if len(questions) == 0 {
		return nil, fmt.Errorf("question usecase: all question generation failed")
	}

	return questions, nil
}

func (u *QuestionUsecase) GradeAnswer(ctx context.Context, input domain.GradeInput, userPlan string) (*domain.GradeResult, error) {
	model := modelForPlan(userPlan)

	q, _, _, err := u.repo.FindByID(ctx, input.QuestionID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, fmt.Errorf("question usecase: question not found: %w", domain.ErrNotFound)
		}
		return nil, fmt.Errorf("question usecase: find question: %w", err)
	}

	if q.QuestionType == domain.QuestionTypeMultipleChoice {
		isCorrect := q.IsCorrect(input.UserAnswer)
		if err := u.repo.UpdateStats(ctx, input.QuestionID, isCorrect); err != nil {
			return nil, fmt.Errorf("question usecase: update stats: %w", err)
		}
		feedback := "不正解です"
		if isCorrect {
			feedback = "正解です"
		}
		score := 0
		if isCorrect {
			score = 100
		}
		return &domain.GradeResult{
			IsCorrect: isCorrect,
			Score:     score,
			Feedback:  feedback,
		}, nil
	}

	gradeResult, err := u.llmClient.GradeAnswer(ctx, q, input.UserAnswer, model)
	if err != nil {
		return nil, fmt.Errorf("question usecase: grade answer: %w", err)
	}

	if err := u.repo.UpdateStats(ctx, input.QuestionID, gradeResult.IsCorrect); err != nil {
		return nil, fmt.Errorf("question usecase: update stats: %w", err)
	}

	return gradeResult, nil
}

func modelForPlan(plan string) string {
	if plan == "pro" {
		return "gemini-1.5-pro"
	}
	return "gemini-1.5-flash"
}
