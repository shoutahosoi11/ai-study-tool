package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

type stubStripeBillingUsecase struct {
	payload   []byte
	signature string
	err       error
}

func (s *stubStripeBillingUsecase) CreateCheckoutSession(ctx context.Context, user *domain.User, email string) (string, error) {
	return "https://checkout.example/session", nil
}

func (s *stubStripeBillingUsecase) HandleWebhook(ctx context.Context, payload []byte, signature string) error {
	s.payload = append([]byte(nil), payload...)
	s.signature = signature
	return s.err
}

func TestStripeWebhookUsesRawBodyAndSignatureHeader(t *testing.T) {
	e := echo.New()
	billing := &stubStripeBillingUsecase{}
	handler := NewStripeHandler(billing, nil)
	raw := `{"id":"evt_1","data":{"object":{"metadata":{"user_id":"u"}}}}`
	req := httptest.NewRequest(http.MethodPost, "/webhooks/stripe", strings.NewReader(raw))
	req.Header.Set("Stripe-Signature", "t=1,v1=sig")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := handler.HandleWebhook(c); err != nil {
		t.Fatalf("HandleWebhook failed: %v", err)
	}
	if string(billing.payload) != raw {
		t.Fatalf("raw body changed: %q", string(billing.payload))
	}
	if billing.signature != "t=1,v1=sig" {
		t.Fatalf("signature header not forwarded: %q", billing.signature)
	}
}

func TestStripeWebhookRejectsInvalidSignature(t *testing.T) {
	e := echo.New()
	handler := NewStripeHandler(&stubStripeBillingUsecase{err: errors.New("bad signature")}, nil)
	req := httptest.NewRequest(http.MethodPost, "/webhooks/stripe", strings.NewReader(`{"id":"evt_1"}`))
	req.Header.Set("Stripe-Signature", "bad")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.HandleWebhook(c)
	if err == nil {
		t.Fatal("expected error")
	}
	httpErr, ok := err.(*echo.HTTPError)
	if !ok || httpErr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %#v", err)
	}
}
