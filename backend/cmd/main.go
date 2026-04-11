package main

import (
	"database/sql"
	"log"
	"os"

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
		AllowOrigins: []string{"http://localhost:3000"},
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
	users.GET("/:id", container.UserHandler.GetUser, authMiddleware)
	users.PUT("/me", container.UserHandler.UpdateProfile, authMiddleware)

	posts := api.Group("/posts", authMiddleware)
	posts.GET("/timeline", container.PostHandler.GetTimeline)
	posts.GET("/:id", container.PostHandler.GetPost)
	posts.POST("", container.PostHandler.CreatePost)

	highlights := api.Group("/highlights", authMiddleware)
	highlights.POST("", container.HighlightHandler.Create)
	highlights.GET("", container.HighlightHandler.List)
	highlights.GET("/:id", container.HighlightHandler.GetByID)
	highlights.DELETE("/:id", container.HighlightHandler.Delete)

	questions := api.Group("/questions", authMiddleware)
	questions.POST("", container.QuestionHandler.GenerateQuestions)
	questions.POST("/:id/grade", container.QuestionHandler.GradeAnswer)

	answers := api.Group("/answers", authMiddleware)
	answers.POST("/:id", container.AnswerHandler.SubmitAnswer)

	notes := api.Group("/notes", authMiddleware)
	notes.POST("", container.NoteHandler.UploadNote)

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
