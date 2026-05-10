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

type QuestionGenerationEnqueuer struct {
	client     *cloudtasks.Client
	queuePath  string
	handlerURL string
}

func NewQuestionGenerationEnqueuerFromEnv(ctx context.Context) (*QuestionGenerationEnqueuer, error) {
	queuePath := strings.TrimSpace(os.Getenv("QUEUE_QUESTION_GENERATION"))
	handlerURL := strings.TrimRight(strings.TrimSpace(os.Getenv("TASK_HANDLER_BASE_URL")), "/")
	if queuePath == "" || handlerURL == "" {
		return nil, nil
	}

	client, err := cloudtasks.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("cloudtasks: new question generation client: %w", err)
	}

	return &QuestionGenerationEnqueuer{
		client:     client,
		queuePath:  queuePath,
		handlerURL: handlerURL,
	}, nil
}

func (e *QuestionGenerationEnqueuer) EnqueueQuestionGeneration(ctx context.Context, jobID uuid.UUID, userID uuid.UUID) error {
	if e == nil || e.client == nil {
		return nil
	}

	payload := map[string]string{
		"job_id":  jobID.String(),
		"user_id": userID.String(),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("cloudtasks: marshal question generation payload: %w", err)
	}

	req := &taskspb.CreateTaskRequest{
		Parent: e.queuePath,
		Task: &taskspb.Task{
			MessageType: &taskspb.Task_HttpRequest{
				HttpRequest: &taskspb.HttpRequest{
					HttpMethod: taskspb.HttpMethod_POST,
					Url:        e.handlerURL + "/internal/tasks/question-generation",
					Headers:    map[string]string{"Content-Type": "application/json"},
					Body:       body,
				},
			},
		},
	}
	if _, err := e.client.CreateTask(ctx, req); err != nil {
		return fmt.Errorf("cloudtasks: create question generation task: %w", err)
	}
	return nil
}

func (e *QuestionGenerationEnqueuer) Close() error {
	if e == nil || e.client == nil {
		return nil
	}
	return e.client.Close()
}
