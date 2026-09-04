package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"

	"github.com/shout/ai-study-tool/backend/internal/domain"
)

type BillingUsecase struct {
	sessionCreator domain.CheckoutSessionCreator
	webhook        domain.StripeWebhookValidator
	billingRepo    domain.BillingRepository
}

func NewBillingUsecase(sessionCreator domain.CheckoutSessionCreator, webhook domain.StripeWebhookValidator, billingRepo domain.BillingRepository) *BillingUsecase {
	return &BillingUsecase{sessionCreator: sessionCreator, webhook: webhook, billingRepo: billingRepo}
}

func (u *BillingUsecase) CreateCheckoutSession(ctx context.Context, user *domain.User, email string) (string, error) {
	return u.sessionCreator.CreateCheckoutSession(ctx, user.ID, email)
}

func (u *BillingUsecase) HandleWebhook(ctx context.Context, payload []byte, signature string) error {
	event, err := u.webhook.ConstructEvent(payload, signature)
	if err != nil {
		// Signature/parse failures must map to 400 so Stripe stops retrying;
		// processing failures below must not, so keep the classes distinct.
		return fmt.Errorf("%w: construct stripe event: %v", domain.ErrInvalidInput, err)
	}

	processed, err := u.billingRepo.ProcessStripeEvent(ctx, event, hashPayload(payload))
	if err != nil {
		return fmt.Errorf("process stripe event %s (%s): %w", event.ID, event.Type, err)
	}
	if !processed {
		slog.Info("stripe_event_skipped", "event_id", event.ID, "event_type", event.Type)
	}

	return nil
}

func hashPayload(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}
