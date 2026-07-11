package stripe

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/shout/ai-study-tool/backend/internal/domain"
	stripeapi "github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/checkout/session"
	"github.com/stripe/stripe-go/v76/subscription"
)

type CheckoutClient struct {
	secretKey  string
	priceID    string
	successURL string
	cancelURL  string
}

func NewCheckoutClientFromEnv() *CheckoutClient {
	return &CheckoutClient{
		secretKey:  strings.TrimSpace(os.Getenv("STRIPE_SECRET_KEY")),
		priceID:    strings.TrimSpace(os.Getenv("STRIPE_PRICE_ID_MONTHLY")),
		successURL: strings.TrimSpace(os.Getenv("STRIPE_SUCCESS_URL")),
		cancelURL:  strings.TrimSpace(os.Getenv("STRIPE_CANCEL_URL")),
	}
}

func (c *CheckoutClient) CreateCheckoutSession(ctx context.Context, userID uuid.UUID, email string) (string, error) {
	if c.secretKey == "" || c.priceID == "" || c.successURL == "" || c.cancelURL == "" {
		return "", domain.ErrInvalidInput
	}

	stripeapi.Key = c.secretKey
	params := &stripeapi.CheckoutSessionParams{
		Mode:              stripeapi.String(string(stripeapi.CheckoutSessionModeSubscription)),
		SuccessURL:        stripeapi.String(c.successURL),
		CancelURL:         stripeapi.String(c.cancelURL),
		ClientReferenceID: stripeapi.String(userID.String()),
		LineItems: []*stripeapi.CheckoutSessionLineItemParams{{
			Price:    stripeapi.String(c.priceID),
			Quantity: stripeapi.Int64(1),
		}},
		Metadata: map[string]string{
			"user_id": userID.String(),
		},
	}
	if normalizedEmail := strings.TrimSpace(email); normalizedEmail != "" {
		params.CustomerEmail = stripeapi.String(normalizedEmail)
	}
	params.Context = ctx

	result, err := session.New(params)
	if err != nil {
		return "", err
	}
	return result.URL, nil
}

// CancelSubscription cancels the subscription immediately. An already
// cancelled or missing subscription is treated as success so account deletion
// stays idempotent across retries.
func (c *CheckoutClient) CancelSubscription(ctx context.Context, subscriptionID string) error {
	if strings.TrimSpace(subscriptionID) == "" {
		return nil
	}
	if c.secretKey == "" {
		return fmt.Errorf("stripe: cancel subscription: %w", domain.ErrInvalidInput)
	}

	stripeapi.Key = c.secretKey
	params := &stripeapi.SubscriptionCancelParams{}
	params.Context = ctx
	if _, err := subscription.Cancel(subscriptionID, params); err != nil {
		var stripeErr *stripeapi.Error
		if errors.As(err, &stripeErr) && stripeErr.Code == stripeapi.ErrorCodeResourceMissing {
			return nil
		}
		return fmt.Errorf("stripe: cancel subscription %s: %w", subscriptionID, err)
	}
	return nil
}
