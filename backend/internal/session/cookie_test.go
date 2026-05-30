package session

import "testing"

func TestCookieSettingsByEnv(t *testing.T) {
	if CookieName("development") != DevelopmentSessionCookieName || SecureCookie("development") {
		t.Fatal("development should use local insecure session cookie")
	}
	if CookieName("production") != HostSessionCookieName || !SecureCookie("production") {
		t.Fatal("production should use secure __Host session cookie")
	}
	if CookieName("staging") != HostSessionCookieName || !SecureCookie("staging") {
		t.Fatal("staging should follow production cookie posture")
	}
}
