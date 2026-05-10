# AI Study Tool

AI Study Tool is an AI-powered learning platform for highlight capture, question generation, quiz sessions, note workflows, and social learning.

## Tech Stack

- Go 1.25 + Echo
- PostgreSQL on Neon + sqlc + database/sql
- React + TypeScript + Tailwind CSS + Vite
- React Native + Expo for the mobile app and iOS share-sheet flows
- Firebase Auth
- Gemini API
- Cloud Run + Secret Manager + GitHub Actions
- Cloud Tasks for asynchronous question generation and highlight import workers
- Cloud Storage signed URLs where direct object upload/download is needed
- Stripe for subscription checkout and webhooks
- Chrome Extension for Kindle Notebook highlight import

## Getting Started

1. Copy `backend/.env.example` to `backend/.env` and fill in the required values.
2. Start the local database and services with Docker Compose as needed.
3. Run the backend with `cd backend && go run ./cmd/main.go`.
4. Run the frontend with `cd frontend && npm run dev`.
5. Run the mobile app with `cd mobile && npx expo start --dev-client --lan --port 8081`.
6. Access the frontend at `http://localhost:3000`.
7. Check backend liveness at `http://localhost:8080/health`.
8. Check backend readiness and DB connectivity at `http://localhost:8080/ready`.

For physical mobile devices, set `EXPO_PUBLIC_API_BASE_URL` to the Mac LAN address and include the `/api/v1` base path.

## Architecture

The project follows Clean Architecture:

- `backend/cmd` contains application entrypoints.
- `backend/internal/domain` contains core entities, constants, domain errors, and interfaces.
- `backend/internal/usecase` contains application-specific business logic.
- `backend/internal/infrastructure` contains adapters for PostgreSQL, Gemini, Firebase, Stripe, Cloud Tasks, Cloud Run, and Cloud Storage.
- `backend/internal/repository/sqlcgen` contains sqlc-generated database access code.
- `backend/internal/handler` contains HTTP handlers and request/response boundary logic.
- `backend/internal/middleware` contains cross-cutting HTTP middleware.
- `backend/internal/router` registers API and internal task routes.
- `frontend/src` contains the web client organized by components, pages, hooks, theme, types, and API access.
- `mobile` contains the Expo app used for iOS share-sheet intake and mobile learning flows.
- `extension` and `kindle-highlights-extension` contain browser-extension work for Kindle Notebook scraping and import experiments.

Backend API routes are under `/api/v1`. Stripe webhooks are exposed at `/webhooks/stripe`. Cloud Tasks call internal endpoints under `/internal/tasks`.

## Highlight Import

Highlights can be imported through Kindle/extension import, mobile share intake, and paste import.

- `POST /api/v1/highlights/import`
- `POST /api/v1/highlights/share`
- `POST /api/v1/highlights/paste`

In production, queued highlight imports are processed asynchronously through Cloud Tasks and the `highlight_import_queue` table. Each task processes one queue row.

## Question Generation

Question generation is event-driven and DB-backed.

The app evaluates generation conditions on sync and after answer completion:

- App startup or polling calls `POST /api/v1/questions/sync`.
- Answer submission calls `POST /api/v1/questions/:id/answer`.
- The backend evaluates each `book_key` and creates rows in `question_generation_jobs` when generation conditions are met.
- Cloud Tasks calls `POST /internal/tasks/question-generation`.
- The worker claims a queued job with DB CAS, generates questions with Gemini, and marks the job completed or failed.

Generation conditions are book-based:

- Condition A: at least 10 pending highlights for a book.
- Condition B: 5 to 9 pending highlights and 0 unanswered active questions for that book.
- Fewer than 5 pending highlights does not trigger generation.

Important invariants:

- `highlights.status = 'pending'` is the generation queue source.
- A generation job processes up to 10 highlights.
- One active question per highlight is maintained with `questions.superseded_at`.
- Answering a question can return the source highlight to `pending`.
- The daily generation limit is enforced by backend budget/quota logic.

Relevant runtime controls:

- `QUESTION_SYNC_DAILY_LIMIT`: max generated questions per user per day.
- `QUESTION_SYNC_STALE_PROCESSING_SECONDS`: retry window for stale processing jobs.
- `QUESTION_WORKER_BATCH_SIZE`: fallback worker batch size.
- `QUESTION_WORKER_MAX_RETRY`: max worker retry count.
- `QUESTION_WORKER_MAX_QUESTIONS_PER_CALL`: max Gemini questions per call.
- `QUESTION_WORKER_MAX_PROMPT_TOKENS`: prompt-size guardrail.
- `QUESTION_WORKER_REQUEST_INTERVAL_MS`: process-level Gemini request spacing.
- `USE_GEMINI_MOCK=true`: use the fake Gemini client for local/staging checks.

## Cloud Tasks

Production async work uses Cloud Tasks:

- `QUEUE_QUESTION_GENERATION`
- `QUEUE_HIGHLIGHT_IMPORT`
- `TASK_HANDLER_BASE_URL`

Queue setup script:

```bash
PROJECT_ID=<gcp-project-id> LOCATION=asia-northeast1 ./deploy/cloudtasks/setup-queues.sh
```

Design details:

- [`docs/phase2/cloudtasks-migration-design.md`](docs/phase2/cloudtasks-migration-design.md)

## Deployment

The API is deployed to Cloud Run through GitHub Actions.

Production runtime dependencies:

- Cloud Run
- Neon PostgreSQL
- Firebase Auth
- Secret Manager
- Cloud Tasks
- Gemini API
- Stripe

Store the Neon connection string in the `DATABASE_URL` Secret Manager secret. Store sensitive API keys such as `GEMINI_API_KEY`, `STRIPE_SECRET_KEY`, and `STRIPE_WEBHOOK_SECRET` in Secret Manager.

Cloud Run should be configured with bounded scale and internal ingress where Cloud Tasks/internal routing is used:

```bash
gcloud run services update <service> \
  --region=asia-northeast1 \
  --max-instances=10 \
  --concurrency=80 \
  --ingress=internal-and-cloud-load-balancing
```

Deployment references:

- [`docs/deploy-phase-1.md`](docs/deploy-phase-1.md)
- [`docs/deployment/cloud-run.md`](docs/deployment/cloud-run.md)
- [`docs/deployment/cloudflare.md`](docs/deployment/cloudflare.md)
- [`docs/deployment/incident-runbook.md`](docs/deployment/incident-runbook.md)

The React Native / Expo mobile app lives in [`mobile/`](mobile/README.md).

## AI Consensus Review

This repository includes a local two-reviewer workflow that runs Codex and Claude on the same diff, shares both reviews back into a consensus pass, and can optionally ask Codex to apply only the mutually agreed findings.

Requirements:

- `codex` CLI installed and authenticated
- `claude` CLI installed and authenticated
- Python 3 available as `python3`

Examples:

- `python3 scripts/consensus_review.py --uncommitted`
- `python3 scripts/consensus_review.py --uncommitted --path backend/internal/middleware/auth.go`
- `python3 scripts/consensus_review.py --base main`
- `python3 scripts/consensus_review.py --uncommitted --apply`
- `python3 scripts/consensus_review.py --uncommitted --changed-under backend/internal --apply`
- `python3 scripts/consensus_review.py --uncommitted --changed-under backend/internal/middleware --changed-under backend/internal/handler --changed-under backend/internal/usecase --apply`
- `python3 scripts/consensus_review.py --resume-report .ai-consensus/<timestamp>`

Reports are written to `.ai-consensus/<timestamp>/` and include the diff, each model's review JSON, each consensus decision JSON, the mutually agreed findings, and a summary markdown file.

If Claude authentication, token expiry, budget, or usage limit issues are detected, the workflow stops in a `paused` state and writes the reason to `.ai-consensus/<timestamp>/status.json` and `summary.md` instead of continuing into the consensus or apply step.

For `--changed-under`, the workflow processes each changed file under the selected directory one by one, writes per-file attempt reports under `.ai-consensus/<timestamp>/files/`, and saves a resumable `batch_state.json`. If Claude later becomes available again, rerun the command with `--resume-report` pointing at the same report directory to continue from the paused file. Saved per-stage artifacts such as `codex_review.json` and `codex_consensus.json` are reused so the resume path can continue without redoing already completed stages for that file.

If you pass `--changed-under` multiple times, the script preserves that directory order. This lets you review in architecture order such as `middleware -> handler -> usecase -> repository`.

For local VS Code usage there is also a `.vscode/tasks.json` task set with workspace review, current-file review, base-branch review, and review-plus-apply commands.
