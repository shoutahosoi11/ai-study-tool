package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

type QuestionGenerationTaskUsecase interface {
	ProcessQuestionGenerationJob(ctx context.Context, jobID uuid.UUID, userID uuid.UUID) error
}

type TaskHandler struct {
	questionWorker QuestionGenerationTaskUsecase
}

type QuestionGenerationTaskRequest struct {
	JobID  string `json:"job_id"`
	UserID string `json:"user_id"`
}

func NewTaskHandler(questionWorker QuestionGenerationTaskUsecase) *TaskHandler {
	return &TaskHandler{questionWorker: questionWorker}
}

func (h *TaskHandler) HandleQuestionGeneration(c echo.Context) error {
	req := new(QuestionGenerationTaskRequest)
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	jobID, err := uuid.Parse(req.JobID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid job_id")
	}
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid user_id")
	}
	if h.questionWorker == nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "question worker is not configured")
	}

	if err := h.questionWorker.ProcessQuestionGenerationJob(c.Request().Context(), jobID, userID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "job not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusAccepted, map[string]string{"status": "accepted"})
}
