package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/shout/ai-study-tool/backend/internal/domain"
	"github.com/shout/ai-study-tool/backend/internal/usecase"
)

type NoteHandler struct {
	noteUsecase *usecase.NoteUsecase
	userUsecase *usecase.UserUsecase
}

func NewNoteHandler(nu *usecase.NoteUsecase, userUsecase *usecase.UserUsecase) *NoteHandler {
	return &NoteHandler{noteUsecase: nu, userUsecase: userUsecase}
}

func (h *NoteHandler) UploadNote(c echo.Context) error {
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

	if err := c.Request().ParseMultipartForm(32 << 20); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "failed to parse form")
	}

	file, header, err := c.Request().FormFile("file")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "file is required")
	}
	defer file.Close()

	title := c.FormValue("title")
	if title == "" {
		title = header.Filename
	}

	note, err := h.noteUsecase.UploadNote(c.Request().Context(), user.ID.String(), title, file, header.Header.Get("Content-Type"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadGateway, "file upload failed")
	}

	return c.JSON(http.StatusCreated, note)
}
