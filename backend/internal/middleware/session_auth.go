package middleware

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	firebaseauth "firebase.google.com/go/v4/auth"
	"github.com/labstack/echo/v4"
	"github.com/shout/ai-study-tool/backend/internal/domain"
	appsession "github.com/shout/ai-study-tool/backend/internal/session"
)

const (
	sessionCookieName = appsession.DevelopmentSessionCookieName
	hostSessionCookie = appsession.HostSessionCookieName
	csrfCookieName    = appsession.CSRFCookieName
)

var isFirebaseSessionCookieClientError = func(err error) bool {
	return firebaseauth.IsSessionCookieRevoked(err) ||
		firebaseauth.IsSessionCookieExpired(err) ||
		firebaseauth.IsSessionCookieInvalid(err) ||
		firebaseauth.IsUserDisabled(err)
}

func IsFirebaseSessionCookieClientError(err error) bool {
	return isFirebaseSessionCookieClientError(err)
}

type SessionAuthMiddleware struct {
	verifier domain.SessionVerifier
	appEnv   string
}

func NewSessionAuthMiddleware(verifier domain.SessionVerifier, appEnv string) (*SessionAuthMiddleware, error) {
	if verifier == nil {
		return nil, errors.New("session auth middleware: verifier is nil")
	}
	return &SessionAuthMiddleware{
		verifier: verifier,
		appEnv:   strings.TrimSpace(appEnv),
	}, nil
}

func (m *SessionAuthMiddleware) Authenticate(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		cookie, err := c.Cookie(SessionCookieName(m.appEnv))
		if err != nil || strings.TrimSpace(cookie.Value) == "" {
			return echo.NewHTTPError(http.StatusUnauthorized, "missing session cookie")
		}

		token, err := m.verifier.VerifySessionCookieAndCheckRevoked(c.Request().Context(), cookie.Value)
		if err != nil {
			return firebaseSessionCookieError(err)
		}
		if token == nil || token.UID == "" {
			return echo.NewHTTPError(http.StatusServiceUnavailable, "authentication service unavailable")
		}

		setSessionAuth(c, token)
		return next(c)
	}
}

func SessionCookieName(appEnv string) string {
	return appsession.CookieName(appEnv)
}

func CSRFCookieName() string {
	return csrfCookieName
}

func SecureCookie(appEnv string) bool {
	return appsession.SecureCookie(appEnv)
}

func firebaseSessionCookieError(err error) *echo.HTTPError {
	if isFirebaseSessionCookieClientError(err) {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid, expired, or revoked session")
	}
	slog.Error("firebase_session_cookie_verify_failed", "error", err.Error())
	return echo.NewHTTPError(http.StatusServiceUnavailable, "authentication service unavailable")
}
