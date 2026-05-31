package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

type adminRepository struct {
	db *sql.DB
}

func NewAdminRepository(db *sql.DB) domain.AdminRepository {
	return &adminRepository{db: db}
}

func (r *adminRepository) FindAdminIdentityByFirebaseUID(ctx context.Context, firebaseUID string) (*domain.AdminIdentity, error) {
	var role string
	var userID uuid.UUID
	err := r.db.QueryRowContext(ctx, `
SELECT u.id, ur.role
FROM users u
JOIN user_roles ur ON ur.user_id = u.id
WHERE u.firebase_uid = $1
LIMIT 1
`, strings.TrimSpace(firebaseUID)).Scan(&userID, &role)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("admin repo: find identity: %w", err)
	}
	adminRole := domain.AdminRole(role)
	if !domain.IsValidAdminRole(adminRole) {
		return nil, domain.ErrForbidden
	}
	return &domain.AdminIdentity{UserID: userID, Role: adminRole}, nil
}

func (r *adminRepository) CreateAuditLog(ctx context.Context, input domain.AdminAuditLogInput) error {
	if input.AdminUserID == uuid.Nil || strings.TrimSpace(input.Action) == "" || strings.TrimSpace(input.TargetType) == "" {
		return domain.ErrInvalidInput
	}
	metadata, err := marshalAuditMetadata(input.Metadata)
	if err != nil {
		return err
	}
	var targetID any
	if strings.TrimSpace(input.TargetID) != "" {
		targetID = strings.TrimSpace(input.TargetID)
	}
	if _, err := r.db.ExecContext(ctx, `
INSERT INTO admin_audit_logs (admin_user_id, action, target_type, target_id, metadata_json)
VALUES ($1, $2, $3, $4, $5::jsonb)
`, input.AdminUserID, strings.TrimSpace(input.Action), strings.TrimSpace(input.TargetType), targetID, metadata); err != nil {
		return fmt.Errorf("admin repo: create audit log: %w", err)
	}
	return nil
}

func (r *adminRepository) ListAuditLogs(ctx context.Context, limit int) ([]domain.AdminAuditLog, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, admin_user_id, action, target_type, target_id, metadata_json, created_at
FROM admin_audit_logs
ORDER BY created_at DESC
LIMIT $1
`, normalizeSQLLimit(limit))
	if err != nil {
		return nil, fmt.Errorf("admin repo: list audit logs: %w", err)
	}
	defer rows.Close()
	return scanAuditLogs(rows)
}

func (r *adminRepository) GetOverview(ctx context.Context, now time.Time) (*domain.AdminOverview, error) {
	budget, err := r.getBudget(ctx, adminBudgetDate(now))
	if err != nil {
		return nil, err
	}
	start, end := adminDayWindow(now)
	usage, err := r.getLLMUsageTotals(ctx, start, end)
	if err != nil {
		return nil, err
	}
	jobs, err := r.getJobStatusCounts(ctx, nil)
	if err != nil {
		return nil, err
	}
	audits, err := r.ListAuditLogs(ctx, 10)
	if err != nil {
		return nil, err
	}

	var overview domain.AdminOverview
	overview.Budget = *budget
	overview.LLMUsageToday = usage
	overview.GenerationJobs = jobs
	overview.CloudTasksQueueEstimate = jobs.Queued + jobs.EnqueueFailed
	overview.StripeWebhookErrorCount = r.countSubscriptionFailures(ctx)
	overview.AdMobSSVErrorCount = 0
	overview.ExtensionImportCount = r.countExtensionImports(ctx, start, end)
	overview.RateLimit429Count = r.countRateLimit429Approx(ctx)
	overview.RecentAuditLogs = audits
	return &overview, nil
}

func (r *adminRepository) SearchUsers(ctx context.Context, query string, limit int) ([]domain.AdminUserSummary, error) {
	rows, err := r.db.QueryContext(ctx, adminUserSummarySQL(`
SELECT u.id, u.firebase_uid, u.email, u.username, u.plan, u.created_at, u.updated_at
FROM users u
WHERE (
  $1 = ''
  OR u.id::text = $1
  OR u.firebase_uid = $1
  OR lower(coalesce(u.email, '')) = lower($1)
  OR coalesce(u.stripe_customer_id, '') = $1
  OR coalesce(u.stripe_subscription_id, '') = $1
)
-- Inner ORDER BY selects the newest matching users before LIMIT; the outer
-- ORDER BY in adminUserSummarySQL preserves display order after joins.
ORDER BY u.created_at DESC
LIMIT $2
`), strings.TrimSpace(query), normalizeSQLLimit(limit))
	if err != nil {
		return nil, fmt.Errorf("admin repo: search users: %w", err)
	}
	defer rows.Close()
	return scanAdminUserSummaries(rows)
}

func (r *adminRepository) GetUser(ctx context.Context, userID uuid.UUID) (*domain.AdminUserSummary, error) {
	row := r.db.QueryRowContext(ctx, adminUserSummarySQL(`
SELECT u.id, u.firebase_uid, u.email, u.username, u.plan, u.created_at, u.updated_at
FROM users u
WHERE u.id = $1
LIMIT 1
`), userID)
	user, err := scanAdminUserSummary(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("admin repo: get user: %w", err)
	}
	return user, nil
}

func (r *adminRepository) ListExtensionTokens(ctx context.Context, userID uuid.UUID) ([]domain.AdminExtensionToken, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, name, scopes, created_at, last_used_at, expires_at, revoked_at
FROM extension_tokens
WHERE user_id = $1
ORDER BY created_at DESC
`, userID)
	if err != nil {
		return nil, fmt.Errorf("admin repo: list extension tokens: %w", err)
	}
	defer rows.Close()

	tokens := make([]domain.AdminExtensionToken, 0)
	for rows.Next() {
		token, err := scanAdminExtensionToken(rows)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, *token)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("admin repo: rows extension tokens: %w", err)
	}
	return tokens, nil
}

func (r *adminRepository) RevokeExtensionToken(ctx context.Context, adminID, userID, tokenID uuid.UUID, now time.Time) error {
	return withAdminTx(ctx, r.db, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `
UPDATE extension_tokens
SET revoked_at = $3
WHERE id = $1
  AND user_id = $2
  AND revoked_at IS NULL
`, tokenID, userID, now)
		if err != nil {
			return fmt.Errorf("admin repo: revoke extension token: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("admin repo: revoke extension token rows: %w", err)
		}
		if affected == 0 {
			return domain.ErrNotFound
		}
		return insertAuditLogTx(ctx, tx, domain.AdminAuditLogInput{
			AdminUserID: adminID,
			Action:      "extension_token_revoke",
			TargetType:  "extension_token",
			TargetID:    tokenID.String(),
			Metadata: map[string]any{
				"user_id": userID.String(),
			},
		})
	})
}

func (r *adminRepository) RevokeAllExtensionTokens(ctx context.Context, adminID, userID uuid.UUID, now time.Time) (int, error) {
	var revoked int
	err := withAdminTx(ctx, r.db, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `
UPDATE extension_tokens
SET revoked_at = $2
WHERE user_id = $1
  AND revoked_at IS NULL
`, userID, now)
		if err != nil {
			return fmt.Errorf("admin repo: revoke all extension tokens: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("admin repo: revoke all rows: %w", err)
		}
		revoked = int(affected)
		return insertAuditLogTx(ctx, tx, domain.AdminAuditLogInput{
			AdminUserID: adminID,
			Action:      "extension_token_revoke_all",
			TargetType:  "user",
			TargetID:    userID.String(),
			Metadata: map[string]any{
				"revoked_count": revoked,
			},
		})
	})
	return revoked, err
}

func (r *adminRepository) GetLLMOverview(ctx context.Context, now time.Time) (*domain.AdminLLMOverview, error) {
	date := adminBudgetDate(now)
	start, end := adminDayWindow(now)
	budget, err := r.getBudget(ctx, date)
	if err != nil {
		return nil, err
	}
	usage, err := r.getLLMUsageTotals(ctx, start, end)
	if err != nil {
		return nil, err
	}
	providerModels, err := r.listProviderModelUsage(ctx, start, end)
	if err != nil {
		return nil, err
	}
	reasons, err := r.listFailedJobReasons(ctx, start, end)
	if err != nil {
		return nil, err
	}
	return &domain.AdminLLMOverview{
		Budget:           *budget,
		UsageToday:       usage,
		ProviderModels:   providerModels,
		FailedJobReasons: reasons,
	}, nil
}

func (r *adminRepository) UpdateGlobalLLMBudget(ctx context.Context, adminID uuid.UUID, input domain.UpdateAdminLLMBudgetInput, now time.Time) (*domain.AdminLLMBudget, error) {
	date := adminBudgetDate(now)
	var updated *domain.AdminLLMBudget
	err := withAdminTx(ctx, r.db, func(tx *sql.Tx) error {
		row := tx.QueryRowContext(ctx, `
INSERT INTO global_llm_budgets (budget_date, max_requests, max_estimated_cost_yen)
VALUES ($1, $2, $3)
ON CONFLICT (budget_date) DO UPDATE
SET max_requests = EXCLUDED.max_requests,
    max_estimated_cost_yen = EXCLUDED.max_estimated_cost_yen,
    updated_at = NOW()
WHERE global_llm_budgets.used_requests <= EXCLUDED.max_requests
  AND global_llm_budgets.used_estimated_cost_yen <= EXCLUDED.max_estimated_cost_yen
RETURNING budget_date, max_requests, used_requests, max_estimated_cost_yen,
          used_estimated_cost_yen, updated_at
`, date, input.MaxRequests, input.MaxEstimatedCostYen)
		budget, err := scanAdminLLMBudget(row)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return domain.ErrInvalidInput
			}
			return fmt.Errorf("admin repo: update global budget: %w", err)
		}
		updated = budget
		return insertAuditLogTx(ctx, tx, domain.AdminAuditLogInput{
			AdminUserID: adminID,
			Action:      "global_llm_budget_update",
			TargetType:  "global_llm_budget",
			TargetID:    date.Format("2006-01-02"),
			Metadata: map[string]any{
				"max_requests":           input.MaxRequests,
				"max_estimated_cost_yen": input.MaxEstimatedCostYen,
			},
		})
	})
	return updated, err
}

func (r *adminRepository) ListGenerationJobs(ctx context.Context, status string, limit int) ([]domain.AdminGenerationJob, error) {
	query := `
SELECT id, user_id, book_key, status, reason, retry_count, created_at, updated_at, failed_at, completed_at
FROM question_generation_jobs
WHERE ($1 = '' OR status = $1)
ORDER BY created_at DESC
LIMIT $2`
	rows, err := r.db.QueryContext(ctx, query, strings.TrimSpace(status), normalizeSQLLimit(limit))
	if err != nil {
		return nil, fmt.Errorf("admin repo: list generation jobs: %w", err)
	}
	defer rows.Close()

	jobs := make([]domain.AdminGenerationJob, 0)
	for rows.Next() {
		job, err := scanAdminGenerationJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, *job)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("admin repo: rows generation jobs: %w", err)
	}
	return jobs, nil
}

func (r *adminRepository) RetryGenerationJob(ctx context.Context, adminID, jobID uuid.UUID, now time.Time) (*domain.AdminGenerationJob, error) {
	var job *domain.AdminGenerationJob
	err := withAdminTx(ctx, r.db, func(tx *sql.Tx) error {
		var err error
		job, err = scanAdminGenerationJob(tx.QueryRowContext(ctx, `
UPDATE question_generation_jobs
SET status = 'queued',
    last_error = NULL,
    processing_started_at = NULL,
    completed_at = NULL,
    failed_at = NULL,
    updated_at = NOW()
WHERE id = $1
  AND status IN ('failed', 'enqueue_failed')
RETURNING id, user_id, book_key, status, reason, retry_count, created_at, updated_at, failed_at, completed_at
`, jobID))
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return domain.ErrNotFound
			}
			return fmt.Errorf("admin repo: retry job: %w", err)
		}
		return insertAuditLogTx(ctx, tx, domain.AdminAuditLogInput{
			AdminUserID: adminID,
			Action:      "generation_job_retry",
			TargetType:  "question_generation_job",
			TargetID:    jobID.String(),
		})
	})
	if err != nil {
		return nil, err
	}
	return job, nil
}

func (r *adminRepository) MarkGenerationJobEnqueueFailed(ctx context.Context, adminID, jobID uuid.UUID, now time.Time) error {
	return withAdminTx(ctx, r.db, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `
UPDATE question_generation_jobs
SET status = 'enqueue_failed',
    last_error = 'admin_retry_enqueue_failed',
    processing_started_at = NULL,
    updated_at = $2
WHERE id = $1
  AND status = 'queued'
`, jobID, now)
		if err != nil {
			return fmt.Errorf("admin repo: mark retry enqueue failed: %w", err)
		}
		if err := ensureRowsAffected(result); err != nil {
			return err
		}
		return insertAuditLogTx(ctx, tx, domain.AdminAuditLogInput{
			AdminUserID: adminID,
			Action:      "generation_job_retry_enqueue_failed",
			TargetType:  "question_generation_job",
			TargetID:    jobID.String(),
		})
	})
}

func (r *adminRepository) CancelGenerationJob(ctx context.Context, adminID, jobID uuid.UUID, now time.Time) error {
	return withAdminTx(ctx, r.db, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `
UPDATE question_generation_jobs
SET status = 'failed',
    last_error = 'cancelled_by_admin',
    processing_started_at = NULL,
    failed_at = $2,
    updated_at = NOW()
WHERE id = $1
  AND status IN ('queued', 'enqueue_failed')
`, jobID, now)
		if err != nil {
			return fmt.Errorf("admin repo: cancel job: %w", err)
		}
		if err := ensureRowsAffected(result); err != nil {
			return err
		}
		return insertAuditLogTx(ctx, tx, domain.AdminAuditLogInput{
			AdminUserID: adminID,
			Action:      "generation_job_cancel",
			TargetType:  "question_generation_job",
			TargetID:    jobID.String(),
		})
	})
}

func (r *adminRepository) ListBilling(ctx context.Context, limit int) (*domain.AdminBillingOverview, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT event_id, event_type, processed_at
FROM stripe_events
ORDER BY processed_at DESC
LIMIT $1
`, normalizeSQLLimit(limit))
	if err != nil {
		return nil, fmt.Errorf("admin repo: list billing: %w", err)
	}
	defer rows.Close()

	events := make([]domain.AdminStripeEvent, 0)
	for rows.Next() {
		var event domain.AdminStripeEvent
		if err := rows.Scan(&event.EventID, &event.EventType, &event.ProcessedAt); err != nil {
			return nil, fmt.Errorf("admin repo: scan billing event: %w", err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("admin repo: rows billing: %w", err)
	}
	return &domain.AdminBillingOverview{
		Events:       events,
		FailureCount: r.countSubscriptionFailures(ctx),
	}, nil
}

func (r *adminRepository) ListAdMob(ctx context.Context, limit int) (*domain.AdminAdMobOverview, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT transaction_id, user_id, reward_amount, verified_at
FROM admob_ssv_events
ORDER BY verified_at DESC
LIMIT $1
`, normalizeSQLLimit(limit))
	if err != nil {
		return nil, fmt.Errorf("admin repo: list admob: %w", err)
	}
	defer rows.Close()

	events := make([]domain.AdminAdMobEvent, 0)
	for rows.Next() {
		var event domain.AdminAdMobEvent
		if err := rows.Scan(&event.TransactionID, &event.UserID, &event.RewardAmount, &event.VerifiedAt); err != nil {
			return nil, fmt.Errorf("admin repo: scan admob event: %w", err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("admin repo: rows admob: %w", err)
	}
	return &domain.AdminAdMobOverview{
		Events:             events,
		DuplicateCount:     0,
		StaleFallbackCount: 0,
	}, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func adminUserSummarySQL(targetUsersSQL string) string {
	return `
WITH target_users AS (
` + targetUsersSQL + `
),
today_budget AS (
  SELECT b.user_id, b.free_used, b.token_used, b.ad_views_today
  FROM question_daily_budgets b
  JOIN target_users tu ON tu.id = b.user_id
  WHERE b.budget_date = CURRENT_DATE
),
token_balance AS (
  SELECT t.user_id, COALESCE(SUM(t.token_count), 0)::int AS available_tokens
  FROM user_ad_tokens t
  JOIN target_users tu ON tu.id = t.user_id
  WHERE t.used_at IS NULL
  GROUP BY t.user_id
),
extension_counts AS (
  SELECT t.user_id, COUNT(*)::int AS token_count, MAX(t.last_used_at) AS last_used_at
  FROM extension_tokens t
  JOIN target_users tu ON tu.id = t.user_id
  WHERE t.revoked_at IS NULL
  GROUP BY t.user_id
),
recent_jobs AS (
  SELECT j.user_id, COUNT(*)::int AS job_count
  FROM question_generation_jobs j
  JOIN target_users tu ON tu.id = j.user_id
  WHERE j.created_at >= NOW() - INTERVAL '7 days'
  GROUP BY j.user_id
),
latest_subscription AS (
  SELECT DISTINCT ON (s.user_id) s.user_id, s.status
  FROM subscriptions s
  JOIN target_users tu ON tu.id = s.user_id
  ORDER BY s.user_id, s.updated_at DESC
)
SELECT
  u.id,
  u.firebase_uid,
  u.email,
  u.username,
  u.plan,
  latest_subscription.status,
  u.created_at,
  GREATEST(u.updated_at, COALESCE(extension_counts.last_used_at, u.updated_at)) AS last_active_at,
  COALESCE(today_budget.free_used, 0)::int,
  COALESCE(today_budget.token_used, 0)::int,
  COALESCE(today_budget.ad_views_today, 0)::int,
  COALESCE(token_balance.available_tokens, 0)::int,
  COALESCE(extension_counts.token_count, 0)::int,
  COALESCE(recent_jobs.job_count, 0)::int
FROM target_users u
LEFT JOIN today_budget ON today_budget.user_id = u.id
LEFT JOIN token_balance ON token_balance.user_id = u.id
LEFT JOIN extension_counts ON extension_counts.user_id = u.id
LEFT JOIN recent_jobs ON recent_jobs.user_id = u.id
LEFT JOIN latest_subscription ON latest_subscription.user_id = u.id
ORDER BY u.created_at DESC
`
}

func scanAdminUserSummaries(rows *sql.Rows) ([]domain.AdminUserSummary, error) {
	users := make([]domain.AdminUserSummary, 0)
	for rows.Next() {
		user, err := scanAdminUserSummary(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, *user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("admin repo: rows user summaries: %w", err)
	}
	return users, nil
}

func scanAdminUserSummary(s scanner) (*domain.AdminUserSummary, error) {
	var user domain.AdminUserSummary
	var email sql.NullString
	var subscriptionStatus sql.NullString
	var lastActiveAt sql.NullTime
	err := s.Scan(
		&user.ID,
		&user.FirebaseUID,
		&email,
		&user.Username,
		&user.Plan,
		&subscriptionStatus,
		&user.CreatedAt,
		&lastActiveAt,
		&user.QuestionBudget.FreeUsedToday,
		&user.QuestionBudget.TokenUsedToday,
		&user.QuestionBudget.AdViewsToday,
		&user.QuestionBudget.AvailableTokens,
		&user.ExtensionTokenCount,
		&user.RecentJobsCount,
	)
	if err != nil {
		return nil, err
	}
	user.Email = adminNullableStringPtr(email)
	user.SubscriptionStatus = adminNullableStringPtr(subscriptionStatus)
	user.LastActiveAt = adminNullableTimePtr(lastActiveAt)
	return &user, nil
}

func scanAdminExtensionToken(s scanner) (*domain.AdminExtensionToken, error) {
	var token domain.AdminExtensionToken
	var name sql.NullString
	var scopes pq.StringArray
	var lastUsedAt sql.NullTime
	var expiresAt sql.NullTime
	var revokedAt sql.NullTime
	if err := s.Scan(&token.ID, &name, &scopes, &token.CreatedAt, &lastUsedAt, &expiresAt, &revokedAt); err != nil {
		return nil, fmt.Errorf("admin repo: scan extension token: %w", err)
	}
	token.Name = adminNullableStringPtr(name)
	token.Scopes = append([]string(nil), scopes...)
	token.LastUsedAt = adminNullableTimePtr(lastUsedAt)
	token.ExpiresAt = adminNullableTimePtr(expiresAt)
	token.RevokedAt = adminNullableTimePtr(revokedAt)
	return &token, nil
}

func (r *adminRepository) getBudget(ctx context.Context, date time.Time) (*domain.AdminLLMBudget, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT budget_date, max_requests, used_requests, max_estimated_cost_yen,
       used_estimated_cost_yen, updated_at
FROM global_llm_budgets
WHERE budget_date = $1
`, date)
	budget, err := scanAdminLLMBudget(row)
	if err == nil {
		return budget, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("admin repo: get budget: %w", err)
	}
	return &domain.AdminLLMBudget{BudgetDate: date}, nil
}

func scanAdminLLMBudget(s scanner) (*domain.AdminLLMBudget, error) {
	var budget domain.AdminLLMBudget
	var updatedAt sql.NullTime
	if err := s.Scan(
		&budget.BudgetDate,
		&budget.MaxRequests,
		&budget.UsedRequests,
		&budget.MaxEstimatedCostYen,
		&budget.UsedEstimatedCostYen,
		&updatedAt,
	); err != nil {
		return nil, err
	}
	budget.UpdatedAt = adminNullableTimePtr(updatedAt)
	return &budget, nil
}

func (r *adminRepository) getLLMUsageTotals(ctx context.Context, start, end time.Time) (domain.AdminLLMUsageTotals, error) {
	var totals domain.AdminLLMUsageTotals
	if err := r.db.QueryRowContext(ctx, `
SELECT COUNT(*)::int,
       COALESCE(SUM(input_tokens), 0)::int,
       COALESCE(SUM(output_tokens), 0)::int,
       COALESCE(SUM(estimated_cost_yen), 0)::float8
FROM llm_usage_logs
WHERE created_at >= $1 AND created_at < $2
`, start, end).Scan(&totals.RequestCount, &totals.InputTokens, &totals.OutputTokens, &totals.EstimatedCostYen); err != nil {
		return totals, fmt.Errorf("admin repo: llm usage totals: %w", err)
	}
	return totals, nil
}

func (r *adminRepository) getJobStatusCounts(ctx context.Context, since *time.Time) (domain.AdminJobStatusCounts, error) {
	query := `
SELECT status, COUNT(*)::int
FROM question_generation_jobs
WHERE ($1::timestamptz IS NULL OR created_at >= $1)
GROUP BY status`
	rows, err := r.db.QueryContext(ctx, query, since)
	if err != nil {
		return domain.AdminJobStatusCounts{}, fmt.Errorf("admin repo: job status counts: %w", err)
	}
	defer rows.Close()
	var counts domain.AdminJobStatusCounts
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return counts, fmt.Errorf("admin repo: scan job status count: %w", err)
		}
		switch status {
		case "queued":
			counts.Queued = count
		case "processing":
			counts.Processing = count
		case "failed":
			counts.Failed = count
		case "completed":
			counts.Completed = count
		case "enqueue_failed":
			counts.EnqueueFailed = count
		}
	}
	if err := rows.Err(); err != nil {
		return counts, fmt.Errorf("admin repo: rows job status count: %w", err)
	}
	return counts, nil
}

func (r *adminRepository) listProviderModelUsage(ctx context.Context, start, end time.Time) ([]domain.AdminLLMProviderUsage, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT provider, model, COUNT(*)::int,
       COALESCE(SUM(input_tokens), 0)::int,
       COALESCE(SUM(output_tokens), 0)::int,
       COALESCE(SUM(estimated_cost_yen), 0)::float8
FROM llm_usage_logs
WHERE created_at >= $1 AND created_at < $2
GROUP BY provider, model
ORDER BY COUNT(*) DESC, provider, model
`, start, end)
	if err != nil {
		return nil, fmt.Errorf("admin repo: provider model usage: %w", err)
	}
	defer rows.Close()
	usages := make([]domain.AdminLLMProviderUsage, 0)
	for rows.Next() {
		var usage domain.AdminLLMProviderUsage
		if err := rows.Scan(&usage.Provider, &usage.Model, &usage.RequestCount, &usage.InputTokens, &usage.OutputTokens, &usage.EstimatedCostYen); err != nil {
			return nil, fmt.Errorf("admin repo: scan provider model usage: %w", err)
		}
		usages = append(usages, usage)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("admin repo: rows provider model usage: %w", err)
	}
	return usages, nil
}

func (r *adminRepository) listFailedJobReasons(ctx context.Context, start, end time.Time) ([]domain.AdminFailedJobReason, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT reason, COUNT(*)::int
FROM question_generation_jobs
WHERE status IN ('failed', 'enqueue_failed')
  AND updated_at >= $1 AND updated_at < $2
GROUP BY reason
ORDER BY COUNT(*) DESC, reason
`, start, end)
	if err != nil {
		return nil, fmt.Errorf("admin repo: failed job reasons: %w", err)
	}
	defer rows.Close()
	reasons := make([]domain.AdminFailedJobReason, 0)
	for rows.Next() {
		var reason domain.AdminFailedJobReason
		if err := rows.Scan(&reason.Reason, &reason.Count); err != nil {
			return nil, fmt.Errorf("admin repo: scan failed reason: %w", err)
		}
		reasons = append(reasons, reason)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("admin repo: rows failed reasons: %w", err)
	}
	return reasons, nil
}

func scanAdminGenerationJob(s scanner) (*domain.AdminGenerationJob, error) {
	var job domain.AdminGenerationJob
	var failedAt sql.NullTime
	var completedAt sql.NullTime
	if err := s.Scan(&job.ID, &job.UserID, &job.BookID, &job.Status, &job.Reason, &job.RetryCount, &job.CreatedAt, &job.UpdatedAt, &failedAt, &completedAt); err != nil {
		return nil, fmt.Errorf("admin repo: scan generation job: %w", err)
	}
	job.FailedAt = adminNullableTimePtr(failedAt)
	job.CompletedAt = adminNullableTimePtr(completedAt)
	return &job, nil
}

func scanAuditLogs(rows *sql.Rows) ([]domain.AdminAuditLog, error) {
	logs := make([]domain.AdminAuditLog, 0)
	for rows.Next() {
		var log domain.AdminAuditLog
		var targetID sql.NullString
		var metadata []byte
		if err := rows.Scan(&log.ID, &log.AdminUserID, &log.Action, &log.TargetType, &targetID, &metadata, &log.CreatedAt); err != nil {
			return nil, fmt.Errorf("admin repo: scan audit log: %w", err)
		}
		log.TargetID = adminNullableStringPtr(targetID)
		log.Metadata = map[string]any{}
		if len(metadata) > 0 {
			_ = json.Unmarshal(metadata, &log.Metadata)
		}
		logs = append(logs, log)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("admin repo: rows audit logs: %w", err)
	}
	return logs, nil
}

func insertAuditLogTx(ctx context.Context, tx *sql.Tx, input domain.AdminAuditLogInput) error {
	metadata, err := marshalAuditMetadata(input.Metadata)
	if err != nil {
		return err
	}
	var targetID any
	if strings.TrimSpace(input.TargetID) != "" {
		targetID = strings.TrimSpace(input.TargetID)
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO admin_audit_logs (admin_user_id, action, target_type, target_id, metadata_json)
VALUES ($1, $2, $3, $4, $5::jsonb)
`, input.AdminUserID, strings.TrimSpace(input.Action), strings.TrimSpace(input.TargetType), targetID, metadata); err != nil {
		return fmt.Errorf("admin repo: insert audit log tx: %w", err)
	}
	return nil
}

func marshalAuditMetadata(metadata map[string]any) (string, error) {
	if len(metadata) == 0 {
		return "{}", nil
	}
	// Audit metadata must stay structured and must not contain free-form user
	// input. This key-based redaction is a final guardrail, not a sanitizer for
	// raw prompts, highlights, tokens, cookies, signatures, or payloads.
	safe := make(map[string]any, len(metadata))
	for key, value := range metadata {
		normalizedKey := strings.ToLower(strings.TrimSpace(key))
		if normalizedKey == "" || strings.Contains(normalizedKey, "token") ||
			strings.Contains(normalizedKey, "secret") ||
			strings.Contains(normalizedKey, "cookie") ||
			strings.Contains(normalizedKey, "signature") ||
			strings.Contains(normalizedKey, "prompt") ||
			strings.Contains(normalizedKey, "highlight") ||
			strings.Contains(normalizedKey, "raw") {
			continue
		}
		safe[normalizedKey] = value
	}
	encoded, err := json.Marshal(safe)
	if err != nil {
		return "", fmt.Errorf("admin repo: marshal audit metadata: %w", err)
	}
	return string(encoded), nil
}

func withAdminTx(ctx context.Context, db *sql.DB, fn func(*sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("admin repo: begin tx: %w", err)
	}
	defer tx.Rollback()
	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("admin repo: commit tx: %w", err)
	}
	return nil
}

func ensureRowsAffected(result sql.Result) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("admin repo: rows affected: %w", err)
	}
	if affected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *adminRepository) countSubscriptionFailures(ctx context.Context) int {
	var count int
	if err := r.db.QueryRowContext(ctx, `
SELECT COUNT(*)::int
FROM subscriptions
WHERE status IN ('past_due', 'canceled', 'expired')
`).Scan(&count); err != nil {
		return 0
	}
	return count
}

func (r *adminRepository) countExtensionImports(ctx context.Context, start, end time.Time) int {
	var count int
	if err := r.db.QueryRowContext(ctx, `
SELECT COUNT(*)::int
FROM highlights
WHERE source = 'extension'
  AND created_at >= $1 AND created_at < $2
`, start, end).Scan(&count); err != nil {
		return 0
	}
	return count
}

func (r *adminRepository) countRateLimit429Approx(ctx context.Context) int {
	var count int
	if err := r.db.QueryRowContext(ctx, `
SELECT COALESCE(SUM(GREATEST(count - 1, 0)), 0)::int
FROM rate_limit_counters
WHERE period = CURRENT_DATE
`).Scan(&count); err != nil {
		return 0
	}
	return count
}

func adminBudgetDate(now time.Time) time.Time {
	loc, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		loc = time.FixedZone("JST", 9*60*60)
	}
	local := now.In(loc)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
}

func adminDayWindow(now time.Time) (time.Time, time.Time) {
	start := adminBudgetDate(now)
	return start, start.Add(24 * time.Hour)
}

func normalizeSQLLimit(limit int) int {
	if limit <= 0 {
		return 50
	}
	if limit > 100 {
		return 100
	}
	return limit
}

func adminNullableStringPtr(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

func adminNullableTimePtr(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}

var _ domain.AdminRepository = (*adminRepository)(nil)
