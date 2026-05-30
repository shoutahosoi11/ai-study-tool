package middleware

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/labstack/echo/v4"
	appconfig "github.com/shout/ai-study-tool/backend/internal/config"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

const appCheckHeader = "X-Firebase-AppCheck"

type AppCheckVerifier interface {
	VerifyAppCheckToken(ctx context.Context, token string) error
}

type AppCheckMiddleware struct {
	verifier            AppCheckVerifier
	enforced            bool
	requiredClientTypes map[domain.AuthClientType]struct{}
}

type AppCheckConfig struct {
	AppEnv              string
	Enforced            bool
	RequiredClientTypes []domain.AuthClientType
}

func AppCheckEnforcementEnabledFromEnv(appEnv string) (bool, error) {
	raw := strings.TrimSpace(os.Getenv("APP_CHECK_ENFORCEMENT"))
	if raw == "" {
		return appconfig.NormalizeAppEnv(appEnv).IsProduction(), nil
	}

	enabled := envBool("APP_CHECK_ENFORCEMENT")
	if appconfig.NormalizeAppEnv(appEnv).IsProduction() && !enabled {
		return false, errors.New("app check middleware: APP_CHECK_ENFORCEMENT=false is not allowed in production")
	}
	return enabled, nil
}

func NewAppCheckMiddlewareFromEnv(appEnv string, verifier AppCheckVerifier) (*AppCheckMiddleware, error) {
	enforced, err := AppCheckEnforcementEnabledFromEnv(appEnv)
	if err != nil {
		return nil, err
	}
	return NewAppCheckMiddleware(verifier, AppCheckConfig{
		AppEnv:              appEnv,
		Enforced:            enforced,
		RequiredClientTypes: []domain.AuthClientType{domain.AuthClientTypeMobile},
	})
}

func NewAppCheckMiddleware(verifier AppCheckVerifier, config AppCheckConfig) (*AppCheckMiddleware, error) {
	appEnv := strings.TrimSpace(config.AppEnv)
	if appconfig.NormalizeAppEnv(appEnv).IsProduction() && !config.Enforced {
		return nil, errors.New("app check middleware: enforcement is required in production")
	}
	if config.Enforced && verifier == nil {
		return nil, errors.New("app check middleware: verifier is nil")
	}

	clientTypes := config.RequiredClientTypes
	if len(clientTypes) == 0 {
		clientTypes = []domain.AuthClientType{domain.AuthClientTypeMobile}
	}
	required := make(map[domain.AuthClientType]struct{}, len(clientTypes))
	for _, clientType := range clientTypes {
		if clientType != "" {
			required[clientType] = struct{}{}
		}
	}

	return &AppCheckMiddleware{
		verifier:            verifier,
		enforced:            config.Enforced,
		requiredClientTypes: required,
	}, nil
}

func (m *AppCheckMiddleware) Require(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if m == nil || !m.enforced {
			return next(c)
		}

		clientType, ok := GetAuthClientType(c)
		if !ok {
			return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
		}
		if _, required := m.requiredClientTypes[clientType]; !required {
			return next(c)
		}

		token := strings.TrimSpace(c.Request().Header.Get(appCheckHeader))
		if token == "" {
			return echo.NewHTTPError(http.StatusUnauthorized, "missing app check token")
		}
		if err := m.verifier.VerifyAppCheckToken(c.Request().Context(), token); err != nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid app check token")
		}

		return next(c)
	}
}
