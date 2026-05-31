package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/shout/ai-study-tool/backend/internal/domain"
	appmiddleware "github.com/shout/ai-study-tool/backend/internal/middleware"
)

type AdminUsecase interface {
	Overview(ctx context.Context) (*domain.AdminOverview, error)
	SearchUsers(ctx context.Context, admin domain.AdminIdentity, query string, limit int) (*domain.AdminUserSearchResult, error)
	GetUser(ctx context.Context, admin domain.AdminIdentity, userID uuid.UUID) (*domain.AdminUserSummary, error)
	ListExtensionTokens(ctx context.Context, admin domain.AdminIdentity, userID uuid.UUID) ([]domain.AdminExtensionToken, error)
	RevokeExtensionToken(ctx context.Context, admin domain.AdminIdentity, userID, tokenID uuid.UUID) error
	RevokeAllExtensionTokens(ctx context.Context, admin domain.AdminIdentity, userID uuid.UUID) (int, error)
	LogoutAll(ctx context.Context, admin domain.AdminIdentity, userID uuid.UUID) error
	LLMOverview(ctx context.Context) (*domain.AdminLLMOverview, error)
	UpdateGlobalLLMBudget(ctx context.Context, admin domain.AdminIdentity, input domain.UpdateAdminLLMBudgetInput) (*domain.AdminLLMBudget, error)
	ListGenerationJobs(ctx context.Context, status string, limit int) ([]domain.AdminGenerationJob, error)
	RetryGenerationJob(ctx context.Context, admin domain.AdminIdentity, jobID uuid.UUID) error
	CancelGenerationJob(ctx context.Context, admin domain.AdminIdentity, jobID uuid.UUID) error
	Billing(ctx context.Context, limit int) (*domain.AdminBillingOverview, error)
	AdMob(ctx context.Context, limit int) (*domain.AdminAdMobOverview, error)
}

type AdminHandler struct {
	usecase AdminUsecase
}

func NewAdminHandler(usecase AdminUsecase) *AdminHandler {
	return &AdminHandler{usecase: usecase}
}

func (h *AdminHandler) Overview(c echo.Context) error {
	overview, err := h.usecase.Overview(c.Request().Context())
	if err != nil {
		return adminInternalError("overview", err)
	}
	return c.JSON(http.StatusOK, toAdminOverviewResponse(overview))
}

func (h *AdminHandler) SearchUsers(c echo.Context) error {
	admin, err := requireAdminIdentity(c)
	if err != nil {
		return err
	}
	result, err := h.usecase.SearchUsers(c.Request().Context(), admin, c.QueryParam("q"), parseAdminLimit(c))
	if err != nil {
		return adminHTTPError("search_users", err)
	}
	return c.JSON(http.StatusOK, adminUserSearchResponse{Users: toAdminUserResponses(result.Users)})
}

func (h *AdminHandler) GetUser(c echo.Context) error {
	admin, err := requireAdminIdentity(c)
	if err != nil {
		return err
	}
	userID, err := parseUUIDParam(c, "id")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid user id")
	}
	user, err := h.usecase.GetUser(c.Request().Context(), admin, userID)
	if err != nil {
		return adminHTTPError("get_user", err)
	}
	if user == nil {
		return adminInternalError("get_user", errors.New("admin usecase returned nil user"))
	}
	return c.JSON(http.StatusOK, toAdminUserResponse(*user))
}

func (h *AdminHandler) ListExtensionTokens(c echo.Context) error {
	admin, err := requireAdminIdentity(c)
	if err != nil {
		return err
	}
	userID, err := parseUUIDParam(c, "id")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid user id")
	}
	tokens, err := h.usecase.ListExtensionTokens(c.Request().Context(), admin, userID)
	if err != nil {
		return adminHTTPError("list_extension_tokens", err)
	}
	return c.JSON(http.StatusOK, adminExtensionTokenListResponse{Tokens: toAdminExtensionTokenResponses(tokens)})
}

func (h *AdminHandler) RevokeExtensionToken(c echo.Context) error {
	admin, err := requireAdminIdentity(c)
	if err != nil {
		return err
	}
	userID, tokenID, err := parseUserAndTokenID(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid revoke request")
	}
	if err := h.usecase.RevokeExtensionToken(c.Request().Context(), admin, userID, tokenID); err != nil {
		return adminHTTPError("revoke_extension_token", err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *AdminHandler) RevokeAllExtensionTokens(c echo.Context) error {
	admin, err := requireAdminIdentity(c)
	if err != nil {
		return err
	}
	userID, err := parseUUIDParam(c, "id")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid user id")
	}
	revoked, err := h.usecase.RevokeAllExtensionTokens(c.Request().Context(), admin, userID)
	if err != nil {
		return adminHTTPError("revoke_all_extension_tokens", err)
	}
	return c.JSON(http.StatusOK, map[string]int{"revoked_count": revoked})
}

func (h *AdminHandler) LogoutAll(c echo.Context) error {
	admin, err := requireAdminIdentity(c)
	if err != nil {
		return err
	}
	userID, err := parseUUIDParam(c, "id")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid user id")
	}
	if err := h.usecase.LogoutAll(c.Request().Context(), admin, userID); err != nil {
		return adminHTTPError("logout_all", err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *AdminHandler) LLM(c echo.Context) error {
	overview, err := h.usecase.LLMOverview(c.Request().Context())
	if err != nil {
		return adminInternalError("llm", err)
	}
	return c.JSON(http.StatusOK, toAdminLLMResponse(overview))
}

type updateGlobalBudgetRequest struct {
	MaxRequests         int `json:"max_requests"`
	MaxEstimatedCostYen int `json:"max_estimated_cost_yen"`
}

func (h *AdminHandler) UpdateGlobalBudget(c echo.Context) error {
	admin, err := requireAdminIdentity(c)
	if err != nil {
		return err
	}
	req := new(updateGlobalBudgetRequest)
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	budget, err := h.usecase.UpdateGlobalLLMBudget(c.Request().Context(), admin, domain.UpdateAdminLLMBudgetInput{
		MaxRequests:         req.MaxRequests,
		MaxEstimatedCostYen: req.MaxEstimatedCostYen,
	})
	if err != nil {
		return adminHTTPError("update_global_budget", err)
	}
	if budget == nil {
		return adminInternalError("update_global_budget", errors.New("admin usecase returned nil budget"))
	}
	return c.JSON(http.StatusOK, toAdminLLMBudgetResponse(*budget))
}

func (h *AdminHandler) Jobs(c echo.Context) error {
	jobs, err := h.usecase.ListGenerationJobs(c.Request().Context(), c.QueryParam("status"), parseAdminLimit(c))
	if err != nil {
		return adminInternalError("jobs", err)
	}
	return c.JSON(http.StatusOK, adminGenerationJobListResponse{Jobs: toAdminGenerationJobResponses(jobs)})
}

func (h *AdminHandler) RetryJob(c echo.Context) error {
	admin, err := requireAdminIdentity(c)
	if err != nil {
		return err
	}
	jobID, err := parseUUIDParam(c, "id")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid job id")
	}
	if err := h.usecase.RetryGenerationJob(c.Request().Context(), admin, jobID); err != nil {
		return adminHTTPError("retry_job", err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *AdminHandler) CancelJob(c echo.Context) error {
	admin, err := requireAdminIdentity(c)
	if err != nil {
		return err
	}
	jobID, err := parseUUIDParam(c, "id")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid job id")
	}
	if err := h.usecase.CancelGenerationJob(c.Request().Context(), admin, jobID); err != nil {
		return adminHTTPError("cancel_job", err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *AdminHandler) Billing(c echo.Context) error {
	billing, err := h.usecase.Billing(c.Request().Context(), parseAdminLimit(c))
	if err != nil {
		return adminInternalError("billing", err)
	}
	return c.JSON(http.StatusOK, toAdminBillingResponse(billing))
}

func (h *AdminHandler) AdMob(c echo.Context) error {
	admob, err := h.usecase.AdMob(c.Request().Context(), parseAdminLimit(c))
	if err != nil {
		return adminInternalError("admob", err)
	}
	return c.JSON(http.StatusOK, toAdminAdMobResponse(admob))
}

func requireAdminIdentity(c echo.Context) (domain.AdminIdentity, error) {
	admin, ok := appmiddleware.GetAdminIdentity(c)
	if !ok {
		return domain.AdminIdentity{}, echo.NewHTTPError(http.StatusForbidden, "admin access required")
	}
	return admin, nil
}

func adminHTTPError(operation string, err error) error {
	if errors.Is(err, domain.ErrInvalidInput) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid admin request")
	}
	if errors.Is(err, domain.ErrNotFound) {
		return echo.NewHTTPError(http.StatusNotFound, "admin resource not found")
	}
	if errors.Is(err, domain.ErrForbidden) {
		return echo.NewHTTPError(http.StatusForbidden, "admin operation is not allowed")
	}
	return adminInternalError(operation, err)
}

func adminInternalError(operation string, err error) error {
	slog.Error("admin_handler_error", "operation", operation, "error", err)
	return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
}

func parseUUIDParam(c echo.Context, name string) (uuid.UUID, error) {
	id, err := uuid.Parse(strings.TrimSpace(c.Param(name)))
	if err != nil || id == uuid.Nil {
		return uuid.Nil, domain.ErrInvalidInput
	}
	return id, nil
}

func parseUserAndTokenID(c echo.Context) (uuid.UUID, uuid.UUID, error) {
	userID, err := parseUUIDParam(c, "id")
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	tokenID, err := parseUUIDParam(c, "token_id")
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	return userID, tokenID, nil
}

func parseAdminLimit(c echo.Context) int {
	limit, err := strconv.Atoi(strings.TrimSpace(c.QueryParam("limit")))
	if err != nil {
		return 0
	}
	return limit
}

type adminOverviewResponse struct {
	Budget                  adminLLMBudgetResponse       `json:"budget"`
	LLMUsageToday           adminLLMUsageTotalsResponse  `json:"llm_usage_today"`
	GenerationJobs          adminJobStatusCountsResponse `json:"generation_jobs"`
	CloudTasksQueueEstimate int                          `json:"cloud_tasks_queue_estimate"`
	StripeWebhookErrorCount int                          `json:"stripe_webhook_error_count"`
	AdMobSSVErrorCount      int                          `json:"admob_ssv_error_count"`
	ExtensionImportCount    int                          `json:"extension_import_count"`
	RateLimit429Count       int                          `json:"rate_limit_429_count"`
	RecentAuditLogs         []adminAuditLogResponse      `json:"recent_audit_logs"`
}

type adminLLMBudgetResponse struct {
	BudgetDate           string     `json:"budget_date"`
	MaxRequests          int        `json:"max_requests"`
	UsedRequests         int        `json:"used_requests"`
	MaxEstimatedCostYen  int        `json:"max_estimated_cost_yen"`
	UsedEstimatedCostYen int        `json:"used_estimated_cost_yen"`
	UpdatedAt            *time.Time `json:"updated_at,omitempty"`
}

type adminLLMUsageTotalsResponse struct {
	RequestCount     int     `json:"request_count"`
	InputTokens      int     `json:"input_tokens"`
	OutputTokens     int     `json:"output_tokens"`
	EstimatedCostYen float64 `json:"estimated_cost_yen"`
}

type adminJobStatusCountsResponse struct {
	Queued        int `json:"queued"`
	Processing    int `json:"processing"`
	Failed        int `json:"failed"`
	Completed     int `json:"completed"`
	EnqueueFailed int `json:"enqueue_failed"`
}

type adminAuditLogResponse struct {
	ID          string         `json:"id"`
	AdminUserID string         `json:"admin_user_id"`
	Action      string         `json:"action"`
	TargetType  string         `json:"target_type"`
	TargetID    *string        `json:"target_id,omitempty"`
	Metadata    map[string]any `json:"metadata"`
	CreatedAt   time.Time      `json:"created_at"`
}

type adminUserSearchResponse struct {
	Users []adminUserResponse `json:"users"`
}

type adminUserResponse struct {
	ID                  string                      `json:"id"`
	FirebaseUID         string                      `json:"firebase_uid"`
	Email               *string                     `json:"email,omitempty"`
	Username            string                      `json:"username"`
	Plan                string                      `json:"plan"`
	SubscriptionStatus  *string                     `json:"subscription_status,omitempty"`
	CreatedAt           time.Time                   `json:"created_at"`
	LastActiveAt        *time.Time                  `json:"last_active_at,omitempty"`
	QuestionBudget      adminQuestionBudgetResponse `json:"question_budget"`
	ExtensionTokenCount int                         `json:"extension_token_count"`
	RecentJobsCount     int                         `json:"recent_jobs_count"`
}

type adminQuestionBudgetResponse struct {
	FreeUsedToday   int `json:"free_used_today"`
	TokenUsedToday  int `json:"token_used_today"`
	AdViewsToday    int `json:"ad_views_today"`
	AvailableTokens int `json:"available_tokens"`
}

type adminExtensionTokenListResponse struct {
	Tokens []adminExtensionTokenResponse `json:"tokens"`
}

type adminExtensionTokenResponse struct {
	ID         string     `json:"id"`
	Name       *string    `json:"name,omitempty"`
	Scopes     []string   `json:"scopes"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
}

type adminLLMResponse struct {
	Budget           adminLLMBudgetResponse         `json:"budget"`
	UsageToday       adminLLMUsageTotalsResponse    `json:"usage_today"`
	ProviderModels   []adminLLMProviderResponse     `json:"provider_models"`
	FailedJobReasons []adminFailedJobReasonResponse `json:"failed_job_reasons"`
}

type adminLLMProviderResponse struct {
	Provider         string  `json:"provider"`
	Model            string  `json:"model"`
	RequestCount     int     `json:"request_count"`
	InputTokens      int     `json:"input_tokens"`
	OutputTokens     int     `json:"output_tokens"`
	EstimatedCostYen float64 `json:"estimated_cost_yen"`
}

type adminFailedJobReasonResponse struct {
	Reason string `json:"reason"`
	Count  int    `json:"count"`
}

type adminGenerationJobListResponse struct {
	Jobs []adminGenerationJobResponse `json:"jobs"`
}

type adminGenerationJobResponse struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	BookID      string     `json:"book_id"`
	Status      string     `json:"status"`
	Reason      string     `json:"reason"`
	RetryCount  int        `json:"retry_count"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	FailedAt    *time.Time `json:"failed_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

type adminBillingResponse struct {
	Events       []adminStripeEventResponse `json:"events"`
	FailureCount int                        `json:"failure_count"`
}

type adminStripeEventResponse struct {
	EventID     string    `json:"event_id"`
	EventType   string    `json:"event_type"`
	ProcessedAt time.Time `json:"processed_at"`
}

type adminAdMobResponse struct {
	Events             []adminAdMobEventResponse `json:"events"`
	DuplicateCount     int                       `json:"duplicate_count"`
	StaleFallbackCount int                       `json:"stale_fallback_count"`
}

type adminAdMobEventResponse struct {
	TransactionID string    `json:"transaction_id"`
	UserID        string    `json:"user_id"`
	RewardAmount  int       `json:"reward_amount"`
	VerifiedAt    time.Time `json:"verified_at"`
}

func toAdminOverviewResponse(overview *domain.AdminOverview) adminOverviewResponse {
	return adminOverviewResponse{
		Budget:                  toAdminLLMBudgetResponse(overview.Budget),
		LLMUsageToday:           toAdminLLMUsageTotalsResponse(overview.LLMUsageToday),
		GenerationJobs:          toAdminJobStatusCountsResponse(overview.GenerationJobs),
		CloudTasksQueueEstimate: overview.CloudTasksQueueEstimate,
		StripeWebhookErrorCount: overview.StripeWebhookErrorCount,
		AdMobSSVErrorCount:      overview.AdMobSSVErrorCount,
		ExtensionImportCount:    overview.ExtensionImportCount,
		RateLimit429Count:       overview.RateLimit429Count,
		RecentAuditLogs:         toAdminAuditLogResponses(overview.RecentAuditLogs),
	}
}

func toAdminLLMBudgetResponse(budget domain.AdminLLMBudget) adminLLMBudgetResponse {
	return adminLLMBudgetResponse{
		BudgetDate:           budget.BudgetDate.Format("2006-01-02"),
		MaxRequests:          budget.MaxRequests,
		UsedRequests:         budget.UsedRequests,
		MaxEstimatedCostYen:  budget.MaxEstimatedCostYen,
		UsedEstimatedCostYen: budget.UsedEstimatedCostYen,
		UpdatedAt:            budget.UpdatedAt,
	}
}

func toAdminLLMUsageTotalsResponse(usage domain.AdminLLMUsageTotals) adminLLMUsageTotalsResponse {
	return adminLLMUsageTotalsResponse{
		RequestCount:     usage.RequestCount,
		InputTokens:      usage.InputTokens,
		OutputTokens:     usage.OutputTokens,
		EstimatedCostYen: usage.EstimatedCostYen,
	}
}

func toAdminJobStatusCountsResponse(counts domain.AdminJobStatusCounts) adminJobStatusCountsResponse {
	return adminJobStatusCountsResponse{
		Queued:        counts.Queued,
		Processing:    counts.Processing,
		Failed:        counts.Failed,
		Completed:     counts.Completed,
		EnqueueFailed: counts.EnqueueFailed,
	}
}

func toAdminAuditLogResponses(logs []domain.AdminAuditLog) []adminAuditLogResponse {
	responses := make([]adminAuditLogResponse, 0, len(logs))
	for _, log := range logs {
		responses = append(responses, adminAuditLogResponse{
			ID:          log.ID.String(),
			AdminUserID: log.AdminUserID.String(),
			Action:      log.Action,
			TargetType:  log.TargetType,
			TargetID:    log.TargetID,
			Metadata:    log.Metadata,
			CreatedAt:   log.CreatedAt,
		})
	}
	return responses
}

func toAdminUserResponses(users []domain.AdminUserSummary) []adminUserResponse {
	responses := make([]adminUserResponse, 0, len(users))
	for _, user := range users {
		responses = append(responses, toAdminUserResponse(user))
	}
	return responses
}

func toAdminUserResponse(user domain.AdminUserSummary) adminUserResponse {
	return adminUserResponse{
		ID:                 user.ID.String(),
		FirebaseUID:        user.FirebaseUID,
		Email:              user.Email,
		Username:           user.Username,
		Plan:               user.Plan,
		SubscriptionStatus: user.SubscriptionStatus,
		CreatedAt:          user.CreatedAt,
		LastActiveAt:       user.LastActiveAt,
		QuestionBudget: adminQuestionBudgetResponse{
			FreeUsedToday:   user.QuestionBudget.FreeUsedToday,
			TokenUsedToday:  user.QuestionBudget.TokenUsedToday,
			AdViewsToday:    user.QuestionBudget.AdViewsToday,
			AvailableTokens: user.QuestionBudget.AvailableTokens,
		},
		ExtensionTokenCount: user.ExtensionTokenCount,
		RecentJobsCount:     user.RecentJobsCount,
	}
}

func toAdminExtensionTokenResponses(tokens []domain.AdminExtensionToken) []adminExtensionTokenResponse {
	responses := make([]adminExtensionTokenResponse, 0, len(tokens))
	for _, token := range tokens {
		responses = append(responses, adminExtensionTokenResponse{
			ID:         token.ID.String(),
			Name:       token.Name,
			Scopes:     token.Scopes,
			CreatedAt:  token.CreatedAt,
			LastUsedAt: token.LastUsedAt,
			ExpiresAt:  token.ExpiresAt,
			RevokedAt:  token.RevokedAt,
		})
	}
	return responses
}

func toAdminLLMResponse(overview *domain.AdminLLMOverview) adminLLMResponse {
	providerModels := make([]adminLLMProviderResponse, 0, len(overview.ProviderModels))
	for _, usage := range overview.ProviderModels {
		providerModels = append(providerModels, adminLLMProviderResponse{
			Provider:         usage.Provider,
			Model:            usage.Model,
			RequestCount:     usage.RequestCount,
			InputTokens:      usage.InputTokens,
			OutputTokens:     usage.OutputTokens,
			EstimatedCostYen: usage.EstimatedCostYen,
		})
	}
	failedReasons := make([]adminFailedJobReasonResponse, 0, len(overview.FailedJobReasons))
	for _, reason := range overview.FailedJobReasons {
		failedReasons = append(failedReasons, adminFailedJobReasonResponse{Reason: reason.Reason, Count: reason.Count})
	}
	return adminLLMResponse{
		Budget:           toAdminLLMBudgetResponse(overview.Budget),
		UsageToday:       toAdminLLMUsageTotalsResponse(overview.UsageToday),
		ProviderModels:   providerModels,
		FailedJobReasons: failedReasons,
	}
}

func toAdminGenerationJobResponses(jobs []domain.AdminGenerationJob) []adminGenerationJobResponse {
	responses := make([]adminGenerationJobResponse, 0, len(jobs))
	for _, job := range jobs {
		responses = append(responses, adminGenerationJobResponse{
			ID:          job.ID.String(),
			UserID:      job.UserID.String(),
			BookID:      job.BookID,
			Status:      job.Status,
			Reason:      job.Reason,
			RetryCount:  job.RetryCount,
			CreatedAt:   job.CreatedAt,
			UpdatedAt:   job.UpdatedAt,
			FailedAt:    job.FailedAt,
			CompletedAt: job.CompletedAt,
		})
	}
	return responses
}

func toAdminBillingResponse(billing *domain.AdminBillingOverview) adminBillingResponse {
	events := make([]adminStripeEventResponse, 0, len(billing.Events))
	for _, event := range billing.Events {
		events = append(events, adminStripeEventResponse{
			EventID:     event.EventID,
			EventType:   event.EventType,
			ProcessedAt: event.ProcessedAt,
		})
	}
	return adminBillingResponse{Events: events, FailureCount: billing.FailureCount}
}

func toAdminAdMobResponse(admob *domain.AdminAdMobOverview) adminAdMobResponse {
	events := make([]adminAdMobEventResponse, 0, len(admob.Events))
	for _, event := range admob.Events {
		events = append(events, adminAdMobEventResponse{
			TransactionID: event.TransactionID,
			UserID:        event.UserID.String(),
			RewardAmount:  event.RewardAmount,
			VerifiedAt:    event.VerifiedAt,
		})
	}
	return adminAdMobResponse{
		Events:             events,
		DuplicateCount:     admob.DuplicateCount,
		StaleFallbackCount: admob.StaleFallbackCount,
	}
}
