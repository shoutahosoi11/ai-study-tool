package middleware

import (
	"crypto/hmac"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

type CSRFMiddleware struct {
	appEnv string
}

func NewCSRFMiddleware(appEnv string) *CSRFMiddleware {
	return &CSRFMiddleware{appEnv: strings.TrimSpace(appEnv)}
}

func (m *CSRFMiddleware) Protect(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		switch c.Request().Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			return next(c)
		}

		if c.Path() == "/api/v1/auth/session" {
			return next(c)
		}

		if !hasSessionCookie(c, m.appEnv) {
			return next(c)
		}

		headerToken := strings.TrimSpace(c.Request().Header.Get("X-CSRF-Token"))
		cookie, err := c.Cookie(CSRFCookieName())
		if err != nil || strings.TrimSpace(cookie.Value) == "" || headerToken == "" {
			return echo.NewHTTPError(http.StatusForbidden, "invalid csrf token")
		}

		if !hmac.Equal([]byte(headerToken), []byte(strings.TrimSpace(cookie.Value))) {
			return echo.NewHTTPError(http.StatusForbidden, "invalid csrf token")
		}

		return next(c)
	}
}

func hasSessionCookie(c echo.Context, appEnv string) bool {
	cookie, err := c.Cookie(SessionCookieName(appEnv))
	return err == nil && strings.TrimSpace(cookie.Value) != ""
}
