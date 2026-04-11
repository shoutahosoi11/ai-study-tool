package firebase

import (
	"context"
	"fmt"
	"io"
	"net/url"

	firebase "firebase.google.com/go/v4"
	gstorage "cloud.google.com/go/storage"
)

type StorageClient struct {
	bucket     *gstorage.BucketHandle
	bucketName string
}

func NewStorageClient(ctx context.Context, app *firebase.App, bucketName string) (*StorageClient, error) {
	client, err := app.Storage(ctx)
	if err != nil {
		return nil, fmt.Errorf("firebase storage: init client: %w", err)
	}

	bucket, err := client.DefaultBucket()
	if err != nil {
		return nil, fmt.Errorf("firebase storage: get bucket: %w", err)
	}

	return &StorageClient{
		bucket:     bucket,
		bucketName: bucketName,
	}, nil
}

func (c *StorageClient) Upload(ctx context.Context, key string, reader io.Reader, contentType string) (string, error) {
	wc := c.bucket.Object(key).NewWriter(ctx)
	wc.ContentType = contentType

	if _, err := io.Copy(wc, reader); err != nil {
		_ = wc.Close()
		return "", fmt.Errorf("firebase storage: write object: %w", err)
	}

	if err := wc.Close(); err != nil {
		return "", fmt.Errorf("firebase storage: close writer: %w", err)
	}

	encodedKey := url.PathEscape(key)
	downloadURL := fmt.Sprintf(
		"https://firebasestorage.googleapis.com/v0/b/%s/o/%s?alt=media",
		c.bucketName, encodedKey,
	)
	return downloadURL, nil
}
