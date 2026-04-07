package di

import (
	"database/sql"
	"os"

	"github.com/shout/ai-study-tool/backend/internal/domain"
	"github.com/shout/ai-study-tool/backend/internal/handler"
	"github.com/shout/ai-study-tool/backend/internal/infrastructure/gemini"
	"github.com/shout/ai-study-tool/backend/internal/infrastructure/persistence"
	infraS3 "github.com/shout/ai-study-tool/backend/internal/infrastructure/s3"
	"github.com/shout/ai-study-tool/backend/internal/usecase"
)

type Container struct {
	QuestionHandler *handler.QuestionHandler
	NoteHandler     *handler.NoteHandler
	AnswerHandler   *handler.AnswerHandler
	SocialHandler   *handler.SocialHandler
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

	s3Client, err := infraS3.NewClient(
		os.Getenv("S3_BUCKET_NAME"),
		os.Getenv("AWS_REGION"),
	)
	if err != nil {
		return nil, err
	}

	questionRepo := persistence.NewQuestionRepository(db)
	answerRepo := persistence.NewAnswerRepository(db)
	socialRepo := persistence.NewSocialRepository(db)

	questionUsecase := usecase.NewQuestionUsecase(questionRepo, geminiClient)
	answerUsecase := usecase.NewAnswerUsecase(answerRepo, questionRepo, geminiClient)
	noteUsecase := usecase.NewNoteUsecase(db, s3Client, ocrClient, questionUsecase)
	socialUsecase := usecase.NewSocialUsecase(socialRepo)

	questionHandler := handler.NewQuestionHandler(questionUsecase, db)
	answerHandler := handler.NewAnswerHandler(answerUsecase, db)
	noteHandler := handler.NewNoteHandler(noteUsecase)
	socialHandler := handler.NewSocialHandler(socialUsecase)

	_ = domain.StorageClient(s3Client)
	_ = domain.OCRClient(ocrClient)

	return &Container{
		QuestionHandler: questionHandler,
		NoteHandler:     noteHandler,
		AnswerHandler:   answerHandler,
		SocialHandler:   socialHandler,
	}, nil
}
