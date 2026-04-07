package domain

type QuestionType string
type SourceType string

const (
	QuestionTypeMultipleChoice QuestionType = "multiple_choice"
	QuestionTypeDescriptive    QuestionType = "descriptive"

	SourceTypeHighlight SourceType = "highlight"
	SourceTypeNote      SourceType = "note"
	SourceTypeManual    SourceType = "manual"
)

type Question struct {
	ID            string
	QuestionType  QuestionType
	Content       string
	Options       []string
	CorrectAnswer string
	Explanation   string
}

func (q *Question) IsCorrect(answer string) bool {
	return q.CorrectAnswer == answer
}
