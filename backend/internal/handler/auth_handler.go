package handler

import (
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	appconfig "github.com/shout/ai-study-tool/backend/internal/config"
	"github.com/shout/ai-study-tool/backend/internal/domain"
	"github.com/shout/ai-study-tool/backend/internal/middleware"
	appsession "github.com/shout/ai-study-tool/backend/internal/session"
)

const (
	sessionCookieLifetime = 14 * 24 * time.Hour
	maxRecentAuthAge      = 5 * time.Minute
)

type AuthHandler struct {
	sessionManager domain.SessionCookieManager
	appEnv         string
	cookieDomain   string
	csrfSecret     string
	csrfUnsigned   bool
	now            func() time.Time
	randomToken    func() (string, error)
	idTokenError   func(error) bool
}

type sessionRequest struct {
	IDToken string `json:"id_token"`
}

type sessionResponse struct {
	CSRFToken string `json:"csrf_token"`
	UID       string `json:"uid"`
}

func NewAuthHandler(sessionManager domain.SessionCookieManager, appEnv string, cookieDomain string) *AuthHandler {
	normalizedEnv := strings.TrimSpace(appEnv)
	return &AuthHandler{
		sessionManager: sessionManager,
		appEnv:         normalizedEnv,
		cookieDomain:   strings.TrimSpace(cookieDomain),
		csrfSecret:     strings.TrimSpace(os.Getenv("CSRF_SECRET")),
		csrfUnsigned:   appconfig.NormalizeAppEnv(normalizedEnv).AllowsDevBypass() && envBool("CSRF_SIGNING_DISABLED"),
		now:            time.Now,
		randomToken:    middleware.GenerateCSRFRawToken,
		idTokenError:   middleware.IsFirebaseIDTokenClientError,
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
	// Refresh intentionally requires both the existing Session Cookie context
	// and a fresh Firebase ID Token. Firebase Admin creates a Session Cookie
	// from an ID Token, and the 5-minute auth_time check below keeps this route
	// scoped to session re-issue immediately after re-authentication.
	currentUID, ok := middleware.GetFirebaseUID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	req := new(sessionRequest)
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	return h.issueSession(c, strings.TrimSpace(req.IDToken), currentUID)
}

func (h *AuthHandler) Logout(c echo.Context) error {
	h.clearAuthCookies(c)
	return c.NoContent(http.StatusNoContent)
}

func (h *AuthHandler) LogoutAll(c echo.Context) error {
	currentUID, ok := middleware.GetFirebaseUID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	if err := h.sessionManager.RevokeRefreshTokens(c.Request().Context(), currentUID); err != nil {
		middleware.RequestLogger(c).Error("auth_revoke_refresh_tokens_failed", "error", err.Error())
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
		middleware.RequestLogger(c).Error("auth_verified_token_missing_uid")
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
		middleware.RequestLogger(c).Error("auth_create_session_cookie_failed", "error", err.Error())
		return echo.NewHTTPError(http.StatusServiceUnavailable, "authentication service unavailable")
	}

	csrfRaw, err := h.randomToken()
	if err != nil {
		middleware.RequestLogger(c).Error("auth_csrf_token_generation_failed", "error", err.Error())
		return echo.NewHTTPError(http.StatusServiceUnavailable, "authentication service unavailable")
	}
	csrfToken := csrfRaw
	if !h.csrfUnsigned {
		signedToken, err := middleware.SignCSRFToken(h.csrfSecret, token.UID, csrfRaw)
		if err != nil {
			middleware.RequestLogger(c).Error("auth_csrf_token_sign_failed", "error", err.Error())
			return echo.NewHTTPError(http.StatusServiceUnavailable, "authentication service unavailable")
		}
		csrfToken = signedToken
	}

	h.setAuthCookies(c, sessionCookie, csrfToken)
	return c.JSON(http.StatusOK, sessionResponse{CSRFToken: csrfToken, UID: token.UID})
}

func (h *AuthHandler) setAuthCookies(c echo.Context, sessionValue string, csrfToken string) {
	secure := appsession.SecureCookie(h.appEnv)
	expires := h.now().Add(sessionCookieLifetime)
	c.SetCookie(&http.Cookie{
		Name:     appsession.CookieName(h.appEnv),
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
		Name:     appsession.CSRFCookieName,
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
	secure := appsession.SecureCookie(h.appEnv)
	expired := time.Unix(0, 0).UTC()
	c.SetCookie(&http.Cookie{
		Name:     appsession.CookieName(h.appEnv),
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
		Name:     appsession.CSRFCookieName,
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
	if appsession.CookieName(h.appEnv) == appsession.HostSessionCookieName {
		return ""
	}
	return h.cookieDomain
}

func (h *AuthHandler) idTokenHTTPError(err error) *echo.HTTPError {
	if h.idTokenError != nil && h.idTokenError(err) {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid id token")
	}
	slog.Error("auth_id_token_verify_failed", "error", err.Error())
	return echo.NewHTTPError(http.StatusServiceUnavailable, "authentication service unavailable")
}

func envBool(key string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
