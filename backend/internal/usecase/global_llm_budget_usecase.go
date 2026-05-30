package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	appconfig "github.com/shout/ai-study-tool/backend/internal/config"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

const (
	defaultGlobalLLMMaxRequestsPerDay = 500
	defaultGlobalLLMMaxCostYenPerDay  = 500
	defaultLLMEstimatedCostYen        = 1
)

type GlobalLLMBudgetUsecase struct {
	repo                    domain.GlobalLLMBudgetRepository
	now                     func() time.Time
	maxRequestsPerDay       int
	maxEstimatedCostYen     int
	estimatedCostPerRequest int
}

type GlobalLLMBudgetConfig struct {
	MaxRequestsPerDay       int
	MaxEstimatedCostYen     int
	EstimatedCostPerRequest int
}

func NewGlobalLLMBudgetUsecaseFromEnv(repo domain.GlobalLLMBudgetRepository, appEnv string) (*GlobalLLMBudgetUsecase, error) {
	config := GlobalLLMBudgetConfig{
		MaxRequestsPerDay:       readPositiveEnvIntOrDefault("GLOBAL_LLM_DAILY_MAX_REQUESTS", defaultGlobalLLMMaxRequestsPerDay),
		MaxEstimatedCostYen:     readPositiveEnvIntOrDefault("GLOBAL_LLM_DAILY_MAX_ESTIMATED_COST_YEN", defaultGlobalLLMMaxCostYenPerDay),
		EstimatedCostPerRequest: readPositiveEnvIntOrDefault("LLM_ESTIMATED_COST_YEN_PER_REQUEST", defaultLLMEstimatedCostYen),
	}
	if appconfig.NormalizeAppEnv(appEnv).IsProduction() {
		warnDangerousGlobalLLMBudget(config)
	}
	return NewGlobalLLMBudgetUsecase(repo, config)
}

func NewGlobalLLMBudgetUsecase(repo domain.GlobalLLMBudgetRepository, config GlobalLLMBudgetConfig) (*GlobalLLMBudgetUsecase, error) {
	if repo == nil {
		return nil, fmt.Errorf("global llm budget usecase: repository is nil")
	}
	if config.MaxRequestsPerDay <= 0 || config.MaxEstimatedCostYen <= 0 || config.EstimatedCostPerRequest <= 0 {
		return nil, fmt.Errorf("global llm budget usecase: invalid config")
	}
	return &GlobalLLMBudgetUsecase{
		repo:                    repo,
		now:                     time.Now,
		maxRequestsPerDay:       config.MaxRequestsPerDay,
		maxEstimatedCostYen:     config.MaxEstimatedCostYen,
		estimatedCostPerRequest: config.EstimatedCostPerRequest,
	}, nil
}

func newGlobalLLMBudgetUsecaseForTest(repo domain.GlobalLLMBudgetRepository, config GlobalLLMBudgetConfig, now func() time.Time) *GlobalLLMBudgetUsecase {
	uc, err := NewGlobalLLMBudgetUsecase(repo, config)
	if err != nil {
		panic(err)
	}
	if now != nil {
		uc.now = now
	}
	return uc
}

func (u *GlobalLLMBudgetUsecase) EstimateRequestCostYen(requestCount int) int {
	if requestCount <= 0 {
		return 0
	}
	return requestCount * u.estimatedCostPerRequest
}

func (u *GlobalLLMBudgetUsecase) Reserve(ctx context.Context, requestCount int, estimatedCostYen int) error {
	if requestCount <= 0 || estimatedCostYen < 0 {
		return domain.ErrInvalidInput
	}
	_, err := u.repo.Reserve(ctx, domain.ReserveGlobalLLMBudgetInput{
		BudgetDate:         globalLLMBudgetDate(u.now()),
		DefaultMaxRequests: u.maxRequestsPerDay,
		DefaultMaxCostYen:  u.maxEstimatedCostYen,
		RequestCount:       requestCount,
		EstimatedCostYen:   estimatedCostYen,
	})
	if err != nil {
		return fmt.Errorf("global llm budget usecase: reserve: %w", err)
	}
	return nil
}

func (u *GlobalLLMBudgetUsecase) RecordUsage(ctx context.Context, input domain.LLMUsageLogInput) error {
	if input.ID == uuid.Nil {
		input.ID = uuid.New()
	}
	if input.CreatedAt.IsZero() {
		input.CreatedAt = u.now().UTC()
	}
	if strings.TrimSpace(input.Provider) == "" {
		input.Provider = "unknown"
	}
	if strings.TrimSpace(input.Model) == "" {
		input.Model = "unknown"
	}
	if input.EstimatedCostYen < 0 || math.IsNaN(input.EstimatedCostYen) || math.IsInf(input.EstimatedCostYen, 0) {
		return domain.ErrInvalidInput
	}
	if err := u.repo.RecordUsage(ctx, input); err != nil {
		return fmt.Errorf("global llm budget usecase: record usage: %w", err)
	}
	return nil
}

func globalLLMBudgetDate(now time.Time) time.Time {
	loc, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		loc = time.FixedZone("JST", 9*60*60)
	}
	local := now.In(loc)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
}

func readPositiveEnvIntOrDefault(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func warnDangerousGlobalLLMBudget(config GlobalLLMBudgetConfig) {
	if config.MaxRequestsPerDay > defaultGlobalLLMMaxRequestsPerDay*100 {
		slog.Warn("global_llm_budget_config_high", "key", "GLOBAL_LLM_DAILY_MAX_REQUESTS", "value", config.MaxRequestsPerDay)
	}
	if config.MaxEstimatedCostYen > defaultGlobalLLMMaxCostYenPerDay*100 {
		slog.Warn("global_llm_budget_config_high", "key", "GLOBAL_LLM_DAILY_MAX_ESTIMATED_COST_YEN", "value", config.MaxEstimatedCostYen)
	}
}
