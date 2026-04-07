package dto

import "github.com/shout/ai-study-tool/backend/internal/domain"

type GenerateQuestionRequest struct {
	SourceType        string `json:"source_type"`
	SourceID          string `json:"source_id"`
	SourceText        string `json:"source_text"`
	QuestionType      string `json:"question_type"`
	CustomInstruction string `json:"custom_instruction"`
}

type QuestionResponse struct {
	ID            string   `json:"id"`
	QuestionType  string   `json:"question_type"`
	Content       string   `json:"content"`
	Options       []string `json:"options"`
	CorrectAnswer string   `json:"correct_answer"`
	Explanation   string   `json:"explanation"`
}

func ToQuestionResponse(q *domain.Question) QuestionResponse {
	return QuestionResponse{
		ID:            q.ID,
		QuestionType:  string(q.QuestionType),
		Content:       q.Content,
		Options:       q.Options,
		CorrectAnswer: q.CorrectAnswer,
		Explanation:   q.Explanation,
	}
}

type GradeAnswerRequest struct {
	UserAnswer string `json:"user_answer"`
}

type GradeAnswerResponse struct {
	IsCorrect bool   `json:"is_correct"`
	Score     int    `json:"score"`
	Feedback  string `json:"feedback"`
}
