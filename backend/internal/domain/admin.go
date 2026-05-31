package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type AdminRole string

const (
	AdminRoleViewer  AdminRole = "viewer"
	AdminRoleSupport AdminRole = "support"
	AdminRoleAdmin   AdminRole = "admin"
)

type AdminIdentity struct {
	UserID uuid.UUID
	Role   AdminRole
}

func IsValidAdminRole(role AdminRole) bool {
	switch role {
	case AdminRoleViewer, AdminRoleSupport, AdminRoleAdmin:
		return true
	default:
		return false
	}
}

func AdminRoleAllows(actual AdminRole, required AdminRole) bool {
	return adminRoleRank(actual) >= adminRoleRank(required) && adminRoleRank(required) > 0
}

func adminRoleRank(role AdminRole) int {
	switch role {
	case AdminRoleViewer:
		return 1
	case AdminRoleSupport:
		return 2
	case AdminRoleAdmin:
		return 3
	default:
		return 0
	}
}

type AdminAuditLog struct {
	ID          uuid.UUID
	AdminUserID uuid.UUID
	Action      string
	TargetType  string
	TargetID    *string
	Metadata    map[string]any
	CreatedAt   time.Time
}

type AdminAuditLogInput struct {
	AdminUserID uuid.UUID
	Action      string
	TargetType  string
	TargetID    string
	Metadata    map[string]any
}

type AdminOverview struct {
	Budget                  AdminLLMBudget
	LLMUsageToday           AdminLLMUsageTotals
	GenerationJobs          AdminJobStatusCounts
	CloudTasksQueueEstimate int
	StripeWebhookErrorCount int
	AdMobSSVErrorCount      int
	ExtensionImportCount    int
	RateLimit429Count       int
	RecentAuditLogs         []AdminAuditLog
}

type AdminLLMBudget struct {
	BudgetDate           time.Time
	MaxRequests          int
	UsedRequests         int
	MaxEstimatedCostYen  int
	UsedEstimatedCostYen int
	UpdatedAt            *time.Time
}

type AdminLLMUsageTotals struct {
	RequestCount     int
	InputTokens      int
	OutputTokens     int
	EstimatedCostYen float64
}

type AdminJobStatusCounts struct {
	Queued        int
	Processing    int
	Failed        int
	Completed     int
	EnqueueFailed int
}

type AdminUserSearchResult struct {
	Users []AdminUserSummary
}

type AdminUserSummary struct {
	ID                  uuid.UUID
	FirebaseUID         string
	Email               *string
	Username            string
	Plan                string
	SubscriptionStatus  *string
	CreatedAt           time.Time
	LastActiveAt        *time.Time
	QuestionBudget      AdminQuestionBudget
	ExtensionTokenCount int
	RecentJobsCount     int
}

type AdminQuestionBudget struct {
	FreeUsedToday   int
	TokenUsedToday  int
	AdViewsToday    int
	AvailableTokens int
}

type AdminExtensionToken struct {
	ID         uuid.UUID
	Name       *string
	Scopes     []string
	CreatedAt  time.Time
	LastUsedAt *time.Time
	ExpiresAt  *time.Time
	RevokedAt  *time.Time
}

type AdminLLMOverview struct {
	Budget           AdminLLMBudget
	UsageToday       AdminLLMUsageTotals
	ProviderModels   []AdminLLMProviderUsage
	FailedJobReasons []AdminFailedJobReason
}

type AdminLLMProviderUsage struct {
	Provider         string
	Model            string
	RequestCount     int
	InputTokens      int
	OutputTokens     int
	EstimatedCostYen float64
}

type AdminFailedJobReason struct {
	Reason string
	Count  int
}

type AdminGenerationJob struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	BookID      string
	Status      string
	Reason      string
	RetryCount  int
	CreatedAt   time.Time
	UpdatedAt   time.Time
	FailedAt    *time.Time
	CompletedAt *time.Time
}

type AdminStripeEvent struct {
	EventID     string
	EventType   string
	ProcessedAt time.Time
}

type AdminBillingOverview struct {
	Events       []AdminStripeEvent
	FailureCount int
}

type AdminAdMobEvent struct {
	TransactionID string
	UserID        uuid.UUID
	RewardAmount  int
	VerifiedAt    time.Time
}

type AdminAdMobOverview struct {
	Events             []AdminAdMobEvent
	DuplicateCount     int
	StaleFallbackCount int
}

type UpdateAdminLLMBudgetInput struct {
	MaxRequests         int
	MaxEstimatedCostYen int
}

type AdminRepository interface {
	FindAdminIdentityByFirebaseUID(ctx context.Context, firebaseUID string) (*AdminIdentity, error)
	CreateAuditLog(ctx context.Context, input AdminAuditLogInput) error
	ListAuditLogs(ctx context.Context, limit int) ([]AdminAuditLog, error)
	GetOverview(ctx context.Context, now time.Time) (*AdminOverview, error)
	SearchUsers(ctx context.Context, query string, limit int) ([]AdminUserSummary, error)
	GetUser(ctx context.Context, userID uuid.UUID) (*AdminUserSummary, error)
	ListExtensionTokens(ctx context.Context, userID uuid.UUID) ([]AdminExtensionToken, error)
	RevokeExtensionToken(ctx context.Context, adminID, userID, tokenID uuid.UUID, now time.Time) error
	RevokeAllExtensionTokens(ctx context.Context, adminID, userID uuid.UUID, now time.Time) (int, error)
	GetLLMOverview(ctx context.Context, now time.Time) (*AdminLLMOverview, error)
	UpdateGlobalLLMBudget(ctx context.Context, adminID uuid.UUID, input UpdateAdminLLMBudgetInput, now time.Time) (*AdminLLMBudget, error)
	ListGenerationJobs(ctx context.Context, status string, limit int) ([]AdminGenerationJob, error)
	RetryGenerationJob(ctx context.Context, adminID, jobID uuid.UUID, now time.Time) (*AdminGenerationJob, error)
	MarkGenerationJobEnqueueFailed(ctx context.Context, adminID, jobID uuid.UUID, now time.Time) error
	CancelGenerationJob(ctx context.Context, adminID, jobID uuid.UUID, now time.Time) error
	ListBilling(ctx context.Context, limit int) (*AdminBillingOverview, error)
	ListAdMob(ctx context.Context, limit int) (*AdminAdMobOverview, error)
}
