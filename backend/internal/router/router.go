package router

import (
	"log"
	"os"
	"strings"

	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	appconfig "github.com/shout/ai-study-tool/backend/internal/config"
	"github.com/shout/ai-study-tool/backend/internal/di"
	"github.com/shout/ai-study-tool/backend/internal/domain"
	appmiddleware "github.com/shout/ai-study-tool/backend/internal/middleware"
)

const (
	bodyLimitPairingStart     = "1K"   // empty extension pairing start request.
	bodyLimitPairingStatus    = "1K"   // pairing_id-only status polling.
	bodyLimitPairingClaim     = "1K"   // pairing_id-only token claim.
	bodyLimitPairingApprove   = "2K"   // user_code approval from Web.
	bodyLimitTokenRevoke      = "1K"   // extension self-revoke request.
	bodyLimitSession          = "4K"   // Firebase ID token session/refresh exchange.
	bodyLimitLogout           = "1K"   // logout/logout-all with no meaningful body.
	bodyLimitTask             = "4K"   // Cloud Tasks JSON envelope.
	bodyLimitUserSmall        = "4K"   // small user setting updates.
	bodyLimitUserProfile      = "16K"  // signup/profile payloads.
	bodyLimitPostCreate       = "32K"  // post creation content.
	bodyLimitCommentCreate    = "4K"   // comment creation content.
	bodyLimitHighlightCheck   = "256K" // highlight hash batch check.
	bodyLimitHighlightImport  = "2M"   // normal highlight import batch.
	bodyLimitExtensionImport  = "1M"   // extension highlight import batch.
	bodyLimitHighlightShare   = "8K"   // mobile/share extension metadata.
	bodyLimitHighlightPaste   = "5K"   // paste import text.
	bodyLimitHighlightExplain = "8K"   // user-written explanation.
	bodyLimitQuestionSync     = "1K"   // generation sync trigger.
	bodyLimitManualGenerate   = "4K"   // manual generation options.
	bodyLimitQuestionWrite    = "8K"   // save/answer lightweight writes.
	bodyLimitQuestionAnswer   = "4K"   // answer submission.
	bodyLimitTokenAward       = "2K"   // legacy dev/test ad reward path.
	bodyLimitWebhookStripe    = "1M"   // Stripe raw webhook payload.
	bodyLimitAdminSmall       = "4K"   // admin operation payloads.
)

func RegisterAPI(e *echo.Echo, container *di.Container) {
	api := e.Group("/api/v1")
	authMiddleware := container.HybridAuthMiddleware.Authenticate
	ingestRateLimit := container.IngestRateLimitMiddleware.Limit
	generationRateLimit := container.GenerationRateLimitMiddleware.Limit
	postRateLimit := container.PostRateLimitMiddleware.Limit
	socialRateLimit := container.SocialRateLimitMiddleware.Limit
	tokenRateLimit := container.TokenRateLimitMiddleware.Limit

	registerAuthRoutes(api, container)
	registerExtensionRoutes(api, container, authMiddleware, ingestRateLimit)
	registerUserRoutes(api, container, authMiddleware, socialRateLimit)
	registerPostRoutes(api, container, authMiddleware, postRateLimit, socialRateLimit)
	registerHighlightRoutes(api, container, authMiddleware, ingestRateLimit)
	registerQuestionRoutes(api, container, authMiddleware, generationRateLimit)
	registerMonetizationRoutes(api, container, authMiddleware, tokenRateLimit)
	registerAdminRoutes(api, container)
	registerInternalTaskRoutes(e, container)
	registerWebhookRoutes(e, container)
}

func registerWebhookRoutes(e *echo.Echo, container *di.Container) {
	e.POST("/webhooks/stripe", container.StripeHandler.HandleWebhook, echomiddleware.BodyLimit(bodyLimitWebhookStripe))
	e.GET("/webhooks/admob/ssv", container.TokenHandler.AwardAdMobSSV, container.AdMobSSVRateLimitMiddleware.Limit)
}

func registerExtensionRoutes(
	api *echo.Group,
	container *di.Container,
	authMiddleware echo.MiddlewareFunc,
	ingestRateLimit echo.MiddlewareFunc,
) {
	extension := api.Group("/extension")
	extension.POST("/pairing/start", container.ExtensionHandler.StartPairing,
		echomiddleware.BodyLimit(bodyLimitPairingStart),
		container.PairingStartRateLimitMiddleware.Limit,
	)
	extension.POST("/pairing/status", container.ExtensionHandler.PairingStatus,
		echomiddleware.BodyLimit(bodyLimitPairingStatus),
	)
	extension.POST("/pairing/claim", container.ExtensionHandler.ClaimPairing,
		echomiddleware.BodyLimit(bodyLimitPairingClaim),
	)
	extension.POST("/pairing/approve", container.ExtensionHandler.ApprovePairing,
		echomiddleware.BodyLimit(bodyLimitPairingApprove),
		container.SessionAuthMiddleware.Authenticate,
		container.CSRFMiddleware.Protect,
		appmiddleware.RequireClientType(domain.AuthClientTypeWeb),
	)
	extension.DELETE("/tokens/self", container.ExtensionHandler.RevokeSelf,
		echomiddleware.BodyLimit(bodyLimitTokenRevoke),
		authMiddleware,
		appmiddleware.RequireScope(domain.ExtensionScopeRevokeSelf),
	)
	extension.POST("/highlights/import", container.HighlightHandler.ImportExtension,
		echomiddleware.BodyLimit(bodyLimitExtensionImport),
		authMiddleware,
		appmiddleware.RequireScope(domain.ExtensionScopeHighlightWrite),
		ingestRateLimit,
	)
}

func registerAuthRoutes(api *echo.Group, container *di.Container) {
	auth := api.Group("/auth")
	auth.POST("/session", container.AuthHandler.CreateSession, echomiddleware.BodyLimit(bodyLimitSession))
	auth.POST("/refresh", container.AuthHandler.Refresh,
		echomiddleware.BodyLimit(bodyLimitSession),
		container.SessionAuthMiddleware.Authenticate,
		container.CSRFMiddleware.Protect,
	)
	auth.POST("/logout", container.AuthHandler.Logout,
		echomiddleware.BodyLimit(bodyLimitLogout),
		container.SessionAuthMiddleware.Authenticate,
		container.CSRFMiddleware.Protect,
	)
	auth.POST("/logout-all", container.AuthHandler.LogoutAll,
		echomiddleware.BodyLimit(bodyLimitLogout),
		container.SessionAuthMiddleware.Authenticate,
		container.CSRFMiddleware.Protect,
		appmiddleware.RequireRecentAuth,
	)
}

func registerInternalTaskRoutes(e *echo.Echo, container *di.Container) {
	requireInternalTaskOIDCInProduction()
	internal := e.Group("/internal/tasks", appmiddleware.RequireInternalTaskAuthWithSecretFallback(
		os.Getenv("INTERNAL_TASK_SECRET"),
		os.Getenv("TASK_HANDLER_BASE_URL"),
		os.Getenv("INTERNAL_TASK_INVOKER_SERVICE_ACCOUNT"),
		allowInternalTaskSecretFallback(),
	))
	// Cloud Run ingress and queue IAM are network/resource controls; the shared
	// secret is a compatibility fallback. Prefer Cloud Tasks OIDC by setting
	// INTERNAL_TASK_INVOKER_SERVICE_ACCOUNT.
	internal.POST("/question-generation", container.TaskHandler.HandleQuestionGeneration, echomiddleware.BodyLimit(bodyLimitTask))
	internal.POST("/highlight-import", container.TaskHandler.HandleHighlightImport, echomiddleware.BodyLimit(bodyLimitTask))
}

func requireInternalTaskOIDCInProduction() {
	if !appconfig.CurrentAppEnv().IsProduction() {
		return
	}
	if strings.TrimSpace(os.Getenv("TASK_HANDLER_BASE_URL")) == "" ||
		strings.TrimSpace(os.Getenv("INTERNAL_TASK_INVOKER_SERVICE_ACCOUNT")) == "" {
		log.Fatal("TASK_HANDLER_BASE_URL and INTERNAL_TASK_INVOKER_SERVICE_ACCOUNT are required in production")
	}
}

func allowInternalTaskSecretFallback() bool {
	return !appconfig.CurrentAppEnv().IsProduction()
}

func registerUserRoutes(api *echo.Group, container *di.Container, authMiddleware echo.MiddlewareFunc, socialRateLimit echo.MiddlewareFunc) {
	users := api.Group("/users", authMiddleware)
	users.POST("/signup", container.UserHandler.SignUp, echomiddleware.BodyLimit(bodyLimitUserProfile), appmiddleware.RequireScope(domain.ExtensionScopeUserWrite))
	users.GET("/me", container.UserHandler.GetMe, appmiddleware.RequireScope(domain.ExtensionScopeUserRead))
	users.PUT("/me/question-settings", container.UserHandler.UpdateQuestionSettings, echomiddleware.BodyLimit(bodyLimitUserSmall), appmiddleware.RequireScope(domain.ExtensionScopeUserWrite))
	users.GET("/:id", container.UserHandler.GetUser, appmiddleware.RequireScope(domain.ExtensionScopeUserRead))
	users.PUT("/me", container.UserHandler.UpdateProfile, echomiddleware.BodyLimit(bodyLimitUserProfile), appmiddleware.RequireScope(domain.ExtensionScopeUserWrite))
	// 退会はWeb/Mobileのみ許可し、直近5分以内の再認証を要求する。
	users.DELETE("/me", container.UserHandler.DeleteMe,
		echomiddleware.BodyLimit(bodyLimitUserSmall),
		appmiddleware.RequireClientType(domain.AuthClientTypeWeb, domain.AuthClientTypeMobile),
		appmiddleware.RequireRecentAuthFor(domain.AuthClientTypeWeb, domain.AuthClientTypeMobile),
	)
	users.POST("/:id/follow", container.SocialHandler.Follow, appmiddleware.RequireScope(domain.ExtensionScopeSocialWrite), socialRateLimit)
	users.DELETE("/:id/follow", container.SocialHandler.Unfollow, appmiddleware.RequireScope(domain.ExtensionScopeSocialWrite), socialRateLimit)
}

func registerPostRoutes(
	api *echo.Group,
	container *di.Container,
	authMiddleware echo.MiddlewareFunc,
	postRateLimit echo.MiddlewareFunc,
	socialRateLimit echo.MiddlewareFunc,
) {
	posts := api.Group("/posts", authMiddleware)
	posts.GET("/timeline", container.PostHandler.GetTimeline, appmiddleware.RequireScope(domain.ExtensionScopePostRead))
	posts.GET("/:id/questions", container.PostHandler.ListQuestions, appmiddleware.RequireScope(domain.ExtensionScopePostRead))
	posts.GET("/:id", container.PostHandler.GetPost, appmiddleware.RequireScope(domain.ExtensionScopePostRead))
	posts.POST("", container.PostHandler.CreatePost, echomiddleware.BodyLimit(bodyLimitPostCreate), appmiddleware.RequireScope(domain.ExtensionScopePostWrite), postRateLimit)
	posts.POST("/:id/like", container.SocialHandler.Like, appmiddleware.RequireScope(domain.ExtensionScopeSocialWrite), socialRateLimit)
	posts.DELETE("/:id/like", container.SocialHandler.Unlike, appmiddleware.RequireScope(domain.ExtensionScopeSocialWrite), socialRateLimit)
	posts.POST("/:id/repost", container.SocialHandler.Repost, appmiddleware.RequireScope(domain.ExtensionScopeSocialWrite), socialRateLimit)
	posts.DELETE("/:id/repost", container.SocialHandler.Unrepost, appmiddleware.RequireScope(domain.ExtensionScopeSocialWrite), socialRateLimit)
	posts.POST("/:id/comments", container.SocialHandler.CreateComment, echomiddleware.BodyLimit(bodyLimitCommentCreate), appmiddleware.RequireScope(domain.ExtensionScopeSocialWrite), socialRateLimit)
	posts.GET("/:id/comments", container.SocialHandler.ListComments, appmiddleware.RequireScope(domain.ExtensionScopePostRead))
}

func registerHighlightRoutes(
	api *echo.Group,
	container *di.Container,
	authMiddleware echo.MiddlewareFunc,
	ingestRateLimit echo.MiddlewareFunc,
) {
	highlights := api.Group("/highlights", authMiddleware)
	highlights.POST("/sync/check", container.HighlightHandler.CheckExistingHashes, echomiddleware.BodyLimit(bodyLimitHighlightCheck), appmiddleware.RequireScope(domain.ExtensionScopeHighlightCheck))
	highlights.POST("/import", container.HighlightHandler.Import, echomiddleware.BodyLimit(bodyLimitHighlightImport), appmiddleware.RequireScope(domain.ExtensionScopeHighlightWrite), ingestRateLimit)
	highlights.POST("/share", container.HighlightHandler.ImportShared, echomiddleware.BodyLimit(bodyLimitHighlightShare), appmiddleware.RequireScope(domain.ExtensionScopeHighlightWrite), ingestRateLimit)
	highlights.POST("/paste", container.HighlightHandler.ImportPaste, echomiddleware.BodyLimit(bodyLimitHighlightPaste), appmiddleware.RequireScope(domain.ExtensionScopeHighlightWrite), ingestRateLimit)
	highlights.GET("/books", container.HighlightHandler.ListBooks, appmiddleware.RequireScope(domain.ExtensionScopeHighlightCheck))
	highlights.GET("/books/search/items", container.HighlightHandler.ListByBookMetadata, appmiddleware.RequireScope(domain.ExtensionScopeHighlightCheck))
	highlights.GET("/books/:asin/items", container.HighlightHandler.ListByASIN, appmiddleware.RequireScope(domain.ExtensionScopeHighlightCheck))
	highlights.PUT("/:id/explanation", container.HighlightHandler.UpdateExplanation, echomiddleware.BodyLimit(bodyLimitHighlightExplain), appmiddleware.RequireScope(domain.ExtensionScopeHighlightExplain))
}

func registerQuestionRoutes(
	api *echo.Group,
	container *di.Container,
	authMiddleware echo.MiddlewareFunc,
	generationRateLimit echo.MiddlewareFunc,
) {
	questions := api.Group("/questions", authMiddleware)
	questions.GET("", container.QuestionHandler.List, appmiddleware.RequireScope(domain.ExtensionScopeQuestionRead))
	questions.GET("/prepared", container.QuestionHandler.ListPrepared, appmiddleware.RequireScope(domain.ExtensionScopeQuestionRead))
	questions.GET("/saved", container.QuestionHandler.ListSaved, appmiddleware.RequireScope(domain.ExtensionScopeQuestionRead))
	questions.GET("/incorrect", container.QuestionHandler.ListIncorrect, appmiddleware.RequireScope(domain.ExtensionScopeQuestionRead))
	questions.POST("/sync", container.QuestionHandler.SyncStock, echomiddleware.BodyLimit(bodyLimitQuestionSync), appmiddleware.RequireScope(domain.ExtensionScopeQuestionGenerate), generationRateLimit)
	questions.POST("/generate/manual", container.QuestionHandler.ManualGenerate, echomiddleware.BodyLimit(bodyLimitManualGenerate), appmiddleware.RequireScope(domain.ExtensionScopeQuestionGenerate), generationRateLimit)
	questions.POST("/:id/save", container.QuestionHandler.SaveQuestion, echomiddleware.BodyLimit(bodyLimitQuestionWrite), appmiddleware.RequireScope(domain.ExtensionScopeQuestionWrite))
	questions.POST("/:id/answer", container.AnswerHandler.SubmitAnswer, echomiddleware.BodyLimit(bodyLimitQuestionAnswer), appmiddleware.RequireScope(domain.ExtensionScopeQuestionWrite))
}

func registerMonetizationRoutes(api *echo.Group, container *di.Container, authMiddleware echo.MiddlewareFunc, tokenRateLimit echo.MiddlewareFunc) {
	tokens := api.Group("/tokens", authMiddleware)
	if !appconfig.CurrentAppEnv().IsProduction() {
		tokens.POST("/award", container.TokenHandler.Award, echomiddleware.BodyLimit(bodyLimitTokenAward), appmiddleware.RequireScope(domain.ExtensionScopeTokenWrite), tokenRateLimit)
	}
	tokens.GET("/balance", container.TokenHandler.Balance, appmiddleware.RequireScope(domain.ExtensionScopeTokenRead))

	checkout := api.Group("/checkout", authMiddleware)
	checkout.POST("/session", container.StripeHandler.CreateCheckoutSession, appmiddleware.RequireScope(domain.ExtensionScopeBillingWrite), appmiddleware.RequireRecentAuthFor(domain.AuthClientTypeWeb, domain.AuthClientTypeMobile))
}

func registerAdminRoutes(api *echo.Group, container *di.Container) {
	admin := api.Group("/admin",
		container.SessionAuthMiddleware.Authenticate,
		appmiddleware.RequireClientType(domain.AuthClientTypeWeb),
		container.AdminMiddleware.RequireAdmin,
	)
	admin.GET("/overview", container.AdminHandler.Overview)
	admin.GET("/users", container.AdminHandler.SearchUsers)
	admin.GET("/users/:id", container.AdminHandler.GetUser)
	admin.GET("/users/:id/extension-tokens", container.AdminHandler.ListExtensionTokens)
	admin.GET("/llm", container.AdminHandler.LLM)
	admin.GET("/jobs", container.AdminHandler.Jobs)
	admin.GET("/billing", container.AdminHandler.Billing)
	admin.GET("/admob", container.AdminHandler.AdMob)

	admin.POST("/users/:id/extension-tokens/:token_id/revoke", container.AdminHandler.RevokeExtensionToken,
		echomiddleware.BodyLimit(bodyLimitAdminSmall),
		container.CSRFMiddleware.Protect,
		container.AdminMiddleware.RequireAdminRole(domain.AdminRoleSupport),
	)
	admin.POST("/users/:id/extension-tokens/revoke-all", container.AdminHandler.RevokeAllExtensionTokens,
		echomiddleware.BodyLimit(bodyLimitAdminSmall),
		container.CSRFMiddleware.Protect,
		appmiddleware.RequireRecentAuth,
		container.AdminMiddleware.RequireAdminRole(domain.AdminRoleAdmin),
	)
	admin.POST("/users/:id/logout-all", container.AdminHandler.LogoutAll,
		echomiddleware.BodyLimit(bodyLimitAdminSmall),
		container.CSRFMiddleware.Protect,
		appmiddleware.RequireRecentAuth,
		container.AdminMiddleware.RequireAdminRole(domain.AdminRoleAdmin),
	)
	admin.PUT("/llm/budget", container.AdminHandler.UpdateGlobalBudget,
		echomiddleware.BodyLimit(bodyLimitAdminSmall),
		container.CSRFMiddleware.Protect,
		appmiddleware.RequireRecentAuth,
		container.AdminMiddleware.RequireAdminRole(domain.AdminRoleAdmin),
	)
	admin.POST("/jobs/:id/retry", container.AdminHandler.RetryJob,
		echomiddleware.BodyLimit(bodyLimitAdminSmall),
		container.CSRFMiddleware.Protect,
		container.AdminMiddleware.RequireAdminRole(domain.AdminRoleSupport),
	)
	admin.POST("/jobs/:id/cancel", container.AdminHandler.CancelJob,
		echomiddleware.BodyLimit(bodyLimitAdminSmall),
		container.CSRFMiddleware.Protect,
		container.AdminMiddleware.RequireAdminRole(domain.AdminRoleSupport),
	)
}
