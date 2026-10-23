package middleware

import (
	"log/slog"

	"github.com/labstack/echo/v4"
)

const requestLoggerContextKey = "app.request_logger"

// SetRequestLogger stores a request-scoped logger (typically annotated with the
// Cloud Logging trace) so handler-level logs can be correlated with the
// per-request access log.
func SetRequestLogger(c echo.Context, logger *slog.Logger) {
	if logger != nil {
		c.Set(requestLoggerContextKey, logger)
	}
}

// RequestLogger returns the request-scoped logger, falling back to the default
// logger when none was attached (tests, unregistered routes).
func RequestLogger(c echo.Context) *slog.Logger {
	if logger, ok := c.Get(requestLoggerContextKey).(*slog.Logger); ok && logger != nil {
		return logger
	}
	return slog.Default()
}
