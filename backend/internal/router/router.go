package router

import (
	"log"
	"os"
	"strings"

	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/shout/ai-study-tool/backend/internal/di"
	appmiddleware "github.com/shout/ai-study-tool/backend/internal/middleware"
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
	registerUserRoutes(api, container, authMiddleware, socialRateLimit)
	registerPostRoutes(api, container, authMiddleware, postRateLimit, socialRateLimit)
	registerHighlightRoutes(api, container, authMiddleware, ingestRateLimit)
	registerQuestionRoutes(api, container, authMiddleware, generationRateLimit)
	registerMonetizationRoutes(api, container, authMiddleware, tokenRateLimit)
	registerInternalTaskRoutes(e, container)
	e.POST("/webhooks/stripe", container.StripeHandler.HandleWebhook, echomiddleware.BodyLimit("1M"))
}

func registerAuthRoutes(api *echo.Group, container *di.Container) {
	auth := api.Group("/auth")
	auth.POST("/session", container.AuthHandler.CreateSession, echomiddleware.BodyLimit("4K"))
	auth.POST("/refresh", container.AuthHandler.Refresh,
		echomiddleware.BodyLimit("4K"),
		container.CSRFMiddleware.Protect,
		container.SessionAuthMiddleware.Authenticate,
	)
	auth.POST("/logout", container.AuthHandler.Logout,
		echomiddleware.BodyLimit("1K"),
		container.CSRFMiddleware.Protect,
		container.SessionAuthMiddleware.Authenticate,
	)
	auth.POST("/logout-all", container.AuthHandler.LogoutAll,
		echomiddleware.BodyLimit("1K"),
		container.CSRFMiddleware.Protect,
		container.SessionAuthMiddleware.Authenticate,
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
	internal.POST("/question-generation", container.TaskHandler.HandleQuestionGeneration, echomiddleware.BodyLimit("4K"))
	internal.POST("/highlight-import", container.TaskHandler.HandleHighlightImport, echomiddleware.BodyLimit("4K"))
}

func requireInternalTaskOIDCInProduction() {
	if os.Getenv("APP_ENV") != "production" {
		return
	}
	if strings.TrimSpace(os.Getenv("TASK_HANDLER_BASE_URL")) == "" ||
		strings.TrimSpace(os.Getenv("INTERNAL_TASK_INVOKER_SERVICE_ACCOUNT")) == "" {
		log.Fatal("TASK_HANDLER_BASE_URL and INTERNAL_TASK_INVOKER_SERVICE_ACCOUNT are required in production")
	}
}

func allowInternalTaskSecretFallback() bool {
	return os.Getenv("APP_ENV") != "production"
}

func registerUserRoutes(api *echo.Group, container *di.Container, authMiddleware echo.MiddlewareFunc, socialRateLimit echo.MiddlewareFunc) {
	users := api.Group("/users", authMiddleware)
	users.POST("/signup", container.UserHandler.SignUp, echomiddleware.BodyLimit("16K"))
	users.GET("/me", container.UserHandler.GetMe)
	users.PUT("/me/question-settings", container.UserHandler.UpdateQuestionSettings, echomiddleware.BodyLimit("4K"))
	users.GET("/:id", container.UserHandler.GetUser)
	users.PUT("/me", container.UserHandler.UpdateProfile, echomiddleware.BodyLimit("16K"))
	users.POST("/:id/follow", container.SocialHandler.Follow, socialRateLimit)
	users.DELETE("/:id/follow", container.SocialHandler.Unfollow, socialRateLimit)
}

func registerPostRoutes(
	api *echo.Group,
	container *di.Container,
	authMiddleware echo.MiddlewareFunc,
	postRateLimit echo.MiddlewareFunc,
	socialRateLimit echo.MiddlewareFunc,
) {
	posts := api.Group("/posts", authMiddleware)
	posts.GET("/timeline", container.PostHandler.GetTimeline)
	posts.GET("/:id/questions", container.PostHandler.ListQuestions)
	posts.GET("/:id", container.PostHandler.GetPost)
	posts.POST("", container.PostHandler.CreatePost, echomiddleware.BodyLimit("32K"), postRateLimit)
	posts.POST("/:id/like", container.SocialHandler.Like, socialRateLimit)
	posts.DELETE("/:id/like", container.SocialHandler.Unlike, socialRateLimit)
	posts.POST("/:id/repost", container.SocialHandler.Repost, socialRateLimit)
	posts.DELETE("/:id/repost", container.SocialHandler.Unrepost, socialRateLimit)
	posts.POST("/:id/comments", container.SocialHandler.CreateComment, echomiddleware.BodyLimit("4K"), socialRateLimit)
	posts.GET("/:id/comments", container.SocialHandler.ListComments)
}

func registerHighlightRoutes(
	api *echo.Group,
	container *di.Container,
	authMiddleware echo.MiddlewareFunc,
	ingestRateLimit echo.MiddlewareFunc,
) {
	highlights := api.Group("/highlights", authMiddleware)
	highlights.POST("/sync/check", container.HighlightHandler.CheckExistingHashes, echomiddleware.BodyLimit("256K"))
	highlights.POST("/import", container.HighlightHandler.Import, echomiddleware.BodyLimit("2M"), ingestRateLimit)
	highlights.POST("/share", container.HighlightHandler.ImportShared, echomiddleware.BodyLimit("8K"), ingestRateLimit)
	highlights.POST("/paste", container.HighlightHandler.ImportPaste, echomiddleware.BodyLimit("5K"), ingestRateLimit)
	highlights.GET("/books", container.HighlightHandler.ListBooks)
	highlights.GET("/books/search/items", container.HighlightHandler.ListByBookMetadata)
	highlights.GET("/books/:asin/items", container.HighlightHandler.ListByASIN)
	highlights.PUT("/:id/explanation", container.HighlightHandler.UpdateExplanation, echomiddleware.BodyLimit("8K"))
}

func registerQuestionRoutes(
	api *echo.Group,
	container *di.Container,
	authMiddleware echo.MiddlewareFunc,
	generationRateLimit echo.MiddlewareFunc,
) {
	questions := api.Group("/questions", authMiddleware)
	questions.GET("", container.QuestionHandler.List)
	questions.GET("/prepared", container.QuestionHandler.ListPrepared)
	questions.GET("/saved", container.QuestionHandler.ListSaved)
	questions.GET("/incorrect", container.QuestionHandler.ListIncorrect)
	questions.POST("/sync", container.QuestionHandler.SyncStock, echomiddleware.BodyLimit("1K"), generationRateLimit)
	questions.POST("/generate/manual", container.QuestionHandler.ManualGenerate, echomiddleware.BodyLimit("4K"), generationRateLimit)
	questions.POST("/:id/save", container.QuestionHandler.SaveQuestion, echomiddleware.BodyLimit("8K"))
	questions.POST("/:id/answer", container.AnswerHandler.SubmitAnswer, echomiddleware.BodyLimit("4K"))
}

func registerMonetizationRoutes(api *echo.Group, container *di.Container, authMiddleware echo.MiddlewareFunc, tokenRateLimit echo.MiddlewareFunc) {
	tokens := api.Group("/tokens", authMiddleware)
	tokens.POST("/award", container.TokenHandler.Award, echomiddleware.BodyLimit("2K"), tokenRateLimit)
	tokens.GET("/balance", container.TokenHandler.Balance)

	checkout := api.Group("/checkout", authMiddleware)
	checkout.POST("/session", container.StripeHandler.CreateCheckoutSession)
}
