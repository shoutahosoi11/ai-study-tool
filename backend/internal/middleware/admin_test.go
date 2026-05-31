package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

type stubAdminRoleStore struct {
	findFunc func(ctx context.Context, firebaseUID string) (*domain.AdminIdentity, error)
}

func (s stubAdminRoleStore) FindAdminIdentityByFirebaseUID(ctx context.Context, firebaseUID string) (*domain.AdminIdentity, error) {
	return s.findFunc(ctx, firebaseUID)
}

func TestAdminMiddlewareAllowsAdminWebSession(t *testing.T) {
	adminID := uuid.New()
	middleware, err := NewAdminMiddleware(stubAdminRoleStore{
		findFunc: func(ctx context.Context, firebaseUID string) (*domain.AdminIdentity, error) {
			if firebaseUID != "firebase-admin" {
				t.Fatalf("unexpected firebase uid: %s", firebaseUID)
			}
			return &domain.AdminIdentity{UserID: adminID, Role: domain.AdminRoleAdmin}, nil
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	c, rec := newAdminMiddlewareContext()
	setSessionAuth(c, &domain.AuthToken{UID: "firebase-admin"})
	err = middleware.RequireAdmin(func(c echo.Context) error {
		admin, ok := GetAdminIdentity(c)
		if !ok || admin.UserID != adminID || admin.Role != domain.AdminRoleAdmin {
			t.Fatalf("unexpected admin identity: %#v", admin)
		}
		return c.NoContent(http.StatusNoContent)
	})(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
}

func TestAdminMiddlewareRejectsNonAdminWebSession(t *testing.T) {
	middleware, err := NewAdminMiddleware(stubAdminRoleStore{
		findFunc: func(ctx context.Context, firebaseUID string) (*domain.AdminIdentity, error) {
			return nil, domain.ErrNotFound
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	c, _ := newAdminMiddlewareContext()
	setSessionAuth(c, &domain.AuthToken{UID: "firebase-user"})
	err = middleware.RequireAdmin(func(c echo.Context) error {
		t.Fatal("next handler should not be called")
		return nil
	})(c)
	assertMiddlewareHTTPErrorCode(t, err, http.StatusForbidden)
}

func TestAdminMiddlewareRejectsExtensionAndMobileClients(t *testing.T) {
	middleware, err := NewAdminMiddleware(stubAdminRoleStore{
		findFunc: func(ctx context.Context, firebaseUID string) (*domain.AdminIdentity, error) {
			t.Fatal("store should not be called for non-web clients")
			return nil, nil
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, clientType := range []domain.AuthClientType{domain.AuthClientTypeExtension, domain.AuthClientTypeMobile} {
		t.Run(string(clientType), func(t *testing.T) {
			c, _ := newAdminMiddlewareContext()
			c.Set(ContextFirebaseUIDKey, "firebase-user")
			c.Set(ContextAuthClientTypeKey, clientType)
			err := middleware.RequireAdmin(func(c echo.Context) error {
				t.Fatal("next handler should not be called")
				return nil
			})(c)
			assertMiddlewareHTTPErrorCode(t, err, http.StatusForbidden)
		})
	}
}

func TestAdminMiddlewareRejectsInsufficientRole(t *testing.T) {
	middleware, err := NewAdminMiddleware(stubAdminRoleStore{
		findFunc: func(ctx context.Context, firebaseUID string) (*domain.AdminIdentity, error) {
			return &domain.AdminIdentity{UserID: uuid.New(), Role: domain.AdminRoleViewer}, nil
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	c, _ := newAdminMiddlewareContext()
	setSessionAuth(c, &domain.AuthToken{UID: "firebase-viewer"})
	err = middleware.RequireAdminRole(domain.AdminRoleAdmin)(func(c echo.Context) error {
		t.Fatal("next handler should not be called")
		return nil
	})(c)
	assertMiddlewareHTTPErrorCode(t, err, http.StatusForbidden)
}

func TestAdminMiddlewareReusesAdminIdentityFromContext(t *testing.T) {
	adminID := uuid.New()
	middleware, err := NewAdminMiddleware(stubAdminRoleStore{
		findFunc: func(ctx context.Context, firebaseUID string) (*domain.AdminIdentity, error) {
			t.Fatal("store should not be called when admin identity is already set")
			return nil, nil
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	c, rec := newAdminMiddlewareContext()
	setSessionAuth(c, &domain.AuthToken{UID: "firebase-admin"})
	c.Set(ContextAdminIdentityKey, domain.AdminIdentity{UserID: adminID, Role: domain.AdminRoleAdmin})
	err = middleware.RequireAdminRole(domain.AdminRoleSupport)(func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
}

func TestAdminMiddlewareReturnsServiceUnavailableOnStoreError(t *testing.T) {
	middleware, err := NewAdminMiddleware(stubAdminRoleStore{
		findFunc: func(ctx context.Context, firebaseUID string) (*domain.AdminIdentity, error) {
			return nil, errors.New("db unavailable")
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	c, _ := newAdminMiddlewareContext()
	setSessionAuth(c, &domain.AuthToken{UID: "firebase-admin"})
	err = middleware.RequireAdmin(func(c echo.Context) error {
		t.Fatal("next handler should not be called")
		return nil
	})(c)
	assertMiddlewareHTTPErrorCode(t, err, http.StatusServiceUnavailable)
}

func newAdminMiddlewareContext() (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/overview", nil)
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec), rec
}
