package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

const testCSRFSecret = "csrf-test-secret"

func TestCSRFSkipsGET(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: SessionCookieName("development"), Value: "session"})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	middleware := newTestCSRFMiddleware(t)
	if err := middleware.Protect(func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCSRFPassesSignedMatchingTokenAndSameOrigin(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	addValidCSRF(req, "firebase-uid-1")
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(ContextFirebaseUIDKey, "firebase-uid-1")

	middleware := newTestCSRFMiddleware(t)
	if err := middleware.Protect(func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCSRFPassesSameSiteFetchMetadata(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	addValidCSRF(req, "firebase-uid-1")
	req.Header.Set("Origin", "https://app.example.com")
	req.Header.Set("Sec-Fetch-Site", "same-site")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(ContextFirebaseUIDKey, "firebase-uid-1")

	middleware := newConfiguredCSRFMiddleware(t, []string{"https://app.example.com"}, false)
	if err := middleware.Protect(func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCSRFAllowsOriginWithTrailingSlash(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	addValidCSRF(req, "firebase-uid-1")
	req.Header.Set("Origin", "https://app.example.com/")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(ContextFirebaseUIDKey, "firebase-uid-1")

	middleware := newConfiguredCSRFMiddleware(t, []string{"https://app.example.com"}, false)
	if err := middleware.Protect(func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCSRFRejectsOriginWithPath(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	addValidCSRF(req, "firebase-uid-1")
	req.Header.Set("Origin", "https://app.example.com/path")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(ContextFirebaseUIDKey, "firebase-uid-1")

	middleware := newConfiguredCSRFMiddleware(t, []string{"https://app.example.com"}, false)
	err := middleware.Protect(func(c echo.Context) error {
		t.Fatal("next handler should not be called")
		return nil
	})(c)
	assertMiddlewareHTTPErrorCode(t, err, http.StatusForbidden)
}

func TestCSRFRejectsMismatchedCookieAndHeader(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	addValidCSRF(req, "firebase-uid-1")
	otherToken, err := SignCSRFToken(testCSRFSecret, "firebase-uid-1", "other-raw")
	if err != nil {
		t.Fatalf("unexpected csrf sign error: %v", err)
	}
	req.Header.Set("X-CSRF-Token", otherToken)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(ContextFirebaseUIDKey, "firebase-uid-1")

	middleware := newTestCSRFMiddleware(t)
	err = middleware.Protect(func(c echo.Context) error {
		t.Fatal("next handler should not be called")
		return nil
	})(c)
	assertMiddlewareHTTPErrorCode(t, err, http.StatusForbidden)
}

func TestCSRFRejectsMissingToken(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.AddCookie(&http.Cookie{Name: SessionCookieName("development"), Value: "session"})
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(ContextFirebaseUIDKey, "firebase-uid-1")

	middleware := newTestCSRFMiddleware(t)
	err := middleware.Protect(func(c echo.Context) error {
		t.Fatal("next handler should not be called")
		return nil
	})(c)
	assertMiddlewareHTTPErrorCode(t, err, http.StatusForbidden)
}

func TestCSRFRejectsInvalidHMAC(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	token := "csrf-raw.invalid-signature"
	req.AddCookie(&http.Cookie{Name: CSRFCookieName(), Value: token})
	req.Header.Set("X-CSRF-Token", token)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(ContextFirebaseUIDKey, "firebase-uid-1")

	middleware := newTestCSRFMiddleware(t)
	err := middleware.Protect(func(c echo.Context) error {
		t.Fatal("next handler should not be called")
		return nil
	})(c)
	assertMiddlewareHTTPErrorCode(t, err, http.StatusForbidden)
}

func TestCSRFRejectsCrossSiteOrigin(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	addValidCSRF(req, "firebase-uid-1")
	req.Header.Set("Origin", "https://evil.example.com")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(ContextFirebaseUIDKey, "firebase-uid-1")

	middleware := newTestCSRFMiddleware(t)
	err := middleware.Protect(func(c echo.Context) error {
		t.Fatal("next handler should not be called")
		return nil
	})(c)
	assertMiddlewareHTTPErrorCode(t, err, http.StatusForbidden)
}

func TestCSRFRejectsCrossSiteFetchMetadata(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	addValidCSRF(req, "firebase-uid-1")
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(ContextFirebaseUIDKey, "firebase-uid-1")

	middleware := newTestCSRFMiddleware(t)
	err := middleware.Protect(func(c echo.Context) error {
		t.Fatal("next handler should not be called")
		return nil
	})(c)
	assertMiddlewareHTTPErrorCode(t, err, http.StatusForbidden)
}

func TestCSRFRejectsNoneFetchMetadataForUnsafeMethod(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	addValidCSRF(req, "firebase-uid-1")
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Sec-Fetch-Site", "none")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(ContextFirebaseUIDKey, "firebase-uid-1")

	middleware := newTestCSRFMiddleware(t)
	err := middleware.Protect(func(c echo.Context) error {
		t.Fatal("next handler should not be called")
		return nil
	})(c)
	assertMiddlewareHTTPErrorCode(t, err, http.StatusForbidden)
}

func TestCSRFRejectsMissingOriginAndFetchMetadataInProduction(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	addValidCSRF(req, "firebase-uid-1")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(ContextFirebaseUIDKey, "firebase-uid-1")

	middleware := newConfiguredCSRFMiddleware(t, []string{"http://localhost:3000"}, false)
	err := middleware.Protect(func(c echo.Context) error {
		t.Fatal("next handler should not be called")
		return nil
	})(c)
	assertMiddlewareHTTPErrorCode(t, err, http.StatusForbidden)
}

func TestNewCSRFMiddlewareWithConfigRequiresSecretInProduction(t *testing.T) {
	middleware, err := NewCSRFMiddlewareWithConfig(CSRFConfig{
		AppEnv:         "production",
		AllowedOrigins: []string{"https://app.example.com"},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if middleware != nil {
		t.Fatal("expected nil middleware")
	}
}

func TestNewCSRFMiddlewareWithConfigRejectsSigningDisabledInProduction(t *testing.T) {
	middleware, err := NewCSRFMiddlewareWithConfig(CSRFConfig{
		AppEnv:          "production",
		Secret:          testCSRFSecret,
		AllowedOrigins:  []string{"https://app.example.com"},
		SigningDisabled: true,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if middleware != nil {
		t.Fatal("expected nil middleware")
	}
}

func TestNewCSRFMiddlewareWithConfigRejectsSigningDisabledInStaging(t *testing.T) {
	middleware, err := NewCSRFMiddlewareWithConfig(CSRFConfig{
		AppEnv:          "staging",
		Secret:          testCSRFSecret,
		AllowedOrigins:  []string{"https://app.example.com"},
		SigningDisabled: true,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if middleware != nil {
		t.Fatal("expected nil middleware")
	}
}

func TestNewCSRFMiddlewareWithConfigAllowsSigningDisabledOutsideProduction(t *testing.T) {
	middleware, err := NewCSRFMiddlewareWithConfig(CSRFConfig{
		AppEnv:                      "development",
		AllowedOrigins:              []string{"http://localhost:3000"},
		AllowMissingSecurityHeaders: true,
		SigningDisabled:             true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if middleware == nil {
		t.Fatal("expected middleware")
	}
	if !middleware.signingDisabled {
		t.Fatal("expected csrf signing disabled")
	}
}

func TestNewCSRFMiddlewareFromEnvRejectsSigningDisabledInProduction(t *testing.T) {
	t.Setenv("CSRF_SECRET", testCSRFSecret)
	t.Setenv("ALLOWED_ORIGINS", "https://app.example.com")
	t.Setenv("CSRF_SIGNING_DISABLED", "true")

	middleware, err := NewCSRFMiddlewareFromEnv("production")
	if err == nil {
		t.Fatal("expected error")
	}
	if middleware != nil {
		t.Fatal("expected nil middleware")
	}
}

func newTestCSRFMiddleware(t *testing.T) *CSRFMiddleware {
	t.Helper()
	return newConfiguredCSRFMiddleware(t, []string{"http://localhost:3000"}, false)
}

func newConfiguredCSRFMiddleware(t *testing.T, origins []string, allowMissingSecurityHeaders bool) *CSRFMiddleware {
	t.Helper()
	middleware, err := NewCSRFMiddlewareWithConfig(CSRFConfig{
		AppEnv:                      "test",
		Secret:                      testCSRFSecret,
		AllowedOrigins:              origins,
		AllowMissingSecurityHeaders: allowMissingSecurityHeaders,
	})
	if err != nil {
		t.Fatalf("unexpected csrf middleware error: %v", err)
	}
	return middleware
}

func addValidCSRF(req *http.Request, uid string) string {
	token, err := SignCSRFToken(testCSRFSecret, uid, "csrf-raw")
	if err != nil {
		panic(err)
	}
	req.AddCookie(&http.Cookie{Name: CSRFCookieName(), Value: token})
	req.Header.Set("X-CSRF-Token", token)
	return token
}
