package firebase

import (
	"context"
	"strings"

	"firebase.google.com/go/v4/appcheck"
)

type AppCheckVerifier struct {
	client *appcheck.Client
}

func NewAppCheckVerifier(client *appcheck.Client) *AppCheckVerifier {
	return &AppCheckVerifier{client: client}
}

func (v *AppCheckVerifier) VerifyAppCheckToken(ctx context.Context, token string) error {
	// Firebase Admin Go App Check VerifyToken does not accept context today;
	// the middleware interface keeps ctx so tests and future SDK support can
	// share the same boundary.
	_ = ctx
	_, err := v.client.VerifyToken(strings.TrimSpace(token))
	return err
}
