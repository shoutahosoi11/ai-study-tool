package middleware

import (
	"context"
	"crypto/subtle"
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"google.golang.org/api/idtoken"
)

const InternalTaskSecretHeader = "X-Internal-Task-Secret"

func RequireInternalTaskSecret(secret string) echo.MiddlewareFunc {
	return RequireInternalTaskAuth(secret, "", "")
}

type internalTaskIDTokenValidator interface {
	Validate(ctx context.Context, token string, audience string) (*idtoken.Payload, error)
}

var newInternalTaskIDTokenValidator = func(ctx context.Context) (internalTaskIDTokenValidator, error) {
	return idtoken.NewValidator(ctx)
}

func RequireInternalTaskAuth(secret string, handlerBaseURL string, expectedServiceAccount string) echo.MiddlewareFunc {
	expected := strings.TrimSpace(secret)
	baseURL := strings.TrimRight(strings.TrimSpace(handlerBaseURL), "/")
	expectedEmail := strings.TrimSpace(expectedServiceAccount)

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if baseURL != "" && hasBearerToken(c.Request()) {
				if err := validateInternalTaskOIDC(c.Request().Context(), c.Request(), baseURL, expectedEmail); err != nil {
					return echo.NewHTTPError(http.StatusUnauthorized, "invalid internal task authentication")
				}
				return next(c)
			}

			if expected == "" {
				return echo.NewHTTPError(http.StatusServiceUnavailable, "internal task authentication is not configured")
			}

			got := strings.TrimSpace(c.Request().Header.Get(InternalTaskSecretHeader))
			if got == "" || subtle.ConstantTimeCompare([]byte(got), []byte(expected)) != 1 {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid internal task authentication")
			}

			return next(c)
		}
	}
}

func hasBearerToken(r *http.Request) bool {
	return strings.HasPrefix(strings.TrimSpace(r.Header.Get("Authorization")), "Bearer ")
}

func validateInternalTaskOIDC(ctx context.Context, r *http.Request, handlerBaseURL string, expectedEmail string) error {
	raw := strings.TrimSpace(r.Header.Get("Authorization"))
	token := strings.TrimSpace(strings.TrimPrefix(raw, "Bearer "))
	if token == "" {
		return fmt.Errorf("missing bearer token")
	}

	validator, err := newInternalTaskIDTokenValidator(ctx)
	if err != nil {
		return fmt.Errorf("create id token validator: %w", err)
	}

	audience := handlerBaseURL + r.URL.Path
	payload, err := validator.Validate(ctx, token, audience)
	if err != nil {
		return fmt.Errorf("validate id token: %w", err)
	}
	if expectedEmail != "" {
		email, _ := payload.Claims["email"].(string)
		if !strings.EqualFold(strings.TrimSpace(email), expectedEmail) {
			return fmt.Errorf("unexpected service account email")
		}
	}
	return nil
}
