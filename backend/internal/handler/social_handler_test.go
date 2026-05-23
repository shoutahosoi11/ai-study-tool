package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/shout/ai-study-tool/backend/internal/domain"
	"github.com/shout/ai-study-tool/backend/internal/middleware"
)

type stubSocialUsecase struct {
	liked bool
}

func (s *stubSocialUsecase) Follow(ctx context.Context, followerID, followingID string) error {
	return nil
}

func (s *stubSocialUsecase) Unfollow(ctx context.Context, followerID, followingID string) error {
	return nil
}

func (s *stubSocialUsecase) Like(ctx context.Context, userID, postID string) error {
	s.liked = true
	return nil
}

func (s *stubSocialUsecase) Unlike(ctx context.Context, userID, postID string) error {
	return nil
}

func (s *stubSocialUsecase) Repost(ctx context.Context, userID, postID string) error {
	return nil
}

func (s *stubSocialUsecase) Unrepost(ctx context.Context, userID, postID string) error {
	return nil
}

func (s *stubSocialUsecase) CreateComment(ctx context.Context, comment *domain.Comment) (*domain.Comment, error) {
	return comment, nil
}

func (s *stubSocialUsecase) ListComments(ctx context.Context, postID string, limit, offset int) ([]*domain.Comment, error) {
	return nil, nil
}

type stubSocialPostUsecase struct {
	err error
}

func (s *stubSocialPostUsecase) EnsureVisible(ctx context.Context, viewerID, postID uuid.UUID) error {
	return s.err
}

func TestLikeRequiresVisiblePost(t *testing.T) {
	e := echo.New()
	userID := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	postID := uuid.NewString()
	social := &stubSocialUsecase{}
	handler := NewSocialHandler(social, &stubSocialPostUsecase{err: domain.ErrNotFound}, questionHandlerUserUsecase(userID))

	req := httptest.NewRequest(http.MethodPost, "/posts/"+postID+"/like", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.ContextFirebaseUIDKey, "firebase-uid-1")
	c.SetParamNames("id")
	c.SetParamValues(postID)

	err := handler.Like(c)
	httpErr, ok := err.(*echo.HTTPError)
	if !ok {
		t.Fatalf("expected echo.HTTPError, got %T", err)
	}
	if httpErr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", httpErr.Code)
	}
	if social.liked {
		t.Fatal("like should not be called when post is not visible")
	}
}
