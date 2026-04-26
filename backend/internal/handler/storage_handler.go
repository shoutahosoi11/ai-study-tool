package handler

import (
	"errors"
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/shout/ai-study-tool/backend/internal/domain"
	"github.com/shout/ai-study-tool/backend/internal/handler/dto"
	"github.com/shout/ai-study-tool/backend/internal/middleware"
	"github.com/shout/ai-study-tool/backend/internal/usecase"
)

type StorageHandler struct {
	storageUsecase *usecase.StorageUsecase
	userUsecase    usecase.UserUsecaseInterface
}

func NewStorageHandler(storageUsecase *usecase.StorageUsecase, userUsecase usecase.UserUsecaseInterface) *StorageHandler {
	return &StorageHandler{
		storageUsecase: storageUsecase,
		userUsecase:    userUsecase,
	}
}

func (h *StorageHandler) CreateUploadSignedURL(c echo.Context) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}

	req := new(dto.CreateUploadSignedURLRequest)
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	result, err := h.storageUsecase.CreateUploadSignedURL(c.Request().Context(), usecase.CreateUploadSignedURLInput{
		UserID:           user.ID,
		FileName:         req.FileName,
		ContentType:      req.ContentType,
		ExpiresInSeconds: req.ExpiresInSeconds,
	})
	if err != nil {
		return storageError(err)
	}

	return c.JSON(http.StatusOK, toSignedURLResponse(result))
}

func (h *StorageHandler) CreateDownloadSignedURL(c echo.Context) error {
	user, err := h.currentUser(c)
	if err != nil {
		return err
	}

	req := new(dto.CreateDownloadSignedURLRequest)
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	result, err := h.storageUsecase.CreateDownloadSignedURL(c.Request().Context(), usecase.CreateDownloadSignedURLInput{
		UserID:           user.ID,
		ObjectName:       req.ObjectName,
		ExpiresInSeconds: req.ExpiresInSeconds,
	})
	if err != nil {
		return storageError(err)
	}

	return c.JSON(http.StatusOK, toSignedURLResponse(result))
}

func (h *StorageHandler) currentUser(c echo.Context) (*domain.User, error) {
	firebaseUID, ok := middleware.GetFirebaseUID(c)
	if !ok {
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	user, err := h.userUsecase.GetByFirebaseUID(c.Request().Context(), firebaseUID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, echo.NewHTTPError(http.StatusNotFound, "user not found")
		}
		log.Printf("storage currentUser error: %v", err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return user, nil
}

func storageError(err error) error {
	if errors.Is(err, domain.ErrInvalidInput) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid storage request")
	}
	if errors.Is(err, domain.ErrStorageNotConfigured) {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "storage is not configured")
	}

	log.Printf("storage signed url error: %v", err)
	return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
}

func toSignedURLResponse(signedURL *domain.SignedURL) dto.SignedURLResponse {
	return dto.SignedURLResponse{
		URL:         signedURL.URL,
		Method:      signedURL.Method,
		Bucket:      signedURL.Bucket,
		ObjectName:  signedURL.ObjectName,
		ContentType: signedURL.ContentType,
		ExpiresAt:   signedURL.ExpiresAt,
		Headers:     signedURL.Headers,
	}
}
