package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/shout/ai-study-tool/backend/internal/domain"
	"github.com/shout/ai-study-tool/backend/internal/handler/dto"
	"github.com/shout/ai-study-tool/backend/internal/usecase"
)

type QuestionHandler struct {
	questionUsecase     QuestionUsecase
	questionSyncUsecase QuestionSyncUsecase
	userUsecase         usecase.UserUsecaseInterface
}

type QuestionUsecase interface {
	ListQuestions(ctx context.Context, creatorID string, limit int) ([]*domain.Question, error)
	ListSavedQuestions(ctx context.Context, userID string, limit int) ([]*domain.SavedQuestion, error)
	ListIncorrectQuestions(ctx context.Context, userID string, limit int) ([]*domain.IncorrectQuestion, error)
	GenerateQuestions(ctx context.Context, input domain.GenerateQuestionsInput) ([]*domain.Question, error)
	ListPreparedQuestions(ctx context.Context, input domain.GenerateQuestionsInput) ([]*domain.Question, error)
	SaveQuestion(ctx context.Context, userID string, questionID string, note string) error
	GradeAnswer(ctx context.Context, input domain.GradeInput, userPlan string) (*domain.GradeResult, error)
}

type QuestionSyncUsecase interface {
	SyncQuestionStock(ctx context.Context, user *domain.User) (*usecase.SyncQuestionStockResult, error)
}

func NewQuestionHandler(
	qu QuestionUsecase,
	questionSyncUsecase QuestionSyncUsecase,
	userUsecase usecase.UserUsecaseInterface,
) *QuestionHandler {
	return &QuestionHandler{
		questionUsecase:     qu,
		questionSyncUsecase: questionSyncUsecase,
		userUsecase:         userUsecase,
	}
}

func (h *QuestionHandler) List(c echo.Context) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}

	questions, err := h.questionUsecase.ListQuestions(c.Request().Context(), user.ID.String(), 50)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	responses := make([]dto.QuestionResponse, 0, len(questions))
	for _, q := range questions {
		responses = append(responses, dto.ToQuestionResponse(q))
	}

	return c.JSON(http.StatusOK, responses)
}

func (h *QuestionHandler) ListSaved(c echo.Context) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}

	savedQuestions, err := h.questionUsecase.ListSavedQuestions(c.Request().Context(), user.ID.String(), 100)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	responses := make([]dto.SavedQuestionResponse, 0, len(savedQuestions))
	for _, q := range savedQuestions {
		responses = append(responses, dto.ToSavedQuestionResponse(q))
	}

	return c.JSON(http.StatusOK, responses)
}

func (h *QuestionHandler) ListIncorrect(c echo.Context) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}

	incorrectQuestions, err := h.questionUsecase.ListIncorrectQuestions(c.Request().Context(), user.ID.String(), 100)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	responses := make([]dto.IncorrectQuestionResponse, 0, len(incorrectQuestions))
	for _, q := range incorrectQuestions {
		responses = append(responses, dto.ToIncorrectQuestionResponse(q))
	}

	return c.JSON(http.StatusOK, responses)
}

func (h *QuestionHandler) GenerateQuestions(c echo.Context) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}

	req := new(dto.GenerateQuestionRequest)
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if strings.TrimSpace(req.SourceID) == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "source_id is required")
	}
	if err := validateQuestionSourceID(domain.SourceType(req.SourceType), req.SourceID); err != nil {
		if errors.Is(err, domain.ErrInvalidSourceType) {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid source type")
		}
		return echo.NewHTTPError(http.StatusBadRequest, "invalid source id")
	}

	input := domain.GenerateQuestionsInput{
		CreatorID:         user.ID.String(),
		SourceType:        domain.SourceType(req.SourceType),
		SourceID:          req.SourceID,
		BookTitle:         req.BookTitle,
		BookAuthor:        req.BookAuthor,
		QuestionCount:     req.QuestionCount,
		QuestionType:      questionTypeOrDefault(req.QuestionType),
		CustomInstruction: req.CustomInstruction,
		UserPlan:          user.Plan,
	}

	questions, err := h.questionUsecase.GenerateQuestions(c.Request().Context(), input)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidSourceType) {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid source type")
		}
		if errors.Is(err, domain.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "source not found")
		}
		if errors.Is(err, domain.ErrSourceTextUnavailable) {
			return echo.NewHTTPError(http.StatusUnprocessableEntity, "source text is unavailable")
		}
		return echo.NewHTTPError(http.StatusBadGateway, err.Error())
	}

	responses := make([]dto.QuestionResponse, 0, len(questions))
	for _, q := range questions {
		responses = append(responses, dto.ToQuestionResponse(q))
	}

	return c.JSON(http.StatusCreated, responses)
}

func (h *QuestionHandler) ListPrepared(c echo.Context) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}

	sourceType := domain.SourceType(strings.TrimSpace(c.QueryParam("source_type")))
	sourceID := strings.TrimSpace(c.QueryParam("source_id"))
	if sourceID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "source_id is required")
	}
	if err := validateQuestionSourceID(sourceType, sourceID); err != nil {
		if errors.Is(err, domain.ErrInvalidSourceType) {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid source type")
		}
		return echo.NewHTTPError(http.StatusBadRequest, "invalid source id")
	}

	questionCount := 0
	if rawLimit := strings.TrimSpace(c.QueryParam("question_count")); rawLimit != "" {
		if parsedLimit, parseErr := strconv.Atoi(rawLimit); parseErr == nil {
			questionCount = parsedLimit
		}
	}

	input := domain.GenerateQuestionsInput{
		CreatorID:     user.ID.String(),
		SourceType:    sourceType,
		SourceID:      sourceID,
		BookTitle:     strings.TrimSpace(c.QueryParam("book_title")),
		BookAuthor:    strings.TrimSpace(c.QueryParam("book_author")),
		QuestionCount: questionCount,
	}

	questions, err := h.questionUsecase.ListPreparedQuestions(c.Request().Context(), input)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidSourceType) {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid source type")
		}
		if errors.Is(err, domain.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "source not found")
		}
		if errors.Is(err, domain.ErrQuestionsPreparing) {
			return echo.NewHTTPError(http.StatusConflict, "questions are still preparing")
		}
		if errors.Is(err, domain.ErrQuestionGenerationFailed) {
			return echo.NewHTTPError(http.StatusConflict, "question generation failed")
		}
		if errors.Is(err, domain.ErrSourceTextUnavailable) {
			return echo.NewHTTPError(http.StatusUnprocessableEntity, "source text is unavailable")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	responses := make([]dto.QuestionResponse, 0, len(questions))
	for _, q := range questions {
		responses = append(responses, dto.ToQuestionResponse(q))
	}

	return c.JSON(http.StatusOK, responses)
}

func (h *QuestionHandler) SyncStock(c echo.Context) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}

	result, err := h.questionSyncUsecase.SyncQuestionStock(c.Request().Context(), user)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	response := dto.SyncQuestionStockResponse{
		Books:                  make([]dto.SyncQuestionStockBookResponse, 0, len(result.Books)),
		QueuedCount:            result.QueuedCount,
		SkippedDueToDailyLimit: result.SkippedDueToDailyLimit,
	}
	for _, book := range result.Books {
		response.Books = append(response.Books, dto.SyncQuestionStockBookResponse{
			BookKey:    book.BookKey,
			BookTitle:  book.BookTitle,
			BookAuthor: book.BookAuthor,
			Stock:      book.Stock,
			Target:     book.Target,
			Preparing:  book.Preparing,
		})
	}

	return c.JSON(http.StatusOK, response)
}

func (h *QuestionHandler) SaveQuestion(c echo.Context) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}

	questionID := strings.TrimSpace(c.Param("id"))
	if questionID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "question id is required")
	}

	req := new(dto.SaveQuestionRequest)
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if err := h.questionUsecase.SaveQuestion(c.Request().Context(), user.ID.String(), questionID, req.Note); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "question not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, dto.SaveQuestionResponse{
		QuestionID: questionID,
		Note:       strings.TrimSpace(req.Note),
		Saved:      true,
	})
}

func questionTypeOrDefault(questionType string) domain.QuestionType {
	switch domain.QuestionType(strings.TrimSpace(questionType)) {
	case domain.QuestionTypeDescriptive:
		return domain.QuestionTypeDescriptive
	case domain.QuestionTypeMultipleChoice:
		return domain.QuestionTypeMultipleChoice
	default:
		return domain.QuestionTypeMultipleChoice
	}
}

func validateQuestionSourceID(sourceType domain.SourceType, sourceID string) error {
	switch sourceType {
	case domain.SourceTypeKindleBook:
		if strings.TrimSpace(sourceID) == "" {
			return domain.ErrInvalidSourceType
		}
		return nil
	default:
		return domain.ErrInvalidSourceType
	}
}

func (h *QuestionHandler) GradeAnswer(c echo.Context) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}

	questionID := c.Param("id")
	if questionID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "question id is required")
	}

	req := new(dto.GradeAnswerRequest)
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	gradeInput := domain.GradeInput{
		QuestionID: questionID,
		UserAnswer: req.UserAnswer,
	}

	result, err := h.questionUsecase.GradeAnswer(c.Request().Context(), gradeInput, user.Plan)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "question not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, dto.GradeAnswerResponse{
		IsCorrect: result.IsCorrect,
		Score:     result.Score,
		Feedback:  result.Feedback,
	})
}

func (h *QuestionHandler) currentUser(c echo.Context) (*domain.User, error) {
	return resolveCurrentUser(c, h.userUsecase, "question")
}
