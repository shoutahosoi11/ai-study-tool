package middleware

import (
	"firebase.google.com/go/v4/auth"
	"github.com/labstack/echo/v4"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

const (
	ContextFirebaseUIDKey   = "firebase_uid"
	ContextFirebaseTokenKey = "firebase_token"
)

// GetFirebaseUID は auth middleware が Context に保存した Firebase UID を取り出す。
// handler では Context のキーを直接読むのではなく、この helper を使う想定。
func GetFirebaseUID(c echo.Context) (string, bool) {
	firebaseUID, ok := c.Get(ContextFirebaseUIDKey).(string)
	return firebaseUID, ok && firebaseUID != ""
}

// GetFirebaseToken は auth middleware が Context に保存した Firebase token を取り出す。
// UID だけで足りる場合は GetFirebaseUID を使い、claims などが必要な時だけ使う。
func GetFirebaseToken(c echo.Context) (*auth.Token, bool) {
	token, ok := c.Get(ContextFirebaseTokenKey).(*auth.Token)
	return token, ok && token != nil
}

func GetAuthClaims(c echo.Context) (map[string]any, bool) {
	switch token := c.Get(ContextFirebaseTokenKey).(type) {
	case *auth.Token:
		if token == nil || token.Claims == nil {
			return nil, false
		}
		return token.Claims, true
	case *domain.AuthToken:
		if token == nil || token.Claims == nil {
			return nil, false
		}
		return token.Claims, true
	default:
		return nil, false
	}
}

// setFirebaseAuth は認証結果を downstream の handler で使えるよう Context に保存する。
func setFirebaseAuth(c echo.Context, token *auth.Token) {
	if token == nil {
		return
	}

	c.Set(ContextFirebaseUIDKey, token.UID)
	c.Set(ContextFirebaseTokenKey, token)
}

func setSessionAuth(c echo.Context, token *domain.AuthToken) {
	if token == nil {
		return
	}

	c.Set(ContextFirebaseUIDKey, token.UID)
	c.Set(ContextFirebaseTokenKey, token)
}
