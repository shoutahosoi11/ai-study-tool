package domain

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID          uuid.UUID `json:"id"`
	FirebaseUID string    `json:"firebase_uid"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
	AvatarURL   *string   `json:"avatar_url,omitempty"`
	Bio         *string   `json:"bio,omitempty"`
	University  *string   `json:"university,omitempty"`
	Faculty     *string   `json:"faculty,omitempty"`
	Grade       *int16    `json:"grade,omitempty"`
	Country     *string   `json:"country,omitempty"`
	Plan        string    `json:"plan"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreateUserInput struct {
	FirebaseUID string  `json:"firebase_uid"`
	Username    string  `json:"username"`
	DisplayName string  `json:"display_name"`
	AvatarURL   *string `json:"avatar_url"`
	Bio         *string `json:"bio"`
	University  *string `json:"university"`
	Faculty     *string `json:"faculty"`
	Grade       *int16  `json:"grade"`
	Country     *string `json:"country"`
}

type UpdateUserInput struct {
	Username    string  `json:"username"`
	DisplayName string  `json:"display_name"`
	AvatarURL   *string `json:"avatar_url"`
	Bio         *string `json:"bio"`
	University  *string `json:"university"`
	Faculty     *string `json:"faculty"`
	Grade       *int16  `json:"grade"`
	Country     *string `json:"country"`
}
