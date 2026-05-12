package usecase

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/shout/ai-study-tool/backend/internal/domain"
)

const (
	maxGeneratedQuestionContentLength     = 2000
	maxGeneratedQuestionOptionLength      = 500
	maxGeneratedQuestionCorrectAnswerSize = 500
	maxGeneratedQuestionExplanationLength = 4000
	requiredMultipleChoiceOptionCount     = 4
)

func normalizeGeneratedQuestion(input domain.GeneratedQuestion, questionType domain.QuestionType) (domain.GeneratedQuestion, error) {
	normalized := domain.GeneratedQuestion{
		Content:       strings.TrimSpace(input.Content),
		Options:       normalizeGeneratedQuestionOptions(input.Options),
		CorrectAnswer: strings.TrimSpace(input.CorrectAnswer),
		Explanation:   strings.TrimSpace(input.Explanation),
	}

	if !validGeneratedText(normalized.Content, maxGeneratedQuestionContentLength) {
		return domain.GeneratedQuestion{}, fmt.Errorf("%w: invalid generated question content", domain.ErrInvalidInput)
	}
	if !validGeneratedText(normalized.CorrectAnswer, maxGeneratedQuestionCorrectAnswerSize) {
		return domain.GeneratedQuestion{}, fmt.Errorf("%w: invalid generated correct answer", domain.ErrInvalidInput)
	}
	if !validGeneratedText(normalized.Explanation, maxGeneratedQuestionExplanationLength) {
		return domain.GeneratedQuestion{}, fmt.Errorf("%w: invalid generated explanation", domain.ErrInvalidInput)
	}

	if questionType == domain.QuestionTypeMultipleChoice {
		if len(normalized.Options) != requiredMultipleChoiceOptionCount {
			return domain.GeneratedQuestion{}, fmt.Errorf("%w: invalid generated options", domain.ErrInvalidInput)
		}
		for _, option := range normalized.Options {
			if !validGeneratedText(option, maxGeneratedQuestionOptionLength) {
				return domain.GeneratedQuestion{}, fmt.Errorf("%w: invalid generated option", domain.ErrInvalidInput)
			}
		}
	}

	return normalized, nil
}

func normalizeGeneratedQuestionOptions(options []string) []string {
	normalized := make([]string, 0, len(options))
	for _, option := range options {
		normalized = append(normalized, strings.TrimSpace(option))
	}
	return normalized
}

func validGeneratedText(value string, maxRunes int) bool {
	length := utf8.RuneCountInString(value)
	return length > 0 && length <= maxRunes
}
