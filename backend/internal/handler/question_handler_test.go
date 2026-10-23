package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/shout/ai-study-tool/backend/internal/domain"
	"github.com/shout/ai-study-tool/backend/internal/middleware"
	"github.com/shout/ai-study-tool/backend/internal/usecase"
)

type stubQuestionUsecase struct {
	saveQuestion func(ctx context.Context, userID string, questionID string, note string) error
}

func (s *stubQuestionUsecase) ListQuestions(ctx context.Context, creatorID string, limit int) ([]*domain.Question, error) {
	return nil, nil
}

func (s *stubQuestionUsecase) ListSavedQuestions(ctx context.Context, userID string, limit int) ([]*domain.SavedQuestion, error) {
	return nil, nil
}

func (s *stubQuestionUsecase) ListIncorrectQuestions(ctx context.Context, userID string, limit int) ([]*domain.IncorrectQuestion, error) {
	return nil, nil
}

func (s *stubQuestionUsecase) ListPreparedQuestions(ctx context.Context, input domain.GenerateQuestionsInput) ([]*domain.Question, error) {
	return nil, nil
}

func (s *stubQuestionUsecase) SaveQuestion(ctx context.Context, userID string, questionID string, note string) error {
	if s.saveQuestion == nil {
		return nil
	}
	return s.saveQuestion(ctx, userID, questionID, note)
}

type stubQuestionSyncUsecase struct {
	syncQuestionStock       func(ctx context.Context, user *domain.User) (*usecase.SyncQuestionStockResult, error)
	evaluateBookAfterAnswer func(ctx context.Context, user *domain.User, questionID string) error
	evaluatedQuestionID     string
}

func (s *stubQuestionSyncUsecase) GetQuestionStock(ctx context.Context, user *domain.User) (*usecase.SyncQuestionStockResult, error) {
	return s.SyncQuestionStock(ctx, user)
}

func (s *stubQuestionSyncUsecase) SyncQuestionStock(ctx context.Context, user *domain.User) (*usecase.SyncQuestionStockResult, error) {
	if s.syncQuestionStock != nil {
		return s.syncQuestionStock(ctx, user)
	}
	return &usecase.SyncQuestionStockResult{}, nil
}

func (s *stubQuestionSyncUsecase) EvaluateBookAfterAnswer(ctx context.Context, user *domain.User, questionID string) error {
	s.evaluatedQuestionID = questionID
	if s.evaluateBookAfterAnswer != nil {
		return s.evaluateBookAfterAnswer(ctx, user, questionID)
	}
	return nil
}

type stubManualGenerationUsecase struct {
	generate func(ctx context.Context, user *domain.User, bookKey string, highlightIDs []uuid.UUID) (*domain.QuestionGenerationJob, error)
}

func (s *stubManualGenerationUsecase) Generate(ctx context.Context, user *domain.User, bookKey string, highlightIDs []uuid.UUID) (*domain.QuestionGenerationJob, error) {
	if s.generate == nil {
		return &domain.QuestionGenerationJob{ID: uuid.New()}, nil
	}
	return s.generate(ctx, user, bookKey, highlightIDs)
}

func TestSaveQuestionTrimsNoteBeforeSaving(t *testing.T) {
	e := echo.New()
	userID := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	var savedNote string
	handler := NewQuestionHandler(&stubQuestionUsecase{
		saveQuestion: func(ctx context.Context, userID string, questionID string, note string) error {
			savedNote = note
			return nil
		},
	}, &stubQuestionSyncUsecase{}, questionHandlerUserUsecase(userID), nil)

	req := httptest.NewRequest(http.MethodPost, "/questions/q-1/save", strings.NewReader(`{"note":"  keep this  "}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.ContextFirebaseUIDKey, "firebase-uid-1")
	c.SetParamNames("id")
	c.SetParamValues("q-1")

	if err := handler.SaveQuestion(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if savedNote != "keep this" {
		t.Fatalf("expected trimmed note to be saved, got %q", savedNote)
	}

	var response map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if response["note"] != savedNote {
		t.Fatalf("response note %q differs from saved note %q", response["note"], savedNote)
	}
}

func TestManualGenerateRejectsTooManyHighlights(t *testing.T) {
	e := echo.New()
	userID := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	manualCalled := false
	handler := NewQuestionHandler(
		&stubQuestionUsecase{},
		&stubQuestionSyncUsecase{},
		questionHandlerUserUsecase(userID),
		&stubManualGenerationUsecase{
			generate: func(ctx context.Context, user *domain.User, bookKey string, highlightIDs []uuid.UUID) (*domain.QuestionGenerationJob, error) {
				manualCalled = true
				return &domain.QuestionGenerationJob{ID: uuid.New()}, nil
			},
		},
	)

	highlightIDs := make([]string, 0, domain.MaxHighlightsPerJob+1)
	for range domain.MaxHighlightsPerJob + 1 {
		highlightIDs = append(highlightIDs, uuid.NewString())
	}
	body, err := json.Marshal(map[string]any{
		"book_key":      "book-a",
		"highlight_ids": highlightIDs,
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/questions/generate/manual", strings.NewReader(string(body)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.ContextFirebaseUIDKey, "firebase-uid-1")

	err = handler.ManualGenerate(c)
	httpErr, ok := err.(*echo.HTTPError)
	if !ok {
		t.Fatalf("expected echo.HTTPError, got %T", err)
	}
	if httpErr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", httpErr.Code)
	}
	if manualCalled {
		t.Fatal("manual usecase should not be called for oversized request")
	}
}

func TestValidateQuestionSourceIDReturnsInvalidInputForEmptyID(t *testing.T) {
	err := validateQuestionSource(domain.SourceTypeKindleBook, "  ")
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func questionHandlerUserUsecase(userID uuid.UUID) *stubUserUsecase {
	return &stubUserUsecase{
		getByFirebaseUID: func(ctx context.Context, firebaseUID string) (*domain.User, error) {
			return &domain.User{ID: userID, FirebaseUID: firebaseUID}, nil
		},
	}
}

func TestListPreparedRejectsNonNumericQuestionCount(t *testing.T) {
	e := echo.New()
	userID := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	handler := NewQuestionHandler(&stubQuestionUsecase{}, &stubQuestionSyncUsecase{}, questionHandlerUserUsecase(userID), nil)

	req := httptest.NewRequest(http.MethodGet, "/questions/prepared?source_type=kindle_book&source_id=book-1&question_count=abc", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.ContextFirebaseUIDKey, "firebase-uid-1")

	err := handler.ListPrepared(c)
	httpErr, ok := err.(*echo.HTTPError)
	if !ok {
		t.Fatalf("expected echo.HTTPError, got %T", err)
	}
	if httpErr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", httpErr.Code)
	}
}

func TestListPreparedRejectsNegativeQuestionCount(t *testing.T) {
	e := echo.New()
	userID := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	handler := NewQuestionHandler(&stubQuestionUsecase{}, &stubQuestionSyncUsecase{}, questionHandlerUserUsecase(userID), nil)

	req := httptest.NewRequest(http.MethodGet, "/questions/prepared?source_type=kindle_book&source_id=book-1&question_count=-1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.ContextFirebaseUIDKey, "firebase-uid-1")

	err := handler.ListPrepared(c)
	httpErr, ok := err.(*echo.HTTPError)
	if !ok {
		t.Fatalf("expected echo.HTTPError, got %T", err)
	}
	if httpErr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", httpErr.Code)
	}
}

func TestListPreparedAllowsZeroQuestionCount(t *testing.T) {
	e := echo.New()
	userID := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	handler := NewQuestionHandler(&stubQuestionUsecase{}, &stubQuestionSyncUsecase{}, questionHandlerUserUsecase(userID), nil)

	req := httptest.NewRequest(http.MethodGet, "/questions/prepared?source_type=kindle_book&source_id=book-1&question_count=0", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.ContextFirebaseUIDKey, "firebase-uid-1")

	err := handler.ListPrepared(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestSyncStockMapsTemporaryErrorToServiceUnavailable(t *testing.T) {
	e := echo.New()
	userID := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	handler := NewQuestionHandler(
		&stubQuestionUsecase{},
		&stubQuestionSyncUsecase{
			syncQuestionStock: func(ctx context.Context, user *domain.User) (*usecase.SyncQuestionStockResult, error) {
				return nil, context.DeadlineExceeded
			},
		},
		questionHandlerUserUsecase(userID),
		nil,
	)

	req := httptest.NewRequest(http.MethodPost, "/questions/sync", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.ContextFirebaseUIDKey, "firebase-uid-1")

	err := handler.SyncStock(c)
	httpErr, ok := err.(*echo.HTTPError)
	if !ok {
		t.Fatalf("expected echo.HTTPError, got %T", err)
	}
	if httpErr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", httpErr.Code)
	}
}
