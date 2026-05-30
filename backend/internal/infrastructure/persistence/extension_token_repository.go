package persistence

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

type extensionTokenRepository struct {
	db *sql.DB
}

func NewExtensionTokenRepository(db *sql.DB) domain.ExtensionTokenRepository {
	return &extensionTokenRepository{db: db}
}

func NewExtensionPairingRepository(db *sql.DB) domain.ExtensionPairingRepository {
	return &extensionTokenRepository{db: db}
}

func (r *extensionTokenRepository) FindActiveByTokenHash(ctx context.Context, tokenHash string, now time.Time) (*domain.ExtensionToken, error) {
	normalizedHash := strings.TrimSpace(tokenHash)
	if normalizedHash == "" {
		return nil, domain.ErrNotFound
	}

	const query = `
SELECT
  et.id,
  et.user_id,
  u.firebase_uid,
  et.token_hash,
  et.name,
  et.scopes,
  et.created_at,
  et.last_used_at,
  et.expires_at,
  et.revoked_at
FROM extension_tokens et
JOIN users u ON u.id = et.user_id
WHERE et.token_hash = $1
  AND et.revoked_at IS NULL
  AND (et.expires_at IS NULL OR et.expires_at > $2)
LIMIT 1`

	var token domain.ExtensionToken
	var name sql.NullString
	var scopes pq.StringArray
	var lastUsedAt sql.NullTime
	var expiresAt sql.NullTime
	var revokedAt sql.NullTime
	if err := r.db.QueryRowContext(ctx, query, normalizedHash, now).Scan(
		&token.ID,
		&token.UserID,
		&token.FirebaseUID,
		&token.TokenHash,
		&name,
		&scopes,
		&token.CreatedAt,
		&lastUsedAt,
		&expiresAt,
		&revokedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("extension token repo: find active by hash: %w", err)
	}

	if name.Valid {
		token.Name = &name.String
	}
	token.Scopes = append([]string(nil), scopes...)
	token.LastUsedAt = extensionNullableTimePtr(lastUsedAt)
	token.ExpiresAt = extensionNullableTimePtr(expiresAt)
	token.RevokedAt = extensionNullableTimePtr(revokedAt)

	lastUsedThreshold := now.Add(-1 * time.Hour)
	if token.LastUsedAt == nil || token.LastUsedAt.Before(lastUsedThreshold) {
		if _, err := r.db.ExecContext(ctx, `
UPDATE extension_tokens
SET last_used_at = $2
WHERE id = $1
  AND (last_used_at IS NULL OR last_used_at < $3)
`, token.ID, now, lastUsedThreshold); err != nil {
			slog.Warn("extension_token_last_used_update_failed", "token_id", token.ID.String(), "error", err)
			return &token, nil
		}
		token.LastUsedAt = &now
	}

	return &token, nil
}

func extensionNullableTimePtr(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	t := value.Time
	return &t
}

func (r *extensionTokenRepository) CreateExtensionPairing(ctx context.Context, userCode string, expiresAt time.Time) (*domain.ExtensionPairing, error) {
	const query = `
INSERT INTO extension_pairings (user_code, expires_at)
VALUES ($1, $2)
RETURNING id, user_code, scopes, created_at, expires_at`

	var pairing domain.ExtensionPairing
	var scopes pq.StringArray
	if err := r.db.QueryRowContext(ctx, query, strings.TrimSpace(userCode), expiresAt).Scan(
		&pairing.ID,
		&pairing.UserCode,
		&scopes,
		&pairing.CreatedAt,
		&pairing.ExpiresAt,
	); err != nil {
		if isUniqueViolation(err) {
			return nil, domain.ErrAlreadyExists
		}
		return nil, fmt.Errorf("extension pairing repo: create: %w", err)
	}
	pairing.Scopes = append([]string(nil), scopes...)
	return &pairing, nil
}

func (r *extensionTokenRepository) GetExtensionPairingStatus(ctx context.Context, pairingID uuid.UUID, now time.Time) (*domain.ExtensionPairing, error) {
	const query = `
SELECT id, user_code, user_id, token_id, scopes, created_at, expires_at, approved_at, used_at
FROM extension_pairings
WHERE id = $1
  AND expires_at > $2
LIMIT 1`

	var pairing domain.ExtensionPairing
	var userID uuid.NullUUID
	var tokenID uuid.NullUUID
	var scopes pq.StringArray
	var approvedAt sql.NullTime
	var usedAt sql.NullTime
	if err := r.db.QueryRowContext(ctx, query, pairingID, now).Scan(
		&pairing.ID,
		&pairing.UserCode,
		&userID,
		&tokenID,
		&scopes,
		&pairing.CreatedAt,
		&pairing.ExpiresAt,
		&approvedAt,
		&usedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("extension pairing repo: status: %w", err)
	}
	if userID.Valid {
		pairing.UserID = &userID.UUID
	}
	if tokenID.Valid {
		pairing.TokenID = &tokenID.UUID
	}
	pairing.Scopes = append([]string(nil), scopes...)
	pairing.ApprovedAt = extensionNullableTimePtr(approvedAt)
	pairing.UsedAt = extensionNullableTimePtr(usedAt)
	return &pairing, nil
}

func (r *extensionTokenRepository) ApproveExtensionPairing(ctx context.Context, userCode string, userID uuid.UUID, scopes []string, now time.Time) error {
	const query = `
UPDATE extension_pairings
SET user_id = $2, scopes = $3, approved_at = $4
WHERE user_code = $1
  AND expires_at > $4
  AND approved_at IS NULL
  AND used_at IS NULL`

	result, err := r.db.ExecContext(ctx, query, strings.TrimSpace(userCode), userID, pq.Array(domain.NormalizeExtensionScopes(scopes)), now)
	if err != nil {
		return fmt.Errorf("extension pairing repo: approve: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("extension pairing repo: approve rows affected: %w", err)
	}
	if affected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *extensionTokenRepository) CreateExtensionTokenForApprovedPairing(ctx context.Context, input domain.CreateExtensionTokenForPairingInput) (*domain.ExtensionToken, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("extension pairing repo: begin consume: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	const selectPairing = `
SELECT user_id, scopes, approved_at, used_at
FROM extension_pairings
WHERE id = $1
  AND expires_at > $2
FOR UPDATE`

	var userID uuid.NullUUID
	var scopes pq.StringArray
	var approvedAt sql.NullTime
	var usedAt sql.NullTime
	if err := tx.QueryRowContext(ctx, selectPairing, input.PairingID, input.Now).Scan(
		&userID,
		&scopes,
		&approvedAt,
		&usedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("extension pairing repo: select consume: %w", err)
	}
	if usedAt.Valid {
		return nil, domain.ErrAlreadyExists
	}
	if !approvedAt.Valid || !userID.Valid {
		return nil, domain.ErrPairingNotApproved
	}

	normalizedScopes := domain.NormalizeExtensionScopes(input.Scopes)
	if len(normalizedScopes) == 0 {
		normalizedScopes = domain.NormalizeExtensionScopes(scopes)
	}
	var nameArg any
	if input.Name != nil {
		nameArg = *input.Name
	}

	const insertToken = `
INSERT INTO extension_tokens (id, user_id, token_hash, name, scopes, expires_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, user_id, token_hash, name, scopes, created_at, last_used_at, expires_at, revoked_at`

	var token domain.ExtensionToken
	var name sql.NullString
	var tokenScopes pq.StringArray
	var lastUsedAt sql.NullTime
	var expiresAt sql.NullTime
	var revokedAt sql.NullTime
	if err := tx.QueryRowContext(ctx, insertToken,
		input.TokenID,
		userID.UUID,
		strings.TrimSpace(input.TokenHash),
		nameArg,
		pq.Array(normalizedScopes),
		input.ExpiresAt,
	).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&name,
		&tokenScopes,
		&token.CreatedAt,
		&lastUsedAt,
		&expiresAt,
		&revokedAt,
	); err != nil {
		return nil, fmt.Errorf("extension pairing repo: insert token: %w", err)
	}

	const markUsed = `
UPDATE extension_pairings
SET used_at = $2, token_id = $3
WHERE id = $1
  AND used_at IS NULL`
	result, err := tx.ExecContext(ctx, markUsed, input.PairingID, input.Now, token.ID)
	if err != nil {
		return nil, fmt.Errorf("extension pairing repo: mark used: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("extension pairing repo: mark used rows affected: %w", err)
	}
	if affected == 0 {
		return nil, domain.ErrAlreadyExists
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("extension pairing repo: commit consume: %w", err)
	}

	if name.Valid {
		token.Name = &name.String
	}
	token.Scopes = append([]string(nil), tokenScopes...)
	token.LastUsedAt = extensionNullableTimePtr(lastUsedAt)
	token.ExpiresAt = extensionNullableTimePtr(expiresAt)
	token.RevokedAt = extensionNullableTimePtr(revokedAt)
	return &token, nil
}

func (r *extensionTokenRepository) RevokeExtensionToken(ctx context.Context, tokenID uuid.UUID, userID uuid.UUID, now time.Time) error {
	const query = `
UPDATE extension_tokens
SET revoked_at = $3
WHERE id = $1
  AND user_id = $2
  AND revoked_at IS NULL`

	result, err := r.db.ExecContext(ctx, query, tokenID, userID, now)
	if err != nil {
		return fmt.Errorf("extension token repo: revoke: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("extension token repo: revoke rows affected: %w", err)
	}
	if affected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

var _ domain.ExtensionTokenRepository = (*extensionTokenRepository)(nil)
var _ domain.ExtensionPairingRepository = (*extensionTokenRepository)(nil)
