package persistence

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

func TestGlobalLLMBudgetRepositoryIntegration(t *testing.T) {
	db := openHighlightLLMIntegrationDB(t)
	defer db.Close()
	rollbackGlobalLLMRepositoryMigrations(t, db)
	applyGlobalLLMRepositoryMigrations(t, db)
	resetHighlightLLMIntegrationDB(t, db)

	ctx := context.Background()
	userID := insertHighlightLLMUser(t, db, "llm")
	repo := NewGlobalLLMBudgetRepository(db)
	budgetDate := time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC)

	budget, err := repo.Reserve(ctx, domain.ReserveGlobalLLMBudgetInput{
		BudgetDate:         budgetDate,
		DefaultMaxRequests: 2,
		DefaultMaxCostYen:  5,
		RequestCount:       1,
		EstimatedCostYen:   2,
	})
	if err != nil {
		t.Fatalf("reserve first: %v", err)
	}
	if budget.UsedRequests != 1 || budget.UsedEstimatedCostYen != 2 {
		t.Fatalf("unexpected first budget: %#v", budget)
	}

	_, err = repo.Reserve(ctx, domain.ReserveGlobalLLMBudgetInput{
		BudgetDate:         budgetDate,
		DefaultMaxRequests: 100,
		DefaultMaxCostYen:  100,
		RequestCount:       2,
		EstimatedCostYen:   1,
	})
	if !errors.Is(err, domain.ErrGlobalLLMBudgetExceeded) {
		t.Fatalf("reserve beyond existing request cap error = %v, want ErrGlobalLLMBudgetExceeded", err)
	}

	if err := repo.RecordUsage(ctx, domain.LLMUsageLogInput{
		ID:               uuid.New(),
		UserID:           userID,
		Provider:         "gemini",
		Model:            "gemini-test",
		InputTokens:      12,
		OutputTokens:     34,
		EstimatedCostYen: 1.5,
		CreatedAt:        time.Now().UTC(),
	}); err != nil {
		t.Fatalf("record usage: %v", err)
	}
}

func TestHighlightRepositoryIntegration(t *testing.T) {
	db := openHighlightLLMIntegrationDB(t)
	defer db.Close()
	rollbackHighlightRepositoryMigrations(t, db)
	applyHighlightRepositoryMigrations(t, db)
	resetHighlightLLMIntegrationDB(t, db)

	ctx := context.Background()
	userID := insertHighlightLLMUser(t, db, "highlight")
	otherUserID := insertHighlightLLMUser(t, db, "otherhighlight")
	repo := NewHighlightRepository(db)

	highlightID := insertHighlightLLMHighlight(t, db, userID, "book-1", "ASIN1", "Deep Work", "Cal Newport", "completed", "A useful focused idea")
	otherHighlightID := insertHighlightLLMHighlight(t, db, otherUserID, "book-1", "ASIN1", "Deep Work", "Cal Newport", "completed", "Other user's idea")
	insertHighlightLLMHighlight(t, db, userID, "book-1", "ASIN1", "Deep Work", "Cal Newport", "pending", "Pending idea")

	byASIN, err := repo.ListByUserIDAndASIN(ctx, userID, "ASIN1")
	if err != nil {
		t.Fatalf("list by asin: %v", err)
	}
	if len(byASIN) != 2 {
		t.Fatalf("list by asin len = %d, want 2", len(byASIN))
	}
	for _, item := range byASIN {
		if item.UserID != userID {
			t.Fatalf("list by asin leaked user id: %#v", item)
		}
	}

	if err := repo.MarkHighlightsProcessing(ctx, otherUserID, []uuid.UUID{highlightID}); err != nil {
		t.Fatalf("mark other user processing: %v", err)
	}
	status := readHighlightLLMStatus(t, db, highlightID)
	if status != "completed" {
		t.Fatalf("wrong-user processing changed status to %s", status)
	}

	explanation := "owned explanation"
	updated, err := repo.UpdateExplanation(ctx, highlightID, userID, &explanation)
	if err != nil {
		t.Fatalf("update explanation: %v", err)
	}
	if updated.Explanation == nil || *updated.Explanation != explanation {
		t.Fatalf("unexpected updated explanation: %#v", updated.Explanation)
	}
	_, err = repo.UpdateExplanation(ctx, otherHighlightID, userID, &explanation)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("wrong-user update explanation error = %v, want ErrNotFound", err)
	}
}

func openHighlightLLMIntegrationDB(t *testing.T) *sql.DB {
	t.Helper()
	databaseURL := os.Getenv("INTEGRATION_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("INTEGRATION_DATABASE_URL is not set")
	}
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.PingContext(context.Background()); err != nil {
		t.Fatalf("ping db: %v", err)
	}
	return db
}

func applyGlobalLLMRepositoryMigrations(t *testing.T, db *sql.DB) {
	t.Helper()
	for _, file := range []string{
		"001_create_fields.sql",
		"002_create_users.sql",
		"003_create_books.sql",
		"006_create_highlights.sql",
		"008_create_questions.sql",
		"032_create_question_generation_jobs.sql",
		"048_create_global_llm_budgets.sql",
	} {
		applyHighlightLLMMigration(t, db, file)
	}
}

func rollbackGlobalLLMRepositoryMigrations(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec(`
DROP TABLE IF EXISTS llm_usage_logs;
DROP TABLE IF EXISTS global_llm_budgets;
DROP TABLE IF EXISTS question_generation_job_highlights;
DROP TABLE IF EXISTS question_generation_jobs;
DROP TABLE IF EXISTS questions;
DROP TABLE IF EXISTS highlights;
DROP TABLE IF EXISTS books;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS fields;
`); err != nil {
		t.Fatalf("rollback global llm migrations: %v", err)
	}
}

func applyHighlightRepositoryMigrations(t *testing.T, db *sql.DB) {
	t.Helper()
	for _, file := range []string{
		"001_create_fields.sql",
		"002_create_users.sql",
		"003_create_books.sql",
		"006_create_highlights.sql",
		"008_create_questions.sql",
		"029_add_async_question_generation.sql",
		"035_add_highlights_book_status_index.sql",
		"037_add_highlight_book_order_index.sql",
		"041_backfill_highlight_book_keys.sql",
	} {
		applyHighlightLLMMigration(t, db, file)
	}
}

func rollbackHighlightRepositoryMigrations(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec(`
DROP INDEX IF EXISTS uq_highlights_book_order_index;
DROP INDEX IF EXISTS idx_highlights_user_book_status;
DROP INDEX IF EXISTS idx_highlights_status_user_requested_at;
DROP INDEX IF EXISTS highlights_user_id_content_hash_idx;
DROP TABLE IF EXISTS regeneration_queue;
DROP TABLE IF EXISTS questions;
DROP TABLE IF EXISTS highlights;
DROP TABLE IF EXISTS books;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS fields;
`); err != nil {
		t.Fatalf("rollback highlight migrations: %v", err)
	}
}

func resetHighlightLLMIntegrationDB(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec(`TRUNCATE users RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("reset highlight llm integration db: %v", err)
	}
}

func applyHighlightLLMMigration(t *testing.T, db *sql.DB, file string) {
	t.Helper()
	sqlBytes, err := os.ReadFile(filepath.Join("..", "..", "..", "db", "migrations", file))
	if err != nil {
		t.Fatalf("read migration %s: %v", file, err)
	}
	if _, err := db.Exec(string(sqlBytes)); err != nil {
		t.Fatalf("apply migration %s: %v", file, err)
	}
}

func insertHighlightLLMUser(t *testing.T, db *sql.DB, suffix string) uuid.UUID {
	t.Helper()
	userID := uuid.New()
	if _, err := db.Exec(`
INSERT INTO users (id, firebase_uid, username, display_name)
VALUES ($1, $2, $3, $4)
`, userID, "firebase-"+suffix+"-"+userID.String(), "user_"+suffix+"_"+userID.String()[:8], "Integration User"); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	return userID
}

func insertHighlightLLMHighlight(
	t *testing.T,
	db *sql.DB,
	userID uuid.UUID,
	bookKey string,
	asin string,
	title string,
	author string,
	status string,
	content string,
) uuid.UUID {
	t.Helper()
	highlightID := uuid.New()
	if _, err := db.Exec(`
INSERT INTO highlights (
  id, user_id, content, book_title, book_author, asin, source,
  status, book_key, content_hash, generation_requested_at, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, 'extension', $7, $8, $9, NOW(), NOW())
`, highlightID, userID, content, title, author, asin, status, bookKey, "hash-"+highlightID.String()); err != nil {
		t.Fatalf("insert highlight: %v", err)
	}
	return highlightID
}

func readHighlightLLMStatus(t *testing.T, db *sql.DB, highlightID uuid.UUID) string {
	t.Helper()
	var status string
	if err := db.QueryRow(`SELECT status FROM highlights WHERE id = $1`, highlightID).Scan(&status); err != nil {
		t.Fatalf("read highlight status: %v", err)
	}
	return status
}
