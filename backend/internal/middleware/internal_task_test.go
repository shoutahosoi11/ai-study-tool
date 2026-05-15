package middleware

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"google.golang.org/api/idtoken"
)

func TestRequireInternalTaskSecret(t *testing.T) {
	tests := []struct {
		name       string
		configured string
		provided   string
		wantStatus int
	}{
		{
			name:       "valid secret passes",
			configured: "secret-value",
			provided:   "secret-value",
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "missing configured secret fails closed",
			configured: "",
			provided:   "secret-value",
			wantStatus: http.StatusServiceUnavailable,
		},
		{
			name:       "missing provided secret is unauthorized",
			configured: "secret-value",
			provided:   "",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "wrong secret is unauthorized",
			configured: "secret-value",
			provided:   "wrong-value",
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/internal/tasks/question-generation", nil)
			if tt.provided != "" {
				req.Header.Set(InternalTaskSecretHeader, tt.provided)
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			handler := RequireInternalTaskSecret(tt.configured)(func(c echo.Context) error {
				return c.NoContent(http.StatusNoContent)
			})

			err := handler(c)
			if err != nil {
				e.HTTPErrorHandler(err, c)
			}
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

type fakeInternalTaskIDTokenValidator struct {
	wantToken    string
	wantAudience string
	email        string
}

func (v fakeInternalTaskIDTokenValidator) Validate(ctx context.Context, token string, audience string) (*idtoken.Payload, error) {
	if token != v.wantToken {
		return nil, fmt.Errorf("unexpected token %q", token)
	}
	if audience != v.wantAudience {
		return nil, fmt.Errorf("unexpected audience %q", audience)
	}
	return &idtoken.Payload{Claims: map[string]interface{}{"email": v.email}}, nil
}

func TestRequireInternalTaskAuthAcceptsOIDC(t *testing.T) {
	original := newInternalTaskIDTokenValidator
	defer func() { newInternalTaskIDTokenValidator = original }()
	newInternalTaskIDTokenValidator = func(ctx context.Context) (internalTaskIDTokenValidator, error) {
		return fakeInternalTaskIDTokenValidator{
			wantToken:    "valid-token",
			wantAudience: "https://api.example.com/internal/tasks/question-generation",
			email:        "task-invoker@example.iam.gserviceaccount.com",
		}, nil
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/internal/tasks/question-generation", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := RequireInternalTaskAuth("", "https://api.example.com", "task-invoker@example.iam.gserviceaccount.com")(func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})

	if err := handler(c); err != nil {
		e.HTTPErrorHandler(err, c)
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}
