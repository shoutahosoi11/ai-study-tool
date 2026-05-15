package domain

import "time"

const (
	QuestionPerspectiveDefinition    = "definition"
	QuestionPerspectiveComparison    = "comparison"
	QuestionPerspectivePractical     = "practical"
	QuestionPerspectiveUnderstanding = "understanding"
	QuestionPerspectivePitfall       = "pitfall"
	QuestionPerspectiveApplication   = "application"
)

var QuestionPerspectiveOrder = []string{
	QuestionPerspectiveDefinition,
	QuestionPerspectiveComparison,
	QuestionPerspectivePractical,
	QuestionPerspectiveUnderstanding,
	QuestionPerspectivePitfall,
	QuestionPerspectiveApplication,
}

type BookStock struct {
	BookKey           string
	BookTitle         string
	BookAuthor        string
	Stock             int
	Preparing         int
	LatestHighlightAt time.Time
}

type QuestionGenerationBookCandidate struct {
	BookKey                 string
	BookTitle               string
	BookAuthor              string
	PendingHighlightCount   int
	UnansweredQuestionCount int
	LatestHighlightAt       time.Time
}
