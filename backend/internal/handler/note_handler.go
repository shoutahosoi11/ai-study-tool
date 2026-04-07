package handler

import (
	"net/http"

	"github.com/shout/ai-study-tool/backend/internal/usecase"
)

type NoteHandler struct {
	noteUsecase *usecase.NoteUsecase
}

func NewNoteHandler(nu *usecase.NoteUsecase) *NoteHandler {
	return &NoteHandler{noteUsecase: nu}
}

func (h *NoteHandler) UploadNote(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "failed to parse form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "file is required")
		return
	}
	defer file.Close()

	title := r.FormValue("title")
	if title == "" {
		title = header.Filename
	}

	note, err := h.noteUsecase.UploadNote(r.Context(), userID, title, file, header.Header.Get("Content-Type"))
	if err != nil {
		writeError(w, http.StatusBadGateway, "S3_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, note)
}
