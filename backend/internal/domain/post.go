package domain

import (
	"time"

	"github.com/google/uuid"
)

type Post struct {
	ID            uuid.UUID  `json:"id"`
	UserID        uuid.UUID  `json:"user_id"`
	QuestionID    *uuid.UUID `json:"question_id,omitempty"`
	BookID        *uuid.UUID `json:"book_id,omitempty"`
	FieldID       *uuid.UUID `json:"field_id,omitempty"`
	Body          *string    `json:"body,omitempty"`
	BookTitle     *string    `json:"book_title,omitempty"`
	QuestionCount int        `json:"question_count"`
	Type          string     `json:"type"`
	RepostCount   int        `json:"repost_count"`
	LikeCount     int        `json:"like_count"`
	CommentCount  int        `json:"comment_count"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type TimelinePost struct {
	Post
	Score       int     `json:"score"`
	Username    string  `json:"username"`
	DisplayName string  `json:"display_name"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
	FieldName   *string `json:"field_name,omitempty"`
}

type PostQuestionItem struct {
	QuestionID uuid.UUID `json:"question_id"`
	SortOrder  int       `json:"sort_order"`
	Note       string    `json:"note"`
}

type PostedQuestion struct {
	Question
	Note      string `json:"note"`
	SortOrder int    `json:"sort_order"`
}

type CreatePostInput struct {
	UserID        uuid.UUID          `json:"user_id"`
	QuestionID    *uuid.UUID         `json:"question_id"`
	BookID        *uuid.UUID         `json:"book_id"`
	FieldID       *uuid.UUID         `json:"field_id"`
	Body          string             `json:"body"`
	BookTitle     string             `json:"book_title"`
	QuestionCount int                `json:"question_count"`
	Questions     []PostQuestionItem `json:"questions"`
	Type          string             `json:"type"`
}

type TimelineParams struct {
	UserID uuid.UUID `json:"user_id"`
	Limit  int       `json:"limit"`
	Offset int       `json:"offset"`
}
