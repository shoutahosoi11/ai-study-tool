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

	questionRepo := persistence.NewQuestionRepository(db)
	answerRepo := persistence.NewAnswerRepository(db)
	socialRepo := persistence.NewSocialRepository(db)
	highlightRepo := persistence.NewHighlightRepository(db)
	userRepo := postgresrepo.NewUserRepository(db)

	questionUsecase := usecase.NewQuestionUsecase(questionRepo, geminiClient)
	answerUsecase := usecase.NewAnswerUsecase(answerRepo, questionRepo, geminiClient)
	noteUsecase := usecase.NewNoteUsecase(db, storageClient, ocrClient, questionUsecase)
	socialUsecase := usecase.NewSocialUsecase(socialRepo)
	highlightUsecase := usecase.NewHighlightUsecase(highlightRepo)
	userUsecase := usecase.NewUserUsecase(userRepo)

	questionHandler := handler.NewQuestionHandler(questionUsecase, db)
	answerHandler := handler.NewAnswerHandler(answerUsecase, db)
	noteHandler := handler.NewNoteHandler(noteUsecase)
	socialHandler := handler.NewSocialHandler(socialUsecase)
	highlightHandler := handler.NewHighlightHandler(highlightUsecase, userUsecase)

	_ = domain.StorageClient(storageClient)
	_ = domain.OCRClient(ocrClient)

	return &Container{
		QuestionHandler:  questionHandler,
		NoteHandler:      noteHandler,
		AnswerHandler:    answerHandler,
		SocialHandler:    socialHandler,
		HighlightHandler: highlightHandler,
	}, nil
}
