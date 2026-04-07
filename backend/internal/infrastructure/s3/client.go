package s3

import (
	"context"
	"fmt"
	"io"
)

type Client struct {
	bucket string
	region string
}

func NewClient(bucket, region string) (*Client, error) {
	if bucket == "" || region == "" {
		return nil, fmt.Errorf("s3: bucket and region are required")
	}

	return &Client{
		bucket: bucket,
		region: region,
	}, nil
}

func (c *Client) Upload(ctx context.Context, key string, reader io.Reader, contentType string) (string, error) {
	_, err := io.Copy(io.Discard, reader)
	if err != nil {
		return "", fmt.Errorf("s3: read upload body failed: %w", err)
	}

	url := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", c.bucket, c.region, key)
	return url, nil
}
