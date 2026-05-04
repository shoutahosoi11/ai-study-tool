package cloudtasks

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	cloudtasksapi "cloud.google.com/go/cloudtasks/apiv2"
	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"github.com/google/uuid"
	"github.com/googleapis/gax-go/v2"
)

type taskCreator interface {
	CreateTask(ctx context.Context, req *cloudtaskspb.CreateTaskRequest, opts ...gax.CallOption) (*cloudtaskspb.Task, error)
}

type QuestionGenerationEnqueuer struct {
	client    taskCreator
	queuePath string
	targetURL string
}

type QuestionGenerationTaskPayload struct {
	JobID  string `json:"job_id"`
	UserID string `json:"user_id"`
}

func NewQuestionGenerationEnqueuer(
	ctx context.Context,
	projectID string,
	locationID string,
	queueID string,
	targetURL string,
) (*QuestionGenerationEnqueuer, error) {
	client, err := cloudtasksapi.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("cloudtasks question generation: new client: %w", err)
	}

	return NewQuestionGenerationEnqueuerWithClient(
		client,
		fmt.Sprintf("projects/%s/locations/%s/queues/%s", strings.TrimSpace(projectID), strings.TrimSpace(locationID), strings.TrimSpace(queueID)),
		targetURL,
	)
}

func NewQuestionGenerationEnqueuerWithClient(client taskCreator, queuePath string, targetURL string) (*QuestionGenerationEnqueuer, error) {
	queuePath = strings.TrimSpace(queuePath)
	targetURL = strings.TrimSpace(targetURL)
	if client == nil || queuePath == "" || targetURL == "" {
		return nil, fmt.Errorf("cloudtasks question generation: invalid configuration")
	}

	return &QuestionGenerationEnqueuer{
		client:    client,
		queuePath: queuePath,
		targetURL: targetURL,
	}, nil
}

func (e *QuestionGenerationEnqueuer) EnqueueQuestionGeneration(ctx context.Context, jobID uuid.UUID, userID uuid.UUID) error {
	if jobID == uuid.Nil || userID == uuid.Nil {
		return fmt.Errorf("cloudtasks question generation: invalid task identity")
	}

	body, err := json.Marshal(QuestionGenerationTaskPayload{
		JobID:  jobID.String(),
		UserID: userID.String(),
	})
	if err != nil {
		return fmt.Errorf("cloudtasks question generation: marshal payload: %w", err)
	}

	_, err = e.client.CreateTask(ctx, &cloudtaskspb.CreateTaskRequest{
		Parent: e.queuePath,
		Task: &cloudtaskspb.Task{
			Name: fmt.Sprintf("%s/tasks/question-generation-%s", e.queuePath, jobID.String()),
			MessageType: &cloudtaskspb.Task_HttpRequest{
				HttpRequest: &cloudtaskspb.HttpRequest{
					HttpMethod: cloudtaskspb.HttpMethod_POST,
					Url:        e.targetURL,
					Headers: map[string]string{
						"Content-Type": "application/json",
					},
					Body: body,
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("cloudtasks question generation: create task: %w", err)
	}

	return nil
}
