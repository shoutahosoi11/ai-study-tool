package main

import (
	"context"
	"database/sql"
	"log"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/shout/ai-study-tool/backend/internal/infrastructure/persistence"
	"github.com/shout/ai-study-tool/backend/internal/usecase"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("highlight-importer: failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("highlight-importer: failed to ping database: %v", err)
	}

	queueRepo := persistence.NewHighlightImportQueueRepository(db)
	highlightRepo := persistence.NewHighlightRepository(db)
	jobUsecase := usecase.NewHighlightImportJobUsecase(queueRepo, highlightRepo)

	ctx := context.Background()
	if err := jobUsecase.ProcessAll(ctx); err != nil {
		log.Fatalf("highlight-importer: ProcessAll failed: %v", err)
	}

	log.Println("highlight-importer: completed successfully")
	os.Exit(0)
}
