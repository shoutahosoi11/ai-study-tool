package middleware

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/labstack/echo/v4"
	appconfig "github.com/shout/ai-study-tool/backend/internal/config"
)

type CSRFMiddleware struct {
	appEnv                      string
	secret                      string
	allowedOrigins              map[string]struct{}
	allowMissingSecurityHeaders bool
	signingDisabled             bool
}

func NewCSRFMiddleware(appEnv string) *CSRFMiddleware {
	middleware, _ := NewCSRFMiddlewareWithConfig(CSRFConfig{
		AppEnv:                      appEnv,
		Secret:                      "test-csrf-secret",
		AllowedOrigins:              defaultDevelopmentOrigins(),
		AllowMissingSecurityHeaders: true,
	})
	return middleware
}

type CSRFConfig struct {
	AppEnv                      string
	Secret                      string
	AllowedOrigins              []string
	AllowMissingSecurityHeaders bool
	SigningDisabled             bool
}

func NewCSRFMiddlewareFromEnv(appEnv string) (*CSRFMiddleware, error) {
	strictEnv := appconfig.NormalizeAppEnv(appEnv).IsStrictSecurity()
	cfg := CSRFConfig{
		AppEnv:                      appEnv,
		Secret:                      strings.TrimSpace(os.Getenv("CSRF_SECRET")),
		AllowedOrigins:              allowedOriginsFromEnv(),
		AllowMissingSecurityHeaders: !strictEnv,
		SigningDisabled:             envBool("CSRF_SIGNING_DISABLED"),
	}
	if len(cfg.AllowedOrigins) == 0 && !strictEnv {
		cfg.AllowedOrigins = defaultDevelopmentOrigins()
	}
	return NewCSRFMiddlewareWithConfig(cfg)
}

func NewCSRFMiddlewareWithConfig(cfg CSRFConfig) (*CSRFMiddleware, error) {
	appEnv := strings.TrimSpace(cfg.AppEnv)
	secret := strings.TrimSpace(cfg.Secret)
	strictEnv := appconfig.NormalizeAppEnv(appEnv).IsStrictSecurity()
	if strictEnv && cfg.SigningDisabled {
		return nil, errors.New("csrf middleware: CSRF_SIGNING_DISABLED is not allowed in production-like environments")
	}
	if strictEnv && secret == "" {
		return nil, errors.New("csrf middleware: CSRF_SECRET is required in production-like environments")
	}
	if !cfg.SigningDisabled && secret == "" {
		return nil, errors.New("csrf middleware: secret is empty")
	}

	origins := normalizeAllowedOrigins(cfg.AllowedOrigins)
	if strictEnv && len(origins) == 0 {
		return nil, errors.New("csrf middleware: ALLOWED_ORIGINS is required in production-like environments")
	}

	signingDisabled := cfg.SigningDisabled && !strictEnv
	if signingDisabled {
		slog.Warn("csrf_signing_disabled", "app_env", appEnv)
	}

	return &CSRFMiddleware{
		appEnv:                      appEnv,
		secret:                      secret,
		allowedOrigins:              origins,
		allowMissingSecurityHeaders: cfg.AllowMissingSecurityHeaders && !strictEnv,
		signingDisabled:             signingDisabled,
	}, nil
}

// Protect verifies CSRF with the double submit cookie pattern.
// Callers must apply it only to Session Cookie authentication paths; Bearer
// compatibility during the hybrid period is controlled by HybridAuthMiddleware.
func (m *CSRFMiddleware) Protect(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		switch c.Request().Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			return next(c)
		}

		if err := m.verifyOrigin(c); err != nil {
			return err
		}
		if err := m.verifyFetchMetadata(c); err != nil {
			return err
		}

		headerToken := strings.TrimSpace(c.Request().Header.Get("X-CSRF-Token"))
		cookie, err := c.Cookie(CSRFCookieName())
		if err != nil || strings.TrimSpace(cookie.Value) == "" || headerToken == "" {
			return echo.NewHTTPError(http.StatusForbidden, "invalid csrf token")
		}

		if !hmac.Equal([]byte(headerToken), []byte(strings.TrimSpace(cookie.Value))) {
			return echo.NewHTTPError(http.StatusForbidden, "invalid csrf token")
		}

		if !m.signingDisabled {
			uid, ok := GetFirebaseUID(c)
			if !ok {
				return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
			}
			if !VerifySignedCSRFToken(m.secret, uid, headerToken) {
				return echo.NewHTTPError(http.StatusForbidden, "invalid csrf token")
			}
		}

		return next(c)
	}
}

func SignCSRFToken(secret string, uid string, raw string) (string, error) {
	secret = strings.TrimSpace(secret)
	uid = strings.TrimSpace(uid)
	raw = strings.TrimSpace(raw)
	if secret == "" || uid == "" || raw == "" {
		return "", errors.New("csrf token: missing input")
	}

	mac := hmac.New(sha256.New, []byte(secret))
	if _, err := mac.Write([]byte(uid + "." + raw)); err != nil {
		return "", fmt.Errorf("csrf token: sign: %w", err)
	}
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	// raw is generated with base64.RawURLEncoding, whose alphabet does not
	// include '.', so csrf_raw.signature can be split unambiguously.
	return raw + "." + signature, nil
}

func VerifySignedCSRFToken(secret string, uid string, token string) bool {
	secret = strings.TrimSpace(secret)
	uid = strings.TrimSpace(uid)
	raw, signature, ok := splitSignedCSRFToken(token)
	if secret == "" || uid == "" || !ok {
		return false
	}

	expected, err := SignCSRFToken(secret, uid, raw)
	if err != nil {
		return false
	}
	_, expectedSignature, ok := splitSignedCSRFToken(expected)
	if !ok {
		return false
	}
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

func GenerateCSRFRawToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate csrf token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func splitSignedCSRFToken(token string) (string, string, bool) {
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func (m *CSRFMiddleware) verifyOrigin(c echo.Context) error {
	origin := normalizeOrigin(c.Request().Header.Get("Origin"))
	if origin == "" {
		if m.allowMissingSecurityHeaders {
			return nil
		}
		return echo.NewHTTPError(http.StatusForbidden, "invalid origin")
	}

	if _, ok := m.allowedOrigins[origin]; !ok {
		return echo.NewHTTPError(http.StatusForbidden, "invalid origin")
	}
	return nil
}

func (m *CSRFMiddleware) verifyFetchMetadata(c echo.Context) error {
	site := strings.ToLower(strings.TrimSpace(c.Request().Header.Get("Sec-Fetch-Site")))
	if site == "" {
		if m.allowMissingSecurityHeaders {
			return nil
		}
		return echo.NewHTTPError(http.StatusForbidden, "invalid fetch metadata")
	}

	switch site {
	case "same-origin", "same-site":
		return nil
	case "cross-site":
		return echo.NewHTTPError(http.StatusForbidden, "invalid fetch metadata")
	case "none":
		return echo.NewHTTPError(http.StatusForbidden, "invalid fetch metadata")
	default:
		return echo.NewHTTPError(http.StatusForbidden, "invalid fetch metadata")
	}
}

func hasSessionCookie(c echo.Context, appEnv string) bool {
	cookie, err := c.Cookie(SessionCookieName(appEnv))
	return err == nil && strings.TrimSpace(cookie.Value) != ""
}

func allowedOriginsFromEnv() []string {
	raw := strings.TrimSpace(os.Getenv("ALLOWED_ORIGINS"))
	if raw == "" {
		raw = strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS"))
	}
	if raw == "" {
		return nil
	}

	parts := strings.Split(raw, ",")
	origins := make([]string, 0, len(parts))
	for _, part := range parts {
		if origin := strings.TrimSpace(part); origin != "" {
			origins = append(origins, origin)
		}
	}
	return origins
}

func normalizeAllowedOrigins(values []string) map[string]struct{} {
	origins := make(map[string]struct{}, len(values))
	for _, value := range values {
		origin := normalizeOrigin(value)
		if origin != "" {
			origins[origin] = struct{}{}
		}
	}
	return origins
}

func normalizeOrigin(value string) string {
	origin := strings.TrimSpace(value)
	if origin == "" {
		return ""
	}
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return ""
	}
	if parsed.Path != "" && parsed.Path != "/" {
		return ""
	}
	return strings.ToLower(parsed.Scheme) + "://" + strings.ToLower(parsed.Host)
}

func csrfStrictEnvironment(appEnv string) bool {
	return appconfig.NormalizeAppEnv(appEnv).IsStrictSecurity()
}

func defaultDevelopmentOrigins() []string {
	return []string{"http://localhost:3000", "http://127.0.0.1:3000"}
}

func envBool(key string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
