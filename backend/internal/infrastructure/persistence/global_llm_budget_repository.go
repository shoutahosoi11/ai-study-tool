package persistence

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/google/uuid"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

type globalLLMBudgetRepository struct {
	db *sql.DB
}

func NewGlobalLLMBudgetRepository(db *sql.DB) domain.GlobalLLMBudgetRepository {
	return &globalLLMBudgetRepository{db: db}
}

func (r *globalLLMBudgetRepository) Reserve(ctx context.Context, input domain.ReserveGlobalLLMBudgetInput) (*domain.GlobalLLMBudget, error) {
	if input.DefaultMaxRequests < 0 || input.DefaultMaxCostYen < 0 || input.RequestCount <= 0 || input.EstimatedCostYen < 0 {
		return nil, domain.ErrInvalidInput
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("global llm budget repo: begin reserve: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
INSERT INTO global_llm_budgets (budget_date, max_requests, max_estimated_cost_yen)
VALUES ($1, $2, $3)
ON CONFLICT (budget_date) DO NOTHING
`, input.BudgetDate, input.DefaultMaxRequests, input.DefaultMaxCostYen); err != nil {
		return nil, fmt.Errorf("global llm budget repo: ensure daily budget: %w", err)
	}

	var budget domain.GlobalLLMBudget
	err = tx.QueryRowContext(ctx, `
UPDATE global_llm_budgets
SET used_requests = used_requests + $2,
    used_estimated_cost_yen = used_estimated_cost_yen + $3,
    updated_at = now()
WHERE budget_date = $1
  AND used_requests + $2 <= max_requests
  AND used_estimated_cost_yen + $3 <= max_estimated_cost_yen
RETURNING budget_date, max_requests, used_requests, max_estimated_cost_yen,
          used_estimated_cost_yen, created_at, updated_at
`, input.BudgetDate, input.RequestCount, input.EstimatedCostYen).Scan(
		&budget.BudgetDate,
		&budget.MaxRequests,
		&budget.UsedRequests,
		&budget.MaxEstimatedCostYen,
		&budget.UsedEstimatedCostYen,
		&budget.CreatedAt,
		&budget.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrGlobalLLMBudgetExceeded
		}
		return nil, fmt.Errorf("global llm budget repo: reserve: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("global llm budget repo: commit reserve: %w", err)
	}
	return &budget, nil
}

func (r *globalLLMBudgetRepository) RecordUsage(ctx context.Context, input domain.LLMUsageLogInput) error {
	if input.ID == uuid.Nil || input.UserID == uuid.Nil ||
		strings.TrimSpace(input.Provider) == "" || strings.TrimSpace(input.Model) == "" ||
		input.InputTokens < 0 || input.OutputTokens < 0 || input.EstimatedCostYen < 0 ||
		math.IsNaN(input.EstimatedCostYen) || math.IsInf(input.EstimatedCostYen, 0) {
		return domain.ErrInvalidInput
	}

	var jobID any
	if input.JobID != nil {
		jobID = *input.JobID
	}

	if _, err := r.db.ExecContext(ctx, `
INSERT INTO llm_usage_logs (
  id, user_id, job_id, provider, model, input_tokens, output_tokens,
  estimated_cost_yen, created_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
`, input.ID, input.UserID, jobID, strings.TrimSpace(input.Provider), strings.TrimSpace(input.Model),
		input.InputTokens, input.OutputTokens, input.EstimatedCostYen, input.CreatedAt); err != nil {
		return fmt.Errorf("global llm budget repo: record usage: %w", err)
	}
	return nil
}

var _ domain.GlobalLLMBudgetRepository = (*globalLLMBudgetRepository)(nil)
