package middleware

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

type ExtensionAuthMiddleware struct {
	store domain.ExtensionTokenRepository
	now   func() time.Time
}

func NewExtensionAuthMiddleware(store domain.ExtensionTokenRepository) (*ExtensionAuthMiddleware, error) {
	if store == nil {
		return nil, errors.New("extension auth middleware: store is nil")
	}

	return &ExtensionAuthMiddleware{
		store: store,
		now:   time.Now,
	}, nil
}

func (m *ExtensionAuthMiddleware) Authenticate(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		rawToken, ok := bearerToken(c)
		if !ok || !domain.IsExtensionRawToken(rawToken) {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid authorization header format")
		}

		tokenHash := domain.HashExtensionToken(rawToken)
		token, err := m.store.FindActiveByTokenHash(c.Request().Context(), tokenHash, m.now().UTC())
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid, expired, or revoked extension token")
			}
			log.Printf("extension auth error: %v", err)
			return echo.NewHTTPError(http.StatusServiceUnavailable, "authentication service unavailable")
		}
		if token == nil || token.FirebaseUID == "" || token.UserID == uuid.Nil {
			return echo.NewHTTPError(http.StatusServiceUnavailable, "authentication service unavailable")
		}

		setExtensionAuth(c, token)
		return next(c)
	}
}
