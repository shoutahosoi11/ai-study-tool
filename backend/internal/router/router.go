package router

import (
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/shout/ai-study-tool/backend/internal/di"
)

func RegisterAPI(e *echo.Echo, container *di.Container) {
	api := e.Group("/api/v1")
	authMiddleware := container.FirebaseMiddleware.Authenticate
	ingestRateLimit := container.RateLimitMiddleware.Limit

	registerUserRoutes(api, container, authMiddleware)
	registerPostRoutes(api, container, authMiddleware)
	registerHighlightRoutes(api, container, authMiddleware, ingestRateLimit)
	registerQuestionRoutes(api, container, authMiddleware)
	registerMonetizationRoutes(api, container, authMiddleware)
	registerInternalTaskRoutes(e, container)
	e.POST("/webhooks/stripe", container.StripeHandler.HandleWebhook, echomiddleware.BodyLimit("1M"))
}

func registerInternalTaskRoutes(e *echo.Echo, container *di.Container) {
	internal := e.Group("/internal/tasks")
	// Cloud Run ingress=internal-and-cloud-load-balancing and Cloud Tasks queue
	// IAM protect these endpoints. OIDC verification can be added later without
	// changing the usecase/repository contracts.
	internal.POST("/question-generation", container.TaskHandler.HandleQuestionGeneration)
	internal.POST("/highlight-import", container.TaskHandler.HandleHighlightImport)
}

func registerUserRoutes(api *echo.Group, container *di.Container, authMiddleware echo.MiddlewareFunc) {
	users := api.Group("/users", authMiddleware)
	users.POST("/signup", container.UserHandler.SignUp)
	users.GET("/me", container.UserHandler.GetMe)
	users.PUT("/me/question-settings", container.UserHandler.UpdateQuestionSettings)
	users.GET("/:id", container.UserHandler.GetUser)
	users.PUT("/me", container.UserHandler.UpdateProfile)
	users.POST("/:id/follow", container.SocialHandler.Follow)
	users.DELETE("/:id/follow", container.SocialHandler.Unfollow)
}

func registerPostRoutes(api *echo.Group, container *di.Container, authMiddleware echo.MiddlewareFunc) {
	posts := api.Group("/posts", authMiddleware)
	posts.GET("/timeline", container.PostHandler.GetTimeline)
	posts.GET("/:id/questions", container.PostHandler.ListQuestions)
	posts.GET("/:id", container.PostHandler.GetPost)
	posts.POST("", container.PostHandler.CreatePost)
	posts.POST("/:id/like", container.SocialHandler.Like)
	posts.DELETE("/:id/like", container.SocialHandler.Unlike)
	posts.POST("/:id/repost", container.SocialHandler.Repost)
	posts.DELETE("/:id/repost", container.SocialHandler.Unrepost)
	posts.POST("/:id/comments", container.SocialHandler.CreateComment)
	posts.GET("/:id/comments", container.SocialHandler.ListComments)
}

func registerHighlightRoutes(
	api *echo.Group,
	container *di.Container,
	authMiddleware echo.MiddlewareFunc,
	ingestRateLimit echo.MiddlewareFunc,
) {
	highlights := api.Group("/highlights", authMiddleware)
	highlights.POST("/sync/check", container.HighlightHandler.CheckExistingHashes)
	highlights.POST("/import", container.HighlightHandler.Import, ingestRateLimit)
	highlights.POST("/share", container.HighlightHandler.ImportShared, ingestRateLimit)
	highlights.POST("/paste", container.HighlightHandler.ImportPaste, echomiddleware.BodyLimit("5K"), ingestRateLimit)
	highlights.GET("/books", container.HighlightHandler.ListBooks)
	highlights.GET("/books/search/items", container.HighlightHandler.ListByBookMetadata)
	highlights.GET("/books/:asin/items", container.HighlightHandler.ListByASIN)
	highlights.PUT("/:id/explanation", container.HighlightHandler.UpdateExplanation)
}

func registerQuestionRoutes(api *echo.Group, container *di.Container, authMiddleware echo.MiddlewareFunc) {
	questions := api.Group("/questions", authMiddleware)
	questions.GET("", container.QuestionHandler.List)
	questions.GET("/prepared", container.QuestionHandler.ListPrepared)
	questions.GET("/saved", container.QuestionHandler.ListSaved)
	questions.GET("/incorrect", container.QuestionHandler.ListIncorrect)
	questions.POST("/sync", container.QuestionHandler.SyncStock)
	questions.POST("/generate/manual", container.QuestionHandler.ManualGenerate)
	questions.POST("/:id/save", container.QuestionHandler.SaveQuestion)
	questions.POST("/:id/answer", container.AnswerHandler.SubmitAnswer)
}

func registerMonetizationRoutes(api *echo.Group, container *di.Container, authMiddleware echo.MiddlewareFunc) {
	tokens := api.Group("/tokens", authMiddleware)
	tokens.POST("/award", container.TokenHandler.Award)
	tokens.GET("/balance", container.TokenHandler.Balance)

	checkout := api.Group("/checkout", authMiddleware)
	checkout.POST("/session", container.StripeHandler.CreateCheckoutSession)
}
