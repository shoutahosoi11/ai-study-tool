package handler

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/shout/ai-study-tool/backend/internal/domain"
	"github.com/shout/ai-study-tool/backend/internal/usecase"
)

type UserHandler struct {
	userUsecase *usecase.UserUsecase
}

func NewUserHandler(userUsecase *usecase.UserUsecase) *UserHandler {
	return &UserHandler{userUsecase: userUsecase}
}

type SignUpRequest struct {
	Username    string  `json:"username" validate:"required,min=3,max=50"`
	DisplayName string  `json:"display_name" validate:"required,max=100"`
	AvatarURL   *string `json:"avatar_url"`
	University  *string `json:"university"`
	Faculty     *string `json:"faculty"`
	Grade       *int16  `json:"grade"`
	Country     *string `json:"country"`
}

func (h *UserHandler) SignUp(c echo.Context) error {
	firebaseUID, ok := c.Get("firebase_uid").(string)
	if !ok || firebaseUID == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	req := new(SignUpRequest)
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	user, err := h.userUsecase.SignUp(c.Request().Context(), domain.CreateUserInput{
		FirebaseUID: firebaseUID,
		Username:    req.Username,
		DisplayName: req.DisplayName,
		AvatarURL:   req.AvatarURL,
		University:  req.University,
		Faculty:     req.Faculty,
		Grade:       req.Grade,
		Country:     req.Country,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, user)
}

func (h *UserHandler) GetMe(c echo.Context) error {
	firebaseUID, ok := c.Get("firebase_uid").(string)
	if !ok || firebaseUID == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	user, err := h.userUsecase.GetByFirebaseUID(c.Request().Context(), firebaseUID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "user not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, user)
}

func (h *UserHandler) GetUser(c echo.Context) error {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid user id")
	}

	user, err := h.userUsecase.GetByID(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "user not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, user)
}

func (h *UserHandler) UpdateProfile(c echo.Context) error {
	firebaseUID, ok := c.Get("firebase_uid").(string)
	if !ok || firebaseUID == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	me, err := h.userUsecase.GetByFirebaseUID(c.Request().Context(), firebaseUID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "user not found")
	}

	req := new(domain.UpdateUserInput)
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	updated, err := h.userUsecase.UpdateProfile(c.Request().Context(), me.ID, *req)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, updated)
}
