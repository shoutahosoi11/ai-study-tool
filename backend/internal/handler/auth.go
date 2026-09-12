package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/shout/ai-study-tool/backend/internal/domain"
	"github.com/shout/ai-study-tool/backend/internal/middleware"
	"github.com/shout/ai-study-tool/backend/internal/usecase"
)

func resolveCurrentUser(c echo.Context, userUsecase usecase.UserUsecaseInterface, logPrefix string) (*domain.User, error) {
	firebaseUID, ok := middleware.GetFirebaseUID(c)
	if !ok {
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	user, err := userUsecase.GetByFirebaseUID(c.Request().Context(), firebaseUID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, echo.NewHTTPError(http.StatusNotFound, "user not found")
		}
		return nil, internalError(c, logPrefix+".resolve_current_user", err)
	}

	return user, nil
}
