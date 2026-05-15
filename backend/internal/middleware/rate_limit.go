package middleware

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
)

const defaultRateLimitRetryAfter = 24 * time.Hour

type RateLimitStore interface {
	IncrementAndCheck(ctx context.Context, userID, bucket string, limit int64) (current int64, exceeded bool, err error)
}

type RateLimitMiddleware struct {
	store  RateLimitStore
	bucket string
	limit  int64
	now    func() time.Time
}

func NewRateLimitMiddleware(store RateLimitStore, bucket string, limit int64) (*RateLimitMiddleware, error) {
	if store == nil {
		return nil, errors.New("rate limit middleware: store is nil")
	}
	if bucket == "" {
		return nil, errors.New("rate limit middleware: bucket is empty")
	}
	if limit <= 0 {
		return nil, errors.New("rate limit middleware: limit must be positive")
	}

	return &RateLimitMiddleware{
		store:  store,
		bucket: bucket,
		limit:  limit,
		now:    time.Now,
	}, nil
}

func (m *RateLimitMiddleware) Limit(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		userID, ok := GetFirebaseUID(c)
		if !ok {
			return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
		}

		_, exceeded, err := m.store.IncrementAndCheck(c.Request().Context(), userID, m.bucket, m.limit)
		if err != nil {
			log.Printf("rate limit increment error: %v", err)
			return echo.NewHTTPError(http.StatusServiceUnavailable, "rate limit service unavailable")
		}
		if exceeded {
			c.Response().Header().Set(echo.HeaderRetryAfter, strconv.FormatInt(retryAfterUntilNextUTCDay(m.now()), 10))
			return echo.NewHTTPError(http.StatusTooManyRequests, "rate limit exceeded")
		}

		return next(c)
	}
}

func retryAfterUntilNextUTCDay(now time.Time) int64 {
	utcNow := now.UTC()
	nextDay := time.Date(utcNow.Year(), utcNow.Month(), utcNow.Day()+1, 0, 0, 0, 0, time.UTC)
	seconds := int64(nextDay.Sub(utcNow).Seconds())
	if seconds <= 0 {
		return int64(defaultRateLimitRetryAfter.Seconds())
	}
	return seconds
}
