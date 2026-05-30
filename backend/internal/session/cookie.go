package session

import "github.com/shout/ai-study-tool/backend/internal/config"

const (
	DevelopmentSessionCookieName = "session"
	HostSessionCookieName        = "__Host-session"
	CSRFCookieName               = "csrf_token"
)

func CookieName(appEnv string) string {
	if config.NormalizeAppEnv(appEnv).IsDevelopment() {
		return DevelopmentSessionCookieName
	}
	return HostSessionCookieName
}

func SecureCookie(appEnv string) bool {
	return !config.NormalizeAppEnv(appEnv).IsDevelopment()
}
