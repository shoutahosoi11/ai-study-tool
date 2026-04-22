package dto

import (
	"time"

	"github.com/shout/ai-study-tool/backend/internal/domain"
)

type GenerateQuestionRequest struct {
	SourceType        string `json:"source_type"`
	SourceID          string `json:"source_id"`
	BookTitle         string `json:"book_title"`
	BookAuthor        string `json:"book_author"`
	QuestionCount     int    `json:"question_count"`
	QuestionType      string `json:"question_type"`
	CustomInstruction string `json:"custom_instruction"`
}

type SaveQuestionRequest struct {
	Note string `json:"note"`
}

type SaveQuestionResponse struct {
	QuestionID string `json:"question_id"`
	Note       string `json:"note"`
	Saved      bool   `json:"saved"`
}

type SavedQuestionResponse struct {
	ID            string    `json:"id"`
	QuestionType  string    `json:"question_type"`
	Content       string    `json:"content"`
	Options       []string  `json:"options"`
	CorrectAnswer string    `json:"correct_answer"`
	Explanation   string    `json:"explanation"`
	Note          string    `json:"note"`
	SavedAt       time.Time `json:"saved_at"`
}

type IncorrectQuestionResponse struct {
	ID            string    `json:"id"`
	QuestionType  string    `json:"question_type"`
	Content       string    `json:"content"`
	Options       []string  `json:"options"`
	CorrectAnswer string    `json:"correct_answer"`
	Explanation   string    `json:"explanation"`
	Note          string    `json:"note"`
	AnsweredAt    time.Time `json:"answered_at"`
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

func ToSavedQuestionResponse(q *domain.SavedQuestion) SavedQuestionResponse {
	return SavedQuestionResponse{
		ID:            q.ID,
		QuestionType:  string(q.QuestionType),
		Content:       q.Content,
		Options:       q.Options,
		CorrectAnswer: q.CorrectAnswer,
		Explanation:   q.Explanation,
		Note:          q.Note,
		SavedAt:       q.SavedAt,
	}
}

func ToIncorrectQuestionResponse(q *domain.IncorrectQuestion) IncorrectQuestionResponse {
	return IncorrectQuestionResponse{
		ID:            q.ID,
		QuestionType:  string(q.QuestionType),
		Content:       q.Content,
		Options:       q.Options,
		CorrectAnswer: q.CorrectAnswer,
		Explanation:   q.Explanation,
		Note:          q.Note,
		AnsweredAt:    q.AnsweredAt,
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
