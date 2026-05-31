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
)

type stubQuestionWorker struct {
	jobID  uuid.UUID
	userID uuid.UUID
	err    error
}

func (s *stubQuestionWorker) ProcessQuestionGenerationJob(ctx context.Context, jobID uuid.UUID, userID uuid.UUID) error {
	s.jobID = jobID
	s.userID = userID
	return s.err
}

type stubHighlightImportJobs struct {
	queueID uuid.UUID
	userID  uuid.UUID
	err     error
}

func (s *stubHighlightImportJobs) ProcessSingle(ctx context.Context, queueID uuid.UUID, userID uuid.UUID) error {
	s.queueID = queueID
	s.userID = userID
	return s.err
}

func TestTaskHandlerHandleQuestionGeneration(t *testing.T) {
	e := echo.New()
	jobID := uuid.New()
	userID := uuid.New()
	worker := &stubQuestionWorker{}
	handler := NewTaskHandler(worker, nil)

	req := httptest.NewRequest(http.MethodPost, "/tasks/question-generation", strings.NewReader(`{"job_id":"`+jobID.String()+`","user_id":"`+userID.String()+`"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := handler.HandleQuestionGeneration(c); err != nil {
		t.Fatalf("handle question generation: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if worker.jobID != jobID || worker.userID != userID {
		t.Fatalf("unexpected worker args job=%s user=%s", worker.jobID, worker.userID)
	}
}

func TestTaskHandlerHandleQuestionGenerationRejectsInvalidJobID(t *testing.T) {
	e := echo.New()
	handler := NewTaskHandler(&stubQuestionWorker{}, nil)
	req := httptest.NewRequest(http.MethodPost, "/tasks/question-generation", strings.NewReader(`{"job_id":"bad","user_id":"`+uuid.NewString()+`"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.HandleQuestionGeneration(c)
	assertHTTPStatus(t, err, http.StatusBadRequest)
}

func TestTaskHandlerHandleHighlightImportMapsForbidden(t *testing.T) {
	e := echo.New()
	jobs := &stubHighlightImportJobs{err: domain.ErrForbidden}
	handler := NewTaskHandler(nil, jobs)
	req := httptest.NewRequest(http.MethodPost, "/tasks/highlight-import", strings.NewReader(`{"queue_id":"`+uuid.NewString()+`","user_id":"`+uuid.NewString()+`"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.HandleHighlightImport(c)
	assertHTTPStatus(t, err, http.StatusForbidden)
}

func TestTaskHandlerHandleHighlightImportMapsWorkerFailure(t *testing.T) {
	e := echo.New()
	jobs := &stubHighlightImportJobs{err: errors.New("queue unavailable")}
	handler := NewTaskHandler(nil, jobs)
	req := httptest.NewRequest(http.MethodPost, "/tasks/highlight-import", strings.NewReader(`{"queue_id":"`+uuid.NewString()+`","user_id":"`+uuid.NewString()+`"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.HandleHighlightImport(c)
	assertHTTPStatus(t, err, http.StatusInternalServerError)
}
