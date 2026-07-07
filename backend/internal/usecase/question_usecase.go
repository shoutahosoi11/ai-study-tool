package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

type QuestionUsecase struct {
	repo           domain.QuestionLearningRepository
	sourceResolver domain.QuestionSourceResolver
}

const (
	maxExplicitQuestionCount = 10
	maxQuestionCountForAll   = 20
)

func NewQuestionUsecase(repo domain.QuestionLearningRepository, sourceResolver domain.QuestionSourceResolver) *QuestionUsecase {
	return &QuestionUsecase{
		repo:           repo,
		sourceResolver: sourceResolver,
	}
}

func (u *QuestionUsecase) ListQuestions(ctx context.Context, creatorID string, limit int) ([]*domain.Question, error) {
	questions, err := u.repo.ListByCreatorID(ctx, creatorID, limit)
	if err != nil {
		return nil, fmt.Errorf("question usecase: list questions: %w", err)
	}
	if questions == nil {
		return make([]*domain.Question, 0), nil
	}

	return questions, nil
}

func (u *QuestionUsecase) ListSavedQuestions(ctx context.Context, userID string, limit int) ([]*domain.SavedQuestion, error) {
	savedQuestions, err := u.repo.ListSavedByUserID(ctx, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("question usecase: list saved questions: %w", err)
	}
	if savedQuestions == nil {
		return make([]*domain.SavedQuestion, 0), nil
	}

	return savedQuestions, nil
}

func (u *QuestionUsecase) ListIncorrectQuestions(ctx context.Context, userID string, limit int) ([]*domain.IncorrectQuestion, error) {
	incorrectQuestions, err := u.repo.ListIncorrectByUserID(ctx, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("question usecase: list incorrect questions: %w", err)
	}
	if incorrectQuestions == nil {
		return make([]*domain.IncorrectQuestion, 0), nil
	}

	return incorrectQuestions, nil
}

func (u *QuestionUsecase) ListPreparedQuestions(ctx context.Context, input domain.GenerateQuestionsInput) ([]*domain.Question, error) {
	if !isSupportedQuestionSourceType(input.SourceType) {
		return nil, domain.ErrInvalidSourceType
	}

	sourceHighlights, err := u.sourceResolver.ResolveHighlights(
		ctx,
		input.CreatorID,
		input.SourceType,
		input.SourceID,
		input.BookTitle,
		input.BookAuthor,
	)
	if err != nil {
		return nil, fmt.Errorf("question usecase: resolve source highlights: %w", err)
	}

	candidates := filterNonEmptyHighlights(sourceHighlights)
	candidates = filterHighlightsByBookOrderIndex(candidates, input.HighlightStartIndex, input.HighlightEndIndex)
	if len(candidates) == 0 {
		return nil, domain.ErrSourceTextUnavailable
	}

	highlightIDs := make([]uuid.UUID, 0, len(candidates))
	hasPending := false
	hasFailed := false
	for _, highlight := range candidates {
		highlightIDs = append(highlightIDs, highlight.ID)
		if highlight.Status == domain.HighlightStatusPending || highlight.Status == domain.HighlightStatusProcessing {
			hasPending = true
		}
		if highlight.Status == domain.HighlightStatusFailed {
			hasFailed = true
		}
	}

	limit := resolveQuestionSelectionCount(len(candidates), input.QuestionCount)
	preparedQuestions, err := u.repo.ListPreparedByUserIDAndHighlightIDs(ctx, input.CreatorID, highlightIDs, limit)
	if err != nil {
		return nil, fmt.Errorf("question usecase: list prepared questions: %w", err)
	}
	if len(preparedQuestions) > 0 {
		return preparedQuestions, nil
	}
	if hasPending {
		return nil, domain.ErrQuestionsPreparing
	}
	if hasFailed {
		return nil, domain.ErrQuestionGenerationFailed
	}

	return nil, domain.ErrSourceTextUnavailable
}

func filterHighlightsByBookOrderIndex(highlights []*domain.Highlight, startIndex int, endIndex int) []*domain.Highlight {
	if startIndex == 0 && endIndex == 0 {
		return highlights
	}
	filtered := make([]*domain.Highlight, 0, len(highlights))
	for _, highlight := range highlights {
		if highlight == nil || highlight.BookOrderIndex == nil {
			continue
		}
		index := *highlight.BookOrderIndex
		if startIndex > 0 && index < startIndex {
			continue
		}
		if endIndex > 0 && index > endIndex {
			continue
		}
		filtered = append(filtered, highlight)
	}
	return filtered
}

func (u *QuestionUsecase) SaveQuestion(ctx context.Context, userID string, questionID string, note string) error {
	_, meta, _, err := u.repo.FindByID(ctx, questionID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("question usecase: get question for save: %w", err)
	}

	normalizedNote := strings.TrimSpace(note)
	if meta != nil && strings.TrimSpace(meta.CreatorID) != strings.TrimSpace(userID) && normalizedNote != "" {
		return domain.ErrForbidden
	}

	if err := u.repo.SaveForUser(ctx, userID, questionID, normalizedNote); err != nil {
		return fmt.Errorf("question usecase: save question: %w", err)
	}

	return nil
}

func isSupportedQuestionSourceType(sourceType domain.SourceType) bool {
	switch sourceType {
	case domain.SourceTypeKindleBook:
		return true
	default:
		return false
	}
}

func sanitizeQuestionCount(questionCount int) int {
	if questionCount <= 0 {
		return 0
	}
	if questionCount > maxExplicitQuestionCount {
		return maxExplicitQuestionCount
	}
	return questionCount
}

func resolveQuestionSelectionCount(candidateCount int, questionCount int) int {
	if candidateCount <= 0 {
		return 0
	}

	if questionCount == 0 {
		if candidateCount > maxQuestionCountForAll {
			return maxQuestionCountForAll
		}
		return candidateCount
	}

	maxQuestions := sanitizeQuestionCount(questionCount)
	if maxQuestions > candidateCount {
		return candidateCount
	}

	return maxQuestions
}

func filterNonEmptyHighlights(highlights []*domain.Highlight) []*domain.Highlight {
	filtered := make([]*domain.Highlight, 0, len(highlights))
	for _, highlight := range highlights {
		if highlight == nil {
			continue
		}
		if strings.TrimSpace(highlight.Content) == "" {
			continue
		}
		filtered = append(filtered, highlight)
	}

	return filtered
}
