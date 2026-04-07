package domain

import "context"

type ExtractedPoint struct {
	Point   string
	Context string
}

type GeneratedQuestion struct {
	Content       string
	Options       []string
	CorrectAnswer string
	Explanation   string
}

type LLMClient interface {
	ExtractPoints(ctx context.Context, text string, model string) ([]ExtractedPoint, error)
	GenerateQuestion(ctx context.Context, point ExtractedPoint, questionType QuestionType, customInstruction string, model string) (*GeneratedQuestion, error)
	GradeAnswer(ctx context.Context, question *Question, userAnswer string, model string) (*GradeResult, error)
}
