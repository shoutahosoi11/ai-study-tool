package handler

import (
	"context"
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

type stubAnswerUsecase struct {
	input  usecase.SubmitAnswerInput
	result *usecase.SubmitAnswerResult
	err    error
}

func (s *stubAnswerUsecase) SubmitAnswer(ctx context.Context, input usecase.SubmitAnswerInput) (*usecase.SubmitAnswerResult, error) {
	s.input = input
	if s.err != nil {
		return nil, s.err
	}
	return s.result, nil
}

func TestAnswerHandlerSubmitAnswerUsesAuthenticatedUser(t *testing.T) {
	e := echo.New()
	userID := uuid.New()
	answer := &stubAnswerUsecase{
		result: &usecase.SubmitAnswerResult{
			IsCorrect:     true,
			CorrectAnswer: "A",
			Explanation:   "Because.",
		},
	}
	sync := &stubQuestionSyncUsecase{}
	handler := NewAnswerHandler(answer, questionHandlerUserUsecase(userID), sync)

	req := httptest.NewRequest(http.MethodPost, "/questions/q1/answers", strings.NewReader(`{"user_answer":"A"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("q1")
	c.Set(middleware.ContextFirebaseUIDKey, "firebase-uid-1")

	if err := handler.SubmitAnswer(c); err != nil {
		t.Fatalf("submit answer: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if answer.input.UserID != userID.String() || answer.input.QuestionID != "q1" || answer.input.UserAnswer != "A" {
		t.Fatalf("unexpected input: %#v", answer.input)
	}
	if sync.evaluatedQuestionID != "q1" {
		t.Fatalf("expected sync for q1, got %q", sync.evaluatedQuestionID)
	}
}

func TestAnswerHandlerSubmitAnswerRejectsEmptyAnswer(t *testing.T) {
	e := echo.New()
	handler := NewAnswerHandler(&stubAnswerUsecase{}, questionHandlerUserUsecase(uuid.New()), nil)
	req := httptest.NewRequest(http.MethodPost, "/questions/q1/answers", strings.NewReader(`{"user_answer":""}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("q1")
	c.Set(middleware.ContextFirebaseUIDKey, "firebase-uid-1")

	err := handler.SubmitAnswer(c)
	assertHTTPStatus(t, err, http.StatusBadRequest)
}

func TestAnswerHandlerSubmitAnswerMapsNotFound(t *testing.T) {
	e := echo.New()
	handler := NewAnswerHandler(&stubAnswerUsecase{err: domain.ErrNotFound}, questionHandlerUserUsecase(uuid.New()), nil)
	req := httptest.NewRequest(http.MethodPost, "/questions/q1/answers", strings.NewReader(`{"user_answer":"A"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("q1")
	c.Set(middleware.ContextFirebaseUIDKey, "firebase-uid-1")

	err := handler.SubmitAnswer(c)
	assertHTTPStatus(t, err, http.StatusNotFound)
}

func TestAnswerHandlerSubmitAnswerToleratesSyncFailure(t *testing.T) {
	e := echo.New()
	handler := NewAnswerHandler(
		&stubAnswerUsecase{result: &usecase.SubmitAnswerResult{}},
		questionHandlerUserUsecase(uuid.New()),
		&stubQuestionSyncUsecase{
			evaluateBookAfterAnswer: func(ctx context.Context, user *domain.User, questionID string) error {
				return errors.New("sync failed")
			},
		},
	)
	req := httptest.NewRequest(http.MethodPost, "/questions/q1/answers", strings.NewReader(`{"user_answer":"A"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("q1")
	c.Set(middleware.ContextFirebaseUIDKey, "firebase-uid-1")

	// The answer is already persisted when the post-answer evaluation runs, so
	// an evaluation failure must not fail the request (it would invite a
	// duplicate submission); it is logged and the grading result is returned.
	if err := handler.SubmitAnswer(c); err != nil {
		t.Fatalf("SubmitAnswer should succeed despite sync failure, got %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
