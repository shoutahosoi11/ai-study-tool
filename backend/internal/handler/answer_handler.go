package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/shout/ai-study-tool/backend/internal/handler/dto"
	"github.com/shout/ai-study-tool/backend/internal/usecase"
)

type AnswerHandler struct {
	answerUsecase *usecase.AnswerUsecase
	db            *sql.DB
}

func NewAnswerHandler(au *usecase.AnswerUsecase, db *sql.DB) *AnswerHandler {
	return &AnswerHandler{answerUsecase: au, db: db}
}

func (h *AnswerHandler) SubmitAnswer(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}

	questionID := r.PathValue("id")
	if questionID == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "question id is required")
		return
	}

	var req dto.SubmitAnswerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	if req.UserAnswer == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "user_answer is required")
		return
	}

	userPlan := getUserPlan(r.Context(), h.db, userID)

	input := usecase.SubmitAnswerInput{
		UserID:     userID,
		QuestionID: questionID,
		UserAnswer: req.UserAnswer,
		UserPlan:   userPlan,
	}

	result, err := h.answerUsecase.SubmitAnswer(r.Context(), input)
	if err != nil {
		errMsg := err.Error()
		if strings.HasPrefix(errMsg, "not_found:") {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "question not found")
			return
		}
		if strings.HasPrefix(errMsg, "llm_error:") {
			writeError(w, http.StatusBadGateway, "GEMINI_ERROR", errMsg)
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", errMsg)
		return
	}

	writeJSON(w, http.StatusOK, dto.SubmitAnswerResponse{
		IsCorrect:     result.IsCorrect,
		CorrectAnswer: result.CorrectAnswer,
		Explanation:   result.Explanation,
		Score:         result.Score,
		Feedback:      result.Feedback,
	})
}
