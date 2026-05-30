package security

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestSecretScanScriptDetectsForbiddenPatterns(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 is not available")
	}

	root := repositoryRoot(t)
	script := filepath.Join(root, "scripts", "secret_scan.py")
	tempDir := t.TempDir()

	cases := []struct {
		name      string
		content   string
		wantError bool
	}{
		{
			name:      "stripe live key",
			content:   "STRIPE" + "_SECRET_KEY=sk" + "_live_" + "abcdefghijklmnopqrstuvwxyz\n",
			wantError: true,
		},
		{
			name:      "vite backend secret",
			content:   "VITE_GEMINI" + "_API_KEY=real-client-bundle-secret\n",
			wantError: true,
		},
		{
			name:      "stripe webhook secret",
			content:   "STRIPE_WEBHOOK" + "_SECRET=wh" + "sec_abcdefghijklmnopqrstuvwxyz\n",
			wantError: true,
		},
		{
			name:      "private key block",
			content:   "FIREBASE_PRIVATE" + "_KEY=-----BEGIN " + "PRIVATE KEY-----\nabc\n-----END PRIVATE KEY-----\n",
			wantError: true,
		},
		{
			name:      "dummy examples",
			content:   "STRIPE" + "_SECRET_KEY=sk_test_YOUR_TEST_KEY\nCSRF" + "_SECRET=\nSTRIPE_WEBHOOK" + "_SECRET=whsec_YOUR_LOCAL_WEBHOOK_SECRET\n",
			wantError: false,
		},
		{
			name:      "secret manager references",
			content:   "GEMINI" + "_API_KEY=GEMINI" + "_API_KEY:latest\nSTRIPE" + "_SECRET_KEY=STRIPE" + "_SECRET_KEY:latest\n",
			wantError: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(tempDir, tc.name+".env")
			if err := os.WriteFile(path, []byte(tc.content), 0o600); err != nil {
				t.Fatalf("write fixture: %v", err)
			}
			cmd := exec.Command("python3", script, path)
			err := cmd.Run()
			if tc.wantError && err == nil {
				t.Fatal("expected secret scan to fail")
			}
			if !tc.wantError && err != nil {
				t.Fatalf("expected secret scan to pass: %v", err)
			}
		})
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve caller")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}
