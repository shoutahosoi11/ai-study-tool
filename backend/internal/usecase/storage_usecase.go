package usecase

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

const (
	defaultSignedURLTTLSeconds = 15 * 60
	maxSignedURLTTLSeconds     = 60 * 60
)

type StorageUsecase struct {
	signer domain.StorageSigner
}

func NewStorageUsecase(signer domain.StorageSigner) *StorageUsecase {
	return &StorageUsecase{signer: signer}
}

type CreateUploadSignedURLInput struct {
	UserID           uuid.UUID
	FileName         string
	ContentType      string
	ExpiresInSeconds int
}

type CreateDownloadSignedURLInput struct {
	UserID           uuid.UUID
	ObjectName       string
	ExpiresInSeconds int
}

func (u *StorageUsecase) CreateUploadSignedURL(ctx context.Context, input CreateUploadSignedURLInput) (*domain.SignedURL, error) {
	if u.signer == nil {
		return nil, domain.ErrStorageNotConfigured
	}

	contentType := normalizeContentType(input.ContentType)
	objectName := buildUserUploadObjectName(input.UserID, input.FileName)
	if objectName == "" {
		return nil, domain.ErrInvalidInput
	}

	return u.signer.SignedURL(ctx, domain.SignedURLOptions{
		ObjectName:  objectName,
		Method:      "PUT",
		ContentType: contentType,
		ExpiresAt:   time.Now().Add(resolveSignedURLTTL(input.ExpiresInSeconds)),
	})
}

func (u *StorageUsecase) CreateDownloadSignedURL(ctx context.Context, input CreateDownloadSignedURLInput) (*domain.SignedURL, error) {
	if u.signer == nil {
		return nil, domain.ErrStorageNotConfigured
	}

	objectName := strings.TrimSpace(input.ObjectName)
	if !isUserUploadObject(input.UserID, objectName) {
		return nil, domain.ErrInvalidInput
	}

	return u.signer.SignedURL(ctx, domain.SignedURLOptions{
		ObjectName: objectName,
		Method:     "GET",
		ExpiresAt:  time.Now().Add(resolveSignedURLTTL(input.ExpiresInSeconds)),
	})
}

func normalizeContentType(contentType string) string {
	contentType = strings.TrimSpace(contentType)
	if contentType == "" {
		return "application/octet-stream"
	}

	return contentType
}

func buildUserUploadObjectName(userID uuid.UUID, fileName string) string {
	cleanName := sanitizeFileName(fileName)
	if cleanName == "" {
		cleanName = "file"
	}

	return fmt.Sprintf("uploads/%s/%s-%s", userID.String(), uuid.New().String(), cleanName)
}

func sanitizeFileName(fileName string) string {
	base := filepath.Base(strings.TrimSpace(fileName))
	if base == "." || base == "/" {
		return ""
	}

	var builder strings.Builder
	for _, r := range base {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r):
			builder.WriteRune(r)
		case r == '.', r == '-', r == '_':
			builder.WriteRune(r)
		default:
			builder.WriteRune('_')
		}
	}

	return strings.Trim(builder.String(), "._-")
}

func isUserUploadObject(userID uuid.UUID, objectName string) bool {
	prefix := fmt.Sprintf("uploads/%s/", userID.String())
	return strings.HasPrefix(objectName, prefix) && !strings.Contains(objectName, "..")
}

func resolveSignedURLTTL(expiresInSeconds int) time.Duration {
	if expiresInSeconds <= 0 {
		expiresInSeconds = defaultSignedURLTTLSeconds
	}
	if expiresInSeconds > maxSignedURLTTLSeconds {
		expiresInSeconds = maxSignedURLTTLSeconds
	}

	return time.Duration(expiresInSeconds) * time.Second
}
