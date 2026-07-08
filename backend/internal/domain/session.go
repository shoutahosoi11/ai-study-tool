package domain

import (
	"context"
	"time"
)

type AuthToken struct {
	UID      string
	AuthTime time.Time
	Claims   map[string]any
}

type SessionVerifier interface {
	VerifySessionCookie(ctx context.Context, sessionCookie string) (*AuthToken, error)
}

type SessionCookieManager interface {
	SessionVerifier
	VerifyIDToken(ctx context.Context, idToken string) (*AuthToken, error)
	CreateSessionCookie(ctx context.Context, idToken string, expiresIn time.Duration) (string, error)
	RevokeRefreshTokens(ctx context.Context, uid string) error
}

// AuthAccountManager covers auth-provider account operations used by
// account deletion: revoking sessions and removing the auth user itself.
type AuthAccountManager interface {
	RevokeRefreshTokens(ctx context.Context, uid string) error
	DeleteUser(ctx context.Context, uid string) error
}
