package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/shout/ai-study-tool/backend/internal/domain"
	appmiddleware "github.com/shout/ai-study-tool/backend/internal/middleware"
	"github.com/shout/ai-study-tool/backend/internal/usecase"
)

type ExtensionUsecase interface {
	StartPairing(ctx context.Context) (*domain.ExtensionPairing, error)
	ApprovePairing(ctx context.Context, userID uuid.UUID, userCode string, clientIdentifier string) error
	PairingStatus(ctx context.Context, pairingID uuid.UUID) (*domain.ExtensionPairing, error)
	ClaimPairing(ctx context.Context, pairingID uuid.UUID, clientIdentifier string) (*usecase.ExtensionTokenIssueResult, error)
	RevokeSelf(ctx context.Context, userID uuid.UUID, tokenID uuid.UUID) error
}

type ExtensionHandler struct {
	extensionUsecase ExtensionUsecase
	userUsecase      usecase.UserUsecaseInterface
}

func NewExtensionHandler(extensionUsecase ExtensionUsecase, userUsecase usecase.UserUsecaseInterface) *ExtensionHandler {
	return &ExtensionHandler{
		extensionUsecase: extensionUsecase,
		userUsecase:      userUsecase,
	}
}

type startExtensionPairingResponse struct {
	PairingID string    `json:"pairing_id"`
	UserCode  string    `json:"user_code"`
	ExpiresAt time.Time `json:"expires_at"`
}

func (h *ExtensionHandler) StartPairing(c echo.Context) error {
	pairing, err := h.extensionUsecase.StartPairing(c.Request().Context())
	if err != nil {
		slog.Error("extension_handler_error", "operation", "start_pairing", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusCreated, startExtensionPairingResponse{
		PairingID: pairing.ID.String(),
		UserCode:  pairing.UserCode,
		ExpiresAt: pairing.ExpiresAt,
	})
}

type approveExtensionPairingRequest struct {
	UserCode string `json:"user_code"`
}

type pairingStatusRequest struct {
	PairingID string `json:"pairing_id"`
}

type pairingStatusResponse struct {
	Status string `json:"status"`
}

type pairingClaimRequest struct {
	PairingID string `json:"pairing_id"`
}

type pairingClaimResponse struct {
	Status    string     `json:"status"`
	Token     string     `json:"token,omitempty"`
	Scopes    []string   `json:"scopes,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

func (h *ExtensionHandler) ApprovePairing(c echo.Context) error {
	req := new(approveExtensionPairingRequest)
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	user, err := resolveCurrentUser(c, h.userUsecase, "extension approve")
	if err != nil {
		return err
	}

	if err := h.extensionUsecase.ApprovePairing(c.Request().Context(), user.ID, req.UserCode, c.RealIP()); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "pairing code not found or expired")
		}
		if errors.Is(err, domain.ErrInvalidInput) {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid pairing request")
		}
		if errors.Is(err, domain.ErrRateLimitExceeded) {
			return echo.NewHTTPError(http.StatusTooManyRequests, "pairing approve rate limit exceeded")
		}
		slog.Error("extension_handler_error", "operation", "approve_pairing", "user_id", user.ID.String(), "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *ExtensionHandler) PairingStatus(c echo.Context) error {
	pairingID, err := bindPairingStatusID(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid pairing_id")
	}

	pairing, err := h.extensionUsecase.PairingStatus(c.Request().Context(), pairingID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "pairing not found or expired")
		}
		if errors.Is(err, domain.ErrInvalidInput) {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid pairing_id")
		}
		slog.Error("extension_handler_error", "operation", "pairing_status", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}
	status := "pending"
	if pairing.UsedAt != nil {
		status = "used"
	} else if pairing.ApprovedAt != nil {
		status = "approved"
	}
	return c.JSON(http.StatusOK, pairingStatusResponse{Status: status})
}

func (h *ExtensionHandler) ClaimPairing(c echo.Context) error {
	pairingID, err := bindPairingClaimID(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid pairing_id")
	}

	result, err := h.extensionUsecase.ClaimPairing(c.Request().Context(), pairingID, c.RealIP())
	if err != nil {
		if errors.Is(err, domain.ErrPairingNotApproved) {
			return c.JSON(http.StatusOK, pairingStatusResponse{Status: "pending"})
		}
		if errors.Is(err, domain.ErrAlreadyExists) {
			return echo.NewHTTPError(http.StatusGone, "pairing already used")
		}
		if errors.Is(err, domain.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "pairing not found or expired")
		}
		if errors.Is(err, domain.ErrInvalidInput) {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid pairing_id")
		}
		if errors.Is(err, domain.ErrRateLimitExceeded) {
			return echo.NewHTTPError(http.StatusTooManyRequests, "pairing claim rate limit exceeded")
		}
		slog.Error("extension_handler_error", "operation", "pairing_claim", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}
	return c.JSON(http.StatusOK, pairingClaimResponse{
		Status:    "approved",
		Token:     result.RawToken,
		Scopes:    result.Scopes,
		ExpiresAt: &result.ExpiresAt,
	})
}

func (h *ExtensionHandler) RevokeSelf(c echo.Context) error {
	user, err := resolveCurrentUser(c, h.userUsecase, "extension revoke")
	if err != nil {
		return err
	}

	tokenIDValue, ok := appmiddleware.GetExtensionTokenID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "extension token required")
	}
	tokenID, err := parseUUID(tokenIDValue)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "extension token required")
	}

	if err := h.extensionUsecase.RevokeSelf(c.Request().Context(), user.ID, tokenID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "extension token not found")
		}
		if errors.Is(err, domain.ErrInvalidInput) {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid revoke request")
		}
		slog.Error("extension_handler_error", "operation", "revoke_self", "user_id", user.ID.String(), "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return c.NoContent(http.StatusNoContent)
}

func bindPairingStatusID(c echo.Context) (uuid.UUID, error) {
	req := new(pairingStatusRequest)
	if err := c.Bind(req); err != nil {
		return uuid.Nil, err
	}
	return parseUUID(req.PairingID)
}

func bindPairingClaimID(c echo.Context) (uuid.UUID, error) {
	req := new(pairingClaimRequest)
	if err := c.Bind(req); err != nil {
		return uuid.Nil, err
	}
	return parseUUID(req.PairingID)
}

func parseUUID(value string) (uuid.UUID, error) {
	parsed, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil || parsed == uuid.Nil {
		return uuid.Nil, domain.ErrInvalidInput
	}
	return parsed, nil
}
