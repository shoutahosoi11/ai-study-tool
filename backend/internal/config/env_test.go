package config

import "testing"

func TestAppEnvHelpers(t *testing.T) {
	if got := NormalizeAppEnv(" Production "); got != AppEnvProduction {
		t.Fatalf("unexpected normalized env: %q", got)
	}
	if !AppEnvProduction.IsProduction() || !AppEnvStaging.IsStrictSecurity() || !AppEnvPreview.IsStrictSecurity() {
		t.Fatal("expected production-like envs to be strict")
	}
	if AppEnvStaging.AllowsDevBypass() || AppEnvPreview.AllowsDevBypass() || AppEnvProduction.AllowsDevBypass() {
		t.Fatal("production-like envs must not allow dev bypass")
	}
	if !AppEnvDevelopment.AllowsDevBypass() || !AppEnvTest.AllowsDevBypass() {
		t.Fatal("development/test should allow explicit dev bypasses")
	}
}
