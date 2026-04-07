package domain

import "context"

type OCRClient interface {
	ExtractText(ctx context.Context, imageURL string) (string, error)
}
