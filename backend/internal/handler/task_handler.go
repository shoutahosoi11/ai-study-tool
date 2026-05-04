package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type TaskHandler struct{}

func NewTaskHandler() *TaskHandler {
	return &TaskHandler{}
}

func (h *TaskHandler) HandleQuestionGeneration(c echo.Context) error {
	return c.JSON(http.StatusAccepted, map[string]string{"status": "accepted"})
}
