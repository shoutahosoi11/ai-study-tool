package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/shout/ai-study-tool/backend/internal/domain"
	appmiddleware "github.com/shout/ai-study-tool/backend/internal/middleware"
)

func TestAdminGetUserRejectsNilUsecaseUser(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/admin/users/"+uuid.NewString(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(uuid.NewString())
	c.Set(appmiddleware.ContextAdminIdentityKey, domain.AdminIdentity{
		UserID: uuid.New(),
		Role:   domain.AdminRoleAdmin,
	})

	handler := NewAdminHandler(&stubAdminHandlerUsecase{})

	err := handler.GetUser(c)
	assertHTTPStatus(t, err, 500)
}

type stubAdminHandlerUsecase struct{}

func (s *stubAdminHandlerUsecase) Overview(ctx context.Context) (*domain.AdminOverview, error) {
	return nil, nil
}

func (s *stubAdminHandlerUsecase) SearchUsers(ctx context.Context, admin domain.AdminIdentity, query string, limit int) (*domain.AdminUserSearchResult, error) {
	return nil, nil
}

func (s *stubAdminHandlerUsecase) GetUser(ctx context.Context, admin domain.AdminIdentity, userID uuid.UUID) (*domain.AdminUserSummary, error) {
	return nil, nil
}

func (s *stubAdminHandlerUsecase) ListExtensionTokens(ctx context.Context, admin domain.AdminIdentity, userID uuid.UUID) ([]domain.AdminExtensionToken, error) {
	return nil, nil
}

func (s *stubAdminHandlerUsecase) RevokeExtensionToken(ctx context.Context, admin domain.AdminIdentity, userID, tokenID uuid.UUID) error {
	return nil
}

func (s *stubAdminHandlerUsecase) RevokeAllExtensionTokens(ctx context.Context, admin domain.AdminIdentity, userID uuid.UUID) (int, error) {
	return 0, nil
}

func (s *stubAdminHandlerUsecase) LogoutAll(ctx context.Context, admin domain.AdminIdentity, userID uuid.UUID) error {
	return nil
}

func (s *stubAdminHandlerUsecase) LLMOverview(ctx context.Context) (*domain.AdminLLMOverview, error) {
	return nil, nil
}

func (s *stubAdminHandlerUsecase) UpdateGlobalLLMBudget(ctx context.Context, admin domain.AdminIdentity, input domain.UpdateAdminLLMBudgetInput) (*domain.AdminLLMBudget, error) {
	return nil, nil
}

func (s *stubAdminHandlerUsecase) ListGenerationJobs(ctx context.Context, status string, limit int) ([]domain.AdminGenerationJob, error) {
	return nil, nil
}

func (s *stubAdminHandlerUsecase) RetryGenerationJob(ctx context.Context, admin domain.AdminIdentity, jobID uuid.UUID) error {
	return nil
}

func (s *stubAdminHandlerUsecase) CancelGenerationJob(ctx context.Context, admin domain.AdminIdentity, jobID uuid.UUID) error {
	return nil
}

func (s *stubAdminHandlerUsecase) Billing(ctx context.Context, limit int) (*domain.AdminBillingOverview, error) {
	return nil, nil
}

func (s *stubAdminHandlerUsecase) AdMob(ctx context.Context, limit int) (*domain.AdminAdMobOverview, error) {
	return nil, nil
}
