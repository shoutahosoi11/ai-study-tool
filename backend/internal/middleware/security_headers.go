package middleware

import (
	"strings"

	"github.com/labstack/echo/v4"
	appconfig "github.com/shout/ai-study-tool/backend/internal/config"
)

type SecurityHeadersMiddleware struct {
	appEnv       string
	cspReportURI string
}

func NewSecurityHeadersMiddleware(appEnv string, cspReportURI string) *SecurityHeadersMiddleware {
	return &SecurityHeadersMiddleware{
		appEnv:       strings.TrimSpace(appEnv),
		cspReportURI: strings.TrimSpace(cspReportURI),
	}
}

func (m *SecurityHeadersMiddleware) Secure(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		headers := c.Response().Header()
		headers.Set(echo.HeaderContentSecurityPolicy, m.contentSecurityPolicy())
		if appconfig.NormalizeAppEnv(m.appEnv).IsProduction() {
			// HSTS preload is production-only because it assumes the domain is
			// ready for long-lived HTTPS enforcement and preload-list operation.
			headers.Set(echo.HeaderStrictTransportSecurity, "max-age=31536000; includeSubDomains; preload")
		}
		headers.Set(echo.HeaderXContentTypeOptions, "nosniff")
		headers.Set(echo.HeaderXFrameOptions, "DENY")
		headers.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		headers.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		return next(c)
	}
}

func (m *SecurityHeadersMiddleware) contentSecurityPolicy() string {
	parts := []string{
		"default-src 'self'",
		"script-src 'self'",
		// The current React screens use inline style attributes extensively.
		// Keep this scoped to style-src; script-src must not allow inline code.
		"style-src 'self' 'unsafe-inline'",
		"img-src 'self' data: https:",
		// Add Stripe.js or other browser SDK origins here when the web client
		// starts calling those providers directly.
		"connect-src 'self' https://*.googleapis.com https://*.firebaseio.com",
		"frame-ancestors 'none'",
		"base-uri 'self'",
		"object-src 'none'",
		"form-action 'self'",
		"upgrade-insecure-requests",
	}
	if m.cspReportURI != "" {
		parts = append(parts, "report-uri "+m.cspReportURI)
	}
	return strings.Join(parts, "; ")
}
