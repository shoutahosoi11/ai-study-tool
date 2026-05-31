package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

const ContextAdminIdentityKey = "admin_identity"

type AdminRoleStore interface {
	FindAdminIdentityByFirebaseUID(ctx context.Context, firebaseUID string) (*domain.AdminIdentity, error)
}

type AdminMiddleware struct {
	store AdminRoleStore
}

func NewAdminMiddleware(store AdminRoleStore) (*AdminMiddleware, error) {
	if store == nil {
		return nil, errors.New("admin middleware: store is nil")
	}
	return &AdminMiddleware{store: store}, nil
}

func (m *AdminMiddleware) RequireAdmin(next echo.HandlerFunc) echo.HandlerFunc {
	return m.RequireAdminRole(domain.AdminRoleViewer)(next)
}

func (m *AdminMiddleware) RequireAdminRole(required domain.AdminRole) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			clientType, ok := GetAuthClientType(c)
			if !ok {
				return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
			}
			if clientType != domain.AuthClientTypeWeb {
				return echo.NewHTTPError(http.StatusForbidden, "admin requires web session")
			}

			if admin, ok := GetAdminIdentity(c); ok {
				if !domain.AdminRoleAllows(admin.Role, required) {
					return echo.NewHTTPError(http.StatusForbidden, "admin role required")
				}
				return next(c)
			}

			firebaseUID, ok := GetFirebaseUID(c)
			if !ok {
				return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
			}

			admin, err := m.store.FindAdminIdentityByFirebaseUID(c.Request().Context(), firebaseUID)
			if err != nil {
				if errors.Is(err, domain.ErrNotFound) {
					return echo.NewHTTPError(http.StatusForbidden, "admin access required")
				}
				return echo.NewHTTPError(http.StatusServiceUnavailable, "admin authorization unavailable")
			}
			if admin == nil || admin.UserID == uuid.Nil || !domain.IsValidAdminRole(admin.Role) {
				return echo.NewHTTPError(http.StatusForbidden, "admin access required")
			}
			if !domain.AdminRoleAllows(admin.Role, required) {
				return echo.NewHTTPError(http.StatusForbidden, "admin role required")
			}

			c.Set(ContextAdminIdentityKey, *admin)
			return next(c)
		}
	}
}

func GetAdminIdentity(c echo.Context) (domain.AdminIdentity, bool) {
	admin, ok := c.Get(ContextAdminIdentityKey).(domain.AdminIdentity)
	return admin, ok && admin.UserID != uuid.Nil && strings.TrimSpace(string(admin.Role)) != ""
}
