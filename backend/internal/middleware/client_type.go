package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

func RequireClientType(allowedTypes ...domain.AuthClientType) echo.MiddlewareFunc {
	allowed := make(map[domain.AuthClientType]struct{}, len(allowedTypes))
	for _, clientType := range allowedTypes {
		allowed[clientType] = struct{}{}
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			clientType, ok := GetAuthClientType(c)
			if !ok {
				return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
			}
			if _, ok := allowed[clientType]; !ok {
				return echo.NewHTTPError(http.StatusForbidden, "client type not allowed")
			}
			return next(c)
		}
	}
}
