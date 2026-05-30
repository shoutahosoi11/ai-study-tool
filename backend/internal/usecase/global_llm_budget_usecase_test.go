package usecase

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

type fakeGlobalLLMBudgetRepository struct {
	mu        sync.Mutex
	usedReq   int
	usedCost  int
	usageLogs []domain.LLMUsageLogInput
}

func (r *fakeGlobalLLMBudgetRepository) Reserve(ctx context.Context, input domain.ReserveGlobalLLMBudgetInput) (*domain.GlobalLLMBudget, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.usedReq+input.RequestCount > input.DefaultMaxRequests ||
		r.usedCost+input.EstimatedCostYen > input.DefaultMaxCostYen {
		return nil, domain.ErrGlobalLLMBudgetExceeded
	}
	r.usedReq += input.RequestCount
	r.usedCost += input.EstimatedCostYen
	return &domain.GlobalLLMBudget{
		BudgetDate:           input.BudgetDate,
		MaxRequests:          input.DefaultMaxRequests,
		UsedRequests:         r.usedReq,
		MaxEstimatedCostYen:  input.DefaultMaxCostYen,
		UsedEstimatedCostYen: r.usedCost,
	}, nil
}

func (r *fakeGlobalLLMBudgetRepository) RecordUsage(ctx context.Context, input domain.LLMUsageLogInput) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.usageLogs = append(r.usageLogs, input)
	return nil
}

func TestGlobalLLMBudgetReserveRejectsCostExceeded(t *testing.T) {
	repo := &fakeGlobalLLMBudgetRepository{}
	uc := newGlobalLLMBudgetUsecaseForTest(repo, GlobalLLMBudgetConfig{
		MaxRequestsPerDay:       10,
		MaxEstimatedCostYen:     1,
		EstimatedCostPerRequest: 2,
	}, func() time.Time { return time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC) })

	err := uc.Reserve(context.Background(), 1, uc.EstimateRequestCostYen(1))
	if !errors.Is(err, domain.ErrGlobalLLMBudgetExceeded) {
		t.Fatalf("expected cost budget exceeded, got %v", err)
	}
}

func TestGlobalLLMBudgetReserveConcurrentDoesNotExceedMaxRequests(t *testing.T) {
	repo := &fakeGlobalLLMBudgetRepository{}
	uc := newGlobalLLMBudgetUsecaseForTest(repo, GlobalLLMBudgetConfig{
		MaxRequestsPerDay:       3,
		MaxEstimatedCostYen:     100,
		EstimatedCostPerRequest: 1,
	}, func() time.Time { return time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC) })

	var wg sync.WaitGroup
	errs := make(chan error, 10)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- uc.Reserve(context.Background(), 1, 1)
		}()
	}
	wg.Wait()
	close(errs)

	successes := 0
	for err := range errs {
		if err == nil {
			successes++
		}
	}
	if successes != 3 {
		t.Fatalf("expected exactly 3 successful reserves, got %d", successes)
	}
	if repo.usedReq != 3 {
		t.Fatalf("used requests exceeded max: %d", repo.usedReq)
	}
}

func TestGlobalLLMBudgetRecordUsageFillsDefaults(t *testing.T) {
	repo := &fakeGlobalLLMBudgetRepository{}
	now := time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC)
	uc := newGlobalLLMBudgetUsecaseForTest(repo, GlobalLLMBudgetConfig{
		MaxRequestsPerDay:       10,
		MaxEstimatedCostYen:     10,
		EstimatedCostPerRequest: 1,
	}, func() time.Time { return now })

	if err := uc.RecordUsage(context.Background(), domain.LLMUsageLogInput{
		UserID:           uuid.New(),
		Provider:         "",
		Model:            "",
		InputTokens:      10,
		OutputTokens:     20,
		EstimatedCostYen: 1,
	}); err != nil {
		t.Fatalf("record usage: %v", err)
	}
	if len(repo.usageLogs) != 1 {
		t.Fatalf("expected one usage log, got %d", len(repo.usageLogs))
	}
	log := repo.usageLogs[0]
	if log.ID == uuid.Nil || !log.CreatedAt.Equal(now) {
		t.Fatalf("expected defaults, got %#v", log)
	}
	if log.Provider != "unknown" || log.Model != "unknown" {
		t.Fatalf("unexpected provider/model defaults: %#v", log)
	}
}
