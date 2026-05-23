package di

import (
	"context"
	"database/sql"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/shout/ai-study-tool/backend/internal/handler"
	"github.com/shout/ai-study-tool/backend/internal/infrastructure/cloudtasks"
	infrafb "github.com/shout/ai-study-tool/backend/internal/infrastructure/firebase"
	"github.com/shout/ai-study-tool/backend/internal/infrastructure/gemini"
	"github.com/shout/ai-study-tool/backend/internal/infrastructure/persistence"
	infrastripes "github.com/shout/ai-study-tool/backend/internal/infrastructure/stripe"
	"github.com/shout/ai-study-tool/backend/internal/middleware"
	postgresrepo "github.com/shout/ai-study-tool/backend/internal/repository/postgres"
	"github.com/shout/ai-study-tool/backend/internal/usecase"
)

type Container struct {
	UserHandler                   *handler.UserHandler
	PostHandler                   *handler.PostHandler
	QuestionHandler               *handler.QuestionHandler
	AnswerHandler                 *handler.AnswerHandler
	SocialHandler                 *handler.SocialHandler
	HighlightHandler              *handler.HighlightHandler
	TokenHandler                  *handler.TokenHandler
	StripeHandler                 *handler.StripeHandler
	TaskHandler                   *handler.TaskHandler
	FirebaseMiddleware            *middleware.FirebaseMiddleware
	IngestRateLimitMiddleware     *middleware.RateLimitMiddleware
	GenerationRateLimitMiddleware *middleware.RateLimitMiddleware
	PostRateLimitMiddleware       *middleware.RateLimitMiddleware
	SocialRateLimitMiddleware     *middleware.RateLimitMiddleware
	TokenRateLimitMiddleware      *middleware.RateLimitMiddleware
	closeCloudTasks               []func() error
	closeLLMClient                gemini.ClientCloser
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

	userRepo := postgresrepo.NewUserRepository(db)
	postRepo := postgresrepo.NewPostRepository(db)
	questionRepo := persistence.NewQuestionRepository(db)
	answerRepo := persistence.NewAnswerRepository(db)
	socialRepo := persistence.NewSocialRepository(db)
	highlightRepo := persistence.NewHighlightRepository(db)
	rateLimitRepo := persistence.NewRateLimitRepository(db)
	questionJobRepo := persistence.NewQuestionGenerationJobRepository(db)
	questionBudgetRepo := persistence.NewQuestionBudgetRepository(db)
	billingRepo := persistence.NewBillingRepository(db)

	ingestRateLimitMiddleware, err := middleware.NewRateLimitMiddleware(rateLimitRepo, "ingest", readEnvInt64OrDefault("HIGHLIGHT_INGEST_DAILY_LIMIT", 100))
	if err != nil {
		return nil, err
	}
	generationRateLimitMiddleware, err := middleware.NewRateLimitMiddleware(rateLimitRepo, "generation", readEnvInt64OrDefault("QUESTION_GENERATION_DAILY_LIMIT", 50))
	if err != nil {
		return nil, err
	}
	postRateLimitMiddleware, err := middleware.NewRateLimitMiddleware(rateLimitRepo, "post", readEnvInt64OrDefault("POST_DAILY_LIMIT", 50))
	if err != nil {
		return nil, err
	}
	socialRateLimitMiddleware, err := middleware.NewRateLimitMiddleware(rateLimitRepo, "social", readEnvInt64OrDefault("SOCIAL_ACTION_DAILY_LIMIT", 500))
	if err != nil {
		return nil, err
	}
	tokenRateLimitMiddleware, err := middleware.NewRateLimitMiddleware(rateLimitRepo, "token_award", readEnvInt64OrDefault("TOKEN_AWARD_DAILY_LIMIT", 10))
	if err != nil {
		return nil, err
	}

	userUsecase := usecase.NewUserUsecase(userRepo)
	postUsecase := usecase.NewPostUsecase(postRepo)
	questionSourceResolver := usecase.NewQuestionSourceResolver(highlightRepo)
	questionUsecase := usecase.NewQuestionUsecase(questionRepo, geminiClient, questionSourceResolver)
	questionWorkerUsecase := usecase.NewQuestionWorkerUsecaseWithJobRepository(highlightRepo, questionRepo, questionJobRepo, geminiClient)
	questionTaskEnqueuer, err := cloudtasks.NewQuestionGenerationEnqueuerFromEnv(ctx)
	if err != nil {
		return nil, err
	}
	questionSyncUsecase := usecase.NewQuestionSyncUsecase(highlightRepo, questionRepo, questionJobRepo, questionTaskEnqueuer)
	manualGenerationUsecase := usecase.NewManualGenerationUsecase(questionJobRepo, highlightRepo, questionBudgetRepo, questionTaskEnqueuer)
	answerUsecase := usecase.NewAnswerUsecase(answerRepo, questionRepo)
	socialUsecase := usecase.NewSocialUsecase(socialRepo)
	importQueueRepo := persistence.NewHighlightImportQueueRepository(db)
	highlightJobTrigger, err := cloudtasks.NewHighlightImportEnqueuerFromEnv(ctx)
	if err != nil {
		return nil, err
	}
	highlightUsecase := usecase.NewHighlightUsecaseWithQueue(highlightRepo, importQueueRepo, highlightJobTrigger)
	highlightImportJobUsecase := usecase.NewHighlightImportJobUsecase(importQueueRepo, highlightRepo)
	tokenUsecase := usecase.NewTokenUsecaseWithAdRewardSecret(questionBudgetRepo, os.Getenv("AD_REWARD_HMAC_SECRET"))
	billingUsecase := usecase.NewBillingUsecase(
		infrastripes.NewCheckoutClientFromEnv(),
		infrastripes.NewWebhookValidatorFromEnv(),
		billingRepo,
	)
	userHandler := handler.NewUserHandler(userUsecase)
	postHandler := handler.NewPostHandler(postUsecase, userUsecase)
	questionHandler := handler.NewQuestionHandler(questionUsecase, questionSyncUsecase, userUsecase, manualGenerationUsecase)
	answerHandler := handler.NewAnswerHandler(answerUsecase, userUsecase, questionSyncUsecase)
	socialHandler := handler.NewSocialHandler(socialUsecase, postUsecase, userUsecase)
	highlightHandler := handler.NewHighlightHandler(highlightUsecase, userUsecase)
	tokenHandler := handler.NewTokenHandler(tokenUsecase, userUsecase)
	stripeHandler := handler.NewStripeHandler(billingUsecase, userUsecase)
	taskHandler := handler.NewTaskHandler(questionWorkerUsecase, highlightImportJobUsecase)
	closeCloudTasks := make([]func() error, 0, 2)
	if questionTaskEnqueuer != nil {
		closeCloudTasks = append(closeCloudTasks, questionTaskEnqueuer.Close)
	}
	if highlightJobTrigger != nil {
		closeCloudTasks = append(closeCloudTasks, highlightJobTrigger.Close)
	}
	return &Container{
		UserHandler:                   userHandler,
		PostHandler:                   postHandler,
		QuestionHandler:               questionHandler,
		AnswerHandler:                 answerHandler,
		SocialHandler:                 socialHandler,
		HighlightHandler:              highlightHandler,
		TokenHandler:                  tokenHandler,
		StripeHandler:                 stripeHandler,
		TaskHandler:                   taskHandler,
		FirebaseMiddleware:            firebaseMiddleware,
		IngestRateLimitMiddleware:     ingestRateLimitMiddleware,
		GenerationRateLimitMiddleware: generationRateLimitMiddleware,
		PostRateLimitMiddleware:       postRateLimitMiddleware,
		SocialRateLimitMiddleware:     socialRateLimitMiddleware,
		TokenRateLimitMiddleware:      tokenRateLimitMiddleware,
		closeCloudTasks:               closeCloudTasks,
		closeLLMClient:                closeLLMClient,
	}, nil
}

func (c *Container) Close() {
	if c == nil {
		return
	}
	for _, closeFn := range c.closeCloudTasks {
		if closeFn == nil {
			continue
		}
		if err := closeFn(); err != nil {
			log.Printf("di container: close cloud tasks client: %v", err)
		}
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
