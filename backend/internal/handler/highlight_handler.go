package handler

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/shout/ai-study-tool/backend/internal/domain"
	"github.com/shout/ai-study-tool/backend/internal/handler/dto"
	"github.com/shout/ai-study-tool/backend/internal/usecase"
)

type HighlightHandler struct {
	highlightUsecase *usecase.HighlightUsecase
	userUsecase      *usecase.UserUsecase
}

func NewHighlightHandler(highlightUsecase *usecase.HighlightUsecase, userUsecase *usecase.UserUsecase) *HighlightHandler {
	return &HighlightHandler{
		highlightUsecase: highlightUsecase,
		userUsecase:      userUsecase,
	}
}

func (h *HighlightHandler) Create(c echo.Context) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}

	req := new(dto.CreateHighlightRequest)
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if strings.TrimSpace(req.Content) == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "content is required")
	}

	input := usecase.CreateHighlightInput{
		BookTitle:     req.BookTitle,
		BookAuthor:    req.BookAuthor,
		ASIN:          req.ASIN,
		Content:       req.Content,
		Location:      req.Location,
		HighlightedAt: req.HighlightedAt,
		Source:        req.Source,
	}

	if req.BookID != nil {
		bookID, err := uuid.Parse(*req.BookID)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid book id")
		}
		input.BookID = &bookID
	}

	highlight, err := h.highlightUsecase.Create(c.Request().Context(), user.ID, input)
	if err != nil {
		log.Printf("highlight create error: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusCreated, toHighlightResponse(highlight))
}

func (h *HighlightHandler) List(c echo.Context) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}

	page, _ := strconv.Atoi(c.QueryParam("page"))
	limit, _ := strconv.Atoi(c.QueryParam("limit"))

	highlights, total, err := h.highlightUsecase.List(c.Request().Context(), user.ID, page, limit)
	if err != nil {
		log.Printf("highlight list error: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	responses := make([]*dto.HighlightResponse, 0, len(highlights))
	for _, highlight := range highlights {
		responses = append(responses, toHighlightResponse(highlight))
	}

	return c.JSON(http.StatusOK, dto.ListHighlightsResponse{
		Highlights: responses,
		Total:      total,
		Page:       page,
		Limit:      limit,
	})
}

func (h *HighlightHandler) GetByID(c echo.Context) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid highlight id")
	}

	highlight, err := h.highlightUsecase.GetByID(c.Request().Context(), id, user.ID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "highlight not found")
		}
		log.Printf("highlight get by id error: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, toHighlightResponse(highlight))
}

func (h *HighlightHandler) Delete(c echo.Context) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid highlight id")
	}

	if err := h.highlightUsecase.Delete(c.Request().Context(), id, user.ID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "highlight not found")
		}
		log.Printf("highlight delete error: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *HighlightHandler) currentUser(c echo.Context) (*domain.User, error) {
	firebaseUID, ok := c.Get("firebase_uid").(string)
	if !ok || firebaseUID == "" {
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	user, err := h.userUsecase.GetByFirebaseUID(c.Request().Context(), firebaseUID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, echo.NewHTTPError(http.StatusNotFound, "user not found")
		}
		log.Printf("currentUser error: %v", err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return user, nil
}

func toHighlightResponse(h *domain.Highlight) *dto.HighlightResponse {
	resp := &dto.HighlightResponse{
		ID:            h.ID.String(),
		BookTitle:     h.BookTitle,
		BookAuthor:    h.BookAuthor,
		ASIN:          h.ASIN,
		Content:       h.Content,
		Location:      h.Location,
		HighlightedAt: h.HighlightedAt,
		Source:        h.Source,
		CreatedAt:     h.CreatedAt,
	}

	if h.BookID != nil {
		bookID := h.BookID.String()
		resp.BookID = &bookID
	}

	return resp
}
