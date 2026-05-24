package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"firebase.google.com/go/v4/auth"
	"github.com/labstack/echo/v4"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

func TestHybridAuthUsesSessionWhenCookieExists(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: SessionCookieName("development"), Value: "session-cookie"})
	req.Header.Set("Authorization", "Bearer bearer-token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	sessionMiddleware, err := NewSessionAuthMiddleware(stubSessionVerifier{
		verifyFunc: func(ctx context.Context, sessionCookie string) (*domain.AuthToken, error) {
			return &domain.AuthToken{UID: "session-uid"}, nil
		},
	}, "development")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	bearerMiddleware, err := NewFirebaseMiddleware(stubTokenVerifier{
		verifyFunc: func(ctx context.Context, idToken string) (*auth.Token, error) {
			t.Fatal("bearer verifier should not be called")
			return nil, nil
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hybrid := NewHybridAuthMiddleware(sessionMiddleware, bearerMiddleware, "development")
	if err := hybrid.Authenticate(func(c echo.Context) error {
		uid, _ := GetFirebaseUID(c)
		if uid != "session-uid" {
			t.Fatalf("unexpected uid: %s", uid)
		}
		return c.NoContent(http.StatusNoContent)
	})(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHybridAuthUsesBearerWhenNoCookie(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer bearer-token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	sessionMiddleware, err := NewSessionAuthMiddleware(stubSessionVerifier{
		verifyFunc: func(ctx context.Context, sessionCookie string) (*domain.AuthToken, error) {
			t.Fatal("session verifier should not be called")
			return nil, nil
		},
	}, "development")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	bearerMiddleware, err := NewFirebaseMiddleware(stubTokenVerifier{
		verifyFunc: func(ctx context.Context, idToken string) (*auth.Token, error) {
			if idToken != "bearer-token" {
				t.Fatalf("unexpected id token: %s", idToken)
			}
			return &auth.Token{UID: "bearer-uid"}, nil
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hybrid := NewHybridAuthMiddleware(sessionMiddleware, bearerMiddleware, "development")
	if err := hybrid.Authenticate(func(c echo.Context) error {
		uid, _ := GetFirebaseUID(c)
		if uid != "bearer-uid" {
			t.Fatalf("unexpected uid: %s", uid)
		}
		return c.NoContent(http.StatusNoContent)
	})(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHybridAuthRejectsMissingCredentials(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	sessionMiddleware, err := NewSessionAuthMiddleware(stubSessionVerifier{
		verifyFunc: func(ctx context.Context, sessionCookie string) (*domain.AuthToken, error) {
			return nil, nil
		},
	}, "development")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	bearerMiddleware, err := NewFirebaseMiddleware(stubTokenVerifier{
		verifyFunc: func(ctx context.Context, idToken string) (*auth.Token, error) {
			return nil, nil
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hybrid := NewHybridAuthMiddleware(sessionMiddleware, bearerMiddleware, "development")
	err = hybrid.Authenticate(func(c echo.Context) error {
		t.Fatal("next handler should not be called")
		return nil
	})(c)
	assertMiddlewareHTTPErrorCode(t, err, http.StatusUnauthorized)
}
