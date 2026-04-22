package gcs

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/shout/ai-study-tool/backend/internal/domain"
)

type SignedURLService struct {
	bucketName          string
	signingGoogleAccess string
	client              *storage.Client
}

func NewSignedURLService(ctx context.Context, bucketName string, signingGoogleAccess string) (*SignedURLService, error) {
	bucketName = strings.TrimSpace(bucketName)
	if bucketName == "" {
		return nil, nil
	}

	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("gcs signed url: new storage client: %w", err)
	}

	return &SignedURLService{
		bucketName:          bucketName,
		signingGoogleAccess: strings.TrimSpace(signingGoogleAccess),
		client:              client,
	}, nil
}

func (s *SignedURLService) SignedURL(ctx context.Context, options domain.SignedURLOptions) (*domain.SignedURL, error) {
	if s == nil || s.client == nil || s.bucketName == "" {
		return nil, domain.ErrStorageNotConfigured
	}

	opts := &storage.SignedURLOptions{
		Method:      options.Method,
		Expires:     options.ExpiresAt,
		ContentType: options.ContentType,
		Scheme:      storage.SigningSchemeV4,
	}
	if s.signingGoogleAccess != "" {
		opts.GoogleAccessID = s.signingGoogleAccess
	}

	url, err := s.client.Bucket(s.bucketName).SignedURL(options.ObjectName, opts)
	if err != nil {
		return nil, fmt.Errorf("gcs signed url: sign %s %s: %w", options.Method, options.ObjectName, err)
	}

	headers := make(map[string]string)
	if options.ContentType != "" {
		headers["Content-Type"] = options.ContentType
	}

	return &domain.SignedURL{
		URL:         url,
		Method:      options.Method,
		Bucket:      s.bucketName,
		ObjectName:  options.ObjectName,
		ContentType: options.ContentType,
		ExpiresAt:   options.ExpiresAt,
		Headers:     headers,
	}, nil
}
