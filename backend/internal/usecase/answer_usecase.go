package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

type AnswerUsecase struct {
	answerRepo   domain.AnswerRepository
	questionRepo domain.AnswerQuestionRepository
	llmClient    domain.LLMClient
}

func NewAnswerUsecase(
	answerRepo domain.AnswerRepository,
	questionRepo domain.AnswerQuestionRepository,
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
	q, meta, _, err := u.questionRepo.FindByID(ctx, input.QuestionID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, fmt.Errorf("answer usecase: question not found: %w", domain.ErrNotFound)
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
		model := u.llmClient.ModelForPlan(input.UserPlan)
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
	if _, err := u.answerRepo.UpsertAndUpdateStats(ctx, upsertInput); err != nil {
		return nil, fmt.Errorf("answer usecase: upsert answer: %w", err)
	}

	if meta != nil && meta.CreatorID == input.UserID && meta.HighlightID != "" {
		highlightID, parseErr := uuid.Parse(meta.HighlightID)
		if parseErr == nil {
			if enqueueErr := u.questionRepo.EnqueueRegeneration(ctx, input.UserID, highlightID, input.QuestionID); enqueueErr != nil {
				log.Printf("answer usecase: enqueue regeneration error: %v", enqueueErr)
			}
		}
	}

	return &SubmitAnswerResult{
		IsCorrect:     isCorrect,
		CorrectAnswer: q.CorrectAnswer,
		Explanation:   q.Explanation,
		Score:         score,
		Feedback:      feedback,
	}, nil
}
