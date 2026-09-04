package firebase

import (
	"context"
	"time"

	"firebase.google.com/go/v4/auth"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

type SessionCookieClient struct {
	client *auth.Client
	cache  *revocationCache
}

func NewSessionCookieClient(client *auth.Client) *SessionCookieClient {
	return &SessionCookieClient{client: client, cache: newRevocationCache(revocationCheckTTL)}
}

func (c *SessionCookieClient) VerifyIDToken(ctx context.Context, idToken string) (*domain.AuthToken, error) {
	token, err := c.client.VerifyIDToken(ctx, idToken)
	if err != nil {
		return nil, err
	}
	return firebaseTokenToDomain(token), nil
}

func (c *SessionCookieClient) CreateSessionCookie(ctx context.Context, idToken string, expiresIn time.Duration) (string, error) {
	return c.client.SessionCookie(ctx, idToken, expiresIn)
}

// VerifySessionCookie verifies the cookie signature locally against cached
// Firebase certs, and re-checks revocation against Firebase at most once per
// TTL per user. Without any revocation check, logout-all and account deletion
// would leave existing cookies valid until expiry (14 days); checking every
// request costs a remote call. The TTL bounds revocation latency instead.
func (c *SessionCookieClient) VerifySessionCookie(ctx context.Context, sessionCookie string) (*domain.AuthToken, error) {
	token, err := c.client.VerifySessionCookie(ctx, sessionCookie)
	if err != nil {
		return nil, err
	}
	if c.cache.fresh(token.UID, time.Now()) {
		return firebaseTokenToDomain(token), nil
	}
	token, err = c.client.VerifySessionCookieAndCheckRevoked(ctx, sessionCookie)
	if err != nil {
		return nil, err
	}
	c.cache.store(token.UID, time.Now())
	return firebaseTokenToDomain(token), nil
}

func (c *SessionCookieClient) RevokeRefreshTokens(ctx context.Context, uid string) error {
	if err := c.client.RevokeRefreshTokens(ctx, uid); err != nil {
		return err
	}
	c.cache.evict(uid)
	return nil
}

func (c *SessionCookieClient) DeleteUser(ctx context.Context, uid string) error {
	if err := c.client.DeleteUser(ctx, uid); err != nil {
		return err
	}
	c.cache.evict(uid)
	return nil
}

func firebaseTokenToDomain(token *auth.Token) *domain.AuthToken {
	if token == nil {
		return nil
	}
	return &domain.AuthToken{
		UID:      token.UID,
		AuthTime: time.Unix(token.AuthTime, 0).UTC(),
		Claims:   token.Claims,
	}
}
