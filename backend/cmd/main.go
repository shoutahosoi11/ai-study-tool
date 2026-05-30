package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	appconfig "github.com/shout/ai-study-tool/backend/internal/config"
	"github.com/shout/ai-study-tool/backend/internal/di"
	dbinfra "github.com/shout/ai-study-tool/backend/internal/infrastructure/db"
	"github.com/shout/ai-study-tool/backend/internal/logging"
	"github.com/shout/ai-study-tool/backend/internal/router"
)

const (
	readinessTimeout       = 2 * time.Second
	startupDatabaseTimeout = 5 * time.Second
	defaultShutdownSeconds = 90
	readHeaderTimeout      = 10 * time.Second
	readTimeout            = 120 * time.Second
	writeTimeout           = 130 * time.Second
	idleTimeout            = 60 * time.Second
)

func main() {
	if !appconfig.CurrentAppEnv().IsProduction() && godotenv.Load() != nil {
		log.Println("No .env file found, using environment variables")
	}

	logging.Setup(os.Getenv("APP_ENV"))

	db, err := dbinfra.Open(os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	configureDatabasePool(db)
	if err := pingDatabase(db); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}

	container, err := di.NewContainer(db)
	if err != nil {
		log.Fatalf("failed to build DI container: %v", err)
	}
	defer container.Close()

	e := echo.New()
	// Cloud Run terminates TCP at Google's frontend, so RemoteAddr is a shared
	// frontend address and cannot be used directly for per-client rate limits.
	// Cloud Run appends the client address to X-Forwarded-For; use the rightmost
	// valid value and fall back to RemoteAddr for local/non-proxy execution.
	e.IPExtractor = cloudRunIPExtractor
	configureServerTimeouts(e)
	e.Use(requestLogger())
	e.Use(echomiddleware.Recover())
	e.Use(container.SecurityHeadersMiddleware.Secure)
	e.Use(echomiddleware.CORSWithConfig(echomiddleware.CORSConfig{
		AllowOrigins:     allowedOrigins(),
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type", "X-CSRF-Token"},
		AllowCredentials: true,
	}))

	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})
	e.GET("/ready", readinessHandler(db))

	router.RegisterAPI(e, container)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- e.Start(":" + port)
	}()

	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-serverErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("server failed: %v", err)
			return
		}
	case <-signalCtx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout())
		defer cancel()
		if err := e.Shutdown(shutdownCtx); err != nil {
			log.Printf("server shutdown error: %v", err)
		}
	}
}

func cloudRunIPExtractor(req *http.Request) string {
	xff := req.Header.Get(echo.HeaderXForwardedFor)
	if xff != "" {
		parts := strings.Split(xff, ",")
		for i := len(parts) - 1; i >= 0; i-- {
			value := strings.TrimSpace(parts[i])
			value = strings.TrimPrefix(value, "[")
			value = strings.TrimSuffix(value, "]")
			if ip := net.ParseIP(value); ip != nil {
				return ip.String()
			}
		}
	}
	return echo.ExtractIPDirect()(req)
}

func pingDatabase(db *sql.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), startupDatabaseTimeout)
	defer cancel()
	return db.PingContext(ctx)
}

func configureServerTimeouts(e *echo.Echo) {
	e.Server.ReadHeaderTimeout = readHeaderTimeout
	e.Server.ReadTimeout = readTimeout
	e.Server.WriteTimeout = writeTimeout
	e.Server.IdleTimeout = idleTimeout
}

func requestLogger() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			path := c.Path()
			if path == "/health" || path == "/ready" {
				return next(c)
			}

			startedAt := time.Now()
			err := next(c)
			if err != nil {
				c.Error(err)
			}

			req := c.Request()
			res := c.Response()
			status := res.Status
			level := slog.LevelInfo
			if status >= http.StatusInternalServerError {
				level = slog.LevelError
			} else if status >= http.StatusBadRequest {
				level = slog.LevelWarn
			}

			args := []any{
				"method", req.Method,
				"path", path,
				"request_path", req.URL.Path,
				"status", status,
				"latency_ms", time.Since(startedAt).Milliseconds(),
				"bytes_in", req.ContentLength,
				"bytes_out", res.Size,
				"remote_ip", c.RealIP(),
				"user_agent", req.UserAgent(),
			}
			if trace := cloudLoggingTrace(req.Header.Get("X-Cloud-Trace-Context")); trace != "" {
				args = append(args, "logging.googleapis.com/trace", trace)
			}
			slog.Log(req.Context(), level, "http_request", args...)
			return nil
		}
	}
}

func cloudLoggingTrace(header string) string {
	projectID := strings.TrimSpace(os.Getenv("GCP_PROJECT_ID"))
	if projectID == "" {
		projectID = strings.TrimSpace(os.Getenv("GOOGLE_CLOUD_PROJECT"))
	}
	if projectID == "" {
		return ""
	}

	traceID := strings.TrimSpace(header)
	if slash := strings.Index(traceID, "/"); slash >= 0 {
		traceID = traceID[:slash]
	}
	if semicolon := strings.Index(traceID, ";"); semicolon >= 0 {
		traceID = traceID[:semicolon]
	}
	if traceID == "" {
		return ""
	}
	return "projects/" + projectID + "/traces/" + traceID
}

func readinessHandler(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx, cancel := context.WithTimeout(c.Request().Context(), readinessTimeout)
		defer cancel()

		if err := db.PingContext(ctx); err != nil {
			log.Printf("readiness check failed: %v", err)
			return c.JSON(http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
		}

		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}
}

func allowedOrigins() []string {
	raw := strings.TrimSpace(os.Getenv("ALLOWED_ORIGINS"))
	if raw == "" {
		raw = strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS"))
	}
	if raw == "" {
		if appconfig.CurrentAppEnv().IsProduction() {
			log.Fatal("ALLOWED_ORIGINS is required in production")
		}
		return []string{"http://localhost:3000", "http://127.0.0.1:3000"}
	}

	parts := strings.Split(raw, ",")
	origins := make([]string, 0, len(parts))
	for _, part := range parts {
		origin := strings.TrimSpace(part)
		if origin != "" {
			origins = append(origins, origin)
		}
	}
	if len(origins) == 0 {
		if appconfig.CurrentAppEnv().IsProduction() {
			log.Fatal("ALLOWED_ORIGINS must include at least one origin in production")
		}
		return []string{"http://localhost:3000", "http://127.0.0.1:3000"}
	}

	return origins
}

func configureDatabasePool(db *sql.DB) {
	db.SetMaxOpenConns(readEnvInt("DB_MAX_OPEN_CONNS", 10))
	db.SetMaxIdleConns(readEnvInt("DB_MAX_IDLE_CONNS", 5))
	db.SetConnMaxLifetime(time.Duration(readEnvInt("DB_CONN_MAX_LIFETIME_SECONDS", 1800)) * time.Second)
	db.SetConnMaxIdleTime(time.Duration(readEnvInt("DB_CONN_MAX_IDLE_SECONDS", 300)) * time.Second)
}

func shutdownTimeout() time.Duration {
	return time.Duration(readEnvInt("SHUTDOWN_TIMEOUT_SECONDS", defaultShutdownSeconds)) * time.Second
}

func readEnvInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return fallback
	}
	return parsed
}
