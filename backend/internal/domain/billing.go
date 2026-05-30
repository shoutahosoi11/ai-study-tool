package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type CheckoutSessionCreator interface {
	CreateCheckoutSession(ctx context.Context, userID uuid.UUID, email string) (string, error)
}

type StripeWebhookValidator interface {
	ConstructEvent(payload []byte, signatureHeader string) (StripeWebhookEvent, error)
}

type StripeWebhookEvent struct {
	ID             string
	Type           string
	CustomerID     string
	SubscriptionID string
	ProductID      string
	Status         string
	UserID         uuid.UUID
	ExpiresAt      *time.Time
}

type BillingRepository interface {
	ProcessStripeEvent(ctx context.Context, event StripeWebhookEvent, payloadHash string) (bool, error)
	MarkCheckoutCompleted(ctx context.Context, userID uuid.UUID, customerID, subscriptionID string, expiresAt *time.Time) error
	UpdateSubscription(ctx context.Context, customerID, subscriptionID string, expiresAt *time.Time) error
	CancelSubscription(ctx context.Context, customerID, subscriptionID string) error
}
