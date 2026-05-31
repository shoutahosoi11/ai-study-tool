package di

import (
	"context"
	"database/sql"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/shout/ai-study-tool/backend/internal/handler"
	infraadmob "github.com/shout/ai-study-tool/backend/internal/infrastructure/admob"
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
	UserHandler                     *handler.UserHandler
	PostHandler                     *handler.PostHandler
	QuestionHandler                 *handler.QuestionHandler
	AnswerHandler                   *handler.AnswerHandler
	SocialHandler                   *handler.SocialHandler
	HighlightHandler                *handler.HighlightHandler
	TokenHandler                    *handler.TokenHandler
	StripeHandler                   *handler.StripeHandler
	AuthHandler                     *handler.AuthHandler
	ExtensionHandler                *handler.ExtensionHandler
	TaskHandler                     *handler.TaskHandler
	AdminHandler                    *handler.AdminHandler
	FirebaseMiddleware              *middleware.FirebaseMiddleware
	SessionAuthMiddleware           *middleware.SessionAuthMiddleware
	CSRFMiddleware                  *middleware.CSRFMiddleware
	AdminMiddleware                 *middleware.AdminMiddleware
	HybridAuthMiddleware            *middleware.HybridAuthMiddleware
	SecurityHeadersMiddleware       *middleware.SecurityHeadersMiddleware
	IngestRateLimitMiddleware       *middleware.RateLimitMiddleware
	GenerationRateLimitMiddleware   *middleware.RateLimitMiddleware
	PostRateLimitMiddleware         *middleware.RateLimitMiddleware
	SocialRateLimitMiddleware       *middleware.RateLimitMiddleware
	TokenRateLimitMiddleware        *middleware.RateLimitMiddleware
	PairingStartRateLimitMiddleware *middleware.ShortWindowRateLimitMiddleware
	AdMobSSVRateLimitMiddleware     *middleware.ShortWindowRateLimitMiddleware
	closeCloudTasks                 []func() error
	closeLLMClient                  gemini.ClientCloser
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
	sessionCookieClient := infrafb.NewSessionCookieClient(authClient)
	appEnv := os.Getenv("APP_ENV")
	sessionAuthMiddleware, err := middleware.NewSessionAuthMiddleware(sessionCookieClient, appEnv)
	if err != nil {
		return nil, err
	}
	extensionTokenRepo := persistence.NewExtensionTokenRepository(db)
	extensionPairingRepo := persistence.NewExtensionPairingRepository(db)
	extensionAuthMiddleware, err := middleware.NewExtensionAuthMiddleware(extensionTokenRepo)
	if err != nil {
		return nil, err
	}
	csrfMiddleware, err := middleware.NewCSRFMiddlewareFromEnv(appEnv)
	if err != nil {
		return nil, err
	}
	appCheckEnforced, err := middleware.AppCheckEnforcementEnabledFromEnv(appEnv)
	if err != nil {
		return nil, err
	}
	var appCheckVerifier middleware.AppCheckVerifier
	if appCheckEnforced {
		appCheckClient, err := firebaseApp.AppCheck(ctx)
		if err != nil {
			return nil, err
		}
		appCheckVerifier = infrafb.NewAppCheckVerifier(appCheckClient)
	}
	appCheckMiddleware, err := middleware.NewAppCheckMiddlewareFromEnv(appEnv, appCheckVerifier)
	if err != nil {
		return nil, err
	}
	appVersionMiddleware := middleware.NewAppVersionMiddlewareFromEnv(appEnv)
	hybridAuthMiddleware, err := middleware.NewHybridAuthMiddleware(
		sessionAuthMiddleware,
		firebaseMiddleware,
		extensionAuthMiddleware,
		csrfMiddleware,
		appEnv,
		appCheckMiddleware.Require,
		appVersionMiddleware.Check,
	)
	if err != nil {
		return nil, err
	}
	securityHeadersMiddleware := middleware.NewSecurityHeadersMiddleware(appEnv, os.Getenv("CSP_REPORT_URI"))

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
	globalLLMBudgetRepo := persistence.NewGlobalLLMBudgetRepository(db)
	billingRepo := persistence.NewBillingRepository(db)
	adminRepo := persistence.NewAdminRepository(db)
	adminMiddleware, err := middleware.NewAdminMiddleware(adminRepo)
	if err != nil {
		return nil, err
	}

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
	pairingStartRateLimitMiddleware, err := middleware.NewShortWindowRateLimitMiddleware(
		rateLimitRepo,
		"extension_pairing_start",
		readEnvInt64OrDefault("EXTENSION_PAIRING_START_PER_MINUTE_LIMIT", 20),
		middleware.ClientIPRateLimitIdentifier,
	)
	if err != nil {
		return nil, err
	}
	adMobSSVRateLimitMiddleware, err := middleware.NewShortWindowRateLimitMiddleware(
		rateLimitRepo,
		"admob_ssv",
		readEnvInt64OrDefault("ADMOB_SSV_PER_MINUTE_LIMIT", 120),
		middleware.ClientIPRateLimitIdentifier,
	)
	if err != nil {
		return nil, err
	}

	userUsecase := usecase.NewUserUsecase(userRepo)
	postUsecase := usecase.NewPostUsecase(postRepo)
	questionSourceResolver := usecase.NewQuestionSourceResolver(highlightRepo)
	questionUsecase := usecase.NewQuestionUsecase(questionRepo, geminiClient, questionSourceResolver)
	globalLLMBudgetUsecase, err := usecase.NewGlobalLLMBudgetUsecaseFromEnv(globalLLMBudgetRepo, appEnv)
	if err != nil {
		return nil, err
	}
	questionWorkerUsecase := usecase.NewQuestionWorkerUsecaseWithJobRepository(highlightRepo, questionRepo, questionJobRepo, geminiClient).
		WithGlobalLLMBudget(globalLLMBudgetUsecase)
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
	tokenUsecase := usecase.NewTokenUsecaseWithAdRewardSecretAndEnv(questionBudgetRepo, infraadmob.NewSSVVerifierFromEnv(), os.Getenv("AD_REWARD_HMAC_SECRET"), appEnv)
	billingUsecase := usecase.NewBillingUsecase(
		infrastripes.NewCheckoutClientFromEnv(),
		infrastripes.NewWebhookValidatorFromEnv(),
		billingRepo,
	)
	extensionUsecase, err := usecase.NewExtensionUsecase(extensionPairingRepo, rateLimitRepo)
	if err != nil {
		return nil, err
	}
	adminUsecase, err := usecase.NewAdminUsecase(adminRepo, sessionCookieClient, questionTaskEnqueuer)
	if err != nil {
		return nil, err
	}
	userHandler := handler.NewUserHandler(userUsecase)
	postHandler := handler.NewPostHandler(postUsecase, userUsecase)
	questionHandler := handler.NewQuestionHandler(questionUsecase, questionSyncUsecase, userUsecase, manualGenerationUsecase)
	answerHandler := handler.NewAnswerHandler(answerUsecase, userUsecase, questionSyncUsecase)
	socialHandler := handler.NewSocialHandler(socialUsecase, postUsecase, userUsecase)
	highlightHandler := handler.NewHighlightHandler(highlightUsecase, userUsecase)
	tokenHandler := handler.NewTokenHandler(tokenUsecase, userUsecase)
	stripeHandler := handler.NewStripeHandler(billingUsecase, userUsecase)
	authHandler := handler.NewAuthHandler(sessionCookieClient, appEnv, os.Getenv("SESSION_COOKIE_DOMAIN"))
	extensionHandler := handler.NewExtensionHandler(extensionUsecase, userUsecase)
	taskHandler := handler.NewTaskHandler(questionWorkerUsecase, highlightImportJobUsecase)
	adminHandler := handler.NewAdminHandler(adminUsecase)
	closeCloudTasks := make([]func() error, 0, 2)
	if questionTaskEnqueuer != nil {
		closeCloudTasks = append(closeCloudTasks, questionTaskEnqueuer.Close)
	}
	if highlightJobTrigger != nil {
		closeCloudTasks = append(closeCloudTasks, highlightJobTrigger.Close)
	}
	return &Container{
		UserHandler:                     userHandler,
		PostHandler:                     postHandler,
		QuestionHandler:                 questionHandler,
		AnswerHandler:                   answerHandler,
		SocialHandler:                   socialHandler,
		HighlightHandler:                highlightHandler,
		TokenHandler:                    tokenHandler,
		StripeHandler:                   stripeHandler,
		AuthHandler:                     authHandler,
		ExtensionHandler:                extensionHandler,
		TaskHandler:                     taskHandler,
		AdminHandler:                    adminHandler,
		FirebaseMiddleware:              firebaseMiddleware,
		SessionAuthMiddleware:           sessionAuthMiddleware,
		CSRFMiddleware:                  csrfMiddleware,
		AdminMiddleware:                 adminMiddleware,
		HybridAuthMiddleware:            hybridAuthMiddleware,
		SecurityHeadersMiddleware:       securityHeadersMiddleware,
		IngestRateLimitMiddleware:       ingestRateLimitMiddleware,
		GenerationRateLimitMiddleware:   generationRateLimitMiddleware,
		PostRateLimitMiddleware:         postRateLimitMiddleware,
		SocialRateLimitMiddleware:       socialRateLimitMiddleware,
		TokenRateLimitMiddleware:        tokenRateLimitMiddleware,
		PairingStartRateLimitMiddleware: pairingStartRateLimitMiddleware,
		AdMobSSVRateLimitMiddleware:     adMobSSVRateLimitMiddleware,
		closeCloudTasks:                 closeCloudTasks,
		closeLLMClient:                  closeLLMClient,
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
