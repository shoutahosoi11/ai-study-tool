package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

func TestAdminUsecaseRetryGenerationJobReenqueuesTask(t *testing.T) {
	jobID := uuid.New()
	userID := uuid.New()
	adminID := uuid.New()
	repo := &stubAdminRepository{
		retryJob: &domain.AdminGenerationJob{
			ID:     jobID,
			UserID: userID,
		},
	}
	enqueuer := &stubQuestionGenerationTaskEnqueuer{}
	uc, err := NewAdminUsecase(repo, nil, enqueuer)
	if err != nil {
		t.Fatalf("new admin usecase: %v", err)
	}

	if err := uc.RetryGenerationJob(context.Background(), domain.AdminIdentity{UserID: adminID, Role: domain.AdminRoleSupport}, jobID); err != nil {
		t.Fatalf("retry generation job: %v", err)
	}
	if repo.retryAdminID != adminID || repo.retryJobID != jobID {
		t.Fatalf("unexpected retry call admin=%s job=%s", repo.retryAdminID, repo.retryJobID)
	}
	if enqueuer.jobID != jobID || enqueuer.userID != userID {
		t.Fatalf("unexpected enqueue call job=%s user=%s", enqueuer.jobID, enqueuer.userID)
	}
}

func TestAdminUsecaseRetryGenerationJobMarksEnqueueFailed(t *testing.T) {
	jobID := uuid.New()
	userID := uuid.New()
	adminID := uuid.New()
	repo := &stubAdminRepository{
		retryJob: &domain.AdminGenerationJob{
			ID:     jobID,
			UserID: userID,
		},
	}
	enqueuer := &stubQuestionGenerationTaskEnqueuer{err: errors.New("queue unavailable")}
	uc, err := NewAdminUsecase(repo, nil, enqueuer)
	if err != nil {
		t.Fatalf("new admin usecase: %v", err)
	}

	err = uc.RetryGenerationJob(context.Background(), domain.AdminIdentity{UserID: adminID, Role: domain.AdminRoleSupport}, jobID)
	if err == nil {
		t.Fatal("expected enqueue error")
	}
	if repo.markEnqueueFailedAdminID != adminID || repo.markEnqueueFailedJobID != jobID {
		t.Fatalf("expected enqueue failed mark, got admin=%s job=%s", repo.markEnqueueFailedAdminID, repo.markEnqueueFailedJobID)
	}
}

func TestAdminUsecaseLogoutAllHandlesNilUser(t *testing.T) {
	adminID := uuid.New()
	repo := &stubAdminRepository{}
	session := &stubSessionCookieManager{}
	uc, err := NewAdminUsecase(repo, session)
	if err != nil {
		t.Fatalf("new admin usecase: %v", err)
	}

	err = uc.LogoutAll(context.Background(), domain.AdminIdentity{UserID: adminID, Role: domain.AdminRoleAdmin}, uuid.New())
	if err == nil {
		t.Fatal("expected error")
	}
	if session.revokeCalled {
		t.Fatal("session revoke should not be called for nil user")
	}
}

func TestNewAdminUsecaseRequiresRepository(t *testing.T) {
	if _, err := NewAdminUsecase(nil, nil); err == nil {
		t.Fatal("expected constructor to fail without repository")
	}
}

type stubAdminRepository struct {
	retryJob                 *domain.AdminGenerationJob
	retryAdminID             uuid.UUID
	retryJobID               uuid.UUID
	markEnqueueFailedAdminID uuid.UUID
	markEnqueueFailedJobID   uuid.UUID
}

func (s *stubAdminRepository) FindAdminIdentityByFirebaseUID(ctx context.Context, firebaseUID string) (*domain.AdminIdentity, error) {
	return nil, domain.ErrNotFound
}

func (s *stubAdminRepository) CreateAuditLog(ctx context.Context, input domain.AdminAuditLogInput) error {
	return nil
}

func (s *stubAdminRepository) ListAuditLogs(ctx context.Context, limit int) ([]domain.AdminAuditLog, error) {
	return nil, nil
}

func (s *stubAdminRepository) GetOverview(ctx context.Context, now time.Time) (*domain.AdminOverview, error) {
	return nil, nil
}

func (s *stubAdminRepository) SearchUsers(ctx context.Context, query string, limit int) ([]domain.AdminUserSummary, error) {
	return nil, nil
}

func (s *stubAdminRepository) GetUser(ctx context.Context, userID uuid.UUID) (*domain.AdminUserSummary, error) {
	return nil, nil
}

func (s *stubAdminRepository) ListExtensionTokens(ctx context.Context, userID uuid.UUID) ([]domain.AdminExtensionToken, error) {
	return nil, nil
}

func (s *stubAdminRepository) RevokeExtensionToken(ctx context.Context, adminID, userID, tokenID uuid.UUID, now time.Time) error {
	return nil
}

func (s *stubAdminRepository) RevokeAllExtensionTokens(ctx context.Context, adminID, userID uuid.UUID, now time.Time) (int, error) {
	return 0, nil
}

func (s *stubAdminRepository) GetLLMOverview(ctx context.Context, now time.Time) (*domain.AdminLLMOverview, error) {
	return nil, nil
}

func (s *stubAdminRepository) UpdateGlobalLLMBudget(ctx context.Context, adminID uuid.UUID, input domain.UpdateAdminLLMBudgetInput, now time.Time) (*domain.AdminLLMBudget, error) {
	return nil, nil
}

func (s *stubAdminRepository) ListGenerationJobs(ctx context.Context, status string, limit int) ([]domain.AdminGenerationJob, error) {
	return nil, nil
}

func (s *stubAdminRepository) RetryGenerationJob(ctx context.Context, adminID, jobID uuid.UUID, now time.Time) (*domain.AdminGenerationJob, error) {
	s.retryAdminID = adminID
	s.retryJobID = jobID
	return s.retryJob, nil
}

func (s *stubAdminRepository) MarkGenerationJobEnqueueFailed(ctx context.Context, adminID, jobID uuid.UUID, now time.Time) error {
	s.markEnqueueFailedAdminID = adminID
	s.markEnqueueFailedJobID = jobID
	return nil
}

func (s *stubAdminRepository) CancelGenerationJob(ctx context.Context, adminID, jobID uuid.UUID, now time.Time) error {
	return nil
}

func (s *stubAdminRepository) ListBilling(ctx context.Context, limit int) (*domain.AdminBillingOverview, error) {
	return nil, nil
}

func (s *stubAdminRepository) ListAdMob(ctx context.Context, limit int) (*domain.AdminAdMobOverview, error) {
	return nil, nil
}

type stubQuestionGenerationTaskEnqueuer struct {
	jobID  uuid.UUID
	userID uuid.UUID
	err    error
}

func (s *stubQuestionGenerationTaskEnqueuer) EnqueueQuestionGeneration(ctx context.Context, jobID uuid.UUID, userID uuid.UUID) error {
	s.jobID = jobID
	s.userID = userID
	return s.err
}

type stubSessionCookieManager struct {
	revokeCalled bool
}

func (s *stubSessionCookieManager) VerifySessionCookieAndCheckRevoked(ctx context.Context, sessionCookie string) (*domain.AuthToken, error) {
	return nil, nil
}

func (s *stubSessionCookieManager) VerifyIDToken(ctx context.Context, idToken string) (*domain.AuthToken, error) {
	return nil, nil
}

func (s *stubSessionCookieManager) CreateSessionCookie(ctx context.Context, idToken string, expiresIn time.Duration) (string, error) {
	return "", nil
}

func (s *stubSessionCookieManager) RevokeRefreshTokens(ctx context.Context, uid string) error {
	s.revokeCalled = true
	return nil
}
