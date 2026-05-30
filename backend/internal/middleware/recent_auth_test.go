package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"firebase.google.com/go/v4/auth"
	"github.com/labstack/echo/v4"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

func TestRequireRecentAuthRejectsOldAuthTime(t *testing.T) {
	now := time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC)
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(ContextFirebaseTokenKey, &domain.AuthToken{
		UID:      "firebase-uid-1",
		AuthTime: now.Add(-6 * time.Minute),
	})
	c.Set(ContextAuthClientTypeKey, domain.AuthClientTypeWeb)

	err := requireRecentAuthWithClock(RecentAuthMaxAge, func() time.Time { return now }, domain.AuthClientTypeWeb)(func(c echo.Context) error {
		t.Fatal("next handler should not be called")
		return nil
	})(c)
	assertMiddlewareHTTPErrorCode(t, err, http.StatusUnauthorized)
}

func TestRequireRecentAuthPassesFreshAuthTime(t *testing.T) {
	now := time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC)
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(ContextFirebaseTokenKey, &domain.AuthToken{
		UID:      "firebase-uid-1",
		AuthTime: now.Add(-4 * time.Minute),
	})
	c.Set(ContextAuthClientTypeKey, domain.AuthClientTypeWeb)

	if err := requireRecentAuthWithClock(RecentAuthMaxAge, func() time.Time { return now }, domain.AuthClientTypeWeb)(func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRequireRecentAuthSkipsNonWebClients(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(ContextAuthClientTypeKey, domain.AuthClientTypeMobile)

	if err := requireRecentAuthWithClock(RecentAuthMaxAge, time.Now, domain.AuthClientTypeWeb)(func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRequireRecentAuthSkipsExtensionClients(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(ContextAuthClientTypeKey, domain.AuthClientTypeExtension)

	if err := requireRecentAuthWithClock(RecentAuthMaxAge, time.Now, domain.AuthClientTypeWeb)(func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetAuthTimeReadsFirebaseTokenForMobileFoundation(t *testing.T) {
	authTime := time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC)
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(ContextFirebaseTokenKey, &auth.Token{
		UID:      "firebase-uid-1",
		AuthTime: authTime.Unix(),
	})
	c.Set(ContextAuthClientTypeKey, domain.AuthClientTypeMobile)

	got, ok := GetAuthTime(c)
	if !ok {
		t.Fatal("expected auth time")
	}
	if !got.Equal(authTime) {
		t.Fatalf("unexpected auth time: %s", got)
	}
}

func TestRequireRecentAuthCanOptIntoMobileAuthTime(t *testing.T) {
	now := time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC)
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(ContextFirebaseTokenKey, &auth.Token{
		UID:      "firebase-uid-1",
		AuthTime: now.Add(-4 * time.Minute).Unix(),
	})
	c.Set(ContextAuthClientTypeKey, domain.AuthClientTypeMobile)

	if err := requireRecentAuthWithClock(RecentAuthMaxAge, func() time.Time { return now }, domain.AuthClientTypeMobile)(func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRequireRecentAuthRejectsOldMobileAuthTimeWhenOptedIn(t *testing.T) {
	now := time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC)
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(ContextFirebaseTokenKey, &auth.Token{
		UID:      "firebase-uid-1",
		AuthTime: now.Add(-6 * time.Minute).Unix(),
	})
	c.Set(ContextAuthClientTypeKey, domain.AuthClientTypeMobile)

	err := requireRecentAuthWithClock(RecentAuthMaxAge, func() time.Time { return now }, domain.AuthClientTypeMobile)(func(c echo.Context) error {
		t.Fatal("next handler should not be called")
		return nil
	})(c)
	assertMiddlewareHTTPErrorCode(t, err, http.StatusUnauthorized)
}
