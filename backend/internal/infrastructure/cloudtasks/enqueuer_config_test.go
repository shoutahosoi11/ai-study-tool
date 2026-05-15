package cloudtasks

import (
	"context"
	"testing"
)

func TestQuestionGenerationEnqueuerRequiresInternalTaskSecret(t *testing.T) {
	t.Setenv("QUEUE_QUESTION_GENERATION", "projects/project/locations/asia-northeast1/queues/question-generation")
	t.Setenv("TASK_HANDLER_BASE_URL", "https://api.example.com")
	t.Setenv("INTERNAL_TASK_SECRET", "")
	t.Setenv("INTERNAL_TASK_INVOKER_SERVICE_ACCOUNT", "")

	enqueuer, err := NewQuestionGenerationEnqueuerFromEnv(context.Background())
	if err == nil {
		t.Fatal("expected error when task authentication is missing")
	}
	if enqueuer != nil {
		t.Fatal("expected nil enqueuer when configuration is invalid")
	}
}

func TestHighlightImportEnqueuerRequiresInternalTaskSecret(t *testing.T) {
	t.Setenv("QUEUE_HIGHLIGHT_IMPORT", "projects/project/locations/asia-northeast1/queues/highlight-import")
	t.Setenv("TASK_HANDLER_BASE_URL", "https://api.example.com")
	t.Setenv("INTERNAL_TASK_SECRET", "")
	t.Setenv("INTERNAL_TASK_INVOKER_SERVICE_ACCOUNT", "")

	enqueuer, err := NewHighlightImportEnqueuerFromEnv(context.Background())
	if err == nil {
		t.Fatal("expected error when task authentication is missing")
	}
	if enqueuer != nil {
		t.Fatal("expected nil enqueuer when configuration is invalid")
	}
}
