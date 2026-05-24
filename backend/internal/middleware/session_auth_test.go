package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

type stubSessionVerifier struct {
	verifyFunc func(ctx context.Context, sessionCookie string) (*domain.AuthToken, error)
}

func (s stubSessionVerifier) VerifySessionCookieAndCheckRevoked(ctx context.Context, sessionCookie string) (*domain.AuthToken, error) {
	return s.verifyFunc(ctx, sessionCookie)
}

func TestSessionAuthPassesValidCookieAndSetsUID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: SessionCookieName("development"), Value: "valid-session"})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	middleware, err := NewSessionAuthMiddleware(stubSessionVerifier{
		verifyFunc: func(ctx context.Context, sessionCookie string) (*domain.AuthToken, error) {
			if sessionCookie != "valid-session" {
				t.Fatalf("unexpected session cookie: %s", sessionCookie)
			}
			return &domain.AuthToken{UID: "firebase-uid-1"}, nil
		},
	}, "development")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	handler := middleware.Authenticate(func(c echo.Context) error {
		uid, ok := GetFirebaseUID(c)
		if !ok || uid != "firebase-uid-1" {
			t.Fatalf("unexpected uid: %s", uid)
		}
		return c.NoContent(http.StatusNoContent)
	})

	if err := handler(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
}

func TestSessionAuthRejectsMissingCookie(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	middleware, err := NewSessionAuthMiddleware(stubSessionVerifier{
		verifyFunc: func(ctx context.Context, sessionCookie string) (*domain.AuthToken, error) {
			t.Fatal("verifier should not be called")
			return nil, nil
		},
	}, "development")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = middleware.Authenticate(func(c echo.Context) error {
		t.Fatal("next handler should not be called")
		return nil
	})(c)
	assertMiddlewareHTTPErrorCode(t, err, http.StatusUnauthorized)
}

func TestSessionAuthRejectsRevokedCookie(t *testing.T) {
	originalClassifier := isFirebaseSessionCookieClientError
	defer func() {
		isFirebaseSessionCookieClientError = originalClassifier
	}()

	revokedErr := errors.New("revoked")
	isFirebaseSessionCookieClientError = func(err error) bool {
		return err == revokedErr
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: SessionCookieName("development"), Value: "revoked-session"})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	middleware, err := NewSessionAuthMiddleware(stubSessionVerifier{
		verifyFunc: func(ctx context.Context, sessionCookie string) (*domain.AuthToken, error) {
			return nil, revokedErr
		},
	}, "development")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = middleware.Authenticate(func(c echo.Context) error {
		t.Fatal("next handler should not be called")
		return nil
	})(c)
	assertMiddlewareHTTPErrorCode(t, err, http.StatusUnauthorized)
}

func assertMiddlewareHTTPErrorCode(t *testing.T, err error, want int) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error")
	}
	httpErr, ok := err.(*echo.HTTPError)
	if !ok {
		t.Fatalf("expected *echo.HTTPError, got %T", err)
	}
	if httpErr.Code != want {
		t.Fatalf("unexpected status: %d", httpErr.Code)
	}
}
