package middleware

import "github.com/labstack/echo/v4"

func InternalOnly() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Cloud Run ingress=internal-and-cloud-load-balancing is the Phase 1
			// boundary. OIDC verification will be added when Cloud Tasks is wired.
			return next(c)
		}
	}
}
