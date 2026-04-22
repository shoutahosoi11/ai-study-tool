package domain

import (
	"context"

	"github.com/google/uuid"
)

type GenerateQuestionsInput struct {
	CreatorID         string
	SourceType        SourceType
	SourceID          string
	BookTitle         string
	BookAuthor        string
	QuestionCount     int
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
	ListByCreatorID(ctx context.Context, creatorID string, limit int) ([]*Question, error)
	ListSavedByUserID(ctx context.Context, userID string, limit int) ([]*SavedQuestion, error)
	ListIncorrectByUserID(ctx context.Context, userID string, limit int) ([]*IncorrectQuestion, error)
	ListUsedHighlightIDsByUserID(ctx context.Context, userID string, highlightIDs []uuid.UUID) ([]uuid.UUID, error)
	FindByID(ctx context.Context, id string) (*Question, *QuestionMeta, *QuestionStats, error)
	GetByID(ctx context.Context, id string) (*Question, error)
	UpdateStats(ctx context.Context, questionID string, isCorrect bool) error
	SaveGeneration(ctx context.Context, userID, sourceType, sourceID, promptUsed, modelUsed string) (string, error)
	SaveForUser(ctx context.Context, userID, questionID, note string) error
}
