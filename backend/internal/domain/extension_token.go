package domain

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	ExtensionTokenPrefix = "ext_"

	// The first three scopes are grantable to Browser Extension tokens.
	ExtensionScopeHighlightWrite = "highlight:write"
	ExtensionScopeHighlightCheck = "highlight:check"
	ExtensionScopeRevokeSelf     = "extension:revoke-self"
	// The remaining scopes are route guard scopes. They are intentionally not
	// grantable to Browser Extension tokens; RequireScope uses them to deny
	// extension credentials on broad Web/Mobile-only routes.
	ExtensionScopeUserRead         = "user:read"
	ExtensionScopeUserWrite        = "user:write"
	ExtensionScopePostRead         = "post:read"
	ExtensionScopePostWrite        = "post:write"
	ExtensionScopeSocialWrite      = "social:write"
	ExtensionScopeQuestionRead     = "question:read"
	ExtensionScopeQuestionWrite    = "question:write"
	ExtensionScopeQuestionGenerate = "question:generate"
	ExtensionScopeTokenRead        = "token:read"
	ExtensionScopeTokenWrite       = "token:write"
	ExtensionScopeBillingWrite     = "billing:write"
	ExtensionScopeHighlightExplain = "highlight:explanation:write"
)

// Route-level scope constants include capabilities that Web/Mobile may use for
// deny-by-default checks. allowedExtensionScopes below is the only list that can
// actually be granted to extension tokens.
var allowedExtensionScopes = map[string]struct{}{
	ExtensionScopeHighlightWrite: {},
	ExtensionScopeHighlightCheck: {},
	ExtensionScopeRevokeSelf:     {},
}

type ExtensionToken struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	FirebaseUID string
	TokenHash   string
	Name        *string
	Scopes      []string
	CreatedAt   time.Time
	LastUsedAt  *time.Time
	ExpiresAt   *time.Time
	RevokedAt   *time.Time
}

type ExtensionTokenRepository interface {
	FindActiveByTokenHash(ctx context.Context, tokenHash string, now time.Time) (*ExtensionToken, error)
}

func DefaultExtensionTokenScopes() []string {
	return []string{
		ExtensionScopeHighlightWrite,
		ExtensionScopeHighlightCheck,
		ExtensionScopeRevokeSelf,
	}
}

func IsExtensionRawToken(rawToken string) bool {
	token := strings.TrimSpace(rawToken)
	return strings.HasPrefix(token, ExtensionTokenPrefix)
}

func HashExtensionToken(rawToken string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(rawToken)))
	return hex.EncodeToString(sum[:])
}

func HasScope(scopes []string, required string) bool {
	required = strings.TrimSpace(required)
	if required == "" {
		return false
	}

	for _, scope := range scopes {
		if subtle.ConstantTimeCompare([]byte(strings.TrimSpace(scope)), []byte(required)) == 1 {
			return true
		}
	}
	return false
}

func NormalizeExtensionScopes(scopes []string) []string {
	normalized := make([]string, 0, len(scopes))
	seen := make(map[string]struct{}, len(scopes))
	for _, scope := range scopes {
		scope = strings.TrimSpace(scope)
		if _, ok := allowedExtensionScopes[scope]; !ok {
			continue
		}
		if _, ok := seen[scope]; ok {
			continue
		}
		seen[scope] = struct{}{}
		normalized = append(normalized, scope)
	}
	return normalized
}
