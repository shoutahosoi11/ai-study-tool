# Event Driven Question Generation Phase 1

## What Changed

- Added forward-only migrations for the event-driven question generation foundation:
  - `question_generation_jobs`
  - `question_generation_job_highlights`
  - `questions.superseded_at`
  - `users.last_sync_at`
  - `highlights.book_key` and `idx_highlights_user_book_status`
- Added domain definitions for job status, job reason, retry constants, repository ports, and the Cloud Tasks enqueue port.
- Added a Postgres-backed `QuestionGenerationJobRepository` with create/get/status update methods and a queued-to-processing CAS claim.
- Added a Cloud Tasks HTTP enqueuer implementation behind the domain interface.
- Added a no-op `InternalOnly` middleware and registered `POST /internal/tasks/question-generation`.
- Added a placeholder task handler that returns `202 Accepted`; actual job processing is intentionally left for Phase 2.

## Why This Design

Phase 1 keeps existing behavior untouched by adding schema and adapters only. The handler/usecase paths that currently serve questions still run the same code, while the new job table and task endpoint are available for Phase 2.

The Cloud Tasks SDK is isolated in `internal/infrastructure/cloudtasks`, and the domain layer exposes only `QuestionGenerationTaskEnqueuer`. That keeps usecases testable and prevents GCP-specific types from leaking into business logic.

The repository owns SQL persistence and CAS mechanics only. It does not decide when jobs should be created or how many highlights should be selected; those rules belong to Phase 2 usecases.

The internal task middleware is no-op for now because the agreed Phase 1 boundary is Cloud Run `ingress=internal-and-cloud-load-balancing`. OIDC verification is explicitly deferred so the task endpoint can be wired in without mixing infrastructure auth policy into this foundation PR.

## Alternatives Considered

- Calling Cloud Tasks directly from usecases: rejected because it would couple business logic to GCP SDKs and make tests heavier.
- Reusing highlight `status` alone as the job queue: rejected because retries, enqueue failures, task idempotency, and active-job uniqueness need a durable job identity.
- Adding one combined migration file: rejected for this phase because the requested split makes each schema concern reviewable independently.
- Functional index for book key: rejected because the requested index targets `book_key`; Phase 1 adds a nullable physical column so Phase 2 can populate it consistently.

## Manual Rollback SQL

```sql
DROP INDEX IF EXISTS idx_highlights_user_book_status;
ALTER TABLE highlights DROP COLUMN IF EXISTS book_key;

ALTER TABLE users DROP COLUMN IF EXISTS last_sync_at;

DROP INDEX IF EXISTS idx_questions_active_by_highlight;
ALTER TABLE questions DROP COLUMN IF EXISTS superseded_at;

DROP TABLE IF EXISTS question_generation_job_highlights;
DROP INDEX IF EXISTS idx_question_generation_jobs_enqueue_failed;
DROP INDEX IF EXISTS uq_question_generation_jobs_active;
DROP TABLE IF EXISTS question_generation_jobs;
```

## Verification

- `cd backend && go build ./...`
- `cd backend && go test ./...`
