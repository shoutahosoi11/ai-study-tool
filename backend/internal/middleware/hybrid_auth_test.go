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

	csrfMiddleware := NewCSRFMiddleware("development")
	hybrid, err := NewHybridAuthMiddleware(sessionMiddleware, bearerMiddleware, csrfMiddleware, "development")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
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

func TestHybridAuthRequiresCSRFForSessionPost(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.AddCookie(&http.Cookie{Name: SessionCookieName("development"), Value: "session-cookie"})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	sessionMiddleware, err := NewSessionAuthMiddleware(stubSessionVerifier{
		verifyFunc: func(ctx context.Context, sessionCookie string) (*domain.AuthToken, error) {
			t.Fatal("session verifier should not be called before csrf passes")
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

	hybrid, err := NewHybridAuthMiddleware(sessionMiddleware, bearerMiddleware, NewCSRFMiddleware("development"), "development")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	err = hybrid.Authenticate(func(c echo.Context) error {
		t.Fatal("next handler should not be called")
		return nil
	})(c)
	assertMiddlewareHTTPErrorCode(t, err, http.StatusForbidden)
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

	csrfMiddleware := NewCSRFMiddleware("development")
	hybrid, err := NewHybridAuthMiddleware(sessionMiddleware, bearerMiddleware, csrfMiddleware, "development")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
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

	csrfMiddleware := NewCSRFMiddleware("development")
	hybrid, err := NewHybridAuthMiddleware(sessionMiddleware, bearerMiddleware, csrfMiddleware, "development")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	err = hybrid.Authenticate(func(c echo.Context) error {
		t.Fatal("next handler should not be called")
		return nil
	})(c)
	assertMiddlewareHTTPErrorCode(t, err, http.StatusUnauthorized)
}

func TestNewHybridAuthMiddlewareRejectsNilDependencies(t *testing.T) {
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
	csrfMiddleware := NewCSRFMiddleware("development")

	testCases := []struct {
		name    string
		session *SessionAuthMiddleware
		bearer  *FirebaseMiddleware
		csrf    *CSRFMiddleware
	}{
		{name: "NilSession", bearer: bearerMiddleware, csrf: csrfMiddleware},
		{name: "NilBearer", session: sessionMiddleware, csrf: csrfMiddleware},
		{name: "NilCSRF", session: sessionMiddleware, bearer: bearerMiddleware},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hybrid, err := NewHybridAuthMiddleware(tc.session, tc.bearer, tc.csrf, "development")
			if err == nil {
				t.Fatal("expected error")
			}
			if hybrid != nil {
				t.Fatal("expected nil middleware")
			}
		})
	}
}
