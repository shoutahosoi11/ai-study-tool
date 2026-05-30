package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

type stubRateLimitStore struct {
	current  int64
	exceeded bool
	err      error
	userID   string
	bucket   string
	limit    int64
}

func (s *stubRateLimitStore) IncrementAndCheck(ctx context.Context, userID, bucket string, limit int64) (int64, bool, error) {
	s.userID = userID
	s.bucket = bucket
	s.limit = limit
	if s.err != nil {
		return 0, false, s.err
	}
	return s.current, s.exceeded, nil
}

func TestRateLimitAllowsRequestWithinLimit(t *testing.T) {
	store := &stubRateLimitStore{current: 1}
	middleware, err := NewRateLimitMiddleware(store, "ingest", 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(ContextFirebaseUIDKey, "firebase-uid-1")

	handler := middleware.Limit(func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})

	if err := handler(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
	if store.userID != "firebase-uid-1" {
		t.Fatalf("unexpected user id: %s", store.userID)
	}
	if store.bucket != "ingest" {
		t.Fatalf("unexpected bucket: %s", store.bucket)
	}
	if store.limit != 100 {
		t.Fatalf("unexpected limit: %d", store.limit)
	}
}

func TestRateLimitRejectsExceededRequest(t *testing.T) {
	store := &stubRateLimitStore{current: 101, exceeded: true}
	middleware, err := NewRateLimitMiddleware(store, "ingest", 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	middleware.now = func() time.Time {
		return time.Date(2026, 5, 15, 23, 59, 0, 0, time.UTC)
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(ContextFirebaseUIDKey, "firebase-uid-1")

	handler := middleware.Limit(func(c echo.Context) error {
		t.Fatal("next handler should not be called")
		return nil
	})

	err = handler(c)
	if err == nil {
		t.Fatal("expected error")
	}
	httpErr, ok := err.(*echo.HTTPError)
	if !ok {
		t.Fatalf("expected *echo.HTTPError, got %T", err)
	}
	if httpErr.Code != http.StatusTooManyRequests {
		t.Fatalf("unexpected status: %d", httpErr.Code)
	}
	if rec.Header().Get(echo.HeaderRetryAfter) != "60" {
		t.Fatalf("unexpected Retry-After: %s", rec.Header().Get(echo.HeaderRetryAfter))
	}
}

func TestRateLimitReturnsServiceUnavailableOnStoreError(t *testing.T) {
	store := &stubRateLimitStore{err: errors.New("database unavailable")}
	middleware, err := NewRateLimitMiddleware(store, "ingest", 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(ContextFirebaseUIDKey, "firebase-uid-1")

	err = middleware.Limit(func(c echo.Context) error {
		t.Fatal("next handler should not be called")
		return nil
	})(c)
	if err == nil {
		t.Fatal("expected error")
	}
	httpErr, ok := err.(*echo.HTTPError)
	if !ok {
		t.Fatalf("expected *echo.HTTPError, got %T", err)
	}
	if httpErr.Code != http.StatusServiceUnavailable {
		t.Fatalf("unexpected status: %d", httpErr.Code)
	}
}

func TestShortWindowRateLimitUsesIdentifierAndMinuteBucket(t *testing.T) {
	store := &stubRateLimitStore{current: 1}
	middleware, err := NewShortWindowRateLimitMiddleware(store, "extension_pairing_start", 5, func(c echo.Context) string {
		return "ip:203.0.113.10"
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	middleware.now = func() time.Time {
		return time.Date(2026, 5, 27, 10, 11, 12, 0, time.UTC)
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err = middleware.Limit(func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.userID != "ip:203.0.113.10" {
		t.Fatalf("unexpected identifier: %s", store.userID)
	}
	if store.bucket != "extension_pairing_start:202605271011" {
		t.Fatalf("unexpected bucket: %s", store.bucket)
	}
	if store.limit != 5 {
		t.Fatalf("unexpected limit: %d", store.limit)
	}
}

func TestClientIPRateLimitIdentifierHashesRawIP(t *testing.T) {
	e := echo.New()
	e.IPExtractor = echo.ExtractIPDirect()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.RemoteAddr = "203.0.113.10:12345"
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	identifier := ClientIPRateLimitIdentifier(c)
	if identifier == "" || identifier == "ip:203.0.113.10" {
		t.Fatalf("expected hashed IP identifier, got %s", identifier)
	}
	if wantPrefix := "ip_hash:"; len(identifier) <= len(wantPrefix) || identifier[:len(wantPrefix)] != wantPrefix {
		t.Fatalf("unexpected identifier prefix: %s", identifier)
	}
	if identifier == ClientIPRateLimitIdentifier(c) {
		return
	}
	t.Fatal("expected stable hash for the same IP")
}

func TestShortWindowRateLimitRejectsExceededRequest(t *testing.T) {
	store := &stubRateLimitStore{current: 6, exceeded: true}
	middleware, err := NewShortWindowRateLimitMiddleware(store, "extension_pairing_start", 5, func(c echo.Context) string {
		return "ip:203.0.113.10"
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err = middleware.Limit(func(c echo.Context) error {
		t.Fatal("next handler should not be called")
		return nil
	})(c)
	if err == nil {
		t.Fatal("expected error")
	}
	httpErr, ok := err.(*echo.HTTPError)
	if !ok {
		t.Fatalf("expected *echo.HTTPError, got %T", err)
	}
	if httpErr.Code != http.StatusTooManyRequests {
		t.Fatalf("unexpected status: %d", httpErr.Code)
	}
	if rec.Header().Get(echo.HeaderRetryAfter) != "60" {
		t.Fatalf("unexpected Retry-After: %s", rec.Header().Get(echo.HeaderRetryAfter))
	}
}

func TestRequireClientTypeAllowsOnlyConfiguredTypes(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(ContextAuthClientTypeKey, domain.AuthClientTypeWeb)

	err := RequireClientType(domain.AuthClientTypeWeb)(func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
}

func TestRequireClientTypeRejectsOtherTypes(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(ContextAuthClientTypeKey, domain.AuthClientTypeMobile)

	err := RequireClientType(domain.AuthClientTypeWeb)(func(c echo.Context) error {
		t.Fatal("next handler should not be called")
		return nil
	})(c)
	if err == nil {
		t.Fatal("expected error")
	}
	httpErr, ok := err.(*echo.HTTPError)
	if !ok {
		t.Fatalf("expected *echo.HTTPError, got %T", err)
	}
	if httpErr.Code != http.StatusForbidden {
		t.Fatalf("unexpected status: %d", httpErr.Code)
	}
}
