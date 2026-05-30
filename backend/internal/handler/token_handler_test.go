package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/shout/ai-study-tool/backend/internal/domain"
	"github.com/shout/ai-study-tool/backend/internal/usecase"
)

type stubTokenUsecase struct {
	adMobErr error
}

func (s stubTokenUsecase) Award(ctx context.Context, user *domain.User, input usecase.AwardAdTokensInput) (*domain.QuestionTokenBalance, error) {
	return nil, errors.New("not implemented")
}

func (s stubTokenUsecase) AwardAdMobSSV(ctx context.Context, rawQuery string) error {
	return s.adMobErr
}

func (s stubTokenUsecase) Balance(ctx context.Context, user *domain.User) (*domain.QuestionTokenBalance, error) {
	return nil, errors.New("not implemented")
}

func TestAwardAdMobSSVSuccessReturnsNoBody(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/webhooks/admob/ssv?transaction_id=txn-1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	handler := NewTokenHandler(stubTokenUsecase{}, nil)

	if err := handler.AwardAdMobSSV(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
	if body := rec.Body.String(); body != "" {
		t.Fatalf("expected empty response body, got %q", body)
	}
}
