package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

type HybridAuthMiddleware struct {
	sessionAuth *SessionAuthMiddleware
	bearerAuth  *FirebaseMiddleware
	csrf        *CSRFMiddleware
	appEnv      string
}

func NewHybridAuthMiddleware(
	sessionAuth *SessionAuthMiddleware,
	bearerAuth *FirebaseMiddleware,
	csrf *CSRFMiddleware,
	appEnv string,
) (*HybridAuthMiddleware, error) {
	if sessionAuth == nil {
		return nil, errors.New("hybrid auth middleware: session auth is nil")
	}
	if bearerAuth == nil {
		return nil, errors.New("hybrid auth middleware: bearer auth is nil")
	}
	if csrf == nil {
		return nil, errors.New("hybrid auth middleware: csrf is nil")
	}
	return &HybridAuthMiddleware{
		sessionAuth: sessionAuth,
		bearerAuth:  bearerAuth,
		csrf:        csrf,
		appEnv:      strings.TrimSpace(appEnv),
	}, nil
}

func (m *HybridAuthMiddleware) Authenticate(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if hasSessionCookie(c, m.appEnv) {
			return m.csrf.Protect(m.sessionAuth.Authenticate(next))(c)
		}

		if strings.TrimSpace(c.Request().Header.Get("Authorization")) != "" {
			return m.bearerAuth.Authenticate(next)(c)
		}

		return echo.NewHTTPError(http.StatusUnauthorized, "missing authentication credentials")
	}
}
