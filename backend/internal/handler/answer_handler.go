package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/shout/ai-study-tool/backend/internal/domain"
	"github.com/shout/ai-study-tool/backend/internal/handler/dto"
	"github.com/shout/ai-study-tool/backend/internal/middleware"
	"github.com/shout/ai-study-tool/backend/internal/usecase"
)

type AnswerHandler struct {
	answerUsecase *usecase.AnswerUsecase
	userUsecase   usecase.UserUsecaseInterface
}

func NewAnswerHandler(au *usecase.AnswerUsecase, userUsecase usecase.UserUsecaseInterface) *AnswerHandler {
	return &AnswerHandler{answerUsecase: au, userUsecase: userUsecase}
}

func (h *AnswerHandler) SubmitAnswer(c echo.Context) error {
	firebaseUID, ok := middleware.GetFirebaseUID(c)
	if !ok {
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

	req := new(dto.SubmitAnswerRequest)
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.UserAnswer == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "user_answer is required")
	}

	input := usecase.SubmitAnswerInput{
		UserID:     user.ID.String(),
		QuestionID: questionID,
		UserAnswer: req.UserAnswer,
		UserPlan:   user.Plan,
	}

	result, err := h.answerUsecase.SubmitAnswer(c.Request().Context(), input)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "question not found")
		}
		if strings.HasPrefix(err.Error(), "llm_error:") {
			return echo.NewHTTPError(http.StatusBadGateway, "AI grading failed")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, dto.SubmitAnswerResponse{
		IsCorrect:     result.IsCorrect,
		CorrectAnswer: result.CorrectAnswer,
		Explanation:   result.Explanation,
		Score:         result.Score,
		Feedback:      result.Feedback,
	})
}
