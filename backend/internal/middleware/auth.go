package middleware

import (
	"context"
	"net/http"
	"strings"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"github.com/labstack/echo/v4"
	"google.golang.org/api/option"
)

type FirebaseMiddleware struct {
	client *auth.Client
}

func NewFirebaseMiddleware(credentialsPath string) (*FirebaseMiddleware, error) {
	app, err := firebase.NewApp(context.Background(), nil, option.WithCredentialsFile(credentialsPath))
	if err != nil {
		return nil, err
	}
	client, err := app.Auth(context.Background())
	if err != nil {
		return nil, err
	}
	return &FirebaseMiddleware{client: client}, nil
}

func (m *FirebaseMiddleware) Authenticate(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		authHeader := c.Request().Header.Get("Authorization")
		if authHeader == "" {
			return echo.NewHTTPError(http.StatusUnauthorized, "missing authorization header")
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid authorization header format")
		}

		token, err := m.client.VerifyIDToken(c.Request().Context(), parts[1])
		if err != nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid or expired token")
		}

		c.Set("firebase_uid", token.UID)
		c.Set("firebase_token", token)
		return next(c)
	}
}
