package cloudtasks

import (
	"context"
	"encoding/json"
	"testing"

	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"github.com/google/uuid"
	"github.com/googleapis/gax-go/v2"
)

type fakeTaskCreator struct {
	request *cloudtaskspb.CreateTaskRequest
}

func (f *fakeTaskCreator) CreateTask(ctx context.Context, req *cloudtaskspb.CreateTaskRequest, opts ...gax.CallOption) (*cloudtaskspb.Task, error) {
	f.request = req
	return req.Task, nil
}

func TestQuestionGenerationEnqueuerUsesInjectedClient(t *testing.T) {
	client := &fakeTaskCreator{}
	enqueuer, err := NewQuestionGenerationEnqueuerWithClient(client, "projects/p/locations/l/queues/q", "https://example.com/internal/tasks/question-generation")
	if err != nil {
		t.Fatalf("new enqueuer: %v", err)
	}

	jobID := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	userID := uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	if err := enqueuer.EnqueueQuestionGeneration(context.Background(), jobID, userID); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	if client.request == nil {
		t.Fatal("expected CreateTask to be called")
	}
	if client.request.Parent != "projects/p/locations/l/queues/q" {
		t.Fatalf("unexpected parent: %s", client.request.Parent)
	}
	task := client.request.Task
	if task.Name != "projects/p/locations/l/queues/q/tasks/question-generation-"+jobID.String() {
		t.Fatalf("unexpected task name: %s", task.Name)
	}

	httpRequest := task.GetHttpRequest()
	if httpRequest.GetHttpMethod() != cloudtaskspb.HttpMethod_POST {
		t.Fatalf("unexpected method: %s", httpRequest.GetHttpMethod())
	}
	if httpRequest.GetUrl() != "https://example.com/internal/tasks/question-generation" {
		t.Fatalf("unexpected url: %s", httpRequest.GetUrl())
	}
	if httpRequest.GetHeaders()["Content-Type"] != "application/json" {
		t.Fatalf("unexpected headers: %#v", httpRequest.GetHeaders())
	}

	var payload QuestionGenerationTaskPayload
	if err := json.Unmarshal(httpRequest.GetBody(), &payload); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if payload.JobID != jobID.String() || payload.UserID != userID.String() {
		t.Fatalf("unexpected payload: %#v", payload)
	}
}
