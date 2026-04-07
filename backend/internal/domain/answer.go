package domain

import "time"

type Answer struct {
	ID          string
	UserID      string
	QuestionID  string
	UserAnswer  string
	IsCorrect   bool
	Score       *int
	Feedback    *string
	GraderModel *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
