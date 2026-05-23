package usecase

import (
	"strings"
	"testing"

	"github.com/shout/ai-study-tool/backend/internal/domain"
)

func TestNormalizeGeneratedQuestionAcceptsCorrectAnswerInOptions(t *testing.T) {
	question, err := normalizeGeneratedQuestion(domain.GeneratedQuestion{
		Content:       " question ",
		Options:       []string{" A ", "B", "C", "D"},
		CorrectAnswer: " A ",
		Explanation:   " explanation ",
	}, domain.QuestionTypeMultipleChoice)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if question.CorrectAnswer != "A" || question.Options[0] != "A" {
		t.Fatalf("expected trimmed answer and option, got %#v", question)
	}
}

func TestNormalizeGeneratedQuestionRequiresCorrectAnswerInOptions(t *testing.T) {
	_, err := normalizeGeneratedQuestion(domain.GeneratedQuestion{
		Content:       "question",
		Options:       []string{"A", "B", "C", "D"},
		CorrectAnswer: "E",
		Explanation:   "explanation",
	}, domain.QuestionTypeMultipleChoice)
	if err == nil {
		t.Fatal("expected invalid generated question")
	}
}

func TestNormalizeGeneratedQuestionDoesNotAcceptPartialOptionMatch(t *testing.T) {
	_, err := normalizeGeneratedQuestion(domain.GeneratedQuestion{
		Content:       "question",
		Options:       []string{"Correct answer with extra text", "B", "C", "D"},
		CorrectAnswer: "Correct answer",
		Explanation:   "explanation",
	}, domain.QuestionTypeMultipleChoice)
	if err == nil {
		t.Fatal("expected partial option match to be rejected")
	}
}

func TestNormalizeGeneratedQuestionRejectsBlankAfterTrimming(t *testing.T) {
	_, err := normalizeGeneratedQuestion(domain.GeneratedQuestion{
		Content:       strings.Repeat(" ", 3),
		Options:       []string{"A", "B", "C", "D"},
		CorrectAnswer: "A",
		Explanation:   "explanation",
	}, domain.QuestionTypeMultipleChoice)
	if err == nil {
		t.Fatal("expected blank content to be rejected")
	}
}
