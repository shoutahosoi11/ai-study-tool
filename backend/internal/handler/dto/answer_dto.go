package dto

type SubmitAnswerRequest struct {
	UserAnswer string `json:"user_answer"`
}

type SubmitAnswerResponse struct {
	IsCorrect     bool    `json:"is_correct"`
	CorrectAnswer string  `json:"correct_answer"`
	Explanation   string  `json:"explanation"`
	Score         *int    `json:"score,omitempty"`
	Feedback      *string `json:"feedback,omitempty"`
}
