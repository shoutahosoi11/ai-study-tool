package handler

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/shout/ai-study-tool/backend/internal/domain"
	"github.com/shout/ai-study-tool/backend/internal/middleware"
)

const (
	sessionCookieLifetime = 14 * 24 * time.Hour
	maxRecentAuthAge      = 5 * time.Minute
	csrfTokenBytes        = 32
)

type AuthHandler struct {
	sessionManager domain.SessionCookieManager
	appEnv         string
	cookieDomain   string
	now            func() time.Time
	randomToken    func() (string, error)
	idTokenError   func(error) bool
	sessionError   func(error) bool
}

type sessionRequest struct {
	IDToken string `json:"id_token"`
}

type sessionResponse struct {
	CSRFToken string `json:"csrf_token"`
	UID       string `json:"uid"`
}

func NewAuthHandler(sessionManager domain.SessionCookieManager, appEnv string, cookieDomain string) *AuthHandler {
	return &AuthHandler{
		sessionManager: sessionManager,
		appEnv:         strings.TrimSpace(appEnv),
		cookieDomain:   strings.TrimSpace(cookieDomain),
		now:            time.Now,
		randomToken:    generateCSRFToken,
		idTokenError:   middleware.IsFirebaseIDTokenClientError,
		sessionError:   middleware.IsFirebaseSessionCookieClientError,
	}
}

func (h *AuthHandler) CreateSession(c echo.Context) error {
	req := new(sessionRequest)
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	return h.issueSession(c, strings.TrimSpace(req.IDToken), "")
}

func (h *AuthHandler) Refresh(c echo.Context) error {
	sessionCookie, err := c.Cookie(middleware.SessionCookieName(h.appEnv))
	if err != nil || strings.TrimSpace(sessionCookie.Value) == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "missing session cookie")
	}
	currentToken, err := h.sessionManager.VerifySessionCookieAndCheckRevoked(c.Request().Context(), sessionCookie.Value)
	if err != nil {
		return h.sessionAuthHTTPError(err)
	}
	if currentToken == nil || currentToken.UID == "" {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "authentication service unavailable")
	}

	req := new(sessionRequest)
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	return h.issueSession(c, strings.TrimSpace(req.IDToken), currentToken.UID)
}

func (h *AuthHandler) Logout(c echo.Context) error {
	h.clearAuthCookies(c)
	return c.NoContent(http.StatusNoContent)
}

func (h *AuthHandler) LogoutAll(c echo.Context) error {
	sessionCookie, err := c.Cookie(middleware.SessionCookieName(h.appEnv))
	if err != nil || strings.TrimSpace(sessionCookie.Value) == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "missing session cookie")
	}

	token, err := h.sessionManager.VerifySessionCookieAndCheckRevoked(c.Request().Context(), sessionCookie.Value)
	if err != nil {
		return h.sessionAuthHTTPError(err)
	}
	if token == nil || token.UID == "" {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "authentication service unavailable")
	}

	if err := h.sessionManager.RevokeRefreshTokens(c.Request().Context(), token.UID); err != nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "authentication service unavailable")
	}

	h.clearAuthCookies(c)
	return c.NoContent(http.StatusNoContent)
}

func (h *AuthHandler) issueSession(c echo.Context, idToken string, expectedUID string) error {
	if idToken == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid id token")
	}

	token, err := h.sessionManager.VerifyIDToken(c.Request().Context(), idToken)
	if err != nil {
		return h.idTokenHTTPError(err)
	}
	if token == nil || token.UID == "" {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "authentication service unavailable")
	}
	if expectedUID != "" && token.UID != expectedUID {
		return echo.NewHTTPError(http.StatusUnauthorized, "id token user mismatch")
	}
	if token.AuthTime.IsZero() || token.AuthTime.Before(h.now().Add(-maxRecentAuthAge)) {
		return echo.NewHTTPError(http.StatusUnauthorized, "recent sign-in required")
	}

	sessionCookie, err := h.sessionManager.CreateSessionCookie(c.Request().Context(), idToken, sessionCookieLifetime)
	if err != nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "authentication service unavailable")
	}

	csrfToken, err := h.randomToken()
	if err != nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "authentication service unavailable")
	}

	h.setAuthCookies(c, sessionCookie, csrfToken)
	return c.JSON(http.StatusOK, sessionResponse{CSRFToken: csrfToken, UID: token.UID})
}

func (h *AuthHandler) setAuthCookies(c echo.Context, sessionValue string, csrfToken string) {
	secure := middleware.SecureCookie(h.appEnv)
	expires := h.now().Add(sessionCookieLifetime)
	c.SetCookie(&http.Cookie{
		Name:     middleware.SessionCookieName(h.appEnv),
		Value:    sessionValue,
		Path:     "/",
		Domain:   h.effectiveCookieDomain(),
		Expires:  expires,
		MaxAge:   int(sessionCookieLifetime.Seconds()),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
	c.SetCookie(&http.Cookie{
		Name:     middleware.CSRFCookieName(),
		Value:    csrfToken,
		Path:     "/",
		Domain:   h.effectiveCookieDomain(),
		Expires:  expires,
		MaxAge:   int(sessionCookieLifetime.Seconds()),
		HttpOnly: false,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *AuthHandler) clearAuthCookies(c echo.Context) {
	secure := middleware.SecureCookie(h.appEnv)
	expired := time.Unix(0, 0).UTC()
	c.SetCookie(&http.Cookie{
		Name:     middleware.SessionCookieName(h.appEnv),
		Value:    "",
		Path:     "/",
		Domain:   h.effectiveCookieDomain(),
		Expires:  expired,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
	c.SetCookie(&http.Cookie{
		Name:     middleware.CSRFCookieName(),
		Value:    "",
		Path:     "/",
		Domain:   h.effectiveCookieDomain(),
		Expires:  expired,
		MaxAge:   -1,
		HttpOnly: false,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *AuthHandler) effectiveCookieDomain() string {
	if middleware.SessionCookieName(h.appEnv) == "__Host-session" {
		return ""
	}
	return h.cookieDomain
}

func generateCSRFToken() (string, error) {
	buf := make([]byte, csrfTokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate csrf token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func (h *AuthHandler) idTokenHTTPError(err error) *echo.HTTPError {
	if h.idTokenError != nil && h.idTokenError(err) {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid id token")
	}
	return echo.NewHTTPError(http.StatusServiceUnavailable, "authentication service unavailable")
}

func (h *AuthHandler) sessionAuthHTTPError(err error) *echo.HTTPError {
	if h.sessionError != nil && h.sessionError(err) {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid, expired, or revoked session")
	}
	return echo.NewHTTPError(http.StatusServiceUnavailable, "authentication service unavailable")
}
