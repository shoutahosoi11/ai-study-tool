package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/shout/ai-study-tool/backend/internal/domain"
	"github.com/shout/ai-study-tool/backend/internal/middleware"
)

// internalError logs the underlying error with request-trace correlation and
// returns a generic 500 so internals never leak to clients.
func internalError(c echo.Context, operation string, err error) error {
	middleware.RequestLogger(c).Error("handler_error", "operation", operation, "error", err.Error())
	return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
}

// asValidationHTTPError converts domain.ValidationError into a 400 carrying the
// user-facing message. Any other error is left for the caller to classify.
func asValidationHTTPError(err error) (error, bool) {
	var ve *domain.ValidationError
	if errors.As(err, &ve) {
		return echo.NewHTTPError(http.StatusBadRequest, ve.Message), true
	}
	return nil, false
}
