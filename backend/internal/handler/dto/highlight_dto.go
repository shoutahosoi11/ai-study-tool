package dto

import "time"

type CreateHighlightRequest struct {
	BookID        *string    `json:"book_id"`
	BookTitle     *string    `json:"book_title"`
	BookAuthor    *string    `json:"book_author"`
	ASIN          *string    `json:"asin"`
	Content       string     `json:"content"`
	Location      *string    `json:"location"`
	HighlightedAt *time.Time `json:"highlighted_at"`
	Source        string     `json:"source"`
}

type HighlightResponse struct {
	ID            string     `json:"id"`
	BookID        *string    `json:"book_id,omitempty"`
	BookTitle     *string    `json:"book_title,omitempty"`
	BookAuthor    *string    `json:"book_author,omitempty"`
	ASIN          *string    `json:"asin,omitempty"`
	Content       string     `json:"content"`
	Location      *string    `json:"location,omitempty"`
	HighlightedAt *time.Time `json:"highlighted_at,omitempty"`
	Source        string     `json:"source"`
	CreatedAt     time.Time  `json:"created_at"`
}

type ListHighlightsResponse struct {
	Highlights []*HighlightResponse `json:"highlights"`
	Total      int64                `json:"total"`
	Page       int                  `json:"page"`
	Limit      int                  `json:"limit"`
}
