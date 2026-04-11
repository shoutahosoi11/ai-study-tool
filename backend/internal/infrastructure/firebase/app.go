package firebase

import (
	"context"
	"fmt"

	firebase "firebase.google.com/go/v4"
	"google.golang.org/api/option"
)

func NewApp(ctx context.Context, credPath, storageBucket string) (*firebase.App, error) {
	if credPath == "" {
		return nil, fmt.Errorf("firebase: FIREBASE_CREDENTIALS_PATH is required")
	}
	app, err := firebase.NewApp(ctx, &firebase.Config{StorageBucket: storageBucket}, option.WithCredentialsFile(credPath))
	if err != nil {
		return nil, fmt.Errorf("firebase: init app: %w", err)
	}
	return app, nil
}
