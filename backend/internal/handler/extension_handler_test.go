package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/shout/ai-study-tool/backend/internal/domain"
	"github.com/shout/ai-study-tool/backend/internal/middleware"
	"github.com/shout/ai-study-tool/backend/internal/usecase"
)

type stubExtensionUsecase struct {
	pairing       *domain.ExtensionPairing
	tokenResult   *usecase.ExtensionTokenIssueResult
	approveUserID uuid.UUID
	revokeUserID  uuid.UUID
	revokeTokenID uuid.UUID
	err           error
}

func (s *stubExtensionUsecase) StartPairing(ctx context.Context) (*domain.ExtensionPairing, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.pairing, nil
}

func (s *stubExtensionUsecase) ApprovePairing(ctx context.Context, userID uuid.UUID, userCode string, clientIdentifier string) error {
	s.approveUserID = userID
	return s.err
}

func (s *stubExtensionUsecase) PairingStatus(ctx context.Context, pairingID uuid.UUID) (*domain.ExtensionPairing, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.pairing, nil
}

func (s *stubExtensionUsecase) ClaimPairing(ctx context.Context, pairingID uuid.UUID, clientIdentifier string) (*usecase.ExtensionTokenIssueResult, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.tokenResult, nil
}

func (s *stubExtensionUsecase) RevokeSelf(ctx context.Context, userID uuid.UUID, tokenID uuid.UUID) error {
	s.revokeUserID = userID
	s.revokeTokenID = tokenID
	return s.err
}

func TestExtensionHandlerStartPairing(t *testing.T) {
	e := echo.New()
	pairingID := uuid.New()
	expiresAt := time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC)
	handler := NewExtensionHandler(&stubExtensionUsecase{
		pairing: &domain.ExtensionPairing{ID: pairingID, UserCode: "ABCDE-FGHJK", ExpiresAt: expiresAt},
	}, nil)
	req := httptest.NewRequest(http.MethodPost, "/extension/pairings", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := handler.StartPairing(c); err != nil {
		t.Fatalf("start pairing: %v", err)
	}
	if rec.Code != http.StatusCreated || !strings.Contains(rec.Body.String(), pairingID.String()) {
		t.Fatalf("unexpected response status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestExtensionHandlerClaimPairingPending(t *testing.T) {
	e := echo.New()
	handler := NewExtensionHandler(&stubExtensionUsecase{err: domain.ErrPairingNotApproved}, nil)
	req := httptest.NewRequest(http.MethodPost, "/extension/pairings/claim", strings.NewReader(`{"pairing_id":"`+uuid.NewString()+`"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := handler.ClaimPairing(c); err != nil {
		t.Fatalf("claim pairing: %v", err)
	}
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"status":"pending"`) {
		t.Fatalf("unexpected response status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestExtensionHandlerApprovePairingRequiresCurrentUser(t *testing.T) {
	e := echo.New()
	handler := NewExtensionHandler(&stubExtensionUsecase{}, &stubUserUsecase{})
	req := httptest.NewRequest(http.MethodPost, "/extension/pairings/approve", strings.NewReader(`{"user_code":"ABCDE-FGHJK"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.ApprovePairing(c)
	assertHTTPStatus(t, err, http.StatusUnauthorized)
}

func TestExtensionHandlerApprovePairingMapsRateLimit(t *testing.T) {
	e := echo.New()
	userID := uuid.New()
	handler := NewExtensionHandler(
		&stubExtensionUsecase{err: domain.ErrRateLimitExceeded},
		questionHandlerUserUsecase(userID),
	)
	req := httptest.NewRequest(http.MethodPost, "/extension/pairings/approve", strings.NewReader(`{"user_code":"ABCDE-FGHJK"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.ContextFirebaseUIDKey, "firebase-uid-1")

	err := handler.ApprovePairing(c)
	assertHTTPStatus(t, err, http.StatusTooManyRequests)
}

func TestExtensionHandlerRevokeSelfUsesExtensionTokenID(t *testing.T) {
	e := echo.New()
	userID := uuid.New()
	tokenID := uuid.New()
	extension := &stubExtensionUsecase{}
	handler := NewExtensionHandler(extension, questionHandlerUserUsecase(userID))
	req := httptest.NewRequest(http.MethodDelete, "/extension/tokens/current", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.ContextFirebaseUIDKey, "firebase-uid-1")
	c.Set(middleware.ContextExtensionTokenIDKey, tokenID.String())

	if err := handler.RevokeSelf(c); err != nil {
		t.Fatalf("revoke self: %v", err)
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if extension.revokeUserID != userID || extension.revokeTokenID != tokenID {
		t.Fatalf("unexpected revoke args user=%s token=%s", extension.revokeUserID, extension.revokeTokenID)
	}
}

func TestExtensionHandlerRevokeSelfMapsNotFound(t *testing.T) {
	e := echo.New()
	handler := NewExtensionHandler(
		&stubExtensionUsecase{err: domain.ErrNotFound},
		questionHandlerUserUsecase(uuid.New()),
	)
	req := httptest.NewRequest(http.MethodDelete, "/extension/tokens/current", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.ContextFirebaseUIDKey, "firebase-uid-1")
	c.Set(middleware.ContextExtensionTokenIDKey, uuid.NewString())

	err := handler.RevokeSelf(c)
	assertHTTPStatus(t, err, http.StatusNotFound)
}

func TestExtensionHandlerRevokeSelfMapsUnexpectedError(t *testing.T) {
	e := echo.New()
	handler := NewExtensionHandler(
		&stubExtensionUsecase{err: errors.New("db down")},
		questionHandlerUserUsecase(uuid.New()),
	)
	req := httptest.NewRequest(http.MethodDelete, "/extension/tokens/current", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.ContextFirebaseUIDKey, "firebase-uid-1")
	c.Set(middleware.ContextExtensionTokenIDKey, uuid.NewString())

	err := handler.RevokeSelf(c)
	assertHTTPStatus(t, err, http.StatusInternalServerError)
}
