package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestGenerateOnceUsesHeaderAPIKey(t *testing.T) {
	const apiKey = "secret-api-key"

	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if strings.Contains(r.URL.RawQuery, apiKey) {
			t.Fatalf("api key leaked in query: %s", r.URL.RawQuery)
		}
		if got := r.Header.Get("x-goog-api-key"); got != apiKey {
			t.Fatalf("unexpected api key header: %q", got)
		}
		if got := r.URL.Path; got != "/v1beta/models/gemini-test:generateContent" {
			t.Fatalf("unexpected path: %s", got)
		}

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		config, ok := payload["generationConfig"].(map[string]any)
		if !ok || config["responseMimeType"] != "application/json" {
			t.Fatalf("expected JSON response mime type, got %#v", payload["generationConfig"])
		}

		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewBufferString(`{"candidates":[{"content":{"parts":[{"text":"{\"questions\":[]}"}]}}]}`)),
		}, nil
	})}

	client := &Client{
		apiKey:          apiKey,
		httpClient:      httpClient,
		maxRetries:      0,
		retryDelay:      time.Millisecond,
		maxRetryDelay:   time.Millisecond,
		endpointBaseURL: "https://gemini.example.com",
	}

	if _, _, _, err := client.generateOnce(context.Background(), "gemini-test", "prompt"); err != nil {
		t.Fatalf("generateOnce returned error: %v", err)
	}
}

func TestParseJSONExtractsObjectFromPreface(t *testing.T) {
	var result struct {
		Questions []struct {
			Content string `json:"content"`
		} `json:"questions"`
	}

	raw := "以下が結果です:\n```json\n{\"questions\":[{\"content\":\"q\"}]}\n```"
	if err := parseJSON(raw, &result); err != nil {
		t.Fatalf("parseJSON returned error: %v", err)
	}
	if len(result.Questions) != 1 || result.Questions[0].Content != "q" {
		t.Fatalf("unexpected parsed result: %#v", result)
	}
}
