package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type ExtensionPairing struct {
	ID         uuid.UUID
	UserCode   string
	UserID     *uuid.UUID
	TokenID    *uuid.UUID
	Scopes     []string
	CreatedAt  time.Time
	ExpiresAt  time.Time
	ApprovedAt *time.Time
	UsedAt     *time.Time
}

type CreateExtensionTokenForPairingInput struct {
	PairingID uuid.UUID
	TokenID   uuid.UUID
	TokenHash string
	Name      *string
	Scopes    []string
	ExpiresAt time.Time
	Now       time.Time
}

type ExtensionPairingRepository interface {
	CreateExtensionPairing(ctx context.Context, userCode string, expiresAt time.Time) (*ExtensionPairing, error)
	GetExtensionPairingStatus(ctx context.Context, pairingID uuid.UUID, now time.Time) (*ExtensionPairing, error)
	ApproveExtensionPairing(ctx context.Context, userCode string, userID uuid.UUID, scopes []string, now time.Time) error
	CreateExtensionTokenForApprovedPairing(ctx context.Context, input CreateExtensionTokenForPairingInput) (*ExtensionToken, error)
	RevokeExtensionToken(ctx context.Context, tokenID uuid.UUID, userID uuid.UUID, now time.Time) error
}
