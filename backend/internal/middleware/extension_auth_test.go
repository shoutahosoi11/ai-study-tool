package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

type stubExtensionTokenRepository struct {
	findFunc func(ctx context.Context, tokenHash string, now time.Time) (*domain.ExtensionToken, error)
}

func (s stubExtensionTokenRepository) FindActiveByTokenHash(ctx context.Context, tokenHash string, now time.Time) (*domain.ExtensionToken, error) {
	return s.findFunc(ctx, tokenHash, now)
}

func TestExtensionAuthSetsClientTypeScopesAndUID(t *testing.T) {
	e := echo.New()
	rawToken := "ext_valid-token"
	userID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+rawToken)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	middleware, err := NewExtensionAuthMiddleware(stubExtensionTokenRepository{
		findFunc: func(ctx context.Context, tokenHash string, now time.Time) (*domain.ExtensionToken, error) {
			if tokenHash != domain.HashExtensionToken(rawToken) {
				t.Fatalf("unexpected token hash: %s", tokenHash)
			}
			if tokenHash == rawToken {
				t.Fatal("raw extension token must not be passed to storage")
			}
			return &domain.ExtensionToken{
				ID:          uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
				UserID:      userID,
				FirebaseUID: "firebase-uid-1",
				Scopes:      []string{domain.ExtensionScopeHighlightWrite, domain.ExtensionScopeHighlightCheck, domain.ExtensionScopeQuestionGenerate},
			}, nil
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := middleware.Authenticate(func(c echo.Context) error {
		uid, ok := GetFirebaseUID(c)
		if !ok || uid != "firebase-uid-1" {
			t.Fatalf("unexpected uid: %q", uid)
		}
		clientType, ok := GetAuthClientType(c)
		if !ok || clientType != domain.AuthClientTypeExtension {
			t.Fatalf("unexpected client type: %q", clientType)
		}
		if !domain.HasScope(GetAuthScopes(c), domain.ExtensionScopeHighlightWrite) {
			t.Fatal("expected highlight write scope")
		}
		if domain.HasScope(GetAuthScopes(c), domain.ExtensionScopeQuestionGenerate) {
			t.Fatal("question generation scope must not be accepted for extension auth")
		}
		currentUser, ok := GetCurrentUser(c)
		if !ok || currentUser.UserID == nil || *currentUser.UserID != userID {
			t.Fatalf("unexpected current user: %#v", currentUser)
		}
		tokenID, ok := GetExtensionTokenID(c)
		if !ok || tokenID != "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" {
			t.Fatalf("unexpected extension token id: %q", tokenID)
		}
		return c.NoContent(http.StatusNoContent)
	})(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
}

func TestExtensionAuthRejectsRevokedOrExpiredToken(t *testing.T) {
	testCases := []struct {
		name string
		err  error
	}{
		{name: "Revoked", err: domain.ErrNotFound},
		{name: "Expired", err: domain.ErrNotFound},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Bearer ext_invalid-token")
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			middleware, err := NewExtensionAuthMiddleware(stubExtensionTokenRepository{
				findFunc: func(ctx context.Context, tokenHash string, now time.Time) (*domain.ExtensionToken, error) {
					return nil, tc.err
				},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			err = middleware.Authenticate(func(c echo.Context) error {
				t.Fatal("next handler should not be called")
				return nil
			})(c)
			assertMiddlewareHTTPErrorCode(t, err, http.StatusUnauthorized)
		})
	}
}

func TestExtensionAuthReturnsServiceUnavailableOnStoreError(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer ext_valid-token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	middleware, err := NewExtensionAuthMiddleware(stubExtensionTokenRepository{
		findFunc: func(ctx context.Context, tokenHash string, now time.Time) (*domain.ExtensionToken, error) {
			return nil, errors.New("db unavailable")
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = middleware.Authenticate(func(c echo.Context) error {
		t.Fatal("next handler should not be called")
		return nil
	})(c)
	assertMiddlewareHTTPErrorCode(t, err, http.StatusServiceUnavailable)
}

func TestRequireScopeRejectsExtensionWhenScopeMissing(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setAuthContext(c, domain.AuthenticatedUser{FirebaseUID: "firebase-uid-1"}, domain.AuthClientTypeExtension, []string{domain.ExtensionScopeHighlightCheck})

	err := RequireScope(domain.ExtensionScopeHighlightWrite)(func(c echo.Context) error {
		t.Fatal("next handler should not be called")
		return nil
	})(c)
	assertMiddlewareHTTPErrorCode(t, err, http.StatusForbidden)
}

func TestRequireScopeAllowsExtensionHighlightImport(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/extension/highlights/import", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setAuthContext(c, domain.AuthenticatedUser{FirebaseUID: "firebase-uid-1"}, domain.AuthClientTypeExtension, domain.DefaultExtensionTokenScopes())

	if err := RequireScope(domain.ExtensionScopeHighlightWrite)(func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRequireScopeRejectsExtensionGenerationAndBilling(t *testing.T) {
	testCases := []struct {
		name  string
		scope string
		route string
	}{
		{name: "QuestionGeneration", scope: domain.ExtensionScopeQuestionGenerate, route: "/api/v1/questions/generate/manual"},
		{name: "Billing", scope: domain.ExtensionScopeBillingWrite, route: "/api/v1/checkout/session"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, tc.route, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			setAuthContext(c, domain.AuthenticatedUser{FirebaseUID: "firebase-uid-1"}, domain.AuthClientTypeExtension, domain.DefaultExtensionTokenScopes())

			err := RequireScope(tc.scope)(func(c echo.Context) error {
				t.Fatal("next handler should not be called")
				return nil
			})(c)
			assertMiddlewareHTTPErrorCode(t, err, http.StatusForbidden)
		})
	}
}

func TestRequireScopeAllowsWebAndMobileWithoutScopes(t *testing.T) {
	testCases := []struct {
		name       string
		clientType domain.AuthClientType
	}{
		{name: "Web", clientType: domain.AuthClientTypeWeb},
		{name: "Mobile", clientType: domain.AuthClientTypeMobile},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			setAuthContext(c, domain.AuthenticatedUser{FirebaseUID: "firebase-uid-1"}, tc.clientType, nil)

			if err := RequireScope(domain.ExtensionScopeHighlightWrite)(func(c echo.Context) error {
				return c.NoContent(http.StatusNoContent)
			})(c); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
