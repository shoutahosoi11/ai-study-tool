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
	"github.com/lib/pq"
	_ "github.com/lib/pq"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

func TestSecurityRepositoriesIntegration(t *testing.T) {
	db := openSecurityRepositoriesIntegrationDB(t)
	defer db.Close()

	rollbackSecurityRepositoryMigrations(t, db)
	applySecurityRepositoryMigrations(t, db)
	resetSecurityRepositoryIntegrationDB(t, db)

	ctx := context.Background()
	userID := insertSecurityRepositoryUser(t, db, "primary")
	otherUserID := insertSecurityRepositoryUser(t, db, "other")
	now := time.Date(2026, 5, 31, 10, 0, 0, 0, time.UTC)

	t.Run("rate limit increments per user bucket and reports limit", func(t *testing.T) {
		repo := NewRateLimitRepository(db)

		count, exceeded, err := repo.IncrementAndCheck(ctx, userID.String(), "extension_pairing_approve", 1)
		if err != nil {
			t.Fatalf("increment first: %v", err)
		}
		if count != 1 || exceeded {
			t.Fatalf("first increment count=%d exceeded=%v", count, exceeded)
		}

		count, exceeded, err = repo.IncrementAndCheck(ctx, userID.String(), "extension_pairing_approve", 1)
		if err != nil {
			t.Fatalf("increment second: %v", err)
		}
		if count != 2 || !exceeded {
			t.Fatalf("second increment count=%d exceeded=%v", count, exceeded)
		}

		count, err = repo.Count(ctx, userID.String(), "extension_pairing_approve")
		if err != nil {
			t.Fatalf("count: %v", err)
		}
		if count != 2 {
			t.Fatalf("count = %d, want 2", count)
		}

		otherBucketCount, err := repo.Count(ctx, userID.String(), "other")
		if err != nil {
			t.Fatalf("count other bucket: %v", err)
		}
		if otherBucketCount != 0 {
			t.Fatalf("other bucket count = %d, want 0", otherBucketCount)
		}
	})

	t.Run("extension tokens enforce active token and owner scoped revoke", func(t *testing.T) {
		repo := &extensionTokenRepository{db: db}
		tokenID := uuid.New()
		tokenHash := "active-token-hash"
		if _, err := db.ExecContext(ctx, `
INSERT INTO extension_tokens (id, user_id, token_hash, name, scopes, expires_at)
VALUES ($1, $2, $3, 'Chrome', $4, $5)
`, tokenID, userID, tokenHash, pq.Array(domain.DefaultExtensionTokenScopes()), now.Add(time.Hour)); err != nil {
			t.Fatalf("insert extension token: %v", err)
		}

		token, err := repo.FindActiveByTokenHash(ctx, " "+tokenHash+" ", now)
		if err != nil {
			t.Fatalf("find active token: %v", err)
		}
		if token.ID != tokenID || token.UserID != userID || token.FirebaseUID == "" {
			t.Fatalf("unexpected active token: %#v", token)
		}
		if !domain.HasScope(token.Scopes, domain.ExtensionScopeHighlightWrite) || token.LastUsedAt == nil {
			t.Fatalf("expected scopes and last used update, got scopes=%v lastUsed=%v", token.Scopes, token.LastUsedAt)
		}

		if err := repo.RevokeExtensionToken(ctx, tokenID, otherUserID, now); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("revoke by other user error = %v, want ErrNotFound", err)
		}
		if _, err := repo.FindActiveByTokenHash(ctx, tokenHash, now); err != nil {
			t.Fatalf("token should still be active after wrong-user revoke: %v", err)
		}
		if err := repo.RevokeExtensionToken(ctx, tokenID, userID, now); err != nil {
			t.Fatalf("revoke by owner: %v", err)
		}
		if _, err := repo.FindActiveByTokenHash(ctx, tokenHash, now); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("find revoked token error = %v, want ErrNotFound", err)
		}
	})

	t.Run("question budget enforces daily ad limit and consumes tokens after free quota", func(t *testing.T) {
		repo := NewQuestionBudgetRepository(db)

		for i := 0; i < domain.MaxAdViewsPerDay; i++ {
			_, err := repo.AwardAdTokens(ctx, userID, domain.AdRewardClaim{
				Provider:   "test",
				Nonce:      uuid.NewString(),
				RewardedAt: now,
			}, now)
			if err != nil {
				t.Fatalf("award ad tokens %d: %v", i, err)
			}
		}

		_, err := repo.AwardAdTokens(ctx, userID, domain.AdRewardClaim{
			Provider:   "test",
			Nonce:      "daily-limit",
			RewardedAt: now,
		}, now)
		if !errors.Is(err, domain.ErrQuestionBudgetExceeded) {
			t.Fatalf("fourth ad award error = %v, want ErrQuestionBudgetExceeded", err)
		}

		balance, err := repo.ReserveQuestions(ctx, userID, "free", domain.FreeDailyQuestionLimit+2, now)
		if err != nil {
			t.Fatalf("reserve questions: %v", err)
		}
		if balance.FreeUsedToday != domain.FreeDailyQuestionLimit || balance.AvailableTokens != domain.AdTokensPerView*domain.MaxAdViewsPerDay-2 {
			t.Fatalf("unexpected balance after reserve: %#v", balance)
		}

		_, err = repo.AwardAdTokens(ctx, otherUserID, domain.AdRewardClaim{
			Provider:   "test",
			Nonce:      "duplicate-nonce",
			RewardedAt: now,
		}, now)
		if err != nil {
			t.Fatalf("award duplicate setup: %v", err)
		}
		_, err = repo.AwardAdTokens(ctx, otherUserID, domain.AdRewardClaim{
			Provider:   "test",
			Nonce:      "duplicate-nonce",
			RewardedAt: now,
		}, now)
		if !errors.Is(err, domain.ErrAlreadyExists) {
			t.Fatalf("duplicate award error = %v, want ErrAlreadyExists", err)
		}
	})
}

func openSecurityRepositoriesIntegrationDB(t *testing.T) *sql.DB {
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

func applySecurityRepositoryMigrations(t *testing.T, db *sql.DB) {
	t.Helper()
	for _, file := range []string{
		"001_create_fields.sql",
		"002_create_users.sql",
		"003_create_books.sql",
		"006_create_highlights.sql",
		"031_security_hardening.sql",
		"032_create_question_generation_jobs.sql",
		"039_add_question_budget_and_tokens.sql",
		"045_create_ad_reward_claims.sql",
		"046_create_extension_tokens.sql",
		"047_create_extension_pairings.sql",
		"049_harden_billing_and_admob.sql",
		"050_add_extension_pairing_user_code.sql",
	} {
		sqlBytes, err := os.ReadFile(filepath.Join("..", "..", "..", "db", "migrations", file))
		if err != nil {
			t.Fatalf("read migration %s: %v", file, err)
		}
		if _, err := db.Exec(string(sqlBytes)); err != nil {
			t.Fatalf("apply migration %s: %v", file, err)
		}
	}
}

func rollbackSecurityRepositoryMigrations(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec(`
DROP TABLE IF EXISTS admob_ssv_events;
DROP TABLE IF EXISTS subscriptions;
DROP TABLE IF EXISTS stripe_events;
DROP TABLE IF EXISTS extension_pairings;
DROP TABLE IF EXISTS extension_tokens;
DROP TABLE IF EXISTS ad_reward_claims;
DROP TABLE IF EXISTS user_ad_tokens;
DROP TABLE IF EXISTS question_daily_budgets;
DROP TABLE IF EXISTS rate_limit_counters;
DROP TABLE IF EXISTS question_generation_job_highlights;
DROP TABLE IF EXISTS question_generation_jobs;
DROP TABLE IF EXISTS highlights;
DROP TABLE IF EXISTS books;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS fields;
`); err != nil {
		t.Fatalf("rollback security repository migrations: %v", err)
	}
}

func resetSecurityRepositoryIntegrationDB(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec(`
TRUNCATE users RESTART IDENTITY CASCADE;
TRUNCATE rate_limit_counters RESTART IDENTITY;
`); err != nil {
		t.Fatalf("reset integration db: %v", err)
	}
}

func insertSecurityRepositoryUser(t *testing.T, db *sql.DB, suffix string) uuid.UUID {
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
