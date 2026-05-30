package router

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/shout/ai-study-tool/backend/internal/di"
	"github.com/shout/ai-study-tool/backend/internal/domain"
	"github.com/shout/ai-study-tool/backend/internal/handler"
	appmiddleware "github.com/shout/ai-study-tool/backend/internal/middleware"
	"github.com/shout/ai-study-tool/backend/internal/usecase"
)

type routerTokenUsecase struct{}

func (routerTokenUsecase) Award(ctx context.Context, user *domain.User, input usecase.AwardAdTokensInput) (*domain.QuestionTokenBalance, error) {
	return nil, errors.New("not implemented")
}

func (routerTokenUsecase) AwardAdMobSSV(ctx context.Context, rawQuery string) error {
	return nil
}

func (routerTokenUsecase) Balance(ctx context.Context, user *domain.User) (*domain.QuestionTokenBalance, error) {
	return nil, errors.New("not implemented")
}

type routerRateLimitStore struct {
	exceeded bool
	userID   string
	bucket   string
}

func (s *routerRateLimitStore) IncrementAndCheck(ctx context.Context, userID, bucket string, limit int64) (int64, bool, error) {
	s.userID = userID
	s.bucket = bucket
	return limit + 1, s.exceeded, nil
}

func TestAdMobSSVWebhookAppliesRateLimit(t *testing.T) {
	store := &routerRateLimitStore{exceeded: true}
	rateLimit, err := appmiddleware.NewShortWindowRateLimitMiddleware(store, "admob_ssv", 1, func(c echo.Context) string {
		return "ip_hash:test"
	})
	if err != nil {
		t.Fatalf("rate limit middleware: %v", err)
	}
	container := &di.Container{
		TokenHandler:                handler.NewTokenHandler(routerTokenUsecase{}, nil),
		StripeHandler:               handler.NewStripeHandler(nil, nil),
		AdMobSSVRateLimitMiddleware: rateLimit,
	}
	e := echo.New()
	registerWebhookRoutes(e, container)

	req := httptest.NewRequest(http.MethodGet, "/webhooks/admob/ssv?transaction_id=txn-1", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rec.Code)
	}
	if store.bucket == "" || store.userID == "" {
		t.Fatal("expected rate limit store to be called")
	}
}

func TestAdMobSSVWebhookPassesWithinRateLimit(t *testing.T) {
	store := &routerRateLimitStore{}
	rateLimit, err := appmiddleware.NewShortWindowRateLimitMiddleware(store, "admob_ssv", 1, func(c echo.Context) string {
		return "ip_hash:test"
	})
	if err != nil {
		t.Fatalf("rate limit middleware: %v", err)
	}
	container := &di.Container{
		TokenHandler:                handler.NewTokenHandler(routerTokenUsecase{}, nil),
		StripeHandler:               handler.NewStripeHandler(nil, nil),
		AdMobSSVRateLimitMiddleware: rateLimit,
	}
	e := echo.New()
	registerWebhookRoutes(e, container)

	req := httptest.NewRequest(http.MethodGet, "/webhooks/admob/ssv?transaction_id=txn-1", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if body := rec.Body.String(); body != "" {
		t.Fatalf("expected empty body, got %q", body)
	}
}

func TestLegacyTokenAwardRouteIsNotRegisteredInProduction(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	e := echo.New()
	api := e.Group("/api/v1")
	container := &di.Container{
		TokenHandler:  handler.NewTokenHandler(routerTokenUsecase{}, nil),
		StripeHandler: handler.NewStripeHandler(nil, nil),
	}
	registerMonetizationRoutes(api, container, passThroughMiddleware, passThroughMiddleware)

	for _, route := range e.Routes() {
		if route.Method == http.MethodPost && route.Path == "/api/v1/tokens/award" {
			t.Fatal("legacy token award route must not be registered in production")
		}
	}
}

func TestLegacyTokenAwardRouteIsRegisteredOutsideProduction(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	e := echo.New()
	api := e.Group("/api/v1")
	container := &di.Container{
		TokenHandler:  handler.NewTokenHandler(routerTokenUsecase{}, nil),
		StripeHandler: handler.NewStripeHandler(nil, nil),
	}
	registerMonetizationRoutes(api, container, passThroughMiddleware, passThroughMiddleware)

	for _, route := range e.Routes() {
		if route.Method == http.MethodPost && route.Path == "/api/v1/tokens/award" {
			return
		}
	}
	t.Fatal("legacy token award route should remain available outside production")
}

func passThroughMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		return next(c)
	}
}
