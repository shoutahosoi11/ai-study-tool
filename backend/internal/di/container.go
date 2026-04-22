package di

import (
	"context"
	"database/sql"
	"os"

	"github.com/shout/ai-study-tool/backend/internal/handler"
	infrafb "github.com/shout/ai-study-tool/backend/internal/infrastructure/firebase"
	"github.com/shout/ai-study-tool/backend/internal/infrastructure/gcs"
	"github.com/shout/ai-study-tool/backend/internal/infrastructure/gemini"
	"github.com/shout/ai-study-tool/backend/internal/infrastructure/persistence"
	"github.com/shout/ai-study-tool/backend/internal/middleware"
	postgresrepo "github.com/shout/ai-study-tool/backend/internal/repository/postgres"
	"github.com/shout/ai-study-tool/backend/internal/usecase"
)

type Container struct {
	UserHandler        *handler.UserHandler
	PostHandler        *handler.PostHandler
	QuestionHandler    *handler.QuestionHandler
	AnswerHandler      *handler.AnswerHandler
	SocialHandler      *handler.SocialHandler
	HighlightHandler   *handler.HighlightHandler
	StorageHandler     *handler.StorageHandler
	FirebaseMiddleware *middleware.FirebaseMiddleware
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

	geminiClient, err := gemini.NewClient(geminiAPIKey)
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

	userUsecase := usecase.NewUserUsecase(userRepo)
	postUsecase := usecase.NewPostUsecase(postRepo)
	questionSourceResolver := usecase.NewQuestionSourceResolver(highlightRepo)
	questionUsecase := usecase.NewQuestionUsecase(questionRepo, geminiClient, questionSourceResolver)
	answerUsecase := usecase.NewAnswerUsecase(answerRepo, questionRepo, geminiClient)
	socialUsecase := usecase.NewSocialUsecase(socialRepo)
	highlightUsecase := usecase.NewHighlightUsecase(highlightRepo)
	storageUsecase := usecase.NewStorageUsecase(storageSigner)

	userHandler := handler.NewUserHandler(userUsecase)
	postHandler := handler.NewPostHandler(postUsecase, userUsecase)
	questionHandler := handler.NewQuestionHandler(questionUsecase, userUsecase)
	answerHandler := handler.NewAnswerHandler(answerUsecase, userUsecase)
	socialHandler := handler.NewSocialHandler(socialUsecase, postUsecase, userUsecase)
	highlightHandler := handler.NewHighlightHandler(highlightUsecase, userUsecase)
	storageHandler := handler.NewStorageHandler(storageUsecase, userUsecase)

	return &Container{
		UserHandler:        userHandler,
		PostHandler:        postHandler,
		QuestionHandler:    questionHandler,
		AnswerHandler:      answerHandler,
		SocialHandler:      socialHandler,
		HighlightHandler:   highlightHandler,
		StorageHandler:     storageHandler,
		FirebaseMiddleware: firebaseMiddleware,
	}, nil
}
