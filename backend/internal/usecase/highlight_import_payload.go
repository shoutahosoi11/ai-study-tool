package usecase

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

const highlightImportPayloadVersion = 1

type highlightImportPayload struct {
	Version    int                          `json:"version"`
	Highlights []highlightImportPayloadItem `json:"highlights"`
}

type highlightImportPayloadItem struct {
	BookID        *uuid.UUID             `json:"book_id,omitempty"`
	BookTitle     *string                `json:"book_title,omitempty"`
	BookAuthor    *string                `json:"book_author,omitempty"`
	BookKey       string                 `json:"book_key"`
	ASIN          *string                `json:"asin,omitempty"`
	Content       string                 `json:"content"`
	Explanation   *string                `json:"explanation,omitempty"`
	ContentHash   *string                `json:"content_hash,omitempty"`
	Location      *string                `json:"location,omitempty"`
	HighlightedAt *time.Time             `json:"highlighted_at,omitempty"`
	Source        string                 `json:"source"`
	SourceApp     *string                `json:"source_app,omitempty"`
	SourceURL     *string                `json:"source_url,omitempty"`
	Status        domain.HighlightStatus `json:"status"`
}

func marshalHighlightImportPayload(highlights []*domain.Highlight) ([]byte, error) {
	payload := highlightImportPayload{
		Version:    highlightImportPayloadVersion,
		Highlights: make([]highlightImportPayloadItem, 0, len(highlights)),
	}
	for _, highlight := range highlights {
		if highlight == nil {
			continue
		}
		payload.Highlights = append(payload.Highlights, highlightImportPayloadItem{
			BookID:        highlight.BookID,
			BookTitle:     highlight.BookTitle,
			BookAuthor:    highlight.BookAuthor,
			BookKey:       highlight.BookKey,
			ASIN:          highlight.ASIN,
			Content:       highlight.Content,
			Explanation:   highlight.Explanation,
			ContentHash:   highlight.ContentHash,
			Location:      highlight.Location,
			HighlightedAt: highlight.HighlightedAt,
			Source:        highlight.Source,
			SourceApp:     highlight.SourceApp,
			SourceURL:     highlight.SourceURL,
			Status:        highlight.Status,
		})
	}
	return json.Marshal(payload)
}

func unmarshalHighlightImportPayload(raw []byte, userID uuid.UUID) ([]*domain.Highlight, error) {
	var payload highlightImportPayload
	if err := json.Unmarshal(raw, &payload); err == nil && (payload.Version != 0 || payload.Highlights != nil) {
		if payload.Version != highlightImportPayloadVersion {
			return nil, fmt.Errorf("unsupported highlight import payload version %d", payload.Version)
		}
		return payload.toDomainHighlights(userID), nil
	}

	// Compatibility for queue rows created before the payload was versioned.
	var legacy []*domain.Highlight
	if err := json.Unmarshal(raw, &legacy); err != nil {
		return nil, fmt.Errorf("unmarshal highlights: %w", err)
	}
	for i := range legacy {
		if legacy[i] != nil {
			legacy[i].UserID = userID
		}
	}
	return legacy, nil
}

func (p highlightImportPayload) toDomainHighlights(userID uuid.UUID) []*domain.Highlight {
	highlights := make([]*domain.Highlight, 0, len(p.Highlights))
	for _, item := range p.Highlights {
		highlights = append(highlights, &domain.Highlight{
			UserID:        userID,
			BookID:        item.BookID,
			BookTitle:     item.BookTitle,
			BookAuthor:    item.BookAuthor,
			BookKey:       item.BookKey,
			ASIN:          item.ASIN,
			Content:       item.Content,
			Explanation:   item.Explanation,
			ContentHash:   item.ContentHash,
			Location:      item.Location,
			HighlightedAt: item.HighlightedAt,
			Source:        item.Source,
			SourceApp:     item.SourceApp,
			SourceURL:     item.SourceURL,
			Status:        item.Status,
		})
	}
	return highlights
}
