package persistence

import "testing"

func TestPromptUsedForStorageRedactsPromptText(t *testing.T) {
	if got := promptUsedForStorage("generate questions from private highlights"); got != "[redacted]" {
		t.Fatalf("promptUsedForStorage() = %q, want redacted placeholder", got)
	}
	if got := promptUsedForStorage(""); got != "" {
		t.Fatalf("promptUsedForStorage(empty) = %q, want empty", got)
	}
}
