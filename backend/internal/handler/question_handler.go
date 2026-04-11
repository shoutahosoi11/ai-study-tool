package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/shout/ai-study-tool/backend/internal/domain"
	"github.com/shout/ai-study-tool/backend/internal/handler/dto"
	"github.com/shout/ai-study-tool/backend/internal/usecase"
)

type QuestionHandler struct {
	questionUsecase *usecase.QuestionUsecase
	userUsecase     *usecase.UserUsecase
}

func NewQuestionHandler(qu *usecase.QuestionUsecase, userUsecase *usecase.UserUsecase) *QuestionHandler {
	return &QuestionHandler{questionUsecase: qu, userUsecase: userUsecase}
}

func (h *QuestionHandler) GenerateQuestions(c echo.Context) error {
	firebaseUID, ok := c.Get("firebase_uid").(string)
	if !ok || firebaseUID == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	user, err := h.userUsecase.GetByFirebaseUID(c.Request().Context(), firebaseUID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "user not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	req := new(dto.GenerateQuestionRequest)
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.SourceText == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "source_text is required")
	}

	input := domain.GenerateQuestionsInput{
		CreatorID:         user.ID.String(),
		SourceType:        domain.SourceType(req.SourceType),
		SourceID:          req.SourceID,
		SourceText:        req.SourceText,
		QuestionType:      domain.QuestionType(req.QuestionType),
		CustomInstruction: req.CustomInstruction,
		UserPlan:          user.Plan,
	}

	questions, err := h.questionUsecase.GenerateQuestions(c.Request().Context(), input)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadGateway, err.Error())
	}

	responses := make([]dto.QuestionResponse, 0, len(questions))
	for _, q := range questions {
		responses = append(responses, dto.ToQuestionResponse(q))
	}

	return c.JSON(http.StatusCreated, responses)
}

func (h *QuestionHandler) GradeAnswer(c echo.Context) error {
	firebaseUID, ok := c.Get("firebase_uid").(string)
	if !ok || firebaseUID == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	user, err := h.userUsecase.GetByFirebaseUID(c.Request().Context(), firebaseUID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "user not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	questionID := c.Param("id")
	if questionID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "question id is required")
	}

	req := new(dto.GradeAnswerRequest)
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	gradeInput := domain.GradeInput{
		QuestionID: questionID,
		UserAnswer: req.UserAnswer,
	}

	result, err := h.questionUsecase.GradeAnswer(c.Request().Context(), gradeInput, user.Plan)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "question not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, dto.GradeAnswerResponse{
		IsCorrect: result.IsCorrect,
		Score:     result.Score,
		Feedback:  result.Feedback,
	})
}
