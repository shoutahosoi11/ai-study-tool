package persistence

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

type billingRepository struct {
	db *sql.DB
}

const updateStripeSubscriptionUserSQL = `
UPDATE users
SET plan = $2,
    stripe_subscription_id = NULLIF($3, ''),
    subscription_expires_at = $4,
    updated_at = NOW()
WHERE id = $1
`

const cancelStripeSubscriptionUserSQL = `
UPDATE users
SET plan = 'free',
    subscription_expires_at = NULL,
    updated_at = NOW()
WHERE id = $1
`

func NewBillingRepository(db *sql.DB) domain.BillingRepository {
	return &billingRepository{db: db}
}

func (r *billingRepository) ProcessStripeEvent(ctx context.Context, event domain.StripeWebhookEvent, payloadHash string) (bool, error) {
	eventID := strings.TrimSpace(event.ID)
	eventType := strings.TrimSpace(event.Type)
	if eventID == "" || eventType == "" || strings.TrimSpace(payloadHash) == "" {
		return false, domain.ErrInvalidInput
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("billing repo: begin stripe event: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, `
INSERT INTO stripe_events (event_id, event_type, payload_hash)
VALUES ($1, $2, $3)
ON CONFLICT (event_id) DO NOTHING
`, eventID, eventType, strings.TrimSpace(payloadHash))
	if err != nil {
		return false, fmt.Errorf("billing repo: insert stripe event: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("billing repo: stripe event rows affected: %w", err)
	}
	if affected == 0 {
		return false, nil
	}

	if err := r.applyStripeEventTx(ctx, tx, event); err != nil {
		return false, err
	}

	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("billing repo: commit stripe event: %w", err)
	}
	return true, nil
}

func (r *billingRepository) applyStripeEventTx(ctx context.Context, tx *sql.Tx, event domain.StripeWebhookEvent) error {
	switch event.Type {
	case "checkout.session.completed":
		if event.UserID == uuid.Nil {
			return domain.ErrInvalidInput
		}
		status := normalizeSubscriptionStatus(event.Status, "active")
		if err := upsertSubscriptionTx(ctx, tx, event.UserID, event.CustomerID, event.SubscriptionID, event.ProductID, status, event.ExpiresAt); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `
UPDATE users
SET plan = 'premium',
    stripe_customer_id = NULLIF($2, ''),
    stripe_subscription_id = NULLIF($3, ''),
    subscription_expires_at = $4,
    updated_at = NOW()
WHERE id = $1
`, event.UserID, strings.TrimSpace(event.CustomerID), strings.TrimSpace(event.SubscriptionID), event.ExpiresAt)
		if err != nil {
			return fmt.Errorf("billing repo: apply checkout completed: %w", err)
		}
	case "customer.subscription.updated":
		userID, err := r.findStripeUserIDTx(ctx, tx, event.CustomerID, event.SubscriptionID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return nil
			}
			return err
		}
		status := normalizeSubscriptionStatus(event.Status, "active")
		if err := upsertSubscriptionTx(ctx, tx, userID, event.CustomerID, event.SubscriptionID, event.ProductID, status, event.ExpiresAt); err != nil {
			return err
		}
		plan := "premium"
		if status == "canceled" || status == "expired" {
			plan = "free"
		}
		_, err = tx.ExecContext(ctx, updateStripeSubscriptionUserSQL, userID, plan, strings.TrimSpace(event.SubscriptionID), event.ExpiresAt)
		if err != nil {
			return fmt.Errorf("billing repo: apply subscription updated: %w", err)
		}
	case "customer.subscription.deleted":
		userID, err := r.findStripeUserIDTx(ctx, tx, event.CustomerID, event.SubscriptionID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return nil
			}
			return err
		}
		if err := upsertSubscriptionTx(ctx, tx, userID, event.CustomerID, event.SubscriptionID, event.ProductID, "canceled", event.ExpiresAt); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, cancelStripeSubscriptionUserSQL, userID)
		if err != nil {
			return fmt.Errorf("billing repo: apply subscription deleted: %w", err)
		}
	}
	return nil
}

func (r *billingRepository) findStripeUserIDTx(ctx context.Context, tx *sql.Tx, customerID, subscriptionID string) (uuid.UUID, error) {
	var userID uuid.UUID
	err := tx.QueryRowContext(ctx, `
SELECT id
FROM users
WHERE stripe_customer_id = $1
   OR stripe_subscription_id = $2
LIMIT 1
`, strings.TrimSpace(customerID), strings.TrimSpace(subscriptionID)).Scan(&userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return uuid.Nil, domain.ErrNotFound
		}
		return uuid.Nil, fmt.Errorf("billing repo: find stripe user: %w", err)
	}
	return userID, nil
}

func upsertSubscriptionTx(ctx context.Context, tx *sql.Tx, userID uuid.UUID, customerID, subscriptionID, productID, status string, currentPeriodEnd *time.Time) error {
	subscriptionID = strings.TrimSpace(subscriptionID)
	if userID == uuid.Nil || subscriptionID == "" {
		return domain.ErrInvalidInput
	}
	_, err := tx.ExecContext(ctx, `
INSERT INTO subscriptions (
  user_id, provider, provider_user_id, subscription_id, product_id,
  status, current_period_end, updated_at
) VALUES ($1, 'stripe', NULLIF($2, ''), $3, NULLIF($4, ''), $5, $6, NOW())
ON CONFLICT (user_id, provider, subscription_id) DO UPDATE
SET provider_user_id = EXCLUDED.provider_user_id,
    product_id = EXCLUDED.product_id,
    status = EXCLUDED.status,
    current_period_end = EXCLUDED.current_period_end,
    updated_at = NOW()
`, userID, strings.TrimSpace(customerID), subscriptionID, strings.TrimSpace(productID), normalizeSubscriptionStatus(status, "active"), currentPeriodEnd)
	if err != nil {
		return fmt.Errorf("billing repo: upsert subscription: %w", err)
	}
	return nil
}

func normalizeSubscriptionStatus(status string, fallback string) string {
	switch strings.TrimSpace(status) {
	case "active", "trialing", "past_due", "canceled", "expired", "grace_period":
		return strings.TrimSpace(status)
	default:
		return fallback
	}
}

func (r *billingRepository) MarkCheckoutCompleted(ctx context.Context, userID uuid.UUID, customerID, subscriptionID string, expiresAt *time.Time) error {
	_, err := r.db.ExecContext(ctx, `
UPDATE users
SET plan = 'premium',
    stripe_customer_id = NULLIF($2, ''),
    stripe_subscription_id = NULLIF($3, ''),
    subscription_expires_at = $4,
    updated_at = NOW()
WHERE id = $1
`, userID, strings.TrimSpace(customerID), strings.TrimSpace(subscriptionID), expiresAt)
	if err != nil {
		return fmt.Errorf("billing repo: mark checkout completed: %w", err)
	}
	return nil
}

func (r *billingRepository) UpdateSubscription(ctx context.Context, customerID, subscriptionID string, expiresAt *time.Time) error {
	_, err := r.db.ExecContext(ctx, `
UPDATE users
SET plan = 'premium',
    stripe_subscription_id = NULLIF($2, ''),
    subscription_expires_at = $3,
    updated_at = NOW()
WHERE stripe_customer_id = $1
   OR stripe_subscription_id = $2
`, strings.TrimSpace(customerID), strings.TrimSpace(subscriptionID), expiresAt)
	if err != nil {
		return fmt.Errorf("billing repo: update subscription: %w", err)
	}
	return nil
}

func (r *billingRepository) CancelSubscription(ctx context.Context, customerID, subscriptionID string) error {
	_, err := r.db.ExecContext(ctx, `
UPDATE users
SET plan = 'free',
    subscription_expires_at = NULL,
    updated_at = NOW()
WHERE stripe_customer_id = $1
   OR stripe_subscription_id = $2
`, strings.TrimSpace(customerID), strings.TrimSpace(subscriptionID))
	if err != nil {
		return fmt.Errorf("billing repo: cancel subscription: %w", err)
	}
	return nil
}
