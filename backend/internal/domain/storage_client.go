package domain

import (
	"context"
	"io"
)

type StorageClient interface {
	Upload(ctx context.Context, key string, reader io.Reader, contentType string) (string, error)
}
