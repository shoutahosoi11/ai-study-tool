package main

import (
	"net/http"
	"testing"
)

func TestCloudLoggingTrace(t *testing.T) {
	t.Setenv("GCP_PROJECT_ID", "project-id")

	got := cloudLoggingTrace("105445aa7843bc8bf206b120001000/1;o=1")
	want := "projects/project-id/traces/105445aa7843bc8bf206b120001000"
	if got != want {
		t.Fatalf("cloudLoggingTrace() = %q, want %q", got, want)
	}
}

func TestCloudLoggingTraceWithoutProject(t *testing.T) {
	t.Setenv("GCP_PROJECT_ID", "")
	t.Setenv("GOOGLE_CLOUD_PROJECT", "")

	if got := cloudLoggingTrace("105445aa7843bc8bf206b120001000/1;o=1"); got != "" {
		t.Fatalf("cloudLoggingTrace() = %q, want empty", got)
	}
}

func TestCloudRunIPExtractorUsesRightmostXForwardedFor(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "/", nil)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	req.RemoteAddr = "10.0.0.10:12345"
	req.Header.Set("X-Forwarded-For", "198.51.100.10, 203.0.113.20")

	if got := cloudRunIPExtractor(req); got != "203.0.113.20" {
		t.Fatalf("cloudRunIPExtractor() = %q, want rightmost XFF client IP", got)
	}
}

func TestCloudRunIPExtractorFallsBackToRemoteAddr(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "/", nil)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	req.RemoteAddr = "192.0.2.30:12345"

	if got := cloudRunIPExtractor(req); got != "192.0.2.30" {
		t.Fatalf("cloudRunIPExtractor() = %q, want RemoteAddr fallback", got)
	}
}
