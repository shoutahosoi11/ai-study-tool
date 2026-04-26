package usecase

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

type fakeStorageSigner struct {
	lastOptions domain.SignedURLOptions
}

func (s *fakeStorageSigner) SignedURL(ctx context.Context, options domain.SignedURLOptions) (*domain.SignedURL, error) {
	s.lastOptions = options
	return &domain.SignedURL{
		URL:         "https://storage.example/signed",
		Method:      options.Method,
		Bucket:      "bucket",
		ObjectName:  options.ObjectName,
		ContentType: options.ContentType,
		ExpiresAt:   options.ExpiresAt,
		Headers: map[string]string{
			"Content-Type": options.ContentType,
		},
	}, nil
}

func TestStorageUsecaseCreateUploadSignedURLScopesObjectToUser(t *testing.T) {
	signer := &fakeStorageSigner{}
	usecase := NewStorageUsecase(signer)
	userID := uuid.New()

	result, err := usecase.CreateUploadSignedURL(context.Background(), CreateUploadSignedURLInput{
		UserID:      userID,
		FileName:    "../avatar image.png",
		ContentType: "image/png",
	})
	if err != nil {
		t.Fatalf("CreateUploadSignedURL returned error: %v", err)
	}

	expectedPrefix := "uploads/" + userID.String() + "/"
	if !strings.HasPrefix(result.ObjectName, expectedPrefix) {
		t.Fatalf("object name = %q, want prefix %q", result.ObjectName, expectedPrefix)
	}
	if result.ContentType != "image/png" {
		t.Fatalf("content type = %q, want image/png", result.ContentType)
	}
	if signer.lastOptions.Method != "PUT" {
		t.Fatalf("method = %q, want PUT", signer.lastOptions.Method)
	}
	if time.Until(result.ExpiresAt) <= 0 {
		t.Fatalf("expires_at should be in the future")
	}
}

func TestStorageUsecaseCreateDownloadSignedURLRejectsOtherUserObject(t *testing.T) {
	usecase := NewStorageUsecase(&fakeStorageSigner{})

	_, err := usecase.CreateDownloadSignedURL(context.Background(), CreateDownloadSignedURLInput{
		UserID:     uuid.New(),
		ObjectName: "uploads/" + uuid.New().String() + "/file.png",
	})
	if err == nil {
		t.Fatalf("CreateDownloadSignedURL succeeded, want error")
	}
	if err != domain.ErrInvalidInput {
		t.Fatalf("error = %v, want ErrInvalidInput", err)
	}
}

func TestStorageUsecaseReturnsNotConfiguredWithoutSigner(t *testing.T) {
	usecase := NewStorageUsecase(nil)

	_, err := usecase.CreateUploadSignedURL(context.Background(), CreateUploadSignedURLInput{
		UserID:   uuid.New(),
		FileName: "file.png",
	})
	if err != domain.ErrStorageNotConfigured {
		t.Fatalf("error = %v, want ErrStorageNotConfigured", err)
	}
}
