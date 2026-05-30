package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"firebase.google.com/go/v4/auth"
	"github.com/google/uuid"
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
	extensionMiddleware := newUnusedExtensionAuthMiddleware(t)
	hybrid, err := NewHybridAuthMiddleware(sessionMiddleware, bearerMiddleware, extensionMiddleware, csrfMiddleware, "development")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := hybrid.Authenticate(func(c echo.Context) error {
		uid, _ := GetFirebaseUID(c)
		if uid != "session-uid" {
			t.Fatalf("unexpected uid: %s", uid)
		}
		clientType, ok := GetAuthClientType(c)
		if !ok || clientType != domain.AuthClientTypeWeb {
			t.Fatalf("unexpected client type: %q", clientType)
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
			if sessionCookie != "session-cookie" {
				t.Fatalf("unexpected session cookie: %s", sessionCookie)
			}
			return &domain.AuthToken{UID: "session-uid"}, nil
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

	hybrid, err := NewHybridAuthMiddleware(sessionMiddleware, bearerMiddleware, newUnusedExtensionAuthMiddleware(t), NewCSRFMiddleware("development"), "development")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	err = hybrid.Authenticate(func(c echo.Context) error {
		t.Fatal("next handler should not be called")
		return nil
	})(c)
	assertMiddlewareHTTPErrorCode(t, err, http.StatusForbidden)
}

func TestHybridAuthUsesBearerWhenNoCookieAndDoesNotRequireCSRF(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
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
	hybrid, err := NewHybridAuthMiddleware(sessionMiddleware, bearerMiddleware, newUnusedExtensionAuthMiddleware(t), csrfMiddleware, "development")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := hybrid.Authenticate(func(c echo.Context) error {
		uid, _ := GetFirebaseUID(c)
		if uid != "bearer-uid" {
			t.Fatalf("unexpected uid: %s", uid)
		}
		clientType, ok := GetAuthClientType(c)
		if !ok || clientType != domain.AuthClientTypeMobile {
			t.Fatalf("unexpected client type: %q", clientType)
		}
		return c.NoContent(http.StatusNoContent)
	})(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHybridAuthAppliesMobileGuardsAfterBearerAuth(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Authorization", "Bearer bearer-token")
	req.Header.Set(appCheckHeader, "valid-app-check-token")
	req.Header.Set(appPlatformHeader, AppPlatformIOS)
	req.Header.Set(appVersionHeader, "1.0.0")
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
			return &auth.Token{UID: "bearer-uid"}, nil
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	appCheckMiddleware, err := NewAppCheckMiddleware(stubAppCheckVerifier{
		verifyFunc: func(ctx context.Context, token string) error {
			if token != "valid-app-check-token" {
				t.Fatalf("unexpected app check token: %s", token)
			}
			return nil
		},
	}, AppCheckConfig{AppEnv: "production", Enforced: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	appVersionMiddleware := NewAppVersionMiddleware(AppVersionConfig{
		MinSupportedIOSVersion:     "1.0.0",
		RejectMissingMobileHeaders: true,
	})

	hybrid, err := NewHybridAuthMiddleware(
		sessionMiddleware,
		bearerMiddleware,
		newUnusedExtensionAuthMiddleware(t),
		NewCSRFMiddleware("development"),
		"development",
		appVersionMiddleware.Check,
		appCheckMiddleware.Require,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := hybrid.Authenticate(func(c echo.Context) error {
		clientType, ok := GetAuthClientType(c)
		if !ok || clientType != domain.AuthClientTypeMobile {
			t.Fatalf("unexpected client type: %q", clientType)
		}
		return c.NoContent(http.StatusNoContent)
	})(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHybridAuthUsesExtensionWhenExtBearerTokenExistsAndDoesNotRequireCSRF(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Authorization", "Bearer ext_valid-token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	userID := uuid.New()

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
			t.Fatal("firebase verifier should not be called")
			return nil, nil
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	extensionMiddleware, err := NewExtensionAuthMiddleware(stubExtensionTokenRepository{
		findFunc: func(ctx context.Context, tokenHash string, now time.Time) (*domain.ExtensionToken, error) {
			if tokenHash != domain.HashExtensionToken("ext_valid-token") {
				t.Fatalf("unexpected token hash: %s", tokenHash)
			}
			return &domain.ExtensionToken{
				ID:          uuid.New(),
				UserID:      userID,
				FirebaseUID: "extension-uid",
				Scopes:      []string{domain.ExtensionScopeHighlightWrite},
			}, nil
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hybrid, err := NewHybridAuthMiddleware(sessionMiddleware, bearerMiddleware, extensionMiddleware, NewCSRFMiddleware("development"), "development")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := hybrid.Authenticate(func(c echo.Context) error {
		uid, _ := GetFirebaseUID(c)
		if uid != "extension-uid" {
			t.Fatalf("unexpected uid: %s", uid)
		}
		clientType, ok := GetAuthClientType(c)
		if !ok || clientType != domain.AuthClientTypeExtension {
			t.Fatalf("unexpected client type: %q", clientType)
		}
		if !domain.HasScope(GetAuthScopes(c), domain.ExtensionScopeHighlightWrite) {
			t.Fatal("expected highlight write scope")
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
	hybrid, err := NewHybridAuthMiddleware(sessionMiddleware, bearerMiddleware, newUnusedExtensionAuthMiddleware(t), csrfMiddleware, "development")
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
	extensionMiddleware := newUnusedExtensionAuthMiddleware(t)

	testCases := []struct {
		name      string
		session   *SessionAuthMiddleware
		bearer    *FirebaseMiddleware
		extension *ExtensionAuthMiddleware
		csrf      *CSRFMiddleware
	}{
		{name: "NilSession", bearer: bearerMiddleware, extension: extensionMiddleware, csrf: csrfMiddleware},
		{name: "NilBearer", session: sessionMiddleware, extension: extensionMiddleware, csrf: csrfMiddleware},
		{name: "NilExtension", session: sessionMiddleware, bearer: bearerMiddleware, csrf: csrfMiddleware},
		{name: "NilCSRF", session: sessionMiddleware, bearer: bearerMiddleware, extension: extensionMiddleware},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hybrid, err := NewHybridAuthMiddleware(tc.session, tc.bearer, tc.extension, tc.csrf, "development")
			if err == nil {
				t.Fatal("expected error")
			}
			if hybrid != nil {
				t.Fatal("expected nil middleware")
			}
		})
	}
}

func newUnusedExtensionAuthMiddleware(t *testing.T) *ExtensionAuthMiddleware {
	t.Helper()
	middleware, err := NewExtensionAuthMiddleware(stubExtensionTokenRepository{
		findFunc: func(ctx context.Context, tokenHash string, now time.Time) (*domain.ExtensionToken, error) {
			t.Fatal("extension token repository should not be called")
			return nil, nil
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return middleware
}
