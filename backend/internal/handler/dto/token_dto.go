package dto

import "github.com/shout/ai-study-tool/backend/internal/domain"

type AwardAdTokensRequest struct {
	Provider   string `json:"provider"`
	Nonce      string `json:"nonce"`
	RewardedAt string `json:"rewarded_at"`
	Signature  string `json:"signature"`
}

type TokenBalanceResponse struct {
	AvailableTokens int    `json:"available_tokens"`
	FreeUsedToday   int    `json:"free_used_today"`
	FreeLimit       int    `json:"free_limit"`
	AdViewsToday    int    `json:"ad_views_today"`
	AdViewsLimit    int    `json:"ad_views_limit"`
	Plan            string `json:"plan"`
}

func ToTokenBalanceResponse(balance *domain.QuestionTokenBalance) TokenBalanceResponse {
	if balance == nil {
		return TokenBalanceResponse{
			FreeLimit:    domain.FreeDailyQuestionLimit,
			AdViewsLimit: domain.MaxAdViewsPerDay,
			Plan:         "free",
		}
	}
	return TokenBalanceResponse{
		AvailableTokens: balance.AvailableTokens,
		FreeUsedToday:   balance.FreeUsedToday,
		FreeLimit:       balance.FreeLimit,
		AdViewsToday:    balance.AdViewsToday,
		AdViewsLimit:    balance.AdViewsLimit,
		Plan:            balance.Plan,
	}
}
