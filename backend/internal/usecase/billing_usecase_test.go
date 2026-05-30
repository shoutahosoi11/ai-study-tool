package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

type stubCheckoutSessionCreator struct {
	gotUserID uuid.UUID
}

func (s *stubCheckoutSessionCreator) CreateCheckoutSession(ctx context.Context, userID uuid.UUID, email string) (string, error) {
	s.gotUserID = userID
	return "https://checkout.example/session", nil
}

type stubStripeWebhookValidator struct {
	event domain.StripeWebhookEvent
	err   error
}

func (s stubStripeWebhookValidator) ConstructEvent(payload []byte, signatureHeader string) (domain.StripeWebhookEvent, error) {
	if s.err != nil {
		return domain.StripeWebhookEvent{}, s.err
	}
	return s.event, nil
}

type stubBillingRepository struct {
	processed       bool
	processCalls    int
	lastPayloadHash string
	lastEvent       domain.StripeWebhookEvent
}

func (s *stubBillingRepository) ProcessStripeEvent(ctx context.Context, event domain.StripeWebhookEvent, payloadHash string) (bool, error) {
	s.processCalls++
	s.lastEvent = event
	s.lastPayloadHash = payloadHash
	return s.processed, nil
}

func (s *stubBillingRepository) MarkCheckoutCompleted(ctx context.Context, userID uuid.UUID, customerID, subscriptionID string, expiresAt *time.Time) error {
	return nil
}

func (s *stubBillingRepository) UpdateSubscription(ctx context.Context, customerID, subscriptionID string, expiresAt *time.Time) error {
	return nil
}

func (s *stubBillingRepository) CancelSubscription(ctx context.Context, customerID, subscriptionID string) error {
	return nil
}

func TestBillingUsecaseProcessesStripeEventOnce(t *testing.T) {
	userID := uuid.New()
	repo := &stubBillingRepository{processed: true}
	uc := NewBillingUsecase(&stubCheckoutSessionCreator{}, stubStripeWebhookValidator{
		event: domain.StripeWebhookEvent{
			ID:             "evt_1",
			Type:           "checkout.session.completed",
			UserID:         userID,
			CustomerID:     "cus_1",
			SubscriptionID: "sub_1",
		},
	}, repo)

	if err := uc.HandleWebhook(context.Background(), []byte(`{"id":"evt_1"}`), "sig"); err != nil {
		t.Fatalf("HandleWebhook failed: %v", err)
	}
	if repo.processCalls != 1 {
		t.Fatalf("expected one process call, got %d", repo.processCalls)
	}
	if repo.lastPayloadHash == "" || repo.lastPayloadHash == `{"id":"evt_1"}` {
		t.Fatalf("expected payload hash, got %q", repo.lastPayloadHash)
	}
}

func TestBillingUsecaseNoOpsDuplicateStripeEvent(t *testing.T) {
	repo := &stubBillingRepository{processed: false}
	uc := NewBillingUsecase(&stubCheckoutSessionCreator{}, stubStripeWebhookValidator{
		event: domain.StripeWebhookEvent{ID: "evt_1", Type: "customer.subscription.updated"},
	}, repo)

	if err := uc.HandleWebhook(context.Background(), []byte(`{"id":"evt_1"}`), "sig"); err != nil {
		t.Fatalf("duplicate webhook should no-op: %v", err)
	}
	if repo.processCalls != 1 {
		t.Fatalf("expected one idempotency check, got %d", repo.processCalls)
	}
}

func TestBillingUsecaseRejectsInvalidStripeSignature(t *testing.T) {
	uc := NewBillingUsecase(&stubCheckoutSessionCreator{}, stubStripeWebhookValidator{err: errors.New("bad signature")}, &stubBillingRepository{})

	if err := uc.HandleWebhook(context.Background(), []byte(`{"id":"evt_1"}`), "bad"); err == nil {
		t.Fatal("expected invalid signature to fail")
	}
}

func TestCheckoutSessionIgnoresClientPrice(t *testing.T) {
	creator := &stubCheckoutSessionCreator{}
	uc := NewBillingUsecase(creator, stubStripeWebhookValidator{}, &stubBillingRepository{})
	userID := uuid.New()

	url, err := uc.CreateCheckoutSession(context.Background(), &domain.User{ID: userID}, "user@example.com")
	if err != nil {
		t.Fatalf("CreateCheckoutSession failed: %v", err)
	}
	if url == "" || creator.gotUserID != userID {
		t.Fatalf("unexpected checkout result url=%q user=%s", url, creator.gotUserID)
	}
}
