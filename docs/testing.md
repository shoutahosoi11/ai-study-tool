# Testing Strategy

This repository uses a small number of explicit gates instead of one broad
catch-all test command. Run the gate that matches the files you changed, and run
the broader gate before opening or merging a PR.

## Required Gates

Backend changes:

```sh
cd backend && go test ./... && go build ./...
```

Concurrency-sensitive backend changes, including async jobs, queues, rate
limits, middleware, or worker orchestration:

```sh
cd backend && go test -race ./internal/usecase ./internal/middleware ./internal/infrastructure/cloudtasks
```

Frontend changes:

```sh
cd frontend && npm test
```

Mobile changes:

```sh
cd mobile && npm test
```

Repository-wide local smoke:

```sh
make test
```

## Integration Tests

Integration tests are opt-in and must only use a disposable local or staging
database. Never point `INTEGRATION_DATABASE_URL` at production or a shared Neon
branch, because the repository integration tests reset tables.

```sh
cd backend && INTEGRATION_DATABASE_URL="postgres://postgres:postgres@localhost:5432/ai_study_tool?sslmode=disable" \
  go test -count=1 ./internal/infrastructure/persistence -run Integration
```

CI runs this command with `INTEGRATION_DATABASE_URL` from GitHub Actions
secrets. If the secret is unset, the tests compile and skip.

## Migration Checks

For DB migrations, apply every file to a fresh disposable database in sorted
order before merging:

```sh
DB_URL="postgres://postgres:postgres@localhost:5432/ai_study_tool?sslmode=disable" make migrate
```

Production migrations should use the Neon direct connection string, not the
pooled runtime connection string. Keep migration verification separate from the
Cloud Run `DATABASE_URL`.

## What To Cover

Add tests at the layer where the behavior belongs:

- `handler`: request parsing, auth context requirements, HTTP status mapping,
  body limits, and response DTO shape.
- `usecase`: business rules, authorization decisions, quota counters,
  transaction ordering, retry/idempotency behavior, and external-service error
  handling.
- `repository`: SQL predicates, ownership filters, pagination, upsert/delete
  idempotency, and transaction consistency.
- `middleware`: fail-closed auth, rate limits, headers, and security guards.
- `frontend` / `mobile`: API-client behavior, token attachment, validation,
  loading/error/empty states, and parsing helpers.

High-risk areas require explicit negative tests:

- Authenticated user ID must come from context, not request bodies.
- Users must not update another user's private highlights, notes, explanations,
  or settings.
- Cloud Tasks retries must be idempotent.
- Queue state transitions must not lose work on enqueue or worker failure.
- Gemini/Stripe/Firebase/Cloud Tasks failures must be wrapped and surfaced
  without leaking secrets.

## CI Contract

`.github/workflows/ci.yml` is the PR gate:

- backend: `go test ./...`, opt-in persistence integration tests, race tests
  for concurrency-sensitive packages, and `go build ./...`
- frontend: `npm test`
- mobile: `npm test`
- Kindle extension: `npm run check`

`.github/workflows/deploy-api.yml` repeats backend `go test ./...` and
`go build ./...` before building and deploying the Cloud Run image.
