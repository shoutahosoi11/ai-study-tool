package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

type HighlightUsecase struct {
	repo domain.HighlightRepository
}

func NewHighlightUsecase(repo domain.HighlightRepository) *HighlightUsecase {
	return &HighlightUsecase{repo: repo}
}

type ImportHighlightItem struct {
	ASIN          string
	BookTitle     string
	BookAuthor    string
	Content       string
	Location      string
	HighlightedAt *time.Time
}

type ImportSharedHighlightInput struct {
	BookTitle  string
	BookAuthor string
	Content    string
	SourceApp  string
	SourceURL  string
	SharedAt   *time.Time
}

type ImportKindleResult struct {
	Saved              int
	DuplicateCount     int
	CopyProtectedCount int
	ResolvedASIN       string
	Highlights         []*domain.Highlight
	Warning            *string
}

type ImportSharedHighlightResult struct {
	Saved     bool
	Duplicate bool
	Highlight *domain.Highlight
}

func (u *HighlightUsecase) ImportKindleHighlights(ctx context.Context, userID uuid.UUID, items []ImportHighlightItem) (*ImportKindleResult, error) {
	if len(items) == 0 {
		return nil, domain.ErrAllCopyProtected
	}

	highlights, copyProtectedCount := buildImportHighlights(userID, items)
	if len(highlights) == 0 {
		return nil, domain.ErrAllCopyProtected
	}

	saved, err := u.repo.BulkUpsert(ctx, highlights)
	if err != nil {
		return nil, fmt.Errorf("highlight usecase: import kindle highlights: %w", err)
	}
	if saved > len(highlights) {
		return nil, fmt.Errorf("highlight usecase: import kindle highlights: invalid saved count %d", saved)
	}

	duplicateCount := len(highlights) - saved

	result := &ImportKindleResult{
		Saved:              saved,
		DuplicateCount:     duplicateCount,
		CopyProtectedCount: copyProtectedCount,
		ResolvedASIN:       resolveImportASIN(highlights),
		Highlights:         collectPersistedHighlights(highlights),
	}
	if copyProtectedCount > 0 {
		warning := "コピー制限により一部のハイライトが読み込めませんでした"
		result.Warning = &warning
	}

	return result, nil
}

func (u *HighlightUsecase) ImportSharedHighlight(ctx context.Context, userID uuid.UUID, input ImportSharedHighlightInput) (*ImportSharedHighlightResult, error) {
	highlight, err := newSharedHighlight(userID, input)
	if err != nil {
		return nil, err
	}

	saved, err := u.repo.BulkUpsert(ctx, []*domain.Highlight{highlight})
	if err != nil {
		return nil, fmt.Errorf("highlight usecase: import shared highlight: %w", err)
	}
	if saved < 0 || saved > 1 {
		return nil, fmt.Errorf("highlight usecase: import shared highlight: invalid saved count %d", saved)
	}

	result := &ImportSharedHighlightResult{
		Saved:     saved == 1,
		Duplicate: saved == 0,
	}
	if result.Saved {
		result.Highlight = highlight
	}

	return result, nil
}

func (u *HighlightUsecase) ListExistingContentHashes(ctx context.Context, userID uuid.UUID, hashes []string) ([]string, error) {
	if len(hashes) == 0 {
		return make([]string, 0), nil
	}

	normalized := normalizeHashList(hashes)
	if len(normalized) == 0 {
		return make([]string, 0), nil
	}

	existing, err := u.repo.ListExistingContentHashesByUserID(ctx, userID, normalized)
	if err != nil {
		return nil, fmt.Errorf("highlight usecase: list existing content hashes: %w", err)
	}
	if existing == nil {
		return make([]string, 0), nil
	}

	return existing, nil
}

func (u *HighlightUsecase) ListKindleBooks(ctx context.Context, userID uuid.UUID) ([]*domain.KindleBook, error) {
	books, err := u.repo.ListBooksWithHighlightsByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("highlight usecase: list kindle books: %w", err)
	}
	if books == nil {
		return make([]*domain.KindleBook, 0), nil
	}

	return books, nil
}

func (u *HighlightUsecase) ListByASIN(ctx context.Context, userID uuid.UUID, asin string) ([]*domain.Highlight, error) {
	highlights, err := u.repo.ListByUserIDAndASIN(ctx, userID, asin)
	if err != nil {
		return nil, fmt.Errorf("highlight usecase: list by asin: %w", err)
	}
	if highlights == nil {
		return make([]*domain.Highlight, 0), nil
	}

	return highlights, nil
}

func (u *HighlightUsecase) ListByBookMetadata(ctx context.Context, userID uuid.UUID, bookTitle, bookAuthor string) ([]*domain.Highlight, error) {
	highlights, err := u.repo.ListByUserIDAndBookMetadata(ctx, userID, bookTitle, bookAuthor)
	if err != nil {
		return nil, fmt.Errorf("highlight usecase: list by book metadata: %w", err)
	}
	if highlights == nil {
		return make([]*domain.Highlight, 0), nil
	}

	return highlights, nil
}

func (u *HighlightUsecase) UpdateExplanation(ctx context.Context, id, userID uuid.UUID, explanation string) (*domain.Highlight, error) {
	normalizedExplanation := optionalString(explanation)

	highlight, err := u.repo.UpdateExplanation(ctx, id, userID, normalizedExplanation)
	if err != nil {
		return nil, fmt.Errorf("highlight usecase: update explanation: %w", err)
	}

	return highlight, nil
}

func buildImportHighlights(userID uuid.UUID, items []ImportHighlightItem) ([]*domain.Highlight, int) {
	highlights := make([]*domain.Highlight, 0, len(items))
	copyProtectedCount := 0

	for _, item := range items {
		highlight, ok := newImportHighlight(userID, item)
		if !ok {
			copyProtectedCount++
			continue
		}

		highlights = append(highlights, highlight)
	}

	return highlights, copyProtectedCount
}

func newImportHighlight(userID uuid.UUID, item ImportHighlightItem) (*domain.Highlight, bool) {
	content := strings.TrimSpace(item.Content)
	if content == "" {
		return nil, false
	}

	asin := strings.TrimSpace(item.ASIN)
	location := strings.TrimSpace(item.Location)
	contentHash := computeKindleContentHash(asin, location, content)
	highlightedAt := sanitizeHighlightedAt(item.HighlightedAt)

	return &domain.Highlight{
		UserID:        userID,
		BookTitle:     optionalString(item.BookTitle),
		BookAuthor:    optionalString(item.BookAuthor),
		ASIN:          optionalString(asin),
		Content:       content,
		ContentHash:   &contentHash,
		Location:      optionalString(location),
		HighlightedAt: highlightedAt,
		Source:        domain.HighlightSourceKindle,
		Status:        domain.HighlightStatusPending,
		RequestedAt:   time.Now().UTC(),
	}, true
}

func newSharedHighlight(userID uuid.UUID, input ImportSharedHighlightInput) (*domain.Highlight, error) {
	content := strings.TrimSpace(input.Content)
	if content == "" {
		return nil, domain.ErrInvalidInput
	}

	sourceApp := strings.TrimSpace(input.SourceApp)
	sourceURL := strings.TrimSpace(input.SourceURL)
	bookTitle := strings.TrimSpace(input.BookTitle)
	bookAuthor := strings.TrimSpace(input.BookAuthor)
	contentHash := computeMobileShareContentHash(sourceApp, sourceURL, bookTitle, bookAuthor, content)

	return &domain.Highlight{
		UserID:        userID,
		BookTitle:     optionalString(bookTitle),
		BookAuthor:    optionalString(bookAuthor),
		Content:       content,
		ContentHash:   &contentHash,
		HighlightedAt: sanitizeHighlightedAt(input.SharedAt),
		Source:        domain.HighlightSourceMobileShare,
		SourceApp:     optionalString(sourceApp),
		SourceURL:     optionalString(sourceURL),
		Status:        domain.HighlightStatusPending,
		RequestedAt:   time.Now().UTC(),
	}, nil
}

func computeKindleContentHash(asin, location, content string) string {
	key := fmt.Sprintf(
		"source:%s:asin:%s:loc:%s:content:%s",
		domain.HighlightSourceKindle,
		strings.TrimSpace(asin),
		strings.TrimSpace(location),
		normalizeContent(content),
	)
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

func computeMobileShareContentHash(sourceApp, sourceURL, bookTitle, bookAuthor, content string) string {
	key := fmt.Sprintf(
		"source:%s:app:%s:url:%s:title:%s:author:%s:content:%s",
		domain.HighlightSourceMobileShare,
		strings.TrimSpace(sourceApp),
		strings.TrimSpace(sourceURL),
		strings.TrimSpace(bookTitle),
		strings.TrimSpace(bookAuthor),
		normalizeContent(content),
	)
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

func normalizeContent(content string) string {
	return strings.Join(strings.Fields(strings.ToLower(content)), " ")
}

func normalizeHashList(hashes []string) []string {
	seen := make(map[string]struct{}, len(hashes))
	items := make([]string, 0, len(hashes))
	for _, hash := range hashes {
		normalized := strings.TrimSpace(hash)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		items = append(items, normalized)
	}
	return items
}

func sanitizeHighlightedAt(highlightedAt *time.Time) *time.Time {
	if highlightedAt == nil {
		return nil
	}
	if highlightedAt.After(time.Now()) {
		return nil
	}

	sanitized := *highlightedAt
	return &sanitized
}

func optionalString(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}

	return &trimmed
}

func collectPersistedHighlights(highlights []*domain.Highlight) []*domain.Highlight {
	persisted := make([]*domain.Highlight, 0, len(highlights))
	for _, highlight := range highlights {
		if highlight == nil || highlight.ID == uuid.Nil {
			continue
		}

		persisted = append(persisted, highlight)
	}

	return persisted
}

func resolveImportASIN(highlights []*domain.Highlight) string {
	for _, highlight := range highlights {
		if highlight == nil || highlight.ASIN == nil {
			continue
		}

		asin := strings.TrimSpace(*highlight.ASIN)
		if asin != "" {
			return asin
		}
	}

	return ""
}
