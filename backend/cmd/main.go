package main

import (
	"database/sql"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	_ "github.com/lib/pq"
	"github.com/shout/ai-study-tool/backend/internal/di"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Printf("warning: database not reachable: %v", err)
	}

	container, err := di.NewContainer(db)
	if err != nil {
		log.Fatalf("failed to build DI container: %v", err)
	}

	e := echo.New()
	e.Use(echomiddleware.Logger())
	e.Use(echomiddleware.Recover())
	e.Use(echomiddleware.CORSWithConfig(echomiddleware.CORSConfig{
		AllowOrigins: allowedOrigins(),
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Authorization", "Content-Type"},
	}))

	e.GET("/health", func(c echo.Context) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	})

	api := e.Group("/api")
	authMiddleware := container.FirebaseMiddleware.Authenticate

	users := api.Group("/users")
	users.POST("/signup", container.UserHandler.SignUp, authMiddleware)
	users.GET("/me", container.UserHandler.GetMe, authMiddleware)
	users.PUT("/me/question-settings", container.UserHandler.UpdateQuestionSettings, authMiddleware)
	users.GET("/:id", container.UserHandler.GetUser, authMiddleware)
	users.PUT("/me", container.UserHandler.UpdateProfile, authMiddleware)

	posts := api.Group("/posts", authMiddleware)
	posts.GET("/timeline", container.PostHandler.GetTimeline)
	posts.GET("/:id/questions", container.PostHandler.ListQuestions)
	posts.GET("/:id", container.PostHandler.GetPost)
	posts.POST("", container.PostHandler.CreatePost)

	highlights := api.Group("/highlights", authMiddleware)
	highlights.POST("/import", container.HighlightHandler.Import)
	highlights.GET("/books", container.HighlightHandler.ListBooks)
	highlights.GET("/books/search/items", container.HighlightHandler.ListByBookMetadata)
	highlights.GET("/books/:asin/items", container.HighlightHandler.ListByASIN)
	highlights.PUT("/:id/explanation", container.HighlightHandler.UpdateExplanation)

	storage := api.Group("/storage", authMiddleware)
	storage.POST("/signed-urls/upload", container.StorageHandler.CreateUploadSignedURL)
	storage.POST("/signed-urls/download", container.StorageHandler.CreateDownloadSignedURL)

	questions := api.Group("/questions", authMiddleware)
	questions.GET("", container.QuestionHandler.List)
	questions.GET("/saved", container.QuestionHandler.ListSaved)
	questions.GET("/incorrect", container.QuestionHandler.ListIncorrect)
	questions.POST("", container.QuestionHandler.GenerateQuestions)
	questions.POST("/:id/save", container.QuestionHandler.SaveQuestion)
	questions.POST("/:id/answer", container.AnswerHandler.SubmitAnswer)
	questions.POST("/:id/grade", container.QuestionHandler.GradeAnswer)

	social := api.Group("", authMiddleware)
	social.POST("/users/:id/follow", container.SocialHandler.Follow)
	social.DELETE("/users/:id/follow", container.SocialHandler.Unfollow)
	social.POST("/posts/:id/like", container.SocialHandler.Like)
	social.DELETE("/posts/:id/like", container.SocialHandler.Unlike)
	social.POST("/posts/:id/repost", container.SocialHandler.Repost)
	social.DELETE("/posts/:id/repost", container.SocialHandler.Unrepost)
	social.POST("/posts/:id/comments", container.SocialHandler.CreateComment)
	social.GET("/posts/:id/comments", container.SocialHandler.ListComments)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	e.Logger.Fatal(e.Start(":" + port))
}

func allowedOrigins() []string {
	raw := strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS"))
	if raw == "" {
		return []string{"http://localhost:3000", "http://127.0.0.1:3000"}
	}

	parts := strings.Split(raw, ",")
	origins := make([]string, 0, len(parts))
	for _, part := range parts {
		origin := strings.TrimSpace(part)
		if origin != "" {
			origins = append(origins, origin)
		}
	}
	if len(origins) == 0 {
		return []string{"http://localhost:3000", "http://127.0.0.1:3000"}
	}

	return origins
}
