package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

type HybridAuthMiddleware struct {
	sessionAuth *SessionAuthMiddleware
	bearerAuth  *FirebaseMiddleware
	appEnv      string
}

func NewHybridAuthMiddleware(sessionAuth *SessionAuthMiddleware, bearerAuth *FirebaseMiddleware, appEnv string) *HybridAuthMiddleware {
	return &HybridAuthMiddleware{
		sessionAuth: sessionAuth,
		bearerAuth:  bearerAuth,
		appEnv:      strings.TrimSpace(appEnv),
	}
}

func (m *HybridAuthMiddleware) Authenticate(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if hasSessionCookie(c, m.appEnv) {
			return m.sessionAuth.Authenticate(next)(c)
		}

		if strings.TrimSpace(c.Request().Header.Get("Authorization")) != "" {
			return m.bearerAuth.Authenticate(next)(c)
		}

		return echo.NewHTTPError(http.StatusUnauthorized, "missing authentication credentials")
	}
}
