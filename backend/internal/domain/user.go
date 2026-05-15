package domain

import (
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

const (
	DefaultQuestionCountAll     int16 = 0
	DefaultQuestionCountDefault int16 = 3
	DefaultQuestionCountMax     int16 = 10
	UsernameMinLength           int   = 3
	UsernameMaxLength           int   = 50
)

var usernamePattern = regexp.MustCompile(`^[a-z0-9_]+$`)

type User struct {
	ID                   uuid.UUID
	FirebaseUID          string
	Username             string
	DisplayName          string
	AvatarURL            *string
	Bio                  *string
	University           *string
	Faculty              *string
	Grade                *int16
	Country              *string
	Plan                 string
	DefaultQuestionCount int16
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type CreateUserInput struct {
	FirebaseUID string
	Username    string
	DisplayName string
	AvatarURL   *string
	Bio         *string
	University  *string
	Faculty     *string
	Grade       *int16
	Country     *string
}

type UpdateUserInput struct {
	Username    OptionalStringUpdate
	DisplayName OptionalStringUpdate
	AvatarURL   OptionalStringUpdate
	Bio         OptionalStringUpdate
	University  OptionalStringUpdate
	Faculty     OptionalStringUpdate
	Grade       OptionalInt16Update
	Country     OptionalStringUpdate
}

type UpdateQuestionSettingsInput struct {
	DefaultQuestionCount int16
}

type OptionalStringUpdate struct {
	Set   bool
	Value *string
}

type OptionalInt16Update struct {
	Set   bool
	Value *int16
}

func SomeStringUpdate(value string) OptionalStringUpdate {
	return OptionalStringUpdate{Set: true, Value: &value}
}

func NullStringUpdate() OptionalStringUpdate {
	return OptionalStringUpdate{Set: true}
}

func SomeInt16Update(value int16) OptionalInt16Update {
	return OptionalInt16Update{Set: true, Value: &value}
}

func NullInt16Update() OptionalInt16Update {
	return OptionalInt16Update{Set: true}
}

func (input UpdateUserInput) HasChanges() bool {
	return input.Username.Set ||
		input.DisplayName.Set ||
		input.AvatarURL.Set ||
		input.Bio.Set ||
		input.University.Set ||
		input.Faculty.Set ||
		input.Grade.Set ||
		input.Country.Set
}

func IsValidDefaultQuestionCount(count int16) bool {
	if count == DefaultQuestionCountAll {
		return true
	}
	return count >= 1 && count <= DefaultQuestionCountMax
}

func NormalizeUsername(username string) string {
	normalized := strings.TrimSpace(username)
	normalized = strings.TrimPrefix(normalized, "@")
	return strings.ToLower(normalized)
}

func IsValidUsername(username string) bool {
	length := utf8.RuneCountInString(username)
	if length < UsernameMinLength || length > UsernameMaxLength {
		return false
	}
	return usernamePattern.MatchString(username)
}
