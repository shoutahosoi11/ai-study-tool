package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"firebase.google.com/go/v4/auth"
	"github.com/labstack/echo/v4"
)

// TokenVerifier は Firebase のトークン検証を抽象化するための interface。
// auth.Client の具体的な実装に直接依存しないようにして、テストで差し替えやすくしている。
type TokenVerifier interface {
	VerifyIDTokenAndCheckRevoked(ctx context.Context, idToken string) (*auth.Token, error)
}

var isFirebaseIDTokenClientError = func(err error) bool {
	return auth.IsIDTokenRevoked(err) ||
		auth.IsUserDisabled(err) ||
		auth.IsIDTokenExpired(err) ||
		auth.IsIDTokenInvalid(err) ||
		auth.IsTenantIDMismatch(err)
}

func IsFirebaseIDTokenClientError(err error) bool {
	return isFirebaseIDTokenClientError(err)
}

type FirebaseMiddleware struct {
	verifier TokenVerifier
}

func NewFirebaseMiddleware(verifier TokenVerifier) (*FirebaseMiddleware, error) {
	if verifier == nil {
		return nil, errors.New("firebase middleware: verifier is nil")
	}

	return &FirebaseMiddleware{verifier: verifier}, nil
}

// Authenticate は Authorization ヘッダーの Firebase ID トークンを検証し、
// 認証に成功したユーザー情報を echo.Context に保存して次の handler に渡す。
func (m *FirebaseMiddleware) Authenticate(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		idToken, ok := bearerToken(c)
		if !ok {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid authorization header format")
		}

		token, err := m.verifier.VerifyIDTokenAndCheckRevoked(c.Request().Context(), idToken)
		if err != nil {
			return firebaseAuthError(err)
		}

		// err が nil でも token が使えない状態なら fail-close で止める。
		// 不完全な認証結果のまま downstream に進めないための防御。
		if token == nil || token.UID == "" {
			return echo.NewHTTPError(http.StatusServiceUnavailable, "authentication service unavailable")
		}

		setFirebaseAuth(c, token)
		return next(c)
	}
}

// firebaseAuthError は、クライアント側のトークン不正と認証基盤側の障害を分けて返す。
// 無効・失効・revoke 済み・disable 済みトークンは 401、想定外の検証失敗は 503 にする。
func firebaseAuthError(err error) *echo.HTTPError {
	if isFirebaseIDTokenClientError(err) {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid, expired, or revoked token")
	}

	// Infrastructure failure: without this log a Firebase outage is invisible
	// beyond the bare 503 status. The SDK error text carries no token material.
	slog.Error("firebase_id_token_verify_failed", "error", err.Error())
	return echo.NewHTTPError(http.StatusServiceUnavailable, "authentication service unavailable")
}

func bearerToken(c echo.Context) (string, bool) {
	authHeader := strings.TrimSpace(c.Request().Header.Get("Authorization"))
	if authHeader == "" {
		return "", false
	}

	parts := strings.Fields(authHeader)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") || strings.TrimSpace(parts[1]) == "" {
		return "", false
	}

	return parts[1], true
}
