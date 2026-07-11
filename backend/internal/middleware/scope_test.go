package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

func newScopeTestContext(t *testing.T) echo.Context {
	t.Helper()
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec)
}

func callRequireScope(t *testing.T, c echo.Context, scope string) error {
	t.Helper()
	called := false
	err := RequireScope(scope)(func(c echo.Context) error {
		called = true
		return nil
	})(c)
	if err == nil && !called {
		t.Fatal("next handler was not called despite nil error")
	}
	if err != nil && called {
		t.Fatal("next handler was called despite error")
	}
	return err
}

func requireScopeHTTPError(t *testing.T, err error, wantCode int) {
	t.Helper()
	var httpErr *echo.HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected *echo.HTTPError, got %T (%v)", err, err)
	}
	if httpErr.Code != wantCode {
		t.Fatalf("expected status %d, got %d", wantCode, httpErr.Code)
	}
}

func TestRequireScopeRejectsMissingClientType(t *testing.T) {
	c := newScopeTestContext(t)

	err := callRequireScope(t, c, domain.ExtensionScopeHighlightWrite)

	requireScopeHTTPError(t, err, http.StatusUnauthorized)
}

func TestRequireScopePassesThroughWebClient(t *testing.T) {
	c := newScopeTestContext(t)
	c.Set(ContextAuthClientTypeKey, domain.AuthClientTypeWeb)

	if err := callRequireScope(t, c, domain.ExtensionScopeHighlightWrite); err != nil {
		t.Fatalf("web client should bypass scope check, got %v", err)
	}
}

func TestRequireScopePassesThroughMobileClient(t *testing.T) {
	c := newScopeTestContext(t)
	c.Set(ContextAuthClientTypeKey, domain.AuthClientTypeMobile)

	if err := callRequireScope(t, c, domain.ExtensionScopeHighlightWrite); err != nil {
		t.Fatalf("mobile client should bypass scope check, got %v", err)
	}
}

func TestRequireScopeAllowsExtensionWithScope(t *testing.T) {
	c := newScopeTestContext(t)
	c.Set(ContextAuthClientTypeKey, domain.AuthClientTypeExtension)
	c.Set(ContextAuthScopesKey, []string{domain.ExtensionScopeHighlightWrite})

	if err := callRequireScope(t, c, domain.ExtensionScopeHighlightWrite); err != nil {
		t.Fatalf("extension with matching scope should pass, got %v", err)
	}
}

func TestRequireScopeRejectsExtensionWithoutScope(t *testing.T) {
	c := newScopeTestContext(t)
	c.Set(ContextAuthClientTypeKey, domain.AuthClientTypeExtension)
	c.Set(ContextAuthScopesKey, []string{domain.ExtensionScopeHighlightCheck})

	err := callRequireScope(t, c, domain.ExtensionScopeHighlightWrite)

	requireScopeHTTPError(t, err, http.StatusForbidden)
}

func TestRequireScopeRejectsExtensionWithNoScopes(t *testing.T) {
	c := newScopeTestContext(t)
	c.Set(ContextAuthClientTypeKey, domain.AuthClientTypeExtension)

	err := callRequireScope(t, c, domain.ExtensionScopeHighlightWrite)

	requireScopeHTTPError(t, err, http.StatusForbidden)
}

func TestRequireScopeTrimsRequiredScope(t *testing.T) {
	c := newScopeTestContext(t)
	c.Set(ContextAuthClientTypeKey, domain.AuthClientTypeExtension)
	c.Set(ContextAuthScopesKey, []string{domain.ExtensionScopeHighlightWrite})

	if err := callRequireScope(t, c, "  "+domain.ExtensionScopeHighlightWrite+"  "); err != nil {
		t.Fatalf("required scope should be trimmed before comparison, got %v", err)
	}
}
