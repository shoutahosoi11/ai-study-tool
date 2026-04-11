package di

import (
	"context"
	"database/sql"
	"os"

	"github.com/shout/ai-study-tool/backend/internal/domain"
	"github.com/shout/ai-study-tool/backend/internal/handler"
	"github.com/shout/ai-study-tool/backend/internal/infrastructure/gemini"
	"github.com/shout/ai-study-tool/backend/internal/infrastructure/persistence"
	infrafb "github.com/shout/ai-study-tool/backend/internal/infrastructure/firebase"
	postgresrepo "github.com/shout/ai-study-tool/backend/internal/repository/postgres"
	"github.com/shout/ai-study-tool/backend/internal/usecase"
)

type Container struct {
	UserHandler      *handler.UserHandler
	PostHandler      *handler.PostHandler
	QuestionHandler  *handler.QuestionHandler
	NoteHandler      *handler.NoteHandler
	AnswerHandler    *handler.AnswerHandler
	SocialHandler    *handler.SocialHandler
	HighlightHandler *handler.HighlightHandler
}

func NewContainer(db *sql.DB) (*Container, error) {
	geminiAPIKey := os.Getenv("GEMINI_API_KEY")

	geminiClient, err := gemini.NewClient(geminiAPIKey)
	if err != nil {
		return nil, err
	}

	ocrClient, err := gemini.NewOCRClient(geminiAPIKey)
	if err != nil {
		return nil, err
	}

	storageClient, err := infrafb.NewStorageClient(
		context.Background(),
		os.Getenv("FIREBASE_CREDENTIALS_PATH"),
		os.Getenv("FIREBASE_STORAGE_BUCKET"),
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
	noteRepo := persistence.NewNoteRepository(db)

	userUsecase := usecase.NewUserUsecase(userRepo)
	postUsecase := usecase.NewPostUsecase(postRepo)
	questionUsecase := usecase.NewQuestionUsecase(questionRepo, geminiClient)
	answerUsecase := usecase.NewAnswerUsecase(answerRepo, questionRepo, geminiClient)
	noteUsecase := usecase.NewNoteUsecase(noteRepo, storageClient, ocrClient, questionUsecase)
	socialUsecase := usecase.NewSocialUsecase(socialRepo)
	highlightUsecase := usecase.NewHighlightUsecase(highlightRepo)

	userHandler := handler.NewUserHandler(userUsecase)
	postHandler := handler.NewPostHandler(postUsecase, userUsecase)
	questionHandler := handler.NewQuestionHandler(questionUsecase, userUsecase)
	answerHandler := handler.NewAnswerHandler(answerUsecase, userUsecase)
	noteHandler := handler.NewNoteHandler(noteUsecase, userUsecase)
	socialHandler := handler.NewSocialHandler(socialUsecase, userUsecase)
	highlightHandler := handler.NewHighlightHandler(highlightUsecase, userUsecase)

	_ = domain.StorageClient(storageClient)
	_ = domain.OCRClient(ocrClient)

	return &Container{
		UserHandler:      userHandler,
		PostHandler:      postHandler,
		QuestionHandler:  questionHandler,
		NoteHandler:      noteHandler,
		AnswerHandler:    answerHandler,
		SocialHandler:    socialHandler,
		HighlightHandler: highlightHandler,
	}, nil
}
