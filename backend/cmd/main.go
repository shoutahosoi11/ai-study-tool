package main

import (
	"database/sql"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	_ "github.com/lib/pq"
	apphandler "github.com/shout/ai-study-tool/backend/internal/handler"
	"github.com/shout/ai-study-tool/backend/internal/infrastructure/persistence"
	appmiddleware "github.com/shout/ai-study-tool/backend/internal/middleware"
	postgresrepo "github.com/shout/ai-study-tool/backend/internal/repository/postgres"
	"github.com/shout/ai-study-tool/backend/internal/usecase"
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

	credPath := os.Getenv("FIREBASE_CREDENTIALS_PATH")
	firebaseMiddleware, err := appmiddleware.NewFirebaseMiddleware(credPath)
	if err != nil {
		log.Printf("warning: firebase init failed (set FIREBASE_CREDENTIALS_PATH): %v", err)
	}

	userRepo := postgresrepo.NewUserRepository(db)
	postRepo := postgresrepo.NewPostRepository(db)
	highlightRepo := persistence.NewHighlightRepository(db)

	userUsecase := usecase.NewUserUsecase(userRepo)
	postUsecase := usecase.NewPostUsecase(postRepo)
	highlightUsecase := usecase.NewHighlightUsecase(highlightRepo)

	userHandler := apphandler.NewUserHandler(userUsecase)
	postHandler := apphandler.NewPostHandler(postUsecase, userUsecase)
	highlightHandler := apphandler.NewHighlightHandler(highlightUsecase, userUsecase)

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

	var authMiddleware echo.MiddlewareFunc
	if firebaseMiddleware != nil {
		authMiddleware = firebaseMiddleware.Authenticate
	} else {
		authMiddleware = func(next echo.HandlerFunc) echo.HandlerFunc {
			return next
		}
	}

	users := api.Group("/users")
	users.POST("/signup", userHandler.SignUp, authMiddleware)
	users.GET("/me", userHandler.GetMe, authMiddleware)
	users.GET("/:id", userHandler.GetUser, authMiddleware)
	users.PUT("/me", userHandler.UpdateProfile, authMiddleware)

	posts := api.Group("/posts", authMiddleware)
	posts.GET("/timeline", postHandler.GetTimeline)
	posts.GET("/:id", postHandler.GetPost)
	posts.POST("", postHandler.CreatePost)
	posts.POST("/:id/like", postHandler.LikePost)
	posts.DELETE("/:id/like", postHandler.UnlikePost)

	highlights := api.Group("/highlights", authMiddleware)
	highlights.POST("", highlightHandler.Create)
	highlights.GET("", highlightHandler.List)
	highlights.GET("/:id", highlightHandler.GetByID)
	highlights.DELETE("/:id", highlightHandler.Delete)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	e.Logger.Fatal(e.Start(":" + port))
}
