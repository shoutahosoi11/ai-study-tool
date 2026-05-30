package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

func TestAppVersionRejectsInvalidPlatform(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set(appPlatformHeader, "web")
	req.Header.Set(appVersionHeader, "1.2.3")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(ContextAuthClientTypeKey, domain.AuthClientTypeMobile)

	middleware := NewAppVersionMiddleware(AppVersionConfig{RejectMissingMobileHeaders: true})
	err := middleware.Check(func(c echo.Context) error {
		t.Fatal("next handler should not be called")
		return nil
	})(c)
	assertMiddlewareHTTPErrorCode(t, err, http.StatusBadRequest)
}

func TestAppVersionRejectsMissingVersionWhenRequired(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set(appPlatformHeader, AppPlatformIOS)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(ContextAuthClientTypeKey, domain.AuthClientTypeMobile)

	middleware := NewAppVersionMiddleware(AppVersionConfig{RejectMissingMobileHeaders: true})
	err := middleware.Check(func(c echo.Context) error {
		t.Fatal("next handler should not be called")
		return nil
	})(c)
	assertMiddlewareHTTPErrorCode(t, err, http.StatusBadRequest)
}

func TestAppVersionAllowsMissingVersionOutsideProductionCompatibility(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(ContextAuthClientTypeKey, domain.AuthClientTypeMobile)

	middleware := NewAppVersionMiddleware(AppVersionConfig{RejectMissingMobileHeaders: false})
	if err := middleware.Check(func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAppVersionRejectsOldVersion(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set(appPlatformHeader, AppPlatformIOS)
	req.Header.Set(appVersionHeader, "1.9.9")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(ContextAuthClientTypeKey, domain.AuthClientTypeMobile)

	middleware := NewAppVersionMiddleware(AppVersionConfig{
		MinSupportedIOSVersion:     "2.0.0",
		MinSupportedAndroidVersion: "1.0.0",
		RejectMissingMobileHeaders: true,
	})
	if err := middleware.Check(func(c echo.Context) error {
		t.Fatal("next handler should not be called")
		return nil
	})(c); err != nil {
		httpErr, ok := err.(*echo.HTTPError)
		if ok {
			t.Fatalf("expected JSON response, got HTTP error: %v", httpErr)
		}
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusUpgradeRequired {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
	if body := rec.Body.String(); body != "{\"error\":\"upgrade_required\",\"minVersion\":\"2.0.0\",\"platform\":\"ios\"}\n" {
		t.Fatalf("unexpected body: %s", body)
	}
}

func TestAppVersionAllowsValidVersionAndStoresContext(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set(appPlatformHeader, AppPlatformAndroid)
	req.Header.Set(appVersionHeader, "2.1.0")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(ContextAuthClientTypeKey, domain.AuthClientTypeMobile)

	middleware := NewAppVersionMiddleware(AppVersionConfig{
		MinSupportedIOSVersion:     "2.0.0",
		MinSupportedAndroidVersion: "2.0.0",
		RejectMissingMobileHeaders: true,
	})
	if err := middleware.Check(func(c echo.Context) error {
		version, ok := GetAppVersion(c)
		if !ok || version != "2.1.0" {
			t.Fatalf("unexpected app version: %q", version)
		}
		platform, ok := GetAppPlatform(c)
		if !ok || platform != AppPlatformAndroid {
			t.Fatalf("unexpected platform: %q", platform)
		}
		return c.NoContent(http.StatusNoContent)
	})(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAppVersionSkipsNonMobileClients(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(ContextAuthClientTypeKey, domain.AuthClientTypeExtension)

	middleware := NewAppVersionMiddleware(AppVersionConfig{RejectMissingMobileHeaders: true})
	if err := middleware.Check(func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
