package persistence

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

func TestSocialRepositoryIntegration(t *testing.T) {
	db := openSocialQueueIntegrationDB(t)
	defer db.Close()
	rollbackSocialRepositoryMigrations(t, db)
	applySocialRepositoryMigrations(t, db)
	resetSocialRepositoryIntegrationDB(t, db)

	ctx := context.Background()
	authorID := insertSocialQueueUser(t, db, "author")
	actorID := insertSocialQueueUser(t, db, "actor")
	postID := insertSocialQueuePost(t, db, authorID)
	repo := NewSocialRepository(db)

	if err := repo.Like(ctx, actorID.String(), postID.String()); err != nil {
		t.Fatalf("like: %v", err)
	}
	if err := repo.Like(ctx, actorID.String(), postID.String()); err != nil {
		t.Fatalf("duplicate like: %v", err)
	}
	if got := readSocialPostCounter(t, db, postID, "like_count"); got != 1 {
		t.Fatalf("like_count = %d, want 1", got)
	}
	if err := repo.Unlike(ctx, actorID.String(), postID.String()); err != nil {
		t.Fatalf("unlike: %v", err)
	}
	if err := repo.Unlike(ctx, actorID.String(), postID.String()); err != nil {
		t.Fatalf("duplicate unlike: %v", err)
	}
	if got := readSocialPostCounter(t, db, postID, "like_count"); got != 0 {
		t.Fatalf("like_count after unlike = %d, want 0", got)
	}

	comment, err := repo.CreateComment(ctx, &domain.Comment{
		PostID:  postID.String(),
		UserID:  actorID.String(),
		Content: "Looks useful",
	})
	if err != nil {
		t.Fatalf("create comment: %v", err)
	}
	if comment.Username == "" || comment.Content != "Looks useful" {
		t.Fatalf("unexpected comment: %#v", comment)
	}
	if got := readSocialPostCounter(t, db, postID, "comment_count"); got != 1 {
		t.Fatalf("comment_count = %d, want 1", got)
	}
	comments, err := repo.ListComments(ctx, domain.ListCommentsInput{PostID: postID.String(), Limit: 10})
	if err != nil {
		t.Fatalf("list comments: %v", err)
	}
	if len(comments) != 1 || comments[0].ID != comment.ID {
		t.Fatalf("unexpected comments: %#v", comments)
	}
}

func TestHighlightImportQueueRepositoryIntegration(t *testing.T) {
	db := openSocialQueueIntegrationDB(t)
	defer db.Close()
	rollbackHighlightImportQueueMigrations(t, db)
	applyHighlightImportQueueMigrations(t, db)
	resetHighlightImportQueueIntegrationDB(t, db)

	ctx := context.Background()
	userID := insertSocialQueueUser(t, db, "queue")
	otherUserID := insertSocialQueueUser(t, db, "otherqueue")
	repo := NewHighlightImportQueueRepository(db)

	id, err := repo.Enqueue(ctx, userID, "kindle", []byte(`{"items":[]}`))
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	item, err := repo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if item.UserID != userID || item.Status != domain.ImportQueueStatusQueued {
		t.Fatalf("unexpected queue item: %#v", item)
	}

	claimed, err := repo.ClaimProcessing(ctx, id)
	if err != nil {
		t.Fatalf("claim processing: %v", err)
	}
	if !claimed {
		t.Fatal("expected first claim to succeed")
	}
	claimed, err = repo.ClaimProcessing(ctx, id)
	if err != nil {
		t.Fatalf("claim processing second: %v", err)
	}
	if claimed {
		t.Fatal("expected second claim to be no-op")
	}

	if err := repo.MarkEnqueueFailed(ctx, id, "cloud task unavailable"); err != nil {
		t.Fatalf("mark enqueue failed: %v", err)
	}
	if _, err := repo.Enqueue(ctx, otherUserID, "kindle", []byte(`{"items":[1]}`)); err != nil {
		t.Fatalf("enqueue other user: %v", err)
	}
	recoverable, err := repo.ListRecoverableEnqueuesByUserID(ctx, userID, time.Now().Add(time.Hour), 10)
	if err != nil {
		t.Fatalf("list recoverable: %v", err)
	}
	if len(recoverable) != 1 || recoverable[0].ID != id {
		t.Fatalf("unexpected recoverable items: %#v", recoverable)
	}
}

func openSocialQueueIntegrationDB(t *testing.T) *sql.DB {
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

func applySocialRepositoryMigrations(t *testing.T, db *sql.DB) {
	t.Helper()
	for _, file := range []string{
		"001_create_fields.sql",
		"002_create_users.sql",
		"003_create_books.sql",
		"008_create_questions.sql",
		"010_create_posts.sql",
		"011_create_social.sql",
		"017_add_unique_constraints_social.sql",
	} {
		applySocialQueueMigration(t, db, file)
	}
}

func rollbackSocialRepositoryMigrations(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec(`
DROP TABLE IF EXISTS comments CASCADE;
DROP TABLE IF EXISTS reposts CASCADE;
DROP TABLE IF EXISTS likes CASCADE;
DROP TABLE IF EXISTS follows CASCADE;
DROP TABLE IF EXISTS posts CASCADE;
DROP TABLE IF EXISTS questions CASCADE;
DROP TABLE IF EXISTS books CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS fields CASCADE;
`); err != nil {
		t.Fatalf("rollback social migrations: %v", err)
	}
}

func resetSocialRepositoryIntegrationDB(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec(`TRUNCATE users RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("reset social integration db: %v", err)
	}
}

func applyHighlightImportQueueMigrations(t *testing.T, db *sql.DB) {
	t.Helper()
	for _, file := range []string{
		"002_create_users.sql",
		"040_create_highlight_import_queue.sql",
		"042_add_highlight_import_queue_updated_at.sql",
		"043_add_highlight_import_queue_constraints.sql",
	} {
		applySocialQueueMigration(t, db, file)
	}
}

func rollbackHighlightImportQueueMigrations(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec(`
DROP TABLE IF EXISTS highlight_import_queue CASCADE;
DROP TABLE IF EXISTS users CASCADE;
`); err != nil {
		t.Fatalf("rollback highlight import queue migrations: %v", err)
	}
}

func resetHighlightImportQueueIntegrationDB(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec(`TRUNCATE users RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("reset highlight import queue integration db: %v", err)
	}
}

func applySocialQueueMigration(t *testing.T, db *sql.DB, file string) {
	t.Helper()
	sqlBytes, err := os.ReadFile(filepath.Join("..", "..", "..", "db", "migrations", file))
	if err != nil {
		t.Fatalf("read migration %s: %v", file, err)
	}
	if _, err := db.Exec(string(sqlBytes)); err != nil {
		t.Fatalf("apply migration %s: %v", file, err)
	}
}

func insertSocialQueueUser(t *testing.T, db *sql.DB, suffix string) uuid.UUID {
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

func insertSocialQueuePost(t *testing.T, db *sql.DB, userID uuid.UUID) uuid.UUID {
	t.Helper()
	postID := uuid.New()
	if _, err := db.Exec(`
INSERT INTO posts (id, user_id, type)
VALUES ($1, $2, 'text')
`, postID, userID); err != nil {
		t.Fatalf("insert post: %v", err)
	}
	return postID
}

func readSocialPostCounter(t *testing.T, db *sql.DB, postID uuid.UUID, column string) int {
	t.Helper()
	var count int
	if err := db.QueryRow(`SELECT `+column+` FROM posts WHERE id = $1`, postID).Scan(&count); err != nil {
		t.Fatalf("read post counter %s: %v", column, err)
	}
	return count
}
