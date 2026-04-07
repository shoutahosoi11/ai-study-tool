package domain

import "context"

type GenerateQuestionsInput struct {
	CreatorID         string
	SourceType        SourceType
	SourceID          string
	SourceText        string
	QuestionType      QuestionType
	CustomInstruction string
	UserPlan          string
}

type GradeInput struct {
	QuestionID string
	UserAnswer string
}

type GradeResult struct {
	IsCorrect bool
	Score     int
	Feedback  string
}

type QuestionRepository interface {
	Save(ctx context.Context, q *Question, meta *QuestionMeta) error
	FindByID(ctx context.Context, id string) (*Question, *QuestionMeta, *QuestionStats, error)
	UpdateStats(ctx context.Context, questionID string, isCorrect bool) error
	SaveGeneration(ctx context.Context, userID, sourceType, sourceID, promptUsed, modelUsed string) (string, error)
}
