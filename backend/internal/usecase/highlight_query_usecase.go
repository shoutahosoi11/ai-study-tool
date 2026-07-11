package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

// HighlightQueryUsecase はハイライトの参照系（一覧・ハッシュ照会・解説更新）を
// 担当する。インポート系は HighlightImportUsecase に分離されている。
type HighlightQueryUsecase struct {
	repo domain.HighlightImportRepository
}

func NewHighlightQueryUsecase(repo domain.HighlightImportRepository) *HighlightQueryUsecase {
	return &HighlightQueryUsecase{repo: repo}
}

func (u *HighlightQueryUsecase) ListExistingContentHashes(ctx context.Context, userID uuid.UUID, hashes []string) ([]string, error) {
	if len(hashes) == 0 {
		return make([]string, 0), nil
	}

	normalized, err := normalizeHashList(hashes)
	if err != nil {
		return nil, err
	}
	if len(normalized) == 0 {
		return make([]string, 0), nil
	}

	existing, err := u.repo.ListExistingContentHashesByUserID(ctx, userID, normalized)
	if err != nil {
		return nil, fmt.Errorf("highlight query usecase: list existing content hashes: %w", err)
	}
	if existing == nil {
		return make([]string, 0), nil
	}

	return existing, nil
}

func (u *HighlightQueryUsecase) ListKindleBooks(ctx context.Context, userID uuid.UUID) ([]*domain.KindleBook, error) {
	books, err := u.repo.ListBooksWithHighlightsByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("highlight query usecase: list kindle books: %w", err)
	}
	if books == nil {
		return make([]*domain.KindleBook, 0), nil
	}

	return books, nil
}

func (u *HighlightQueryUsecase) ListByASIN(ctx context.Context, userID uuid.UUID, asin string) ([]*domain.Highlight, error) {
	highlights, err := u.repo.ListByUserIDAndASIN(ctx, userID, asin)
	if err != nil {
		return nil, fmt.Errorf("highlight query usecase: list by asin: %w", err)
	}
	if highlights == nil {
		return make([]*domain.Highlight, 0), nil
	}

	return highlights, nil
}

func (u *HighlightQueryUsecase) ListByBookMetadata(ctx context.Context, userID uuid.UUID, bookTitle, bookAuthor string) ([]*domain.Highlight, error) {
	highlights, err := u.repo.ListByUserIDAndBookMetadata(ctx, userID, bookTitle, bookAuthor)
	if err != nil {
		return nil, fmt.Errorf("highlight query usecase: list by book metadata: %w", err)
	}
	if highlights == nil {
		return make([]*domain.Highlight, 0), nil
	}

	return highlights, nil
}

func (u *HighlightQueryUsecase) UpdateExplanation(ctx context.Context, id, userID uuid.UUID, explanation string) (*domain.Highlight, error) {
	normalizedExplanation := optionalString(explanation)

	highlight, err := u.repo.UpdateExplanation(ctx, id, userID, normalizedExplanation)
	if err != nil {
		return nil, fmt.Errorf("highlight query usecase: update explanation: %w", err)
	}

	return highlight, nil
}


func normalizeHashList(hashes []string) ([]string, error) {
	if len(hashes) > maxHashCheckItems {
		return nil, fmt.Errorf("%w: hashes must be at most %d items", domain.ErrInvalidInput, maxHashCheckItems)
	}

	seen := make(map[string]struct{}, len(hashes))
	items := make([]string, 0, len(hashes))
	for _, hash := range hashes {
		normalized := strings.ToLower(strings.TrimSpace(hash))
		if normalized == "" {
			continue
		}
		if !isSHA256Hex(normalized) {
			return nil, fmt.Errorf("%w: invalid content hash", domain.ErrInvalidInput)
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		items = append(items, normalized)
	}
	return items, nil
}

func isSHA256Hex(value string) bool {
	if len(value) != sha256HexLength {
		return false
	}
	for _, r := range value {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}
