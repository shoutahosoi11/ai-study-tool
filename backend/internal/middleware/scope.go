package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

// RequireScope enforces scopes only for Browser Extension tokens.
// Web session and Mobile bearer clients are authorized by their existing route
// rules; extension access is narrowed here so extension tokens cannot reach
// routes outside their explicit scope set.
func RequireScope(scope string) echo.MiddlewareFunc {
	requiredScope := strings.TrimSpace(scope)
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			clientType, ok := GetAuthClientType(c)
			if !ok {
				return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
			}

			if clientType != domain.AuthClientTypeExtension {
				return next(c)
			}

			if !domain.HasScope(GetAuthScopes(c), requiredScope) {
				return echo.NewHTTPError(http.StatusForbidden, "insufficient scope")
			}

			return next(c)
		}
	}
}
