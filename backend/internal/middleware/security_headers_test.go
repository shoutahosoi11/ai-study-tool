package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestSecurityHeadersMiddlewareAddsPrimaryHeaders(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	middleware := NewSecurityHeadersMiddleware("production", "")
	handler := middleware.Secure(func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"ok": "true"})
	})

	if err := handler(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	headers := rec.Result().Header
	expected := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
		"Permissions-Policy":     "camera=(), microphone=(), geolocation=()",
	}
	for name, want := range expected {
		if got := headers.Get(name); got != want {
			t.Fatalf("%s = %q, want %q", name, got, want)
		}
	}
	if got := headers.Get("Strict-Transport-Security"); got == "" {
		t.Fatal("expected HSTS header in production")
	}

	csp := headers.Get("Content-Security-Policy")
	if csp == "" {
		t.Fatal("expected CSP header")
	}
	if strings.Contains(csp, "script-src *") {
		t.Fatalf("CSP allows wildcard scripts: %s", csp)
	}
	if strings.Contains(csp, "script-src 'unsafe-inline'") {
		t.Fatalf("CSP allows inline scripts: %s", csp)
	}
	if !strings.Contains(csp, "object-src 'none'") || !strings.Contains(csp, "frame-ancestors 'none'") {
		t.Fatalf("CSP is missing object/frame restrictions: %s", csp)
	}
}
