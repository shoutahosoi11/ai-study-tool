package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/shout/ai-study-tool/backend/internal/domain"
	"github.com/shout/ai-study-tool/backend/internal/middleware"
	"github.com/shout/ai-study-tool/backend/internal/usecase"
)

type PostHandler struct {
	postUsecase *usecase.PostUsecase
	userUsecase usecase.UserUsecaseInterface
}

func NewPostHandler(postUsecase *usecase.PostUsecase, userUsecase usecase.UserUsecaseInterface) *PostHandler {
	return &PostHandler{postUsecase: postUsecase, userUsecase: userUsecase}
}

func (h *PostHandler) currentUser(c echo.Context) (*domain.User, error) {
	firebaseUID, ok := middleware.GetFirebaseUID(c)
	if !ok {
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	user, err := h.userUsecase.GetByFirebaseUID(c.Request().Context(), firebaseUID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, echo.NewHTTPError(http.StatusNotFound, "user not found")
		}
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return user, nil
}

func (h *PostHandler) GetTimeline(c echo.Context) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
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
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid post id")
	}

	if err := h.postUsecase.EnsureVisible(c.Request().Context(), user.ID, id); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "post not found")
	}

	post, err := h.postUsecase.GetByID(c.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "post not found")
	}

	return c.JSON(http.StatusOK, post)
}

type CreatePostRequest struct {
	QuestionID    *string                     `json:"question_id"`
	BookID        *string                     `json:"book_id"`
	FieldID       *string                     `json:"field_id"`
	Body          string                      `json:"body"`
	BookTitle     string                      `json:"book_title"`
	QuestionCount int                         `json:"question_count"`
	Questions     []CreatePostQuestionRequest `json:"questions"`
	Type          string                      `json:"type"`
}

type CreatePostQuestionRequest struct {
	QuestionID string `json:"question_id"`
	SortOrder  int    `json:"sort_order"`
	Note       string `json:"note"`
}

type PostQuestionResponse struct {
	ID            string   `json:"id"`
	QuestionType  string   `json:"question_type"`
	Content       string   `json:"content"`
	Options       []string `json:"options"`
	CorrectAnswer string   `json:"correct_answer"`
	Explanation   string   `json:"explanation"`
	Note          string   `json:"note"`
	SortOrder     int      `json:"sort_order"`
}

func toPostQuestionResponse(question *domain.PostedQuestion) PostQuestionResponse {
	return PostQuestionResponse{
		ID:            question.ID,
		QuestionType:  string(question.QuestionType),
		Content:       question.Content,
		Options:       question.Options,
		CorrectAnswer: question.CorrectAnswer,
		Explanation:   question.Explanation,
		Note:          question.Note,
		SortOrder:     question.SortOrder,
	}
}

func (h *PostHandler) CreatePost(c echo.Context) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}

	req := new(CreatePostRequest)
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	input := domain.CreatePostInput{
		UserID:        user.ID,
		Type:          strings.TrimSpace(req.Type),
		Body:          req.Body,
		BookTitle:     req.BookTitle,
		QuestionCount: req.QuestionCount,
	}

	if req.QuestionID != nil {
		id, err := uuid.Parse(*req.QuestionID)
		if err == nil {
			input.QuestionID = &id
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
	for index, question := range req.Questions {
		id, err := uuid.Parse(question.QuestionID)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid question id")
		}
		sortOrder := question.SortOrder
		if sortOrder < 0 {
			sortOrder = index
		}
		input.Questions = append(input.Questions, domain.PostQuestionItem{
			QuestionID: id,
			SortOrder:  sortOrder,
			Note:       question.Note,
		})
	}

	post, err := h.postUsecase.CreatePost(c.Request().Context(), input)
	if err != nil {
		if strings.HasPrefix(err.Error(), "validation:") {
			return echo.NewHTTPError(http.StatusBadRequest, strings.TrimPrefix(err.Error(), "validation: "))
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusCreated, post)
}

func (h *PostHandler) ListQuestions(c echo.Context) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}

	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid post id")
	}

	if err := h.postUsecase.EnsureVisible(c.Request().Context(), user.ID, postID); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "post not found")
	}

	questions, err := h.postUsecase.ListQuestionsByPostID(c.Request().Context(), postID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "post not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	responses := make([]PostQuestionResponse, 0, len(questions))
	for _, question := range questions {
		responses = append(responses, toPostQuestionResponse(question))
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"questions": responses,
	})
}
