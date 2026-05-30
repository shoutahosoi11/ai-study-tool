package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/shout/ai-study-tool/backend/internal/domain"
	"github.com/shout/ai-study-tool/backend/internal/middleware"
)

const testAuthHandlerCSRFSecret = "auth-handler-csrf-secret"

type stubSessionCookieManager struct {
	verifyIDTokenFunc                 func(ctx context.Context, idToken string) (*domain.AuthToken, error)
	createSessionCookieFunc           func(ctx context.Context, idToken string, expiresIn time.Duration) (string, error)
	verifySessionCookieAndRevokedFunc func(ctx context.Context, sessionCookie string) (*domain.AuthToken, error)
	revokeRefreshTokensFunc           func(ctx context.Context, uid string) error
}

func (s *stubSessionCookieManager) VerifyIDToken(ctx context.Context, idToken string) (*domain.AuthToken, error) {
	return s.verifyIDTokenFunc(ctx, idToken)
}

func (s *stubSessionCookieManager) CreateSessionCookie(ctx context.Context, idToken string, expiresIn time.Duration) (string, error) {
	return s.createSessionCookieFunc(ctx, idToken, expiresIn)
}

func (s *stubSessionCookieManager) VerifySessionCookieAndCheckRevoked(ctx context.Context, sessionCookie string) (*domain.AuthToken, error) {
	return s.verifySessionCookieAndRevokedFunc(ctx, sessionCookie)
}

func (s *stubSessionCookieManager) RevokeRefreshTokens(ctx context.Context, uid string) error {
	return s.revokeRefreshTokensFunc(ctx, uid)
}

func TestCreateSessionIssuesSessionAndCSRFCookies(t *testing.T) {
	now := time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC)
	manager := &stubSessionCookieManager{
		verifyIDTokenFunc: func(ctx context.Context, idToken string) (*domain.AuthToken, error) {
			if idToken != "id-token" {
				t.Fatalf("unexpected id token: %s", idToken)
			}
			return &domain.AuthToken{UID: "firebase-uid-1", AuthTime: now.Add(-time.Minute)}, nil
		},
		createSessionCookieFunc: func(ctx context.Context, idToken string, expiresIn time.Duration) (string, error) {
			if expiresIn != sessionCookieLifetime {
				t.Fatalf("unexpected expiresIn: %s", expiresIn)
			}
			return "session-cookie", nil
		},
	}
	handler := NewAuthHandler(manager, "production", "")
	handler.now = func() time.Time { return now }
	handler.csrfSecret = testAuthHandlerCSRFSecret
	handler.randomToken = func() (string, error) { return "csrf-raw", nil }
	expectedCSRFToken, err := middleware.SignCSRFToken(testAuthHandlerCSRFSecret, "firebase-uid-1", "csrf-raw")
	if err != nil {
		t.Fatalf("unexpected csrf sign error: %v", err)
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/session", strings.NewReader(`{"id_token":"id-token"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := handler.CreateSession(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rec.Code)
	}

	cookies := rec.Result().Cookies()
	sessionCookie := findCookie(cookies, middleware.SessionCookieName("production"))
	if sessionCookie == nil {
		t.Fatal("expected session cookie")
	}
	if sessionCookie.Value != "session-cookie" || !sessionCookie.HttpOnly || !sessionCookie.Secure || sessionCookie.Path != "/" {
		t.Fatalf("unexpected session cookie: %#v", sessionCookie)
	}
	csrfCookie := findCookie(cookies, middleware.CSRFCookieName())
	if csrfCookie == nil {
		t.Fatal("expected csrf cookie")
	}
	if csrfCookie.Value != expectedCSRFToken || csrfCookie.HttpOnly || !csrfCookie.Secure || csrfCookie.Path != "/" {
		t.Fatalf("unexpected csrf cookie: %#v", csrfCookie)
	}
	if !strings.Contains(rec.Body.String(), `"csrf_token":"`+expectedCSRFToken+`"`) || !strings.Contains(rec.Body.String(), `"uid":"firebase-uid-1"`) {
		t.Fatalf("unexpected response body: %s", rec.Body.String())
	}
}

func TestCreateSessionRejectsInvalidIDToken(t *testing.T) {
	invalidErr := errors.New("invalid token")
	manager := &stubSessionCookieManager{
		verifyIDTokenFunc: func(ctx context.Context, idToken string) (*domain.AuthToken, error) {
			return nil, invalidErr
		},
	}
	handler := NewAuthHandler(manager, "development", "")
	handler.idTokenError = func(err error) bool {
		return err == invalidErr
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/session", strings.NewReader(`{"id_token":"bad"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.CreateSession(c)
	assertHTTPErrorCode(t, err, http.StatusUnauthorized)
}

func TestCreateSessionReturnsServiceUnavailableForVerifierFailure(t *testing.T) {
	manager := &stubSessionCookieManager{
		verifyIDTokenFunc: func(ctx context.Context, idToken string) (*domain.AuthToken, error) {
			return nil, errors.New("firebase unavailable")
		},
	}
	handler := NewAuthHandler(manager, "development", "")

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/session", strings.NewReader(`{"id_token":"id-token"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.CreateSession(c)
	assertHTTPErrorCode(t, err, http.StatusServiceUnavailable)
}

func TestCreateSessionRejectsOldAuthTime(t *testing.T) {
	now := time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC)
	manager := &stubSessionCookieManager{
		verifyIDTokenFunc: func(ctx context.Context, idToken string) (*domain.AuthToken, error) {
			return &domain.AuthToken{UID: "firebase-uid-1", AuthTime: now.Add(-6 * time.Minute)}, nil
		},
	}
	handler := NewAuthHandler(manager, "development", "")
	handler.now = func() time.Time { return now }

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/session", strings.NewReader(`{"id_token":"id-token"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.CreateSession(c)
	assertHTTPErrorCode(t, err, http.StatusUnauthorized)
}

func TestCreateSessionFailsClosedForInvalidTokenResult(t *testing.T) {
	now := time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC)
	testCases := []struct {
		name  string
		token *domain.AuthToken
	}{
		{name: "NilToken"},
		{name: "EmptyUID", token: &domain.AuthToken{AuthTime: now}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			manager := &stubSessionCookieManager{
				verifyIDTokenFunc: func(ctx context.Context, idToken string) (*domain.AuthToken, error) {
					return tc.token, nil
				},
			}
			handler := NewAuthHandler(manager, "development", "")
			handler.now = func() time.Time { return now }

			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/session", strings.NewReader(`{"id_token":"id-token"}`))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := handler.CreateSession(c)
			assertHTTPErrorCode(t, err, http.StatusServiceUnavailable)
		})
	}
}

func TestCreateSessionReturnsServiceUnavailableWhenCookieOrCSRFGenerationFails(t *testing.T) {
	now := time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC)
	testCases := []struct {
		name                    string
		createSessionCookieFunc func(ctx context.Context, idToken string, expiresIn time.Duration) (string, error)
		randomToken             func() (string, error)
	}{
		{
			name: "SessionCookieFailure",
			createSessionCookieFunc: func(ctx context.Context, idToken string, expiresIn time.Duration) (string, error) {
				return "", errors.New("firebase unavailable")
			},
			randomToken: func() (string, error) { return "csrf-token", nil },
		},
		{
			name: "CSRFGenerationFailure",
			createSessionCookieFunc: func(ctx context.Context, idToken string, expiresIn time.Duration) (string, error) {
				return "session-cookie", nil
			},
			randomToken: func() (string, error) { return "", errors.New("entropy unavailable") },
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			manager := &stubSessionCookieManager{
				verifyIDTokenFunc: func(ctx context.Context, idToken string) (*domain.AuthToken, error) {
					return &domain.AuthToken{UID: "firebase-uid-1", AuthTime: now.Add(-time.Minute)}, nil
				},
				createSessionCookieFunc: tc.createSessionCookieFunc,
			}
			handler := NewAuthHandler(manager, "development", "")
			handler.now = func() time.Time { return now }
			handler.csrfSecret = testAuthHandlerCSRFSecret
			handler.randomToken = tc.randomToken

			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/session", strings.NewReader(`{"id_token":"id-token"}`))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := handler.CreateSession(c)
			assertHTTPErrorCode(t, err, http.StatusServiceUnavailable)
		})
	}
}

func TestRefreshRejectsIDTokenUIDMismatch(t *testing.T) {
	now := time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC)
	manager := &stubSessionCookieManager{
		verifyIDTokenFunc: func(ctx context.Context, idToken string) (*domain.AuthToken, error) {
			return &domain.AuthToken{UID: "other-uid", AuthTime: now.Add(-time.Minute)}, nil
		},
	}
	handler := NewAuthHandler(manager, "development", "")
	handler.now = func() time.Time { return now }

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", strings.NewReader(`{"id_token":"id-token"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.AddCookie(&http.Cookie{Name: middleware.SessionCookieName("development"), Value: "session-cookie"})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.ContextFirebaseUIDKey, "session-uid")

	err := handler.Refresh(c)
	assertHTTPErrorCode(t, err, http.StatusUnauthorized)
}

func TestRefreshRequiresSessionAuthContext(t *testing.T) {
	handler := NewAuthHandler(&stubSessionCookieManager{}, "development", "")

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", strings.NewReader(`{"id_token":"id-token"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.Refresh(c)
	assertHTTPErrorCode(t, err, http.StatusUnauthorized)
}

func TestLogoutClearsCookies(t *testing.T) {
	handler := NewAuthHandler(&stubSessionCookieManager{}, "development", "")

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := handler.Logout(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("unexpected status: %d", rec.Code)
	}

	sessionCookie := findCookie(rec.Result().Cookies(), middleware.SessionCookieName("development"))
	if sessionCookie == nil || sessionCookie.MaxAge != -1 {
		t.Fatalf("expected deleted session cookie, got %#v", sessionCookie)
	}
	csrfCookie := findCookie(rec.Result().Cookies(), middleware.CSRFCookieName())
	if csrfCookie == nil || csrfCookie.MaxAge != -1 {
		t.Fatalf("expected deleted csrf cookie, got %#v", csrfCookie)
	}
}

func TestLogoutDoesNotRevokeRefreshTokens(t *testing.T) {
	revokeCalled := false
	manager := &stubSessionCookieManager{
		revokeRefreshTokensFunc: func(ctx context.Context, uid string) error {
			revokeCalled = true
			return nil
		},
	}
	handler := NewAuthHandler(manager, "development", "")

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.ContextFirebaseUIDKey, "firebase-uid-1")

	if err := handler.Logout(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if revokeCalled {
		t.Fatal("normal logout must only clear current browser cookies; logout-all performs Firebase revocation")
	}
}

func TestLogoutAllRevokesRefreshTokens(t *testing.T) {
	revokedUID := ""
	manager := &stubSessionCookieManager{
		revokeRefreshTokensFunc: func(ctx context.Context, uid string) error {
			revokedUID = uid
			return nil
		},
	}
	handler := NewAuthHandler(manager, "development", "")

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout-all", nil)
	req.AddCookie(&http.Cookie{Name: middleware.SessionCookieName("development"), Value: "session-cookie"})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.ContextFirebaseUIDKey, "firebase-uid-1")

	if err := handler.LogoutAll(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if revokedUID != "firebase-uid-1" {
		t.Fatalf("unexpected revoked uid: %s", revokedUID)
	}
}

func TestLogoutAllRequiresSessionAuthContext(t *testing.T) {
	handler := NewAuthHandler(&stubSessionCookieManager{}, "development", "")

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout-all", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.LogoutAll(c)
	assertHTTPErrorCode(t, err, http.StatusUnauthorized)
}

func findCookie(cookies []*http.Cookie, name string) *http.Cookie {
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}

func assertHTTPErrorCode(t *testing.T, err error, want int) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error")
	}
	httpErr, ok := err.(*echo.HTTPError)
	if !ok {
		t.Fatalf("expected *echo.HTTPError, got %T", err)
	}
	if httpErr.Code != want {
		t.Fatalf("unexpected status: %d", httpErr.Code)
	}
}
