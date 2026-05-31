package persistence

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMarshalAuditMetadataDropsSensitiveKeys(t *testing.T) {
	metadata, err := marshalAuditMetadata(map[string]any{
		"token":          "raw-token",
		"cookie_value":   "session",
		"prompt_text":    "prompt",
		"highlight_body": "highlight",
		"raw_payload":    "payload",
		"query_kind":     "uuid",
		"result_count":   2,
	})
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}
	for _, forbidden := range []string{"raw-token", "session", "prompt", "highlight", "payload"} {
		if strings.Contains(metadata, forbidden) {
			t.Fatalf("metadata contains sensitive value %q: %s", forbidden, metadata)
		}
	}

	var decoded map[string]any
	if err := json.Unmarshal([]byte(metadata), &decoded); err != nil {
		t.Fatalf("decode metadata: %v", err)
	}
	if decoded["query_kind"] != "uuid" || decoded["result_count"].(float64) != 2 {
		t.Fatalf("safe metadata missing: %#v", decoded)
	}
}
