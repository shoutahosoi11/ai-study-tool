package firebase

import (
	"context"
	"time"

	"firebase.google.com/go/v4/auth"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

type SessionCookieClient struct {
	client *auth.Client
}

func NewSessionCookieClient(client *auth.Client) *SessionCookieClient {
	return &SessionCookieClient{client: client}
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

func (c *SessionCookieClient) VerifySessionCookieAndCheckRevoked(ctx context.Context, sessionCookie string) (*domain.AuthToken, error) {
	token, err := c.client.VerifySessionCookieAndCheckRevoked(ctx, sessionCookie)
	if err != nil {
		return nil, err
	}
	return firebaseTokenToDomain(token), nil
}

func (c *SessionCookieClient) RevokeRefreshTokens(ctx context.Context, uid string) error {
	return c.client.RevokeRefreshTokens(ctx, uid)
}

func (c *SessionCookieClient) DeleteUser(ctx context.Context, uid string) error {
	return c.client.DeleteUser(ctx, uid)
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
