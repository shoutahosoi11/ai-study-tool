package domain

import (
	"context"

	"github.com/google/uuid"
)

type UserRepository interface {
	GetByFirebaseUID(ctx context.Context, firebaseUID string) (*User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
	Create(ctx context.Context, input CreateUserInput) (*User, error)
	Update(ctx context.Context, id uuid.UUID, input UpdateUserInput) (*User, error)
	UpdateQuestionSettings(ctx context.Context, id uuid.UUID, input UpdateQuestionSettingsInput) (*User, error)
}

// UserAccountDeleter deletes a user row; the schema cascades to all owned
// data, and admin identities block deletion via ON DELETE RESTRICT.
type UserAccountDeleter interface {
	DeleteByID(ctx context.Context, id uuid.UUID) error
	// GetStripeSubscriptionID returns the user's active Stripe subscription id,
	// or "" when the user has none. Read before DeleteByID: the column is gone
	// once the row is deleted.
	GetStripeSubscriptionID(ctx context.Context, id uuid.UUID) (string, error)
}
