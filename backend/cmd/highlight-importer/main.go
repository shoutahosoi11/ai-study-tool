package main

import (
	"context"
<<<<<<< feat/phase2-item-1-neon-tls
=======
	"database/sql"
>>>>>>> main
	"log"
	"os"

	"github.com/joho/godotenv"
<<<<<<< feat/phase2-item-1-neon-tls
	dbinfra "github.com/shout/ai-study-tool/backend/internal/infrastructure/db"
=======
	_ "github.com/lib/pq"
>>>>>>> main
	"github.com/shout/ai-study-tool/backend/internal/infrastructure/persistence"
	"github.com/shout/ai-study-tool/backend/internal/usecase"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

<<<<<<< feat/phase2-item-1-neon-tls
	db, err := dbinfra.Open(os.Getenv("DATABASE_URL"))
=======
	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
>>>>>>> main
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
