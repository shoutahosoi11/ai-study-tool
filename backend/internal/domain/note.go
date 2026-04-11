package domain

import "time"

type Note struct {
	ID        string
	UserID    string
	Title     string
	FileURL   string
	CreatedAt time.Time
}
