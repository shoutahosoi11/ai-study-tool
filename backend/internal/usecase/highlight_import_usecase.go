package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

const (
	maxKindleImportItems = 1000
	maxHashCheckItems    = 2000
	maxSourceAppLength   = 100
	sha256HexLength      = 64
)

type HighlightImportUsecase struct {
	repo        domain.HighlightImportRepository
	importQueue domain.HighlightImportQueueRepository
	jobTrigger  domain.HighlightImportJobTrigger
}

func NewHighlightImportUsecase(repo domain.HighlightImportRepository) *HighlightImportUsecase {
	return &HighlightImportUsecase{repo: repo}
}

func NewHighlightImportUsecaseWithQueue(
	repo domain.HighlightImportRepository,
	importQueue domain.HighlightImportQueueRepository,
	jobTrigger domain.HighlightImportJobTrigger,
) *HighlightImportUsecase {
	return &HighlightImportUsecase{
		repo:        repo,
		importQueue: importQueue,
		jobTrigger:  jobTrigger,
	}
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

type ImportPastedHighlightInput struct {
	BookTitle  string
	BookAuthor string
	Content    string
	SourceApp  string
	SourceURL  string
}

type ImportKindleResult struct {
	// キューモード: QueueID と QueuedCount が設定される
	QueueID     uuid.UUID
	QueuedCount int
	// 同期モード: Saved 以下が設定される
	Saved          int
	DuplicateCount int
	// 共通
	CopyProtectedCount int
	ResolvedASIN       string
	Highlights         []*domain.Highlight
	Warning            *string
	InvalidItemCount   int
}

type ImportSharedHighlightResult struct {
	Saved     bool
	Duplicate bool
	Highlight *domain.Highlight
}

type ImportPastedHighlightResult struct {
	ID        uuid.UUID
	Duplicate bool
	Highlight *domain.Highlight
}

func (u *HighlightImportUsecase) ImportKindleHighlights(ctx context.Context, userID uuid.UUID, items []ImportHighlightItem) (*ImportKindleResult, error) {
	if len(items) == 0 {
		return nil, domain.NewValidationError("highlights must not be empty")
	}
	if len(items) > maxKindleImportItems {
		return nil, domain.NewValidationError(fmt.Sprintf("highlights must be at most %d items", maxKindleImportItems))
	}

	// キューが設定されている場合は非同期処理に委譲する
	if u.importQueue != nil {
		return u.enqueueKindleImport(ctx, userID, items)
	}

	// キューなし（後方互換: NewHighlightImportUsecase 経由）
	return u.importKindleHighlightsDirect(ctx, userID, items)
}

func (u *HighlightImportUsecase) enqueueKindleImport(ctx context.Context, userID uuid.UUID, items []ImportHighlightItem) (*ImportKindleResult, error) {
	u.recoverFailedImportEnqueues(ctx, userID)

	// 既存の検証・正規化ロジックを使いコピープロテクトを除外する
	highlights, copyProtectedCount, invalidItemCount, err := buildImportHighlights(userID, items)
	if err != nil {
		return nil, err
	}
	if len(highlights) == 0 {
		if invalidItemCount > 0 {
			return nil, domain.ErrInvalidInput
		}
		return nil, domain.ErrAllCopyProtected
	}

	// 検証済み highlights を安定した queue payload に変換して積む。
	payload, err := marshalHighlightImportPayload(highlights)
	if err != nil {
		return nil, fmt.Errorf("highlight usecase: marshal kindle items: %w", err)
	}

	queueID, err := u.importQueue.Enqueue(ctx, userID, domain.ImportQueueSourceKindle, payload)
	if err != nil {
		return nil, fmt.Errorf("highlight usecase: enqueue kindle import: %w", err)
	}

	if u.jobTrigger != nil {
		if triggerErr := u.jobTrigger.TriggerHighlightImportJob(ctx, queueID, userID); triggerErr != nil {
			if markErr := u.importQueue.MarkEnqueueFailed(ctx, queueID, fmt.Sprintf("trigger import task: %v", triggerErr)); markErr != nil {
				slog.Error("highlight_import_queue_mark_enqueue_failed_error", "queue_id", queueID.String(), "error", markErr)
			}
			return nil, fmt.Errorf("highlight usecase: trigger kindle import task: %w", triggerErr)
		}
	}

	result := &ImportKindleResult{
		QueueID:            queueID,
		QueuedCount:        len(highlights),
		CopyProtectedCount: copyProtectedCount,
		InvalidItemCount:   invalidItemCount,
	}
	if copyProtectedCount > 0 || invalidItemCount > 0 {
		warning := importWarning(copyProtectedCount, invalidItemCount)
		result.Warning = &warning
	}
	return result, nil
}

func (u *HighlightImportUsecase) importKindleHighlightsDirect(ctx context.Context, userID uuid.UUID, items []ImportHighlightItem) (*ImportKindleResult, error) {
	highlights, copyProtectedCount, invalidItemCount, err := buildImportHighlights(userID, items)
	if err != nil {
		return nil, err
	}
	if len(highlights) == 0 {
		if invalidItemCount > 0 {
			return nil, domain.ErrInvalidInput
		}
		return nil, domain.ErrAllCopyProtected
	}

	saved, err := u.repo.BulkUpsert(ctx, highlights)
	if err != nil {
		return nil, fmt.Errorf("highlight usecase: import kindle highlights: %w", err)
	}
	if saved > len(highlights) {
		return nil, fmt.Errorf("highlight usecase: import kindle highlights: invalid saved count %d", saved)
	}

	result := &ImportKindleResult{
		Saved:              saved,
		DuplicateCount:     len(highlights) - saved,
		CopyProtectedCount: copyProtectedCount,
		ResolvedASIN:       resolveImportASIN(highlights),
		Highlights:         collectPersistedHighlights(highlights),
		InvalidItemCount:   invalidItemCount,
	}
	if copyProtectedCount > 0 || invalidItemCount > 0 {
		warning := importWarning(copyProtectedCount, invalidItemCount)
		result.Warning = &warning
	}
	return result, nil
}

func (u *HighlightImportUsecase) ImportPastedHighlight(ctx context.Context, userID uuid.UUID, input ImportPastedHighlightInput) (*ImportPastedHighlightResult, error) {
	highlight, err := newPastedHighlight(userID, input)
	if err != nil {
		return nil, err
	}

	saved, err := u.repo.BulkUpsert(ctx, []*domain.Highlight{highlight})
	if err != nil {
		return nil, fmt.Errorf("highlight usecase: import pasted highlight: %w", err)
	}
	if saved < 0 || saved > 1 {
		return nil, fmt.Errorf("highlight usecase: import pasted highlight: invalid saved count %d", saved)
	}
	if saved == 1 {
		return &ImportPastedHighlightResult{
			ID:        highlight.ID,
			Duplicate: false,
			Highlight: highlight,
		}, nil
	}

	existing, err := u.repo.FindByUserIDAndContentHash(ctx, userID, *highlight.ContentHash)
	if err != nil {
		return nil, fmt.Errorf("highlight usecase: find pasted duplicate: %w", err)
	}

	return &ImportPastedHighlightResult{
		ID:        existing.ID,
		Duplicate: true,
		Highlight: existing,
	}, nil
}

func (u *HighlightImportUsecase) ImportSharedHighlight(ctx context.Context, userID uuid.UUID, input ImportSharedHighlightInput) (*ImportSharedHighlightResult, error) {
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

func buildImportHighlights(userID uuid.UUID, items []ImportHighlightItem) ([]*domain.Highlight, int, int, error) {
	highlights := make([]*domain.Highlight, 0, len(items))
	copyProtectedCount := 0
	invalidItemCount := 0

	for index, item := range items {
		highlight, ok, err := newImportHighlight(userID, item)
		if err != nil {
			if errors.Is(err, domain.ErrInvalidInput) {
				invalidItemCount++
				slog.Warn("highlight_import_item_invalid", "index", index, "error", err)
				continue
			}
			return nil, copyProtectedCount, invalidItemCount, err
		}
		if !ok {
			copyProtectedCount++
			continue
		}

		highlights = append(highlights, highlight)
	}

	return highlights, copyProtectedCount, invalidItemCount, nil
}

func newImportHighlight(userID uuid.UUID, item ImportHighlightItem) (*domain.Highlight, bool, error) {
	content := strings.TrimSpace(item.Content)
	if content == "" {
		return nil, false, nil
	}

	normalizedContent, err := normalizeAndValidateHighlightContent(content)
	if err != nil {
		return nil, false, err
	}

	bookTitle, err := normalizeAndValidateOptionalMetaText(item.BookTitle, 200)
	if err != nil {
		return nil, false, err
	}
	bookAuthor, err := normalizeAndValidateOptionalMetaText(item.BookAuthor, 100)
	if err != nil {
		return nil, false, err
	}

	asin := strings.TrimSpace(item.ASIN)
	location := strings.TrimSpace(item.Location)
	contentHash := computeContentHash(normalizedContent)
	highlightedAt := sanitizeHighlightedAt(item.HighlightedAt)
	bookKey := resolveHighlightBookKey(asin, bookTitle, bookAuthor)

	return &domain.Highlight{
		UserID:        userID,
		BookTitle:     bookTitle,
		BookAuthor:    bookAuthor,
		BookKey:       bookKey,
		ASIN:          optionalString(asin),
		Content:       normalizedContent,
		ContentHash:   &contentHash,
		Location:      optionalString(location),
		HighlightedAt: highlightedAt,
		Source:        domain.HighlightSourceExtension,
		Status:        domain.HighlightStatusPending,
		RequestedAt:   time.Now().UTC(),
	}, true, nil
}

func newSharedHighlight(userID uuid.UUID, input ImportSharedHighlightInput) (*domain.Highlight, error) {
	content := strings.TrimSpace(input.Content)
	if content == "" {
		return nil, domain.ErrInvalidInput
	}

	normalizedContent, err := normalizeAndValidateHighlightContent(content)
	if err != nil {
		return nil, err
	}

	sourceApp := strings.ToLower(strings.TrimSpace(input.SourceApp))
	if sourceApp != "" {
		// Share import preserves client-provided app names for diagnostics; unlike paste,
		// it is length-limited rather than allowlisted because mobile share sources vary.
		if err := domain.ValidateRequiredTextLength(sourceApp, maxSourceAppLength); err != nil {
			return nil, err
		}
	}
	sourceURL := strings.TrimSpace(input.SourceURL)
	if sourceURL != "" {
		if err := domain.ValidateSourceURL(sourceURL); err != nil {
			return nil, err
		}
	}

	bookTitle, err := normalizeAndValidateOptionalMetaText(input.BookTitle, 200)
	if err != nil {
		return nil, err
	}
	bookAuthor, err := normalizeAndValidateOptionalMetaText(input.BookAuthor, 100)
	if err != nil {
		return nil, err
	}
	contentHash := computeContentHash(normalizedContent)
	bookKey := resolveHighlightBookKey("", bookTitle, bookAuthor)

	return &domain.Highlight{
		UserID:        userID,
		BookTitle:     bookTitle,
		BookAuthor:    bookAuthor,
		BookKey:       bookKey,
		Content:       normalizedContent,
		ContentHash:   &contentHash,
		HighlightedAt: sanitizeHighlightedAt(input.SharedAt),
		Source:        domain.HighlightSourceShare,
		SourceApp:     optionalString(sourceApp),
		SourceURL:     optionalString(sourceURL),
		Status:        domain.HighlightStatusPending,
		RequestedAt:   time.Now().UTC(),
	}, nil
}

func newPastedHighlight(userID uuid.UUID, input ImportPastedHighlightInput) (*domain.Highlight, error) {
	content := strings.TrimSpace(input.Content)
	if content == "" {
		return nil, domain.ErrInvalidInput
	}

	normalizedContent, err := normalizeAndValidateHighlightContent(content)
	if err != nil {
		return nil, err
	}

	sourceApp := strings.TrimSpace(input.SourceApp)
	if sourceApp != "" && !isAllowedPasteSourceApp(sourceApp) {
		return nil, domain.ErrInvalidInput
	}
	sourceApp = strings.ToLower(sourceApp)

	sourceURL := strings.TrimSpace(input.SourceURL)
	if sourceURL != "" {
		if err := domain.ValidateSourceURL(sourceURL); err != nil {
			return nil, err
		}
	}

	bookTitle, err := normalizeAndValidateOptionalMetaText(input.BookTitle, 200)
	if err != nil {
		return nil, err
	}
	bookAuthor, err := normalizeAndValidateOptionalMetaText(input.BookAuthor, 100)
	if err != nil {
		return nil, err
	}
	contentHash := computeContentHash(normalizedContent)
	bookKey := resolveHighlightBookKey("", bookTitle, bookAuthor)

	return &domain.Highlight{
		UserID:      userID,
		BookTitle:   bookTitle,
		BookAuthor:  bookAuthor,
		BookKey:     bookKey,
		Content:     normalizedContent,
		ContentHash: &contentHash,
		Source:      domain.HighlightSourcePaste,
		SourceApp:   optionalString(sourceApp),
		SourceURL:   optionalString(sourceURL),
		Status:      domain.HighlightStatusPending,
		RequestedAt: time.Now().UTC(),
	}, nil
}

func resolveHighlightBookKey(asin string, bookTitle *string, bookAuthor *string) string {
	normalizedASIN := strings.TrimSpace(asin)
	if normalizedASIN != "" {
		return normalizedASIN
	}

	title := ""
	if bookTitle != nil {
		title = strings.TrimSpace(*bookTitle)
	}
	author := ""
	if bookAuthor != nil {
		author = strings.TrimSpace(*bookAuthor)
	}
	return "metadata:" + title + ":" + author
}

func normalizeAndValidateHighlightContent(content string) (string, error) {
	if err := validateHighlightContent(content); err != nil {
		return "", err
	}

	normalized := domain.NormalizeText(content)
	if err := validateHighlightContent(normalized); err != nil {
		return "", err
	}

	return normalized, nil
}

func validateHighlightContent(content string) error {
	if err := domain.ValidateRequiredTextLength(content, 300); err != nil {
		return err
	}
	if err := domain.ValidateLineCount(content, 20); err != nil {
		return err
	}
	if err := domain.ValidateMaxLineLength(content, 300); err != nil {
		return err
	}

	return nil
}

func normalizeAndValidateOptionalMetaText(value string, max int) (*string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}
	if err := domain.ValidateRequiredTextLength(trimmed, max); err != nil {
		return nil, err
	}

	normalized := domain.NormalizeMetaText(trimmed)
	if err := domain.ValidateRequiredTextLength(normalized, max); err != nil {
		return nil, err
	}

	return &normalized, nil
}

func isAllowedPasteSourceApp(sourceApp string) bool {
	switch strings.ToLower(strings.TrimSpace(sourceApp)) {
	case "kindle", "x", "web":
		return true
	default:
		return false
	}
}

func computeContentHash(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

func sanitizeHighlightedAt(highlightedAt *time.Time) *time.Time {
	if highlightedAt == nil {
		return nil
	}
	if highlightedAt.Before(minHighlightedAt()) || highlightedAt.After(time.Now()) {
		return nil
	}

	sanitized := *highlightedAt
	return &sanitized
}

func minHighlightedAt() time.Time {
	return time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
}

func importWarning(copyProtectedCount, invalidItemCount int) string {
	if copyProtectedCount > 0 && invalidItemCount > 0 {
		return "コピー制限または入力不備により一部のハイライトが読み込めませんでした"
	}
	if invalidItemCount > 0 {
		return "入力不備により一部のハイライトが読み込めませんでした"
	}
	return "コピー制限により一部のハイライトが読み込めませんでした"
}

func (u *HighlightImportUsecase) recoverFailedImportEnqueues(ctx context.Context, userID uuid.UUID) {
	if u.importQueue == nil || u.jobTrigger == nil {
		return
	}

	cutoff := time.Now().UTC().Add(-domain.ImportQueueStaleQueuedTimeout)
	items, err := u.importQueue.ListRecoverableEnqueuesByUserID(ctx, userID, cutoff, domain.ImportQueueRecoverLimit)
	if err != nil {
		slog.Warn("highlight_import_recoverable_list_error", "user_id", userID.String(), "error", err)
		return
	}

	for _, item := range items {
		if item == nil {
			continue
		}
		if item.Status == domain.ImportQueueStatusEnqueueFailed {
			if err := u.importQueue.MarkQueued(ctx, item.ID); err != nil {
				slog.Warn("highlight_import_recoverable_mark_queued_error", "queue_id", item.ID.String(), "error", err)
				continue
			}
		}
		if err := u.jobTrigger.TriggerHighlightImportJob(ctx, item.ID, item.UserID); err != nil {
			if markErr := u.importQueue.MarkEnqueueFailed(ctx, item.ID, fmt.Sprintf("retry trigger import task: %v", err)); markErr != nil {
				slog.Warn("highlight_import_recoverable_mark_enqueue_failed_error", "queue_id", item.ID.String(), "error", markErr)
			}
			continue
		}
		slog.Info("highlight_import_recoverable_enqueued", "queue_id", item.ID.String(), "status", item.Status)
	}
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
