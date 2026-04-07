package usecase

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/shout/ai-study-tool/backend/internal/domain"
)

type AnswerUsecase struct {
	answerRepo   domain.AnswerRepository
	questionRepo domain.QuestionRepository
	llmClient    domain.LLMClient
}

func NewAnswerUsecase(
	answerRepo domain.AnswerRepository,
	questionRepo domain.QuestionRepository,
	llmClient domain.LLMClient,
) *AnswerUsecase {
	return &AnswerUsecase{
		answerRepo:   answerRepo,
		questionRepo: questionRepo,
		llmClient:    llmClient,
	}
}

type SubmitAnswerInput struct {
	UserID     string
	QuestionID string
	UserAnswer string
	UserPlan   string
}

type SubmitAnswerResult struct {
	IsCorrect     bool
	CorrectAnswer string
	Explanation   string
	Score         *int
	Feedback      *string
}

func (u *AnswerUsecase) SubmitAnswer(ctx context.Context, input SubmitAnswerInput) (*SubmitAnswerResult, error) {
	q, err := u.questionRepo.GetByID(ctx, input.QuestionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("not_found: question not found")
		}
		return nil, fmt.Errorf("answer usecase: get question: %w", err)
	}

	var (
		isCorrect   bool
		score       *int
		feedback    *string
		graderModel *string
	)

	switch q.QuestionType {
	case domain.QuestionTypeMultipleChoice:
		isCorrect = q.IsCorrect(input.UserAnswer)
	case domain.QuestionTypeDescriptive:
		model := modelForPlan(input.UserPlan)
		gradeResult, err := u.llmClient.GradeAnswer(ctx, q, input.UserAnswer, model)
		if err != nil {
			return nil, fmt.Errorf("llm_error: %w", err)
		}
		isCorrect = gradeResult.IsCorrect
		score = &gradeResult.Score
		feedback = &gradeResult.Feedback
		graderModel = &model
	default:
		isCorrect = q.IsCorrect(input.UserAnswer)
	}

	upsertInput := domain.AnswerUpsertInput{
		UserID:      input.UserID,
		QuestionID:  input.QuestionID,
		UserAnswer:  input.UserAnswer,
		IsCorrect:   isCorrect,
		Score:       score,
		Feedback:    feedback,
		GraderModel: graderModel,
	}
	if _, err := u.answerRepo.Upsert(ctx, upsertInput); err != nil {
		return nil, fmt.Errorf("answer usecase: upsert answer: %w", err)
	}

	if err := u.questionRepo.IncrementStats(ctx, input.QuestionID, isCorrect); err != nil {
		return nil, fmt.Errorf("answer usecase: increment stats: %w", err)
	}

	return &SubmitAnswerResult{
		IsCorrect:     isCorrect,
		CorrectAnswer: q.CorrectAnswer,
		Explanation:   q.Explanation,
		Score:         score,
		Feedback:      feedback,
	}, nil
}
