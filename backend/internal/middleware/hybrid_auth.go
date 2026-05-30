package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

type HybridAuthMiddleware struct {
	sessionAuth   *SessionAuthMiddleware
	bearerAuth    *FirebaseMiddleware
	extensionAuth *ExtensionAuthMiddleware
	csrf          *CSRFMiddleware
	appEnv        string
	mobileGuards  []echo.MiddlewareFunc
}

func NewHybridAuthMiddleware(
	sessionAuth *SessionAuthMiddleware,
	bearerAuth *FirebaseMiddleware,
	extensionAuth *ExtensionAuthMiddleware,
	csrf *CSRFMiddleware,
	appEnv string,
	mobileGuards ...echo.MiddlewareFunc,
) (*HybridAuthMiddleware, error) {
	if sessionAuth == nil {
		return nil, errors.New("hybrid auth middleware: session auth is nil")
	}
	if bearerAuth == nil {
		return nil, errors.New("hybrid auth middleware: bearer auth is nil")
	}
	if extensionAuth == nil {
		return nil, errors.New("hybrid auth middleware: extension auth is nil")
	}
	if csrf == nil {
		return nil, errors.New("hybrid auth middleware: csrf is nil")
	}
	return &HybridAuthMiddleware{
		sessionAuth:   sessionAuth,
		bearerAuth:    bearerAuth,
		extensionAuth: extensionAuth,
		csrf:          csrf,
		appEnv:        strings.TrimSpace(appEnv),
		mobileGuards:  append([]echo.MiddlewareFunc(nil), mobileGuards...),
	}, nil
}

func (m *HybridAuthMiddleware) Authenticate(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Cookie credentials are treated as Web even when Authorization is also
		// present. Browser cookie requests need CSRF protection; choosing the
		// Bearer path first would let a mixed request bypass that Web-only guard.
		if hasSessionCookie(c, m.appEnv) {
			return m.sessionAuth.Authenticate(m.csrf.Protect(next))(c)
		}

		if rawToken, ok := bearerToken(c); ok {
			// Browser Extensions use dedicated ext_ tokens with route scopes. They
			// do not carry Firebase App Check because extension storage is scoped by
			// the extension token design rather than by a native app attestation.
			if domain.IsExtensionRawToken(rawToken) {
				return m.extensionAuth.Authenticate(next)(c)
			}
			// Non-ext_ Bearer credentials are Mobile Firebase ID Tokens. Mobile
			// requests do not need CSRF, but they do need App Check and version
			// gates so native API access can be rejected independently from Web.
			return m.bearerAuth.Authenticate(chainMiddleware(next, m.mobileGuards...))(c)
		}

		return echo.NewHTTPError(http.StatusUnauthorized, "missing authentication credentials")
	}
}

func chainMiddleware(next echo.HandlerFunc, middlewares ...echo.MiddlewareFunc) echo.HandlerFunc {
	handler := next
	for i := len(middlewares) - 1; i >= 0; i-- {
		if middlewares[i] == nil {
			continue
		}
		handler = middlewares[i](handler)
	}
	return handler
}
