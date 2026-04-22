package domain

import (
	"time"

	"github.com/google/uuid"
)

const (
	HighlightSourceKindle = "kindle"
)

type Highlight struct {
	ID            uuid.UUID
	UserID        uuid.UUID
	BookID        *uuid.UUID
	BookTitle     *string
	BookAuthor    *string
	ASIN          *string
	Content       string
	Explanation   *string
	ContentHash   *string
	Location      *string
	HighlightedAt *time.Time
	Source        string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
