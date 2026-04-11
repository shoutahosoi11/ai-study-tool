package handler

import (
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/shout/ai-study-tool/backend/internal/domain"
	"github.com/shout/ai-study-tool/backend/internal/usecase"
)

type PostHandler struct {
	postUsecase *usecase.PostUsecase
	userUsecase *usecase.UserUsecase
}

func NewPostHandler(postUsecase *usecase.PostUsecase, userUsecase *usecase.UserUsecase) *PostHandler {
	return &PostHandler{postUsecase: postUsecase, userUsecase: userUsecase}
}

func (h *PostHandler) GetTimeline(c echo.Context) error {
	firebaseUID, ok := c.Get("firebase_uid").(string)
	if !ok || firebaseUID == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	user, err := h.userUsecase.GetByFirebaseUID(c.Request().Context(), firebaseUID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "user not found")
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))

	posts, err := h.postUsecase.GetTimeline(c.Request().Context(), user.ID, limit, offset)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if posts == nil {
		posts = []*domain.TimelinePost{}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"posts":  posts,
		"limit":  limit,
		"offset": offset,
	})
}

func (h *PostHandler) GetPost(c echo.Context) error {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid post id")
	}

	post, err := h.postUsecase.GetByID(c.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "post not found")
	}

	return c.JSON(http.StatusOK, post)
}

type CreatePostRequest struct {
	QuestionID *string `json:"question_id"`
	NoteID     *string `json:"note_id"`
	BookID     *string `json:"book_id"`
	FieldID    *string `json:"field_id"`
	Type       string  `json:"type"`
}

func (h *PostHandler) CreatePost(c echo.Context) error {
	firebaseUID, ok := c.Get("firebase_uid").(string)
	if !ok || firebaseUID == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	user, err := h.userUsecase.GetByFirebaseUID(c.Request().Context(), firebaseUID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "user not found")
	}

	req := new(CreatePostRequest)
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	input := domain.CreatePostInput{
		UserID: user.ID,
		Type:   req.Type,
	}

	if req.QuestionID != nil {
		id, err := uuid.Parse(*req.QuestionID)
		if err == nil {
			input.QuestionID = &id
		}
	}
	if req.NoteID != nil {
		id, err := uuid.Parse(*req.NoteID)
		if err == nil {
			input.NoteID = &id
		}
	}
	if req.BookID != nil {
		id, err := uuid.Parse(*req.BookID)
		if err == nil {
			input.BookID = &id
		}
	}
	if req.FieldID != nil {
		id, err := uuid.Parse(*req.FieldID)
		if err == nil {
			input.FieldID = &id
		}
	}

	post, err := h.postUsecase.CreatePost(c.Request().Context(), input)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusCreated, post)
}

