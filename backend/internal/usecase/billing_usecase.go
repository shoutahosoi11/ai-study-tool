package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

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
		return err
	}

	processed, err := u.billingRepo.ProcessStripeEvent(ctx, event, hashPayload(payload))
	if err != nil || !processed {
		return err
	}

	return nil
}

func hashPayload(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}
