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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type HighlightImportEnqueuer struct {
	client     *cloudtasks.Client
	queuePath  string
	handlerURL string
	taskSecret string
	invokerSA  string
}

func NewHighlightImportEnqueuerFromEnv(ctx context.Context) (*HighlightImportEnqueuer, error) {
	queuePath := strings.TrimSpace(os.Getenv("QUEUE_HIGHLIGHT_IMPORT"))
	handlerURL := strings.TrimRight(strings.TrimSpace(os.Getenv("TASK_HANDLER_BASE_URL")), "/")
	taskSecret := strings.TrimSpace(os.Getenv("INTERNAL_TASK_SECRET"))
	invokerSA := strings.TrimSpace(os.Getenv("INTERNAL_TASK_INVOKER_SERVICE_ACCOUNT"))
	if queuePath == "" || handlerURL == "" {
		return nil, nil
	}
	if taskSecret == "" && invokerSA == "" {
		return nil, fmt.Errorf("cloudtasks: INTERNAL_TASK_SECRET or INTERNAL_TASK_INVOKER_SERVICE_ACCOUNT is required when highlight import queue is configured")
	}

	client, err := cloudtasks.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("cloudtasks: new highlight import client: %w", err)
	}

	return &HighlightImportEnqueuer{
		client:     client,
		queuePath:  queuePath,
		handlerURL: handlerURL,
		taskSecret: taskSecret,
		invokerSA:  invokerSA,
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

	taskURL := e.handlerURL + "/internal/tasks/highlight-import"
	httpRequest := &taskspb.HttpRequest{
		HttpMethod: taskspb.HttpMethod_POST,
		Url:        taskURL,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: body,
	}
	if e.invokerSA != "" {
		httpRequest.AuthorizationHeader = &taskspb.HttpRequest_OidcToken{
			OidcToken: &taskspb.OidcToken{
				ServiceAccountEmail: e.invokerSA,
				Audience:            taskURL,
			},
		}
	} else {
		httpRequest.Headers["X-Internal-Task-Secret"] = e.taskSecret
	}

	req := &taskspb.CreateTaskRequest{
		Parent: e.queuePath,
		Task: &taskspb.Task{
			Name: e.queuePath + "/tasks/highlight-import-" + queueID.String(),
			MessageType: &taskspb.Task_HttpRequest{
				HttpRequest: httpRequest,
			},
		},
	}
	if _, err := e.client.CreateTask(ctx, req); err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return nil
		}
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
