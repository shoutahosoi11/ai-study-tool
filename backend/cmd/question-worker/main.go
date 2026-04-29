package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/shout/ai-study-tool/backend/internal/infrastructure/gemini"
	"github.com/shout/ai-study-tool/backend/internal/infrastructure/persistence"
	"github.com/shout/ai-study-tool/backend/internal/usecase"
)

const defaultPollInterval = 10 * time.Minute

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
		log.Fatalf("failed to ping database: %v", err)
	}

	llmClient, closeLLMClient, err := gemini.NewConfiguredClient(os.Getenv("GEMINI_API_KEY"))
	if err != nil {
		log.Fatalf("failed to create gemini client: %v", err)
	}
	defer closeLLMClient()

	highlightRepo := persistence.NewHighlightRepository(db)
	questionRepo := persistence.NewQuestionRepository(db)
	worker := usecase.NewQuestionWorkerUsecase(highlightRepo, questionRepo, llmClient)

	if os.Getenv("WORKER_RUN_ONCE") == "1" {
		if err := worker.RunOnce(context.Background()); err != nil {
			log.Fatalf("question worker run failed: %v", err)
		}
		return
	}

	pollInterval := readPollInterval()
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	log.Printf("question worker started with poll interval %s", pollInterval)
	for {
		if err := worker.RunOnce(context.Background()); err != nil {
			log.Printf("question worker run error: %v", err)
		}
		<-ticker.C
	}
}

func readPollInterval() time.Duration {
	raw := os.Getenv("QUESTION_WORKER_POLL_INTERVAL_SECONDS")
	if raw == "" {
		return defaultPollInterval
	}

	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds <= 0 {
		return defaultPollInterval
	}

	return time.Duration(seconds) * time.Second
}
