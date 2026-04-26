# AI Study Tool

AI Study Tool is an AI-powered learning platform for study workflows, question generation, note capture, and social learning.

## Tech Stack

- Go + Echo
- PostgreSQL + sqlc
- React + TypeScript + Tailwind CSS + Vite
- Firebase Auth
- AWS S3
- Gemini API

## Getting Started

1. Copy `backend/.env.example` to `backend/.env` and fill in the required values.
2. Start the development stack with `docker compose up --build`.
3. Access the frontend at `http://localhost:3000` and the backend health check at `http://localhost:8080/health`.

## Architecture

The project follows Clean Architecture:

- `backend/cmd` contains the application entrypoint.
- `backend/internal/domain` contains core business entities and rules.
- `backend/internal/usecase` contains application-specific business logic.
- `backend/internal/repository` contains persistence adapters and sqlc-generated access code.
- `backend/internal/handler` contains HTTP handlers.
- `backend/internal/middleware` contains cross-cutting HTTP middleware.
- `frontend/src` contains the client application organized by components, pages, hooks, theme, types, and API access.

## Deployment

Phase 1 deployment targets Cloud Run, Neon PostgreSQL, Firebase Auth,
Cloud Storage signed URLs, and GitHub Actions. See
[`docs/deploy-phase-1.md`](docs/deploy-phase-1.md).

For the upcoming iOS / Android share-sheet intake flow, see
[`docs/mobile-share-api.md`](docs/mobile-share-api.md).

The React Native / Expo mobile app scaffold lives in [`mobile/`](mobile/README.md).

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
