package router

import (
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/shout/ai-study-tool/backend/internal/di"
	appmiddleware "github.com/shout/ai-study-tool/backend/internal/middleware"
)

func RegisterAPI(e *echo.Echo, container *di.Container) {
	api := e.Group("/api")
	authMiddleware := container.FirebaseMiddleware.Authenticate
	ingestRateLimit := container.RateLimitMiddleware.Limit

	registerUserRoutes(api, container, authMiddleware)
	registerPostRoutes(api, container, authMiddleware)
	registerHighlightRoutes(api, container, authMiddleware, ingestRateLimit)
	registerStorageRoutes(api, container, authMiddleware)
	registerQuestionRoutes(api, container, authMiddleware)
	registerSocialRoutes(api, container, authMiddleware)
	registerInternalTaskRoutes(e, container)
}

func registerUserRoutes(api *echo.Group, container *di.Container, authMiddleware echo.MiddlewareFunc) {
	users := api.Group("/users", authMiddleware)
	users.POST("/signup", container.UserHandler.SignUp)
	users.GET("/me", container.UserHandler.GetMe)
	users.PUT("/me/question-settings", container.UserHandler.UpdateQuestionSettings)
	users.GET("/:id", container.UserHandler.GetUser)
	users.PUT("/me", container.UserHandler.UpdateProfile)
}

func registerPostRoutes(api *echo.Group, container *di.Container, authMiddleware echo.MiddlewareFunc) {
	posts := api.Group("/posts", authMiddleware)
	posts.GET("/timeline", container.PostHandler.GetTimeline)
	posts.GET("/:id/questions", container.PostHandler.ListQuestions)
	posts.GET("/:id", container.PostHandler.GetPost)
	posts.POST("", container.PostHandler.CreatePost)
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

func registerStorageRoutes(api *echo.Group, container *di.Container, authMiddleware echo.MiddlewareFunc) {
	storage := api.Group("/storage", authMiddleware)
	// TODO: Revisit whether uploads should proxy through the backend. Signed URLs
	// keep API instances out of the data path, while backend uploads centralize
	// validation and scanning at higher Cloud Run cost.
	storage.POST("/signed-urls/upload", container.StorageHandler.CreateUploadSignedURL)
	storage.POST("/signed-urls/download", container.StorageHandler.CreateDownloadSignedURL)
}

func registerQuestionRoutes(api *echo.Group, container *di.Container, authMiddleware echo.MiddlewareFunc) {
	questions := api.Group("/questions", authMiddleware)
	questions.GET("", container.QuestionHandler.List)
	questions.GET("/prepared", container.QuestionHandler.ListPrepared)
	questions.GET("/saved", container.QuestionHandler.ListSaved)
	questions.GET("/incorrect", container.QuestionHandler.ListIncorrect)
	questions.POST("/sync", container.QuestionHandler.SyncStock)
	questions.POST("", container.QuestionHandler.GenerateQuestions)
	questions.POST("/:id/save", container.QuestionHandler.SaveQuestion)
	questions.POST("/:id/answer", container.AnswerHandler.SubmitAnswer)
	questions.POST("/:id/grade", container.QuestionHandler.GradeAnswer)
}

func registerSocialRoutes(api *echo.Group, container *di.Container, authMiddleware echo.MiddlewareFunc) {
	social := api.Group("", authMiddleware)
	social.POST("/users/:id/follow", container.SocialHandler.Follow)
	social.DELETE("/users/:id/follow", container.SocialHandler.Unfollow)
	social.POST("/posts/:id/like", container.SocialHandler.Like)
	social.DELETE("/posts/:id/like", container.SocialHandler.Unlike)
	social.POST("/posts/:id/repost", container.SocialHandler.Repost)
	social.DELETE("/posts/:id/repost", container.SocialHandler.Unrepost)
	social.POST("/posts/:id/comments", container.SocialHandler.CreateComment)
	social.GET("/posts/:id/comments", container.SocialHandler.ListComments)
}

func registerInternalTaskRoutes(e *echo.Echo, container *di.Container) {
	internal := e.Group("/internal", appmiddleware.InternalOnly())
	// TODO: Keep Cloud Run ingress set to internal-and-cloud-load-balancing before
	// enabling Cloud Tasks traffic. OIDC verification will be added in a later phase.
	internal.POST("/tasks/question-generation", container.TaskHandler.HandleQuestionGeneration)
}
