package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type GlobalLLMBudget struct {
	BudgetDate           time.Time
	MaxRequests          int
	UsedRequests         int
	MaxEstimatedCostYen  int
	UsedEstimatedCostYen int
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type ReserveGlobalLLMBudgetInput struct {
	BudgetDate         time.Time
	DefaultMaxRequests int
	DefaultMaxCostYen  int
	RequestCount       int
	EstimatedCostYen   int
}

type LLMUsageLogInput struct {
	ID               uuid.UUID
	UserID           uuid.UUID
	JobID            *uuid.UUID
	Provider         string
	Model            string
	InputTokens      int
	OutputTokens     int
	EstimatedCostYen float64
	CreatedAt        time.Time
}

type GlobalLLMBudgetRepository interface {
	Reserve(ctx context.Context, input ReserveGlobalLLMBudgetInput) (*GlobalLLMBudget, error)
	RecordUsage(ctx context.Context, input LLMUsageLogInput) error
}

type LLMUsage struct {
	InputTokens      int
	OutputTokens     int
	EstimatedCostYen float64
}

type LLMUsageReporter interface {
	LastUsage() (LLMUsage, bool)
}

type LLMProviderNamer interface {
	ProviderName() string
}
