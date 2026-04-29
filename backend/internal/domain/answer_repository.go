package domain

import "context"

type AnswerUpsertInput struct {
	UserID      string
	QuestionID  string
	UserAnswer  string
	IsCorrect   bool
	Score       *int
	Feedback    *string
	GraderModel *string
}

type AnswerRepository interface {
	Upsert(ctx context.Context, input AnswerUpsertInput) (*Answer, error)
	UpsertAndUpdateStats(ctx context.Context, input AnswerUpsertInput) (*Answer, error)
}
