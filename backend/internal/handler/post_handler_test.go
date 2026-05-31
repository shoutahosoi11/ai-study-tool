package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/shout/ai-study-tool/backend/internal/domain"
	"github.com/shout/ai-study-tool/backend/internal/middleware"
)

type stubPostUsecase struct {
	timelineViewerID uuid.UUID
	createInput      domain.CreatePostInput
	ensureVisibleErr error
	createErr        error
	listQuestionsErr error
}

func (s *stubPostUsecase) GetTimeline(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.TimelinePost, error) {
	s.timelineViewerID = userID
	return []*domain.TimelinePost{}, nil
}

func (s *stubPostUsecase) GetByID(ctx context.Context, id uuid.UUID) (*domain.TimelinePost, error) {
	return &domain.TimelinePost{
		Post: domain.Post{
			ID:        id,
			UserID:    uuid.New(),
			Type:      "text",
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		},
		Username:    "alice",
		DisplayName: "Alice",
	}, nil
}

func (s *stubPostUsecase) CreatePost(ctx context.Context, input domain.CreatePostInput) (*domain.Post, error) {
	s.createInput = input
	if s.createErr != nil {
		return nil, s.createErr
	}
	return &domain.Post{
		ID:        uuid.New(),
		UserID:    input.UserID,
		Type:      input.Type,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}, nil
}

func (s *stubPostUsecase) ListQuestionsByPostID(ctx context.Context, postID uuid.UUID) ([]*domain.PostedQuestion, error) {
	if s.listQuestionsErr != nil {
		return nil, s.listQuestionsErr
	}
	return []*domain.PostedQuestion{}, nil
}

func (s *stubPostUsecase) EnsureVisible(ctx context.Context, viewerID, postID uuid.UUID) error {
	return s.ensureVisibleErr
}

func TestPostHandlerGetTimelineUsesAuthenticatedUser(t *testing.T) {
	e := echo.New()
	userID := uuid.New()
	postUsecase := &stubPostUsecase{}
	handler := NewPostHandler(postUsecase, questionHandlerUserUsecase(userID))

	req := httptest.NewRequest(http.MethodGet, "/posts/timeline?limit=10&offset=5", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.ContextFirebaseUIDKey, "firebase-uid-1")

	if err := handler.GetTimeline(c); err != nil {
		t.Fatalf("get timeline: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if postUsecase.timelineViewerID != userID {
		t.Fatalf("timeline user = %s, want %s", postUsecase.timelineViewerID, userID)
	}
}

func TestPostHandlerCreatePostValidatesQuestionIDs(t *testing.T) {
	e := echo.New()
	handler := NewPostHandler(&stubPostUsecase{}, questionHandlerUserUsecase(uuid.New()))
	req := httptest.NewRequest(http.MethodPost, "/posts", strings.NewReader(`{"type":"question","questions":[{"question_id":"bad"}]}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.ContextFirebaseUIDKey, "firebase-uid-1")

	err := handler.CreatePost(c)
	assertHTTPStatus(t, err, http.StatusBadRequest)
}

func TestPostHandlerCreatePostMapsForbidden(t *testing.T) {
	e := echo.New()
	handler := NewPostHandler(&stubPostUsecase{createErr: domain.ErrForbidden}, questionHandlerUserUsecase(uuid.New()))
	req := httptest.NewRequest(http.MethodPost, "/posts", strings.NewReader(`{"type":"question"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.ContextFirebaseUIDKey, "firebase-uid-1")

	err := handler.CreatePost(c)
	assertHTTPStatus(t, err, http.StatusForbidden)
}

func TestPostHandlerGetPostHidesInvisiblePostsAsNotFound(t *testing.T) {
	e := echo.New()
	handler := NewPostHandler(&stubPostUsecase{ensureVisibleErr: domain.ErrForbidden}, questionHandlerUserUsecase(uuid.New()))
	req := httptest.NewRequest(http.MethodGet, "/posts/"+uuid.NewString(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(uuid.NewString())
	c.Set(middleware.ContextFirebaseUIDKey, "firebase-uid-1")

	err := handler.GetPost(c)
	assertHTTPStatus(t, err, http.StatusNotFound)
}

func TestPostHandlerListQuestionsMapsUnexpectedError(t *testing.T) {
	e := echo.New()
	handler := NewPostHandler(&stubPostUsecase{listQuestionsErr: errors.New("db down")}, questionHandlerUserUsecase(uuid.New()))
	req := httptest.NewRequest(http.MethodGet, "/posts/"+uuid.NewString()+"/questions", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(uuid.NewString())
	c.Set(middleware.ContextFirebaseUIDKey, "firebase-uid-1")

	err := handler.ListQuestions(c)
	assertHTTPStatus(t, err, http.StatusInternalServerError)
}
