package dto

import "time"

type ImportHighlightItem struct {
	ASIN          string     `json:"asin"`
	BookTitle     string     `json:"book_title"`
	BookAuthor    string     `json:"book_author"`
	Content       string     `json:"content"`
	Location      string     `json:"location"`
	HighlightedAt *time.Time `json:"highlighted_at"`
}

type ImportHighlightsRequest struct {
	Highlights []ImportHighlightItem `json:"highlights"`
}

type UpdateHighlightExplanationRequest struct {
	Explanation string `json:"explanation"`
}

type HighlightResponse struct {
	ID            string     `json:"id"`
	BookID        *string    `json:"book_id,omitempty"`
	BookTitle     *string    `json:"book_title,omitempty"`
	BookAuthor    *string    `json:"book_author,omitempty"`
	ASIN          *string    `json:"asin,omitempty"`
	Content       string     `json:"content"`
	Explanation   *string    `json:"explanation,omitempty"`
	Location      *string    `json:"location,omitempty"`
	HighlightedAt *time.Time `json:"highlighted_at,omitempty"`
	Source        string     `json:"source"`
	CreatedAt     time.Time  `json:"created_at"`
}

type ImportHighlightsResponse struct {
	SavedCount         int                  `json:"saved_count"`
	DuplicateCount     int                  `json:"duplicate_count"`
	CopyProtectedCount int                  `json:"copy_protected_count"`
	ResolvedASIN       string               `json:"resolved_asin"`
	Highlights         []*HighlightResponse `json:"highlights"`
	Warning            *string              `json:"warning,omitempty"`
}

type ListBookHighlightsResponse struct {
	Highlights []*HighlightResponse `json:"highlights"`
}

type KindleBookResponse struct {
	ASIN           string `json:"asin"`
	BookTitle      string `json:"book_title"`
	BookAuthor     string `json:"book_author"`
	HighlightCount int    `json:"highlight_count"`
	Source         string `json:"source"`
}

type ListKindleBooksResponse struct {
	Books []*KindleBookResponse `json:"books"`
}
