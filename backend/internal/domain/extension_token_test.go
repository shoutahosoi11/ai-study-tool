package domain

import "testing"

func TestHashExtensionTokenDoesNotExposeRawToken(t *testing.T) {
	rawToken := "ext_raw-secret-token"
	hash := HashExtensionToken(rawToken)

	if hash == rawToken {
		t.Fatal("expected hashed token to differ from raw token")
	}
	if !IsExtensionRawToken(rawToken) {
		t.Fatal("expected extension token prefix")
	}
	if IsExtensionRawToken(hash) {
		t.Fatal("hashed token must not retain extension token prefix")
	}
}

func TestHasScope(t *testing.T) {
	scopes := []string{ExtensionScopeHighlightCheck, ExtensionScopeHighlightWrite}
	if !HasScope(scopes, ExtensionScopeHighlightWrite) {
		t.Fatal("expected scope")
	}
	if HasScope(scopes, ExtensionScopeQuestionGenerate) {
		t.Fatal("did not expect question generation scope")
	}
}

func TestNormalizeExtensionScopesKeepsOnlyAllowedScopes(t *testing.T) {
	scopes := NormalizeExtensionScopes([]string{
		ExtensionScopeHighlightCheck,
		ExtensionScopeQuestionGenerate,
		ExtensionScopeHighlightCheck,
		ExtensionScopeRevokeSelf,
	})

	if len(scopes) != 2 {
		t.Fatalf("unexpected scopes: %#v", scopes)
	}
	if !HasScope(scopes, ExtensionScopeHighlightCheck) {
		t.Fatal("expected highlight check scope")
	}
	if !HasScope(scopes, ExtensionScopeRevokeSelf) {
		t.Fatal("expected revoke self scope")
	}
	if HasScope(scopes, ExtensionScopeQuestionGenerate) {
		t.Fatal("question generation must not be an allowed extension scope")
	}
}
