package di

import (
	"context"
	"database/sql"
	"os"
	"strconv"
	"strings"

	"github.com/shout/ai-study-tool/backend/internal/handler"
	infrafb "github.com/shout/ai-study-tool/backend/internal/infrastructure/firebase"
	"github.com/shout/ai-study-tool/backend/internal/infrastructure/gcs"
	"github.com/shout/ai-study-tool/backend/internal/infrastructure/gemini"
	"github.com/shout/ai-study-tool/backend/internal/infrastructure/inprocess"
	"github.com/shout/ai-study-tool/backend/internal/infrastructure/persistence"
	"github.com/shout/ai-study-tool/backend/internal/middleware"
	postgresrepo "github.com/shout/ai-study-tool/backend/internal/repository/postgres"
	"github.com/shout/ai-study-tool/backend/internal/usecase"
)

type Container struct {
	UserHandler         *handler.UserHandler
	PostHandler         *handler.PostHandler
	QuestionHandler     *handler.QuestionHandler
	AnswerHandler       *handler.AnswerHandler
	SocialHandler       *handler.SocialHandler
	HighlightHandler    *handler.HighlightHandler
	StorageHandler      *handler.StorageHandler
	QuestionDispatcher  *inprocess.QuestionGenerationDispatcher
	FirebaseMiddleware  *middleware.FirebaseMiddleware
	RateLimitMiddleware *middleware.RateLimitMiddleware
	closeLLMClient      gemini.ClientCloser
}

func NewContainer(db *sql.DB) (*Container, error) {
	ctx := context.Background()
	credPath := os.Getenv("FIREBASE_CREDENTIALS_PATH")

	firebaseApp, err := infrafb.NewApp(ctx, credPath)
	if err != nil {
		return nil, err
	}

	authClient, err := firebaseApp.Auth(ctx)
	if err != nil {
		return nil, err
	}

	firebaseMiddleware, err := middleware.NewFirebaseMiddleware(authClient)
	if err != nil {
		return nil, err
	}

	geminiAPIKey := os.Getenv("GEMINI_API_KEY")

	geminiClient, closeLLMClient, err := gemini.NewConfiguredClient(geminiAPIKey)
	if err != nil {
		return nil, err
	}

	storageSigner, err := gcs.NewSignedURLService(
		ctx,
		os.Getenv("GCS_BUCKET_NAME"),
		os.Getenv("GCS_SIGNING_SERVICE_ACCOUNT"),
	)
	if err != nil {
		return nil, err
	}

	userRepo := postgresrepo.NewUserRepository(db)
	postRepo := postgresrepo.NewPostRepository(db)
	questionRepo := persistence.NewQuestionRepository(db)
	answerRepo := persistence.NewAnswerRepository(db)
	socialRepo := persistence.NewSocialRepository(db)
	highlightRepo := persistence.NewHighlightRepository(db)
	rateLimitRepo := persistence.NewRateLimitRepository(db)
	questionJobRepo := persistence.NewQuestionGenerationJobRepository(db)

	rateLimitMiddleware, err := middleware.NewRateLimitMiddleware(rateLimitRepo, "ingest", readEnvInt64OrDefault("HIGHLIGHT_INGEST_DAILY_LIMIT", 100))
	if err != nil {
		return nil, err
	}

	userUsecase := usecase.NewUserUsecase(userRepo)
	postUsecase := usecase.NewPostUsecase(postRepo)
	questionSourceResolver := usecase.NewQuestionSourceResolver(highlightRepo)
	questionUsecase := usecase.NewQuestionUsecase(questionRepo, geminiClient, questionSourceResolver)
	questionWorkerUsecase := usecase.NewQuestionWorkerUsecaseWithJobRepository(highlightRepo, questionRepo, questionJobRepo, geminiClient)
	questionDispatcher := inprocess.NewQuestionGenerationDispatcher(
		questionWorkerUsecase,
		readEnvIntOrDefault("QUESTION_DISPATCHER_MAX_CONCURRENT", 3),
	)
	questionSyncUsecase := usecase.NewQuestionSyncUsecase(highlightRepo, questionRepo, questionJobRepo, questionDispatcher)
	answerUsecase := usecase.NewAnswerUsecase(answerRepo, questionRepo, geminiClient)
	socialUsecase := usecase.NewSocialUsecase(socialRepo)
	highlightUsecase := usecase.NewHighlightUsecase(highlightRepo)
	storageUsecase := usecase.NewStorageUsecase(storageSigner)

	userHandler := handler.NewUserHandler(userUsecase)
	postHandler := handler.NewPostHandler(postUsecase, userUsecase)
	questionHandler := handler.NewQuestionHandler(questionUsecase, questionSyncUsecase, userUsecase)
	answerHandler := handler.NewAnswerHandler(answerUsecase, userUsecase, questionSyncUsecase)
	socialHandler := handler.NewSocialHandler(socialUsecase, postUsecase, userUsecase)
	highlightHandler := handler.NewHighlightHandler(highlightUsecase, userUsecase)
	storageHandler := handler.NewStorageHandler(storageUsecase, userUsecase)

	return &Container{
		UserHandler:         userHandler,
		PostHandler:         postHandler,
		QuestionHandler:     questionHandler,
		AnswerHandler:       answerHandler,
		SocialHandler:       socialHandler,
		HighlightHandler:    highlightHandler,
		StorageHandler:      storageHandler,
		QuestionDispatcher:  questionDispatcher,
		FirebaseMiddleware:  firebaseMiddleware,
		RateLimitMiddleware: rateLimitMiddleware,
		closeLLMClient:      closeLLMClient,
	}, nil
}

func (c *Container) Close() {
	if c == nil {
		return
	}
	if c.QuestionDispatcher != nil {
		c.QuestionDispatcher.Wait()
	}
	if c.closeLLMClient != nil {
		c.closeLLMClient()
	}
}

func readEnvIntOrDefault(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}

	return parsed
}

func readEnvInt64OrDefault(key string, fallback int64) int64 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed <= 0 {
		return fallback
	}

	return parsed
}
