package stripe

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shout/ai-study-tool/backend/internal/domain"
	stripeapi "github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/webhook"
)

const testWebhookSecret = "whsec_test_secret"

func signedHeader(t *testing.T, payload []byte, at time.Time, secret string) string {
	t.Helper()
	signature := webhook.ComputeSignature(at, payload, secret)
	return fmt.Sprintf("t=%d,v1=%x", at.Unix(), signature)
}

func eventPayload(eventType string, data string) []byte {
	return []byte(fmt.Sprintf(`{"id":"evt_test_1","api_version":"2023-10-16","type":"%s","data":{"object":%s}}`, eventType, data))
}

func TestConstructEventRejectsEmptySecret(t *testing.T) {
	validator := &WebhookValidator{secret: ""}

	_, err := validator.ConstructEvent([]byte(`{}`), "t=1,v1=deadbeef")

	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected domain.ErrInvalidInput, got %v", err)
	}
}

func TestConstructEventRejectsTamperedPayload(t *testing.T) {
	validator := &WebhookValidator{secret: testWebhookSecret}
	payload := eventPayload("checkout.session.completed", `{"client_reference_id":"`+uuid.NewString()+`"}`)
	header := signedHeader(t, payload, time.Now(), testWebhookSecret)

	tampered := append([]byte{}, payload...)
	tampered[len(tampered)-2] = 'X'

	if _, err := validator.ConstructEvent(tampered, header); err == nil {
		t.Fatal("expected signature verification error for tampered payload")
	}
}

func TestConstructEventRejectsExpiredTimestamp(t *testing.T) {
	validator := &WebhookValidator{secret: testWebhookSecret}
	payload := eventPayload("checkout.session.completed", `{"client_reference_id":"`+uuid.NewString()+`"}`)
	header := signedHeader(t, payload, time.Now().Add(-time.Hour), testWebhookSecret)

	if _, err := validator.ConstructEvent(payload, header); err == nil {
		t.Fatal("expected tolerance error for hour-old signature")
	}
}

func TestConstructEventRejectsWrongSecret(t *testing.T) {
	validator := &WebhookValidator{secret: testWebhookSecret}
	payload := eventPayload("checkout.session.completed", `{"client_reference_id":"`+uuid.NewString()+`"}`)
	header := signedHeader(t, payload, time.Now(), "whsec_other_secret")

	if _, err := validator.ConstructEvent(payload, header); err == nil {
		t.Fatal("expected signature verification error for wrong secret")
	}
}

func TestConstructEventParsesCheckoutCompleted(t *testing.T) {
	validator := &WebhookValidator{secret: testWebhookSecret}
	userID := uuid.New()
	data := fmt.Sprintf(`{"client_reference_id":"%s","customer":{"id":"cus_123"},"subscription":{"id":"sub_123"}}`, userID)
	payload := eventPayload("checkout.session.completed", data)
	header := signedHeader(t, payload, time.Now(), testWebhookSecret)

	event, err := validator.ConstructEvent(payload, header)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if event.UserID != userID {
		t.Fatalf("expected user id %s, got %s", userID, event.UserID)
	}
	if event.CustomerID != "cus_123" || event.SubscriptionID != "sub_123" {
		t.Fatalf("unexpected customer/subscription: %q %q", event.CustomerID, event.SubscriptionID)
	}
	if event.Status != "active" {
		t.Fatalf("expected status active, got %q", event.Status)
	}
}

func TestConstructEventCheckoutFallsBackToMetadataUserID(t *testing.T) {
	validator := &WebhookValidator{secret: testWebhookSecret}
	userID := uuid.New()
	data := fmt.Sprintf(`{"metadata":{"user_id":"%s"}}`, userID)
	payload := eventPayload("checkout.session.completed", data)
	header := signedHeader(t, payload, time.Now(), testWebhookSecret)

	event, err := validator.ConstructEvent(payload, header)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.UserID != userID {
		t.Fatalf("expected metadata user id %s, got %s", userID, event.UserID)
	}
}

func TestConstructEventCheckoutWithoutUserIDFails(t *testing.T) {
	validator := &WebhookValidator{secret: testWebhookSecret}
	payload := eventPayload("checkout.session.completed", `{"customer":{"id":"cus_123"}}`)
	header := signedHeader(t, payload, time.Now(), testWebhookSecret)

	if _, err := validator.ConstructEvent(payload, header); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected domain.ErrInvalidInput for missing user id, got %v", err)
	}
}

func TestConstructEventParsesSubscriptionUpdate(t *testing.T) {
	validator := &WebhookValidator{secret: testWebhookSecret}
	periodEnd := time.Now().Add(30 * 24 * time.Hour).Unix()
	data := fmt.Sprintf(`{
		"id": "sub_456",
		"customer": {"id": "cus_456"},
		"status": "past_due",
		"current_period_end": %d,
		"items": {"data": [{"price": {"id": "price_pro"}}]}
	}`, periodEnd)
	payload := eventPayload("customer.subscription.updated", data)
	header := signedHeader(t, payload, time.Now(), testWebhookSecret)

	event, err := validator.ConstructEvent(payload, header)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if event.SubscriptionID != "sub_456" || event.CustomerID != "cus_456" {
		t.Fatalf("unexpected ids: %q %q", event.SubscriptionID, event.CustomerID)
	}
	if event.Status != "past_due" {
		t.Fatalf("expected normalized status past_due, got %q", event.Status)
	}
	if event.ProductID != "price_pro" {
		t.Fatalf("expected product id price_pro, got %q", event.ProductID)
	}
	if event.ExpiresAt == nil || event.ExpiresAt.Unix() != periodEnd {
		t.Fatalf("expected expires_at %d, got %v", periodEnd, event.ExpiresAt)
	}
}

func TestConstructEventParsesInvoicePaymentFailed(t *testing.T) {
	validator := &WebhookValidator{secret: testWebhookSecret}
	data := `{"customer":{"id":"cus_789"},"subscription":{"id":"sub_789"}}`
	payload := eventPayload("invoice.payment_failed", data)
	header := signedHeader(t, payload, time.Now(), testWebhookSecret)

	event, err := validator.ConstructEvent(payload, header)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.CustomerID != "cus_789" || event.SubscriptionID != "sub_789" {
		t.Fatalf("unexpected ids: %q %q", event.CustomerID, event.SubscriptionID)
	}
}

func TestNormalizeStripeSubscriptionStatus(t *testing.T) {
	cases := map[string]string{
		"active":             "active",
		"trialing":           "trialing",
		"past_due":           "past_due",
		"unpaid":             "past_due",
		"incomplete":         "past_due",
		"incomplete_expired": "past_due",
		"canceled":           "canceled",
		"paused":             "expired",
	}
	for raw, want := range cases {
		if got := normalizeStripeSubscriptionStatus(stripeapi.SubscriptionStatus(raw)); got != want {
			t.Fatalf("status %q: expected %q, got %q", raw, want, got)
		}
	}
}
