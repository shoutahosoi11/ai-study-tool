package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

const defaultRateLimitRetryAfter = 24 * time.Hour
const defaultShortWindowRateLimitRetryAfter = time.Minute

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
			slog.Error("rate_limit_increment_failed", "error", err.Error())
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

type RateLimitIdentifierFunc func(echo.Context) string

type ShortWindowRateLimitMiddleware struct {
	store      RateLimitStore
	bucket     string
	limit      int64
	identifier RateLimitIdentifierFunc
	now        func() time.Time
}

func NewShortWindowRateLimitMiddleware(store RateLimitStore, bucket string, limit int64, identifier RateLimitIdentifierFunc) (*ShortWindowRateLimitMiddleware, error) {
	if store == nil {
		return nil, errors.New("short window rate limit middleware: store is nil")
	}
	if strings.TrimSpace(bucket) == "" {
		return nil, errors.New("short window rate limit middleware: bucket is empty")
	}
	if limit <= 0 {
		return nil, errors.New("short window rate limit middleware: limit must be positive")
	}
	if identifier == nil {
		return nil, errors.New("short window rate limit middleware: identifier is nil")
	}

	return &ShortWindowRateLimitMiddleware{
		store:      store,
		bucket:     strings.TrimSpace(bucket),
		limit:      limit,
		identifier: identifier,
		now:        time.Now,
	}, nil
}

func (m *ShortWindowRateLimitMiddleware) Limit(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		identifier := strings.TrimSpace(m.identifier(c))
		if identifier == "" {
			return echo.NewHTTPError(http.StatusTooManyRequests, "rate limit identifier unavailable")
		}

		bucket := m.bucket + ":" + m.now().UTC().Format("200601021504")
		_, exceeded, err := m.store.IncrementAndCheck(c.Request().Context(), identifier, bucket, m.limit)
		if err != nil {
			slog.Error("rate_limit_increment_failed", "error", err.Error())
			return echo.NewHTTPError(http.StatusServiceUnavailable, "rate limit service unavailable")
		}
		if exceeded {
			c.Response().Header().Set(echo.HeaderRetryAfter, strconv.FormatInt(int64(defaultShortWindowRateLimitRetryAfter.Seconds()), 10))
			return echo.NewHTTPError(http.StatusTooManyRequests, "rate limit exceeded")
		}

		return next(c)
	}
}

func ClientIPRateLimitIdentifier(c echo.Context) string {
	ip := strings.TrimSpace(c.RealIP())
	if ip == "" {
		return ""
	}
	return "ip_hash:" + hashRateLimitIdentifier(ip)
}

func hashRateLimitIdentifier(value string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(value)))
	return hex.EncodeToString(sum[:])
}
