package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestCSRFSkipsGET(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: SessionCookieName("development"), Value: "session"})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	middleware := NewCSRFMiddleware("development")
	if err := middleware.Protect(func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCSRFPassesMatchingToken(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.AddCookie(&http.Cookie{Name: SessionCookieName("development"), Value: "session"})
	req.AddCookie(&http.Cookie{Name: CSRFCookieName(), Value: "csrf-token"})
	req.Header.Set("X-CSRF-Token", "csrf-token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	middleware := NewCSRFMiddleware("development")
	if err := middleware.Protect(func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCSRFRejectsMismatchedToken(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.AddCookie(&http.Cookie{Name: SessionCookieName("development"), Value: "session"})
	req.AddCookie(&http.Cookie{Name: CSRFCookieName(), Value: "csrf-token"})
	req.Header.Set("X-CSRF-Token", "other-token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	middleware := NewCSRFMiddleware("development")
	err := middleware.Protect(func(c echo.Context) error {
		t.Fatal("next handler should not be called")
		return nil
	})(c)
	assertMiddlewareHTTPErrorCode(t, err, http.StatusForbidden)
}

func TestCSRFRejectsMissingToken(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.AddCookie(&http.Cookie{Name: SessionCookieName("development"), Value: "session"})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	middleware := NewCSRFMiddleware("development")
	err := middleware.Protect(func(c echo.Context) error {
		t.Fatal("next handler should not be called")
		return nil
	})(c)
	assertMiddlewareHTTPErrorCode(t, err, http.StatusForbidden)
}
