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

type stubAppCheckVerifier struct {
	verifyFunc func(ctx context.Context, token string) error
}

func (s stubAppCheckVerifier) VerifyAppCheckToken(ctx context.Context, token string) error {
	return s.verifyFunc(ctx, token)
}

func TestAppCheckRequiresTokenForMobileWhenEnforced(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(ContextAuthClientTypeKey, domain.AuthClientTypeMobile)

	middleware, err := NewAppCheckMiddleware(stubAppCheckVerifier{
		verifyFunc: func(ctx context.Context, token string) error {
			t.Fatal("verifier should not be called")
			return nil
		},
	}, AppCheckConfig{AppEnv: "production", Enforced: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = middleware.Require(func(c echo.Context) error {
		t.Fatal("next handler should not be called")
		return nil
	})(c)
	assertMiddlewareHTTPErrorCode(t, err, http.StatusUnauthorized)
}

func TestAppCheckRejectsInvalidToken(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set(appCheckHeader, "invalid-app-check-token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(ContextAuthClientTypeKey, domain.AuthClientTypeMobile)

	middleware, err := NewAppCheckMiddleware(stubAppCheckVerifier{
		verifyFunc: func(ctx context.Context, token string) error {
			if token != "invalid-app-check-token" {
				t.Fatalf("unexpected app check token: %s", token)
			}
			return errors.New("invalid app check")
		},
	}, AppCheckConfig{AppEnv: "production", Enforced: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = middleware.Require(func(c echo.Context) error {
		t.Fatal("next handler should not be called")
		return nil
	})(c)
	assertMiddlewareHTTPErrorCode(t, err, http.StatusUnauthorized)
}

func TestAppCheckPassesValidToken(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set(appCheckHeader, "valid-app-check-token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(ContextAuthClientTypeKey, domain.AuthClientTypeMobile)

	middleware, err := NewAppCheckMiddleware(stubAppCheckVerifier{
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

	if err := middleware.Require(func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAppCheckCanBeSkippedOutsideProduction(t *testing.T) {
	middleware, err := NewAppCheckMiddleware(nil, AppCheckConfig{
		AppEnv:   "development",
		Enforced: false,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(ContextAuthClientTypeKey, domain.AuthClientTypeMobile)

	if err := middleware.Require(func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAppCheckRejectsDisabledEnforcementInProduction(t *testing.T) {
	middleware, err := NewAppCheckMiddleware(nil, AppCheckConfig{
		AppEnv:   "production",
		Enforced: false,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if middleware != nil {
		t.Fatal("expected nil middleware")
	}
}

func TestAppCheckEnforcementFromEnvRejectsFalseInProduction(t *testing.T) {
	t.Setenv("APP_CHECK_ENFORCEMENT", "false")

	_, err := AppCheckEnforcementEnabledFromEnv("production")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAppCheckSkipsWebAndExtensionClients(t *testing.T) {
	testCases := []struct {
		name       string
		clientType domain.AuthClientType
	}{
		{name: "Web", clientType: domain.AuthClientTypeWeb},
		{name: "Extension", clientType: domain.AuthClientTypeExtension},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			middleware, err := NewAppCheckMiddleware(stubAppCheckVerifier{
				verifyFunc: func(ctx context.Context, token string) error {
					t.Fatal("verifier should not be called")
					return nil
				},
			}, AppCheckConfig{AppEnv: "production", Enforced: true})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.Set(ContextAuthClientTypeKey, tc.clientType)

			if err := middleware.Require(func(c echo.Context) error {
				return c.NoContent(http.StatusNoContent)
			})(c); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
