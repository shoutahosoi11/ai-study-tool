package middleware

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

const RecentAuthMaxAge = 5 * time.Minute

func RequireRecentAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return RequireRecentAuthFor(domain.AuthClientTypeWeb)(next)
}

func RequireRecentAuthFor(clientTypes ...domain.AuthClientType) echo.MiddlewareFunc {
	return requireRecentAuthWithClock(RecentAuthMaxAge, time.Now, clientTypes...)
}

func requireRecentAuthWithClock(maxAge time.Duration, now func() time.Time, clientTypes ...domain.AuthClientType) echo.MiddlewareFunc {
	requiredClientTypes := make(map[domain.AuthClientType]struct{}, len(clientTypes))
	for _, clientType := range clientTypes {
		if clientType != "" {
			requiredClientTypes[clientType] = struct{}{}
		}
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			clientType, hasClientType := GetAuthClientType(c)
			if hasClientType {
				if _, required := requiredClientTypes[clientType]; !required {
					return next(c)
				}
			}
			if hasClientType && clientType == domain.AuthClientTypeExtension {
				return next(c)
			}

			authTime, ok := GetAuthTime(c)
			if !ok {
				return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
			}

			if authTime.IsZero() || authTime.Before(now().UTC().Add(-maxAge)) {
				return echo.NewHTTPError(http.StatusUnauthorized, "recent_sign_in_required")
			}

			return next(c)
		}
	}
}
