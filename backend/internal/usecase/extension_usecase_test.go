package usecase

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

type fakeExtensionPairingRepository struct {
	pairings  map[uuid.UUID]*domain.ExtensionPairing
	userCodes map[string]uuid.UUID
	tokens    map[uuid.UUID]*domain.ExtensionToken
	hashes    []string
	createErr error
	creates   int
}

type fakeExtensionRateLimitRepository struct {
	exceeded bool
	calls    []fakeExtensionRateLimitCall
	counts   map[string]int64
}

type fakeExtensionRateLimitCall struct {
	userID string
	bucket string
	limit  int64
}

func (r *fakeExtensionRateLimitRepository) IncrementAndCheck(ctx context.Context, userID, bucket string, limit int64) (int64, bool, error) {
	r.calls = append(r.calls, fakeExtensionRateLimitCall{
		userID: userID,
		bucket: bucket,
		limit:  limit,
	})
	return limit, r.exceeded, nil
}

func (r *fakeExtensionRateLimitRepository) Count(ctx context.Context, userID, bucket string) (int64, error) {
	if r.counts == nil {
		return 0, nil
	}
	return r.counts[userID+"|"+bucket], nil
}

func newExtensionUsecaseForTest(repo domain.ExtensionPairingRepository, now func() time.Time, random io.Reader, rateLimit ...domain.RateLimitRepository) *ExtensionUsecase {
	rateLimitRepo := domain.RateLimitRepository(&fakeExtensionRateLimitRepository{})
	if len(rateLimit) > 0 {
		rateLimitRepo = rateLimit[0]
	}
	uc := NewExtensionUsecase(repo, rateLimitRepo)
	if now != nil {
		uc.now = now
	}
	if random != nil {
		uc.random = random
	}
	return uc
}

func newFakeExtensionPairingRepository() *fakeExtensionPairingRepository {
	return &fakeExtensionPairingRepository{
		pairings:  make(map[uuid.UUID]*domain.ExtensionPairing),
		userCodes: make(map[string]uuid.UUID),
		tokens:    make(map[uuid.UUID]*domain.ExtensionToken),
	}
}

func TestNewExtensionUsecaseRequiresRateLimitRepository(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected constructor to fail without rate limit repository")
		}
	}()

	_ = NewExtensionUsecase(newFakeExtensionPairingRepository(), nil)
}

func (r *fakeExtensionPairingRepository) CreateExtensionPairing(ctx context.Context, userCode string, expiresAt time.Time) (*domain.ExtensionPairing, error) {
	r.creates++
	if r.createErr != nil {
		return nil, r.createErr
	}
	if _, exists := r.userCodes[userCode]; exists {
		return nil, domain.ErrAlreadyExists
	}
	pairing := &domain.ExtensionPairing{
		ID:        uuid.New(),
		UserCode:  userCode,
		Scopes:    domain.DefaultExtensionTokenScopes(),
		CreatedAt: expiresAt.Add(-defaultExtensionPairingTTL),
		ExpiresAt: expiresAt,
	}
	r.pairings[pairing.ID] = pairing
	r.userCodes[userCode] = pairing.ID
	return cloneExtensionPairing(pairing), nil
}

func (r *fakeExtensionPairingRepository) GetExtensionPairingStatus(ctx context.Context, pairingID uuid.UUID, now time.Time) (*domain.ExtensionPairing, error) {
	pairing, ok := r.pairings[pairingID]
	if !ok || !pairing.ExpiresAt.After(now) {
		return nil, domain.ErrNotFound
	}
	return cloneExtensionPairing(pairing), nil
}

func (r *fakeExtensionPairingRepository) ApproveExtensionPairing(ctx context.Context, userCode string, userID uuid.UUID, scopes []string, now time.Time) error {
	pairingID, exists := r.userCodes[userCode]
	if !exists {
		return domain.ErrNotFound
	}
	pairing, ok := r.pairings[pairingID]
	if !ok || !pairing.ExpiresAt.After(now) || pairing.ApprovedAt != nil || pairing.UsedAt != nil {
		return domain.ErrNotFound
	}
	pairing.UserID = &userID
	pairing.Scopes = domain.NormalizeExtensionScopes(scopes)
	pairing.ApprovedAt = &now
	return nil
}

func (r *fakeExtensionPairingRepository) CreateExtensionTokenForApprovedPairing(ctx context.Context, input domain.CreateExtensionTokenForPairingInput) (*domain.ExtensionToken, error) {
	pairing, ok := r.pairings[input.PairingID]
	if !ok || !pairing.ExpiresAt.After(input.Now) {
		return nil, domain.ErrNotFound
	}
	if pairing.UsedAt != nil {
		return nil, domain.ErrAlreadyExists
	}
	if pairing.UserID == nil || pairing.ApprovedAt == nil {
		return nil, domain.ErrPairingNotApproved
	}
	scopes := domain.NormalizeExtensionScopes(input.Scopes)
	token := &domain.ExtensionToken{
		ID:        input.TokenID,
		UserID:    *pairing.UserID,
		TokenHash: input.TokenHash,
		Scopes:    scopes,
		CreatedAt: input.Now,
		ExpiresAt: &input.ExpiresAt,
	}
	r.tokens[token.ID] = token
	r.hashes = append(r.hashes, input.TokenHash)
	pairing.TokenID = &token.ID
	pairing.UsedAt = &input.Now
	return token, nil
}

func (r *fakeExtensionPairingRepository) RevokeExtensionToken(ctx context.Context, tokenID uuid.UUID, userID uuid.UUID, now time.Time) error {
	token, ok := r.tokens[tokenID]
	if !ok || token.UserID != userID || token.RevokedAt != nil {
		return domain.ErrNotFound
	}
	token.RevokedAt = &now
	return nil
}

func TestExtensionPairingGeneratesTenCharacterUserCodeFromRandomSource(t *testing.T) {
	now := time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC)
	repo := newFakeExtensionPairingRepository()
	uc := newExtensionUsecaseForTest(repo, func() time.Time { return now }, bytes.NewReader([]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}))

	pairing, err := uc.StartPairing(context.Background())
	if err != nil {
		t.Fatalf("start pairing: %v", err)
	}
	if pairing.UserCode != "ABCDE-FGHJK" {
		t.Fatalf("unexpected generated user code: %s", pairing.UserCode)
	}
	if strings.ContainsAny(pairing.UserCode, "01IO") {
		t.Fatalf("user code contains ambiguous characters: %s", pairing.UserCode)
	}
}

func TestExtensionPairingStartRetriesWrappedDuplicateUserCode(t *testing.T) {
	now := time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC)
	repo := newFakeExtensionPairingRepository()
	repo.createErr = fmt.Errorf("wrapped: %w", domain.ErrAlreadyExists)
	uc := newExtensionUsecaseForTest(repo, func() time.Time { return now }, bytes.NewReader(make([]byte, 64)))

	_, err := uc.StartPairing(context.Background())
	if !errors.Is(err, domain.ErrAlreadyExists) {
		t.Fatalf("expected wrapped ErrAlreadyExists after retries, got %v", err)
	}
	if repo.creates != extensionUserCodeRetries {
		t.Fatalf("expected %d retries, got %d", extensionUserCodeRetries, repo.creates)
	}
}

func TestExtensionPairingExpires(t *testing.T) {
	now := time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC)
	repo := newFakeExtensionPairingRepository()
	uc := newExtensionUsecaseForTest(repo, func() time.Time { return now }, bytes.NewReader(make([]byte, 64)))

	pairing, err := uc.StartPairing(context.Background())
	if err != nil {
		t.Fatalf("start pairing: %v", err)
	}

	now = now.Add(defaultExtensionPairingTTL + time.Second)
	err = uc.ApprovePairing(context.Background(), uuid.New(), pairing.UserCode, "203.0.113.10")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected expired pairing to be not found, got %v", err)
	}
}

func TestExtensionPairingRequiresApprovalBeforeToken(t *testing.T) {
	now := time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC)
	repo := newFakeExtensionPairingRepository()
	uc := newExtensionUsecaseForTest(repo, func() time.Time { return now }, bytes.NewReader(make([]byte, 64)))

	pairing, err := uc.StartPairing(context.Background())
	if err != nil {
		t.Fatalf("start pairing: %v", err)
	}

	result, err := uc.ClaimPairing(context.Background(), pairing.ID, "203.0.113.10")
	if !errors.Is(err, domain.ErrPairingNotApproved) {
		t.Fatalf("expected pending pairing to be not approved, got result=%#v err=%v", result, err)
	}
}

func TestExtensionPairingStatusDoesNotConsumeToken(t *testing.T) {
	now := time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC)
	repo := newFakeExtensionPairingRepository()
	uc := newExtensionUsecaseForTest(repo, func() time.Time { return now }, bytes.NewReader(make([]byte, 128)))
	userID := uuid.New()

	pairing, err := uc.StartPairing(context.Background())
	if err != nil {
		t.Fatalf("start pairing: %v", err)
	}
	if err := uc.ApprovePairing(context.Background(), userID, pairing.UserCode, "203.0.113.10"); err != nil {
		t.Fatalf("approve pairing: %v", err)
	}

	status, err := uc.PairingStatus(context.Background(), pairing.ID)
	if err != nil {
		t.Fatalf("pairing status: %v", err)
	}
	if status.ApprovedAt == nil || status.UsedAt != nil || status.TokenID != nil {
		t.Fatalf("status should not consume pairing, got %#v", status)
	}

	result, err := uc.ClaimPairing(context.Background(), pairing.ID, "203.0.113.10")
	if err != nil {
		t.Fatalf("claim pairing after status: %v", err)
	}
	if !domain.IsExtensionRawToken(result.RawToken) {
		t.Fatalf("expected ext_ token, got %q", result.RawToken)
	}
}

func TestExtensionPairingClaimRateLimitExceeded(t *testing.T) {
	now := time.Date(2026, 5, 26, 12, 34, 0, 0, time.UTC)
	repo := newFakeExtensionPairingRepository()
	rateLimit := &fakeExtensionRateLimitRepository{exceeded: true}
	uc := newExtensionUsecaseForTest(repo, func() time.Time { return now }, bytes.NewReader(make([]byte, 128)), rateLimit)

	pairing, err := uc.StartPairing(context.Background())
	if err != nil {
		t.Fatalf("start pairing: %v", err)
	}

	result, err := uc.ClaimPairing(context.Background(), pairing.ID, "203.0.113.10")
	if !errors.Is(err, domain.ErrRateLimitExceeded) {
		t.Fatalf("expected rate limit exceeded, got result=%#v err=%v", result, err)
	}
	if len(rateLimit.calls) != 1 {
		t.Fatalf("expected one rate limit call, got %d", len(rateLimit.calls))
	}
	if !strings.HasPrefix(rateLimit.calls[0].userID, "hash:") {
		t.Fatalf("expected hashed rate limit identifier, got %s", rateLimit.calls[0].userID)
	}
	if strings.Contains(rateLimit.calls[0].userID, pairing.ID.String()) {
		t.Fatalf("rate limit identifier exposed raw pairing id: %s", rateLimit.calls[0].userID)
	}
	if strings.Contains(rateLimit.calls[0].userID, "203.0.113.10") {
		t.Fatalf("rate limit identifier exposed raw client identifier: %s", rateLimit.calls[0].userID)
	}
	if rateLimit.calls[0].bucket != "extension_pairing_claim:202605261234" {
		t.Fatalf("unexpected rate limit bucket: %s", rateLimit.calls[0].bucket)
	}
	if rateLimit.calls[0].limit != extensionPairingLimit {
		t.Fatalf("unexpected rate limit: %d", rateLimit.calls[0].limit)
	}
}

func TestExtensionPairingClaimChecksPairingAndClientRateLimit(t *testing.T) {
	now := time.Date(2026, 5, 26, 12, 34, 0, 0, time.UTC)
	repo := newFakeExtensionPairingRepository()
	rateLimit := &fakeExtensionRateLimitRepository{}
	uc := newExtensionUsecaseForTest(repo, func() time.Time { return now }, bytes.NewReader(make([]byte, 128)), rateLimit)

	pairing, err := uc.StartPairing(context.Background())
	if err != nil {
		t.Fatalf("start pairing: %v", err)
	}

	_, err = uc.ClaimPairing(context.Background(), pairing.ID, "203.0.113.10")
	if !errors.Is(err, domain.ErrPairingNotApproved) {
		t.Fatalf("expected not approved, got %v", err)
	}
	if len(rateLimit.calls) != 2 {
		t.Fatalf("expected pairing and client rate limit calls, got %d", len(rateLimit.calls))
	}
	if rateLimit.calls[0].userID == rateLimit.calls[1].userID {
		t.Fatal("expected separate pairing and client rate limit identifiers")
	}
	for _, call := range rateLimit.calls {
		if !strings.HasPrefix(call.userID, "hash:") {
			t.Fatalf("expected hashed rate limit identifier, got %s", call.userID)
		}
		if strings.Contains(call.userID, pairing.ID.String()) || strings.Contains(call.userID, "203.0.113.10") {
			t.Fatalf("rate limit identifier exposed raw input: %s", call.userID)
		}
	}
}

func TestExtensionPairingClaimRejectsEmptyClientIdentifier(t *testing.T) {
	now := time.Date(2026, 5, 26, 12, 34, 0, 0, time.UTC)
	repo := newFakeExtensionPairingRepository()
	rateLimit := &fakeExtensionRateLimitRepository{}
	uc := newExtensionUsecaseForTest(repo, func() time.Time { return now }, bytes.NewReader(make([]byte, 128)), rateLimit)

	result, err := uc.ClaimPairing(context.Background(), uuid.New(), "")
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected invalid input, got result=%#v err=%v", result, err)
	}
	if len(rateLimit.calls) != 0 {
		t.Fatalf("expected no rate limit call for invalid client identifier, got %d", len(rateLimit.calls))
	}
}

func TestExtensionPairingRateLimitChecksPreviousBucket(t *testing.T) {
	now := time.Date(2026, 5, 26, 12, 34, 0, 0, time.UTC)
	repo := newFakeExtensionPairingRepository()
	rateLimit := &fakeExtensionRateLimitRepository{
		counts: map[string]int64{},
	}
	uc := newExtensionUsecaseForTest(repo, func() time.Time { return now }, bytes.NewReader(make([]byte, 128)), rateLimit)

	err := uc.checkPairingRateLimit(context.Background(), "extension_pairing_approve", "code:ABCDE-FGHJK")
	if err != nil {
		t.Fatalf("first rate limit check: %v", err)
	}
	identifier := rateLimit.calls[0].userID
	rateLimit.counts[identifier+"|extension_pairing_approve:202605261233"] = 1

	err = uc.checkPairingRateLimit(context.Background(), "extension_pairing_approve", "code:ABCDE-FGHJK")
	if !errors.Is(err, domain.ErrRateLimitExceeded) {
		t.Fatalf("expected previous bucket to count toward limit, got %v", err)
	}
}

func TestNormalizeUserCodeRequiresTenCharacters(t *testing.T) {
	got, err := normalizeUserCode("abcde-fghjk")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "ABCDE-FGHJK" {
		t.Fatalf("unexpected normalized code: %s", got)
	}
	if _, err := normalizeUserCode("ABCD-EFGH"); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected 8-character code to be rejected, got %v", err)
	}
	if _, err := normalizeUserCode("ABCDE-FGHIJ"); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ambiguous character to be rejected, got %v", err)
	}
}

func TestExtensionPairingApproveFailureRateLimitedByCodeAndClient(t *testing.T) {
	now := time.Date(2026, 5, 26, 12, 34, 0, 0, time.UTC)
	repo := newFakeExtensionPairingRepository()
	rateLimit := &fakeExtensionRateLimitRepository{exceeded: true}
	uc := newExtensionUsecaseForTest(repo, func() time.Time { return now }, bytes.NewReader(make([]byte, 128)), rateLimit)

	err := uc.ApprovePairing(context.Background(), uuid.New(), "ABCDE-FGHJK", "203.0.113.10")
	if !errors.Is(err, domain.ErrRateLimitExceeded) {
		t.Fatalf("expected rate limit exceeded, got %v", err)
	}
	if len(rateLimit.calls) != 1 {
		t.Fatalf("expected code rate limit to stop before repo access, got %d calls", len(rateLimit.calls))
	}
	if !strings.HasPrefix(rateLimit.calls[0].userID, "hash:") {
		t.Fatalf("expected hashed rate limit identifier, got %s", rateLimit.calls[0].userID)
	}
	if strings.Contains(rateLimit.calls[0].userID, "ABCDE-FGHJK") || strings.Contains(rateLimit.calls[0].userID, "203.0.113.10") {
		t.Fatalf("rate limit identifier exposed raw approve input: %s", rateLimit.calls[0].userID)
	}
	if rateLimit.calls[0].bucket != "extension_pairing_approve:202605261234" {
		t.Fatalf("unexpected rate limit bucket: %s", rateLimit.calls[0].bucket)
	}
}

func TestExtensionPairingApproveChecksCodeAndClientRateLimit(t *testing.T) {
	now := time.Date(2026, 5, 26, 12, 34, 0, 0, time.UTC)
	repo := newFakeExtensionPairingRepository()
	rateLimit := &fakeExtensionRateLimitRepository{}
	uc := newExtensionUsecaseForTest(repo, func() time.Time { return now }, bytes.NewReader(make([]byte, 128)), rateLimit)

	err := uc.ApprovePairing(context.Background(), uuid.New(), "ABCDE-FGHJK", "203.0.113.10")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected unknown code to be not found, got %v", err)
	}
	if len(rateLimit.calls) != 2 {
		t.Fatalf("expected code and client rate limit calls, got %d", len(rateLimit.calls))
	}
	for _, call := range rateLimit.calls {
		if !strings.HasPrefix(call.userID, "hash:") {
			t.Fatalf("expected hashed rate limit identifier, got %s", call.userID)
		}
		if strings.Contains(call.userID, "ABCDE-FGHJK") || strings.Contains(call.userID, "203.0.113.10") {
			t.Fatalf("rate limit identifier exposed raw approve input: %s", call.userID)
		}
		if call.bucket != "extension_pairing_approve:202605261234" {
			t.Fatalf("unexpected rate limit bucket: %s", call.bucket)
		}
		if call.limit != extensionPairingLimit {
			t.Fatalf("unexpected rate limit: %d", call.limit)
		}
	}
	if rateLimit.calls[0].userID == rateLimit.calls[1].userID {
		t.Fatal("expected separate code and client rate limit identifiers")
	}
}

func TestExtensionPairingApproveRejectsEmptyClientIdentifier(t *testing.T) {
	now := time.Date(2026, 5, 26, 12, 34, 0, 0, time.UTC)
	repo := newFakeExtensionPairingRepository()
	rateLimit := &fakeExtensionRateLimitRepository{}
	uc := newExtensionUsecaseForTest(repo, func() time.Time { return now }, bytes.NewReader(make([]byte, 128)), rateLimit)

	err := uc.ApprovePairing(context.Background(), uuid.New(), "ABCDE-FGHJK", " ")
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected invalid input, got %v", err)
	}
	if len(rateLimit.calls) != 0 {
		t.Fatalf("empty client identifier must not skip into rate limit/repo flow, got %d calls", len(rateLimit.calls))
	}
}

func TestApprovedExtensionPairingReturnsTokenOnce(t *testing.T) {
	now := time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC)
	repo := newFakeExtensionPairingRepository()
	uc := newExtensionUsecaseForTest(repo, func() time.Time { return now }, bytes.NewReader(make([]byte, 128)))
	userID := uuid.New()

	pairing, err := uc.StartPairing(context.Background())
	if err != nil {
		t.Fatalf("start pairing: %v", err)
	}
	if err := uc.ApprovePairing(context.Background(), userID, pairing.UserCode, "203.0.113.10"); err != nil {
		t.Fatalf("approve pairing: %v", err)
	}

	result, err := uc.ClaimPairing(context.Background(), pairing.ID, "203.0.113.10")
	if err != nil {
		t.Fatalf("consume pairing: %v", err)
	}
	if !domain.IsExtensionRawToken(result.RawToken) {
		t.Fatalf("expected ext_ token, got %q", result.RawToken)
	}
	if !domain.HasScope(result.Scopes, domain.ExtensionScopeHighlightWrite) {
		t.Fatalf("expected highlight write scope, got %#v", result.Scopes)
	}

	second, err := uc.ClaimPairing(context.Background(), pairing.ID, "203.0.113.10")
	if !errors.Is(err, domain.ErrAlreadyExists) {
		t.Fatalf("expected already used pairing, got result=%#v err=%v", second, err)
	}
}

func TestExtensionTokenStoresHashOnly(t *testing.T) {
	now := time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC)
	repo := newFakeExtensionPairingRepository()
	uc := newExtensionUsecaseForTest(repo, func() time.Time { return now }, bytes.NewReader(make([]byte, 128)))

	pairing, err := uc.StartPairing(context.Background())
	if err != nil {
		t.Fatalf("start pairing: %v", err)
	}
	if err := uc.ApprovePairing(context.Background(), uuid.New(), pairing.UserCode, "203.0.113.10"); err != nil {
		t.Fatalf("approve pairing: %v", err)
	}
	result, err := uc.ClaimPairing(context.Background(), pairing.ID, "203.0.113.10")
	if err != nil {
		t.Fatalf("consume pairing: %v", err)
	}

	if len(repo.hashes) != 1 {
		t.Fatalf("expected one stored hash, got %d", len(repo.hashes))
	}
	if repo.hashes[0] == result.RawToken {
		t.Fatal("raw extension token must not be stored")
	}
	if repo.hashes[0] != domain.HashExtensionToken(result.RawToken) {
		t.Fatalf("unexpected stored hash: %s", repo.hashes[0])
	}
}

func TestRevokeSelfRejectsOtherUsersExtensionToken(t *testing.T) {
	now := time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC)
	repo := newFakeExtensionPairingRepository()
	ownerID := uuid.New()
	otherID := uuid.New()
	tokenID := uuid.New()
	repo.tokens[tokenID] = &domain.ExtensionToken{
		ID:     tokenID,
		UserID: ownerID,
	}
	uc := newExtensionUsecaseForTest(repo, func() time.Time { return now }, bytes.NewReader(make([]byte, 128)))

	err := uc.RevokeSelf(context.Background(), otherID, tokenID)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for another user's token, got %v", err)
	}
	if repo.tokens[tokenID].RevokedAt != nil {
		t.Fatal("another user's extension token must not be revoked")
	}
}

func cloneExtensionPairing(pairing *domain.ExtensionPairing) *domain.ExtensionPairing {
	copied := *pairing
	copied.Scopes = append([]string(nil), pairing.Scopes...)
	return &copied
}
