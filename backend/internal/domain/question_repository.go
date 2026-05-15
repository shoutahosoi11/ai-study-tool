package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type GenerateQuestionsInput struct {
	CreatorID           string
	SourceType          SourceType
	SourceID            string
	BookTitle           string
	BookAuthor          string
	QuestionCount       int
	QuestionType        QuestionType
	CustomInstruction   string
	UserPlan            string
	HighlightStartIndex int
	HighlightEndIndex   int
}

type QuestionCatalogReader interface {
	Save(ctx context.Context, q *Question, meta *QuestionMeta) error
	SupersedeActiveQuestionsForHighlight(ctx context.Context, userID uuid.UUID, highlightID uuid.UUID) error
	ListByCreatorID(ctx context.Context, creatorID string, limit int) ([]*Question, error)
	ListSavedByUserID(ctx context.Context, userID string, limit int) ([]*SavedQuestion, error)
	ListIncorrectByUserID(ctx context.Context, userID string, limit int) ([]*IncorrectQuestion, error)
	ListPreparedByUserIDAndHighlightIDs(ctx context.Context, userID string, highlightIDs []uuid.UUID, limit int) ([]*Question, error)
	ListUsedHighlightIDsByUserID(ctx context.Context, userID string, highlightIDs []uuid.UUID) ([]uuid.UUID, error)
	FindByID(ctx context.Context, id string) (*Question, *QuestionMeta, *QuestionStats, error)
	GetByID(ctx context.Context, id string) (*Question, error)
	SaveForUser(ctx context.Context, userID, questionID, note string) error
}

type QuestionGenerationRepository interface {
	Save(ctx context.Context, q *Question, meta *QuestionMeta) error
	SupersedeActiveQuestionsForHighlight(ctx context.Context, userID uuid.UUID, highlightID uuid.UUID) error
	SaveGeneration(ctx context.Context, userID, sourceType, sourceID, promptUsed, modelUsed string) (string, error)
	ListPerspectivesByHighlightID(ctx context.Context, userID string, highlightID uuid.UUID) ([]string, error)
}

type QuestionDailyQuotaRepository interface {
	GetDailyGeneratedCount(ctx context.Context, userID uuid.UUID, day time.Time) (int, error)
	ReserveDailyGeneratedCount(ctx context.Context, userID uuid.UUID, day time.Time, delta int, limit int) (bool, error)
	ReleaseDailyGeneratedCount(ctx context.Context, userID uuid.UUID, day time.Time, delta int) error
}

type QuestionSyncStateRepository interface {
	GetUserLastQuestionSyncAt(ctx context.Context, userID uuid.UUID) (*time.Time, error)
	UpdateUserLastQuestionSyncAt(ctx context.Context, userID uuid.UUID, syncedAt time.Time) error
}

type QuestionRepository interface {
	QuestionCatalogReader
	QuestionGenerationRepository
	QuestionDailyQuotaRepository
	QuestionSyncStateRepository
	ReplaceActiveQuestionsForHighlights(ctx context.Context, userID uuid.UUID, replacements []QuestionReplacement) error
	CompleteQuestionGenerationJob(ctx context.Context, userID uuid.UUID, jobID uuid.UUID, replacements []QuestionReplacement, highlightIDs []uuid.UUID) error
}

type QuestionUsecaseRepository interface {
	QuestionCatalogReader
	QuestionGenerationRepository
}

type AnswerQuestionRepository interface {
	FindByID(ctx context.Context, id string) (*Question, *QuestionMeta, *QuestionStats, error)
}

type QuestionSyncQuestionRepository interface {
	QuestionDailyQuotaRepository
	QuestionSyncStateRepository
	ListPerspectivesByHighlightID(ctx context.Context, userID string, highlightID uuid.UUID) ([]string, error)
	FindByID(ctx context.Context, id string) (*Question, *QuestionMeta, *QuestionStats, error)
	SupersedeActiveQuestionsForHighlight(ctx context.Context, userID uuid.UUID, highlightID uuid.UUID) error
}

type QuestionWorkerRepository interface {
	QuestionGenerationRepository
	QuestionDailyQuotaRepository
	ReplaceActiveQuestionsForHighlights(ctx context.Context, userID uuid.UUID, replacements []QuestionReplacement) error
	CompleteQuestionGenerationJob(ctx context.Context, userID uuid.UUID, jobID uuid.UUID, replacements []QuestionReplacement, highlightIDs []uuid.UUID) error
}

type QuestionReplacement struct {
	HighlightID uuid.UUID
	Question    *Question
	Meta        *QuestionMeta
}
