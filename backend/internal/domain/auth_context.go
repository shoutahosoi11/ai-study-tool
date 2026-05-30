package domain

import "github.com/google/uuid"

type AuthClientType string

const (
	AuthClientTypeWeb       AuthClientType = "web"
	AuthClientTypeMobile    AuthClientType = "mobile"
	AuthClientTypeExtension AuthClientType = "extension"
)

type AuthenticatedUser struct {
	FirebaseUID string
	UserID      *uuid.UUID
}
