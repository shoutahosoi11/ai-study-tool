package main

import "testing"

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
