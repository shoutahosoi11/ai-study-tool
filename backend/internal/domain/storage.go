package domain

import (
	"context"
	"time"
)

type SignedURLOptions struct {
	ObjectName  string
	Method      string
	ContentType string
	ExpiresAt   time.Time
}

type SignedURL struct {
	URL         string
	Method      string
	Bucket      string
	ObjectName  string
	ContentType string
	ExpiresAt   time.Time
	Headers     map[string]string
}

type StorageSigner interface {
	SignedURL(ctx context.Context, options SignedURLOptions) (*SignedURL, error)
}
