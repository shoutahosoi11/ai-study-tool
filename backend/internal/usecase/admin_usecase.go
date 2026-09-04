package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

const (
	defaultAdminListLimit        = 50
	maxAdminListLimit            = 100
	maxAdminGlobalBudgetRequests = 1000000
	maxAdminGlobalBudgetCostYen  = 100000000
)

type AdminUsecase struct {
	repo           domain.AdminRepository
	sessionManager domain.SessionCookieManager
	taskEnqueuer   domain.QuestionGenerationTaskEnqueuer
	now            func() time.Time
}

func NewAdminUsecase(repo domain.AdminRepository, sessionManager domain.SessionCookieManager, taskEnqueuers ...domain.QuestionGenerationTaskEnqueuer) (*AdminUsecase, error) {
	if repo == nil {
		return nil, errors.New("admin usecase: repository is nil")
	}
	var taskEnqueuer domain.QuestionGenerationTaskEnqueuer
	if len(taskEnqueuers) > 0 {
		taskEnqueuer = taskEnqueuers[0]
	}
	return &AdminUsecase{
		repo:           repo,
		sessionManager: sessionManager,
		taskEnqueuer:   taskEnqueuer,
		now:            time.Now,
	}, nil
}

func (u *AdminUsecase) Overview(ctx context.Context) (*domain.AdminOverview, error) {
	return u.repo.GetOverview(ctx, u.now().UTC())
}

// Read-side audit logs are best-effort here so admin reads stay available if
// audit insertion has a transient failure. Mutations keep operation and audit
// writes together inside repository transactions.
func (u *AdminUsecase) SearchUsers(ctx context.Context, admin domain.AdminIdentity, query string, limit int) (*domain.AdminUserSearchResult, error) {
	users, err := u.repo.SearchUsers(ctx, query, normalizeAdminLimit(limit))
	if err != nil {
		return nil, fmt.Errorf("admin usecase: search users: %w", err)
	}
	u.auditLog(ctx, domain.AdminAuditLogInput{
		AdminUserID: admin.UserID,
		Action:      "user_lookup",
		TargetType:  "user",
		Metadata: map[string]any{
			"query_kind":   classifyUserLookupQuery(query),
			"result_count": len(users),
		},
	})
	return &domain.AdminUserSearchResult{Users: users}, nil
}

func (u *AdminUsecase) GetUser(ctx context.Context, admin domain.AdminIdentity, userID uuid.UUID) (*domain.AdminUserSummary, error) {
	if userID == uuid.Nil {
		return nil, domain.ErrInvalidInput
	}
	user, err := u.repo.GetUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("admin usecase: get user: %w", err)
	}
	u.auditLog(ctx, domain.AdminAuditLogInput{
		AdminUserID: admin.UserID,
		Action:      "user_detail_view",
		TargetType:  "user",
		TargetID:    userID.String(),
	})
	return user, nil
}

func (u *AdminUsecase) ListExtensionTokens(ctx context.Context, admin domain.AdminIdentity, userID uuid.UUID) ([]domain.AdminExtensionToken, error) {
	if userID == uuid.Nil {
		return nil, domain.ErrInvalidInput
	}
	tokens, err := u.repo.ListExtensionTokens(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("admin usecase: list extension tokens: %w", err)
	}
	u.auditLog(ctx, domain.AdminAuditLogInput{
		AdminUserID: admin.UserID,
		Action:      "extension_token_list",
		TargetType:  "user",
		TargetID:    userID.String(),
		Metadata: map[string]any{
			"token_count": len(tokens),
		},
	})
	return tokens, nil
}

func (u *AdminUsecase) RevokeExtensionToken(ctx context.Context, admin domain.AdminIdentity, userID, tokenID uuid.UUID) error {
	if userID == uuid.Nil || tokenID == uuid.Nil {
		return domain.ErrInvalidInput
	}
	return u.repo.RevokeExtensionToken(ctx, admin.UserID, userID, tokenID, u.now().UTC())
}

func (u *AdminUsecase) RevokeAllExtensionTokens(ctx context.Context, admin domain.AdminIdentity, userID uuid.UUID) (int, error) {
	if userID == uuid.Nil {
		return 0, domain.ErrInvalidInput
	}
	return u.repo.RevokeAllExtensionTokens(ctx, admin.UserID, userID, u.now().UTC())
}

func (u *AdminUsecase) LogoutAll(ctx context.Context, admin domain.AdminIdentity, userID uuid.UUID) error {
	if userID == uuid.Nil {
		return domain.ErrInvalidInput
	}
	if u.sessionManager == nil {
		return domain.ErrForbidden
	}
	user, err := u.repo.GetUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("admin usecase: logout all get user: %w", err)
	}
	if user == nil {
		return fmt.Errorf("admin usecase: logout all get user: %w", domain.ErrNotFound)
	}
	if err := u.sessionManager.RevokeRefreshTokens(ctx, user.FirebaseUID); err != nil {
		return fmt.Errorf("admin usecase: revoke refresh tokens: %w", err)
	}
	if err := u.repo.CreateAuditLog(ctx, domain.AdminAuditLogInput{
		AdminUserID: admin.UserID,
		Action:      "user_logout_all",
		TargetType:  "user",
		TargetID:    userID.String(),
	}); err != nil {
		return fmt.Errorf("admin usecase: audit logout all: %w", err)
	}
	return nil
}

func (u *AdminUsecase) LLMOverview(ctx context.Context) (*domain.AdminLLMOverview, error) {
	return u.repo.GetLLMOverview(ctx, u.now().UTC())
}

func (u *AdminUsecase) UpdateGlobalLLMBudget(ctx context.Context, admin domain.AdminIdentity, input domain.UpdateAdminLLMBudgetInput) (*domain.AdminLLMBudget, error) {
	if input.MaxRequests < 0 || input.MaxEstimatedCostYen < 0 ||
		input.MaxRequests > maxAdminGlobalBudgetRequests ||
		input.MaxEstimatedCostYen > maxAdminGlobalBudgetCostYen {
		return nil, domain.ErrInvalidInput
	}
	updated, err := u.repo.UpdateGlobalLLMBudget(ctx, admin.UserID, input, u.now().UTC())
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			return nil, err
		}
		return nil, fmt.Errorf("admin usecase: update global llm budget: %w", err)
	}
	return updated, nil
}

func (u *AdminUsecase) ListGenerationJobs(ctx context.Context, status string, limit int) ([]domain.AdminGenerationJob, error) {
	return u.repo.ListGenerationJobs(ctx, strings.TrimSpace(status), normalizeAdminLimit(limit))
}

func (u *AdminUsecase) RetryGenerationJob(ctx context.Context, admin domain.AdminIdentity, jobID uuid.UUID) error {
	if jobID == uuid.Nil {
		return domain.ErrInvalidInput
	}
	now := u.now().UTC()
	job, err := u.repo.RetryGenerationJob(ctx, admin.UserID, jobID, now)
	if err != nil {
		return fmt.Errorf("admin usecase: retry generation job: %w", err)
	}
	if job == nil {
		return fmt.Errorf("admin usecase: retry generation job: %w", domain.ErrNotFound)
	}
	if u.taskEnqueuer == nil {
		return nil
	}
	if err := u.taskEnqueuer.EnqueueQuestionGeneration(ctx, job.ID, job.UserID, job.RetryCount); err != nil {
		if markErr := u.repo.MarkGenerationJobEnqueueFailed(ctx, admin.UserID, job.ID, u.now().UTC()); markErr != nil {
			slog.Error("admin_mark_enqueue_failed_error", "job_id", job.ID.String(), "error", markErr.Error())
		}
		return fmt.Errorf("admin usecase: enqueue retry generation job: %w", err)
	}
	return nil
}

func (u *AdminUsecase) CancelGenerationJob(ctx context.Context, admin domain.AdminIdentity, jobID uuid.UUID) error {
	if jobID == uuid.Nil {
		return domain.ErrInvalidInput
	}
	return u.repo.CancelGenerationJob(ctx, admin.UserID, jobID, u.now().UTC())
}

func (u *AdminUsecase) Billing(ctx context.Context, limit int) (*domain.AdminBillingOverview, error) {
	return u.repo.ListBilling(ctx, normalizeAdminLimit(limit))
}

func (u *AdminUsecase) AdMob(ctx context.Context, limit int) (*domain.AdminAdMobOverview, error) {
	return u.repo.ListAdMob(ctx, normalizeAdminLimit(limit))
}

func normalizeAdminLimit(limit int) int {
	if limit <= 0 {
		return defaultAdminListLimit
	}
	if limit > maxAdminListLimit {
		return maxAdminListLimit
	}
	return limit
}

func classifyUserLookupQuery(query string) string {
	query = strings.TrimSpace(query)
	if query == "" {
		return "recent"
	}
	if _, err := uuid.Parse(query); err == nil {
		return "uuid"
	}
	if strings.Contains(query, "@") {
		return "email"
	}
	if strings.HasPrefix(query, "cus_") {
		return "stripe_customer"
	}
	if strings.HasPrefix(query, "sub_") {
		return "stripe_subscription"
	}
	return "firebase_uid"
}

// auditLog records a read-side admin action. Failures must not fail the read,
// but a persistently broken audit insert has to be visible in logs.
func (u *AdminUsecase) auditLog(ctx context.Context, input domain.AdminAuditLogInput) {
	if err := u.repo.CreateAuditLog(ctx, input); err != nil {
		slog.Warn("admin_audit_log_failed", "action", input.Action, "error", err.Error())
	}
}
