package middleware

import (
	"time"

	"firebase.google.com/go/v4/auth"
	"github.com/labstack/echo/v4"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

const (
	ContextFirebaseUIDKey      = "firebase_uid"
	ContextFirebaseTokenKey    = "firebase_token"
	ContextCurrentUserKey      = "current_user"
	ContextAuthClientTypeKey   = "auth_client_type"
	ContextAuthScopesKey       = "auth_scopes"
	ContextExtensionTokenIDKey = "extension_token_id"
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

func GetAuthTime(c echo.Context) (time.Time, bool) {
	switch token := c.Get(ContextFirebaseTokenKey).(type) {
	case *auth.Token:
		if token == nil || token.AuthTime <= 0 {
			return time.Time{}, false
		}
		return time.Unix(token.AuthTime, 0).UTC(), true
	case *domain.AuthToken:
		if token == nil || token.AuthTime.IsZero() {
			return time.Time{}, false
		}
		return token.AuthTime.UTC(), true
	default:
		return time.Time{}, false
	}
}

func GetCurrentUser(c echo.Context) (domain.AuthenticatedUser, bool) {
	user, ok := c.Get(ContextCurrentUserKey).(domain.AuthenticatedUser)
	return user, ok && user.FirebaseUID != ""
}

func GetAuthClientType(c echo.Context) (domain.AuthClientType, bool) {
	clientType, ok := c.Get(ContextAuthClientTypeKey).(domain.AuthClientType)
	return clientType, ok && clientType != ""
}

func GetAuthScopes(c echo.Context) []string {
	scopes, ok := c.Get(ContextAuthScopesKey).([]string)
	if !ok || len(scopes) == 0 {
		return nil
	}

	copied := make([]string, len(scopes))
	copy(copied, scopes)
	return copied
}

func GetExtensionTokenID(c echo.Context) (string, bool) {
	tokenID, ok := c.Get(ContextExtensionTokenIDKey).(string)
	return tokenID, ok && tokenID != ""
}

// setFirebaseAuth は認証結果を downstream の handler で使えるよう Context に保存する。
func setFirebaseAuth(c echo.Context, token *auth.Token) {
	if token == nil {
		return
	}

	c.Set(ContextFirebaseUIDKey, token.UID)
	c.Set(ContextFirebaseTokenKey, token)
	setAuthContext(c, domain.AuthenticatedUser{FirebaseUID: token.UID}, domain.AuthClientTypeMobile, nil)
}

func setSessionAuth(c echo.Context, token *domain.AuthToken) {
	if token == nil {
		return
	}

	c.Set(ContextFirebaseUIDKey, token.UID)
	c.Set(ContextFirebaseTokenKey, token)
	setAuthContext(c, domain.AuthenticatedUser{FirebaseUID: token.UID}, domain.AuthClientTypeWeb, nil)
}

func setExtensionAuth(c echo.Context, token *domain.ExtensionToken) {
	if token == nil {
		return
	}

	userID := token.UserID
	c.Set(ContextExtensionTokenIDKey, token.ID.String())
	c.Set(ContextFirebaseUIDKey, token.FirebaseUID)
	setAuthContext(c, domain.AuthenticatedUser{
		FirebaseUID: token.FirebaseUID,
		UserID:      &userID,
	}, domain.AuthClientTypeExtension, domain.NormalizeExtensionScopes(token.Scopes))
}

func setAuthContext(c echo.Context, user domain.AuthenticatedUser, clientType domain.AuthClientType, scopes []string) {
	c.Set(ContextCurrentUserKey, user)
	c.Set(ContextAuthClientTypeKey, clientType)
	if len(scopes) == 0 {
		c.Set(ContextAuthScopesKey, []string{})
		return
	}

	copied := make([]string, len(scopes))
	copy(copied, scopes)
	c.Set(ContextAuthScopesKey, copied)
}
