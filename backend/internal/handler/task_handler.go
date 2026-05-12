package handler

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

type TaskHandler struct {
	questionWorker QuestionWorker
	importJobs     HighlightImportJobs
}

type QuestionWorker interface {
	ProcessQuestionGenerationJob(ctx context.Context, jobID uuid.UUID, userID uuid.UUID) error
}

type HighlightImportJobs interface {
	ProcessSingle(ctx context.Context, queueID uuid.UUID, userID uuid.UUID) error
}

func NewTaskHandler(questionWorker QuestionWorker, importJobs HighlightImportJobs) *TaskHandler {
	return &TaskHandler{questionWorker: questionWorker, importJobs: importJobs}
}

type questionGenerationTaskRequest struct {
	JobID  string `json:"job_id"`
	UserID string `json:"user_id"`
}

func (h *TaskHandler) HandleQuestionGeneration(c echo.Context) error {
	if h == nil || h.questionWorker == nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "task handler is not configured")
	}

	var req questionGenerationTaskRequest
	if err := c.Bind(&req); err != nil {
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

	if err := h.questionWorker.ProcessQuestionGenerationJob(c.Request().Context(), jobID, userID); err != nil {
		log.Printf("task handler: question generation failed job_id=%s user_id=%s err=%v", jobID, userID, err)
		return echo.NewHTTPError(http.StatusInternalServerError, "question generation task failed")
	}
	return c.NoContent(http.StatusOK)
}

type highlightImportTaskRequest struct {
	QueueID string `json:"queue_id"`
	UserID  string `json:"user_id"`
}

func (h *TaskHandler) HandleHighlightImport(c echo.Context) error {
	if h == nil || h.importJobs == nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "task handler is not configured")
	}

	var req highlightImportTaskRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	queueID, err := uuid.Parse(req.QueueID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid queue_id")
	}
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid user_id")
	}

	if err := h.importJobs.ProcessSingle(c.Request().Context(), queueID, userID); err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			return echo.NewHTTPError(http.StatusForbidden, "highlight import task user mismatch")
		}
		log.Printf("task handler: highlight import failed queue_id=%s user_id=%s err=%v", queueID, req.UserID, err)
		return echo.NewHTTPError(http.StatusInternalServerError, "highlight import task failed")
	}
	return c.NoContent(http.StatusOK)
}
