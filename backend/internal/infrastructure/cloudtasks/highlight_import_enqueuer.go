package cloudtasks

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	cloudtasks "cloud.google.com/go/cloudtasks/apiv2"
	taskspb "cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"github.com/google/uuid"
)

type HighlightImportEnqueuer struct {
	client     *cloudtasks.Client
	queuePath  string
	handlerURL string
}

func NewHighlightImportEnqueuerFromEnv(ctx context.Context) (*HighlightImportEnqueuer, error) {
	queuePath := strings.TrimSpace(os.Getenv("QUEUE_HIGHLIGHT_IMPORT"))
	handlerURL := strings.TrimRight(strings.TrimSpace(os.Getenv("TASK_HANDLER_BASE_URL")), "/")
	if queuePath == "" || handlerURL == "" {
		return nil, nil
	}

	client, err := cloudtasks.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("cloudtasks: new highlight import client: %w", err)
	}

	return &HighlightImportEnqueuer{
		client:     client,
		queuePath:  queuePath,
		handlerURL: handlerURL,
	}, nil
}

func (e *HighlightImportEnqueuer) TriggerHighlightImportJob(ctx context.Context, queueID uuid.UUID, userID uuid.UUID) error {
	if e == nil || e.client == nil {
		return nil
	}

	payload := map[string]string{
		"queue_id": queueID.String(),
		"user_id":  userID.String(),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("cloudtasks: marshal highlight import payload: %w", err)
	}

	req := &taskspb.CreateTaskRequest{
		Parent: e.queuePath,
		Task: &taskspb.Task{
			MessageType: &taskspb.Task_HttpRequest{
				HttpRequest: &taskspb.HttpRequest{
					HttpMethod: taskspb.HttpMethod_POST,
					Url:        e.handlerURL + "/internal/tasks/highlight-import",
					Headers:    map[string]string{"Content-Type": "application/json"},
					Body:       body,
				},
			},
		},
	}
	if _, err := e.client.CreateTask(ctx, req); err != nil {
		return fmt.Errorf("cloudtasks: create highlight import task: %w", err)
	}
	return nil
}

func (e *HighlightImportEnqueuer) Close() error {
	if e == nil || e.client == nil {
		return nil
	}
	return e.client.Close()
}
