package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/shout/ai-study-tool/backend/internal/domain"
	"github.com/shout/ai-study-tool/backend/internal/handler/dto"
	"github.com/shout/ai-study-tool/backend/internal/usecase"
)

type QuestionHandler struct {
	questionUsecase *usecase.QuestionUsecase
	db              *sql.DB
}

func NewQuestionHandler(qu *usecase.QuestionUsecase, db *sql.DB) *QuestionHandler {
	return &QuestionHandler{questionUsecase: qu, db: db}
}

func (h *QuestionHandler) GenerateQuestions(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}

	userPlan := getUserPlan(r.Context(), h.db, userID)

	var req dto.GenerateQuestionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	if req.SourceText == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "source_text is required")
		return
	}

	input := domain.GenerateQuestionsInput{
		CreatorID:         userID,
		SourceType:        domain.SourceType(req.SourceType),
		SourceID:          req.SourceID,
		SourceText:        req.SourceText,
		QuestionType:      domain.QuestionType(req.QuestionType),
		CustomInstruction: req.CustomInstruction,
		UserPlan:          userPlan,
	}

	questions, err := h.questionUsecase.GenerateQuestions(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusBadGateway, "GEMINI_ERROR", err.Error())
		return
	}

	responses := make([]dto.QuestionResponse, 0, len(questions))
	for _, q := range questions {
		responses = append(responses, dto.ToQuestionResponse(q))
	}

	writeJSON(w, http.StatusCreated, responses)
}

func (h *QuestionHandler) GradeAnswer(w http.ResponseWriter, r *http.Request) {
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

	var req dto.GradeAnswerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	userPlan := getUserPlan(r.Context(), h.db, userID)

	gradeInput := domain.GradeInput{
		QuestionID: questionID,
		UserAnswer: req.UserAnswer,
	}

	result, err := h.questionUsecase.GradeAnswer(r.Context(), gradeInput, userPlan)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, dto.GradeAnswerResponse{
		IsCorrect: result.IsCorrect,
		Score:     result.Score,
		Feedback:  result.Feedback,
	})
}
