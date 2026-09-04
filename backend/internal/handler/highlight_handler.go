package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/shout/ai-study-tool/backend/internal/domain"
	"github.com/shout/ai-study-tool/backend/internal/handler/dto"
	"github.com/shout/ai-study-tool/backend/internal/usecase"
)

type HighlightHandler struct {
	importUsecase HighlightImportUsecase
	queryUsecase  HighlightQueryUsecase
	userUsecase   usecase.UserUsecaseInterface
}

type HighlightImportUsecase interface {
	ImportKindleHighlights(ctx context.Context, userID uuid.UUID, items []usecase.ImportHighlightItem) (*usecase.ImportKindleResult, error)
	ImportSharedHighlight(ctx context.Context, userID uuid.UUID, input usecase.ImportSharedHighlightInput) (*usecase.ImportSharedHighlightResult, error)
	ImportPastedHighlight(ctx context.Context, userID uuid.UUID, input usecase.ImportPastedHighlightInput) (*usecase.ImportPastedHighlightResult, error)
}

type HighlightQueryUsecase interface {
	ListExistingContentHashes(ctx context.Context, userID uuid.UUID, hashes []string) ([]string, error)
	ListKindleBooks(ctx context.Context, userID uuid.UUID) ([]*domain.KindleBook, error)
	ListByASIN(ctx context.Context, userID uuid.UUID, asin string) ([]*domain.Highlight, error)
	ListByBookMetadata(ctx context.Context, userID uuid.UUID, bookTitle, bookAuthor string) ([]*domain.Highlight, error)
	UpdateExplanation(ctx context.Context, id, userID uuid.UUID, explanation string) (*domain.Highlight, error)
}

func NewHighlightHandler(importUsecase HighlightImportUsecase, queryUsecase HighlightQueryUsecase, userUsecase usecase.UserUsecaseInterface) *HighlightHandler {
	return &HighlightHandler{
		importUsecase: importUsecase,
		queryUsecase:  queryUsecase,
		userUsecase:   userUsecase,
	}
}

func (h *HighlightHandler) Import(c echo.Context) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}

	req := new(dto.ImportHighlightsRequest)
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	items := make([]usecase.ImportHighlightItem, 0, len(req.Highlights))
	for _, item := range req.Highlights {
		items = append(items, usecase.ImportHighlightItem{
			ASIN:          item.ASIN,
			BookTitle:     item.BookTitle,
			BookAuthor:    item.BookAuthor,
			Content:       item.Content,
			Location:      item.Location,
			HighlightedAt: item.HighlightedAt,
		})
	}

	result, err := h.importUsecase.ImportKindleHighlights(c.Request().Context(), user.ID, items)
	if err != nil {
		if httpErr, ok := asValidationHTTPError(err); ok {
			return httpErr
		}
		if errors.Is(err, domain.ErrInvalidInput) {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid input")
		}
		if errors.Is(err, domain.ErrAllCopyProtected) {
			return echo.NewHTTPError(http.StatusUnprocessableEntity, "コピー制限によりハイライトを取得できませんでした")
		}
		slog.Error("highlight_handler_error", "operation", "import", "user_id", user.ID.String(), "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	if result.QueuedCount > 0 {
		return c.JSON(http.StatusAccepted, dto.ImportHighlightsResponse{
			Queued:             true,
			QueueID:            result.QueueID.String(),
			QueuedCount:        result.QueuedCount,
			CopyProtectedCount: result.CopyProtectedCount,
			InvalidItemCount:   result.InvalidItemCount,
			Warning:            result.Warning,
		})
	}

	responses := make([]*dto.HighlightResponse, 0, len(result.Highlights))
	for _, highlight := range result.Highlights {
		responses = append(responses, toHighlightResponse(highlight))
	}

	return c.JSON(http.StatusOK, dto.ImportHighlightsResponse{
		Queued:             false,
		SavedCount:         result.Saved,
		DuplicateCount:     result.DuplicateCount,
		CopyProtectedCount: result.CopyProtectedCount,
		InvalidItemCount:   result.InvalidItemCount,
		ResolvedASIN:       result.ResolvedASIN,
		Highlights:         responses,
		Warning:            result.Warning,
	})
}

func (h *HighlightHandler) ImportExtension(c echo.Context) error {
	// Extension imports intentionally reuse the normal import path after
	// route-level scope and rate-limit checks, so downstream generation rules
	// remain backend-owned.
	return h.Import(c)
}

func (h *HighlightHandler) CheckExistingHashes(c echo.Context) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}

	req := new(dto.CheckHighlightHashesRequest)
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	existing, err := h.queryUsecase.ListExistingContentHashes(c.Request().Context(), user.ID, req.Hashes)
	if err != nil {
		if httpErr, ok := asValidationHTTPError(err); ok {
			return httpErr
		}
		if errors.Is(err, domain.ErrInvalidInput) {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid input")
		}
		slog.Error("highlight_handler_error", "operation", "check_existing_hashes", "user_id", user.ID.String(), "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, dto.CheckHighlightHashesResponse{
		ExistingHashes: existing,
	})
}

func (h *HighlightHandler) ImportShared(c echo.Context) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}

	req := new(dto.ImportSharedHighlightRequest)
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	result, err := h.importUsecase.ImportSharedHighlight(c.Request().Context(), user.ID, usecase.ImportSharedHighlightInput{
		BookTitle:  req.BookTitle,
		BookAuthor: req.BookAuthor,
		Content:    req.Content,
		SourceApp:  req.SourceApp,
		SourceURL:  req.SourceURL,
		SharedAt:   req.SharedAt,
	})
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid highlight input")
		}
		slog.Error("highlight_handler_error", "operation", "import_shared", "user_id", user.ID.String(), "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	var responseHighlight *dto.HighlightResponse
	if result.Highlight != nil {
		responseHighlight = toHighlightResponse(result.Highlight)
	}

	return c.JSON(http.StatusOK, dto.ImportSharedHighlightResponse{
		Saved:     result.Saved,
		Duplicate: result.Duplicate,
		Highlight: responseHighlight,
	})
}

func (h *HighlightHandler) ImportPaste(c echo.Context) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}

	req := new(dto.ImportPastedHighlightRequest)
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	result, err := h.importUsecase.ImportPastedHighlight(c.Request().Context(), user.ID, usecase.ImportPastedHighlightInput{
		BookTitle:  req.BookTitle,
		BookAuthor: req.BookAuthor,
		Content:    req.Content,
		SourceApp:  req.SourceApp,
		SourceURL:  req.SourceURL,
	})
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid highlight input")
		}
		slog.Error("highlight_handler_error", "operation", "import_paste", "user_id", user.ID.String(), "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	status := http.StatusCreated
	if result.Duplicate {
		status = http.StatusOK
	}

	return c.JSON(status, dto.ImportPastedHighlightResponse{
		ID:        result.ID.String(),
		Duplicate: result.Duplicate,
	})
}

func (h *HighlightHandler) ListBooks(c echo.Context) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}

	books, err := h.queryUsecase.ListKindleBooks(c.Request().Context(), user.ID)
	if err != nil {
		slog.Error("highlight_handler_error", "operation", "list_books", "user_id", user.ID.String(), "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	responses := make([]*dto.KindleBookResponse, 0, len(books))
	for _, book := range books {
		responses = append(responses, toKindleBookResponse(book))
	}

	return c.JSON(http.StatusOK, dto.ListKindleBooksResponse{
		Books: responses,
	})
}

func (h *HighlightHandler) ListByASIN(c echo.Context) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}

	asin := strings.TrimSpace(c.Param("asin"))
	if asin == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "asin is required")
	}

	highlights, err := h.queryUsecase.ListByASIN(c.Request().Context(), user.ID, asin)
	if err != nil {
		slog.Error("highlight_handler_error", "operation", "list_by_asin", "user_id", user.ID.String(), "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	responses := make([]*dto.HighlightResponse, 0, len(highlights))
	for _, highlight := range highlights {
		responses = append(responses, toHighlightResponse(highlight))
	}

	return c.JSON(http.StatusOK, dto.ListBookHighlightsResponse{
		Highlights: responses,
	})
}

func (h *HighlightHandler) ListByBookMetadata(c echo.Context) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}

	bookTitle := strings.TrimSpace(c.QueryParam("title"))
	if bookTitle == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "title is required")
	}

	bookAuthor := strings.TrimSpace(c.QueryParam("author"))
	highlights, err := h.queryUsecase.ListByBookMetadata(c.Request().Context(), user.ID, bookTitle, bookAuthor)
	if err != nil {
		slog.Error("highlight_handler_error", "operation", "list_by_book_metadata", "user_id", user.ID.String(), "has_author", bookAuthor != "", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	responses := make([]*dto.HighlightResponse, 0, len(highlights))
	for _, highlight := range highlights {
		responses = append(responses, toHighlightResponse(highlight))
	}

	return c.JSON(http.StatusOK, dto.ListBookHighlightsResponse{
		Highlights: responses,
	})
}

func (h *HighlightHandler) UpdateExplanation(c echo.Context) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid highlight id")
	}

	req := new(dto.UpdateHighlightExplanationRequest)
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	highlight, err := h.queryUsecase.UpdateExplanation(c.Request().Context(), id, user.ID, req.Explanation)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "highlight not found")
		}
		slog.Error("highlight_handler_error", "operation", "update_explanation", "user_id", user.ID.String(), "highlight_id", id.String(), "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, toHighlightResponse(highlight))
}

func (h *HighlightHandler) currentUser(c echo.Context) (*domain.User, error) {
	return resolveCurrentUser(c, h.userUsecase, "highlight")
}

func toHighlightResponse(h *domain.Highlight) *dto.HighlightResponse {
	resp := &dto.HighlightResponse{
		ID:             h.ID.String(),
		BookTitle:      h.BookTitle,
		BookAuthor:     h.BookAuthor,
		ASIN:           h.ASIN,
		Content:        h.Content,
		Explanation:    h.Explanation,
		Location:       h.Location,
		HighlightedAt:  h.HighlightedAt,
		Source:         h.Source,
		SourceApp:      h.SourceApp,
		SourceURL:      h.SourceURL,
		BookOrderIndex: h.BookOrderIndex,
		CreatedAt:      h.CreatedAt,
	}

	if h.BookID != nil {
		bookID := h.BookID.String()
		resp.BookID = &bookID
	}

	return resp
}

func toKindleBookResponse(book *domain.KindleBook) *dto.KindleBookResponse {
	return &dto.KindleBookResponse{
		ASIN:           book.ASIN,
		BookTitle:      book.BookTitle,
		BookAuthor:     book.BookAuthor,
		HighlightCount: book.HighlightCount,
		Source:         book.Source,
	}
}
