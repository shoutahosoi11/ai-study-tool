# Production Deploy: Cloud Run + Neon + Firebase Auth + Cloud Tasks

This guide sets up the first production-like environment:

- Cloud Run: `api-service`
- Neon PostgreSQL: managed Postgres reached through a standard `DATABASE_URL`
- Secret Manager: `DATABASE_URL`, `GEMINI_API_KEY`, optional
  `MIGRATION_DATABASE_URL`, and task authentication secrets when OIDC is not
  used
- Firebase Auth: frontend issues ID tokens, backend verifies them
- Cloud Tasks: asynchronous question generation and highlight import workers
- GitHub Actions: build, test, push image, deploy Cloud Run

The backend reads a standard lib/pq `DATABASE_URL`. For production, store the
Neon connection string in Secret Manager. Cloud SQL connectors, Cloud SQL Auth
Proxy, and `--add-cloudsql-instances` are not part of the current deployment.

## 1. Create Google Cloud resources

Choose values:

```sh
PROJECT_ID="your-gcp-project"
REGION="asia-northeast1"
SERVICE="api-service"
GAR_REPOSITORY="ai-study-tool"
```

Enable APIs:

```sh
gcloud services enable \
  run.googleapis.com \
  artifactregistry.googleapis.com \
  secretmanager.googleapis.com \
  iamcredentials.googleapis.com \
  cloudtasks.googleapis.com \
  cloudbuild.googleapis.com \
  --project="${PROJECT_ID}"
```

Create Artifact Registry:

```sh
gcloud artifacts repositories create "${GAR_REPOSITORY}" \
  --repository-format=docker \
  --location="${REGION}" \
  --project="${PROJECT_ID}"
```

Create Cloud Tasks queues:

```sh
PROJECT_ID="${PROJECT_ID}" LOCATION="${REGION}" ./deploy/cloudtasks/setup-queues.sh
```

## 2. Create Neon PostgreSQL

Create a Neon project and database from the Neon dashboard. Use PostgreSQL 16
for parity with local Docker development.

Recommended connection choices:

- Runtime API: use Neon's pooled connection string unless a transaction-heavy
  code path needs the direct endpoint.
- Migrations: use Neon's direct connection string.
- SSL: keep `sslmode=require` in production.

Example:

```text
postgresql://USER:PASSWORD@HOST.neon.tech/DB_NAME?sslmode=require
```

## 3. Store secrets

Create required Secret Manager secrets. Store the runtime pooled Neon URL in
`DATABASE_URL`; if possible, store the direct Neon URL for migrations in
`MIGRATION_DATABASE_URL`.

```sh
DATABASE_URL="postgresql://USER:PASSWORD@HOST.neon.tech/DB_NAME?sslmode=require"
MIGRATION_DATABASE_URL="postgresql://USER:PASSWORD@HOST.neon.tech/DB_NAME?sslmode=require"
INTERNAL_TASK_SECRET="$(openssl rand -base64 32)"

printf '%s' "${DATABASE_URL}" | gcloud secrets create DATABASE_URL \
  --data-file=- \
  --project="${PROJECT_ID}"

printf '%s' "${MIGRATION_DATABASE_URL:-${DATABASE_URL}}" | gcloud secrets create MIGRATION_DATABASE_URL \
  --data-file=- \
  --project="${PROJECT_ID}"

printf '%s' 'PASTE_GEMINI_API_KEY_HERE' | gcloud secrets create GEMINI_API_KEY \
  --data-file=- \
  --project="${PROJECT_ID}"

printf '%s' "${INTERNAL_TASK_SECRET}" | gcloud secrets create INTERNAL_TASK_SECRET \
  --data-file=- \
  --project="${PROJECT_ID}"
```

If Stripe is enabled, also create `STRIPE_SECRET_KEY` and
`STRIPE_WEBHOOK_SECRET`. The deploy workflow treats those as optional and only
binds them when they exist.

If a secret already exists, add a new version:

```sh
printf '%s' "${DATABASE_URL}" | gcloud secrets versions add DATABASE_URL \
  --data-file=- \
  --project="${PROJECT_ID}"

printf '%s' "${MIGRATION_DATABASE_URL:-${DATABASE_URL}}" | gcloud secrets versions add MIGRATION_DATABASE_URL \
  --data-file=- \
  --project="${PROJECT_ID}"

printf '%s' 'PASTE_GEMINI_API_KEY_HERE' | gcloud secrets versions add GEMINI_API_KEY \
  --data-file=- \
  --project="${PROJECT_ID}"

printf '%s' "${INTERNAL_TASK_SECRET}" | gcloud secrets versions add INTERNAL_TASK_SECRET \
  --data-file=- \
  --project="${PROJECT_ID}"
```

## 4. Service accounts and IAM

Runtime service account:

```sh
gcloud iam service-accounts create cloud-run-api \
  --display-name="Cloud Run API runtime" \
  --project="${PROJECT_ID}"

RUNTIME_SA="cloud-run-api@${PROJECT_ID}.iam.gserviceaccount.com"
```

Grant runtime permissions:

```sh
PROJECT_ID="${PROJECT_ID}" \
SA_EMAIL="${RUNTIME_SA}" \
LOCATION="${REGION}" \
./deploy/setup-service-accounts.sh
```

This grants per-secret Secret Manager access and queue-level Cloud Tasks enqueue
access. It does not grant project-wide Secret Manager or storage permissions.

Deploy service account for GitHub Actions:

```sh
gcloud iam service-accounts create github-deployer \
  --display-name="GitHub Actions deployer" \
  --project="${PROJECT_ID}"

DEPLOYER_SA="github-deployer@${PROJECT_ID}.iam.gserviceaccount.com"

gcloud projects add-iam-policy-binding "${PROJECT_ID}" \
  --member="serviceAccount:${DEPLOYER_SA}" \
  --role="roles/run.admin"

gcloud projects add-iam-policy-binding "${PROJECT_ID}" \
  --member="serviceAccount:${DEPLOYER_SA}" \
  --role="roles/artifactregistry.writer"

gcloud projects add-iam-policy-binding "${PROJECT_ID}" \
  --member="serviceAccount:${DEPLOYER_SA}" \
  --role="roles/iam.serviceAccountUser"

for secret in DATABASE_URL MIGRATION_DATABASE_URL GEMINI_API_KEY INTERNAL_TASK_SECRET STRIPE_SECRET_KEY STRIPE_WEBHOOK_SECRET; do
  if gcloud secrets describe "${secret}" --project="${PROJECT_ID}" >/dev/null 2>&1; then
    gcloud secrets add-iam-policy-binding "${secret}" \
      --project="${PROJECT_ID}" \
      --member="serviceAccount:${DEPLOYER_SA}" \
      --role="roles/secretmanager.viewer" \
      --condition=None
  fi
done

for secret in MIGRATION_DATABASE_URL DATABASE_URL; do
  if gcloud secrets describe "${secret}" --project="${PROJECT_ID}" >/dev/null 2>&1; then
    gcloud secrets add-iam-policy-binding "${secret}" \
      --project="${PROJECT_ID}" \
      --member="serviceAccount:${DEPLOYER_SA}" \
      --role="roles/secretmanager.secretAccessor" \
      --condition=None
    break
  fi
done
```

The deployer needs Secret Manager metadata visibility to validate runtime
secret existence. It also needs `roles/secretmanager.secretAccessor` on
`MIGRATION_DATABASE_URL` or `DATABASE_URL`, because the deploy workflow runs
database migrations before deploying a new Cloud Run revision.

Set up GitHub Workload Identity Federation using the official
`google-github-actions/auth` setup. Restrict the provider to your repository,
then store these GitHub secrets:

- `GCP_WORKLOAD_IDENTITY_PROVIDER`
- `GCP_DEPLOYER_SERVICE_ACCOUNT`

## 5. GitHub repository variables

Set these GitHub Actions variables:

| Variable | Example |
| --- | --- |
| `GCP_PROJECT_ID` | `your-gcp-project` |
| `GCP_REGION` | `asia-northeast1` |
| `CLOUD_RUN_API_SERVICE` | `api-service` |
| `ARTIFACT_REGISTRY_REPOSITORY` | `ai-study-tool` |
| `CLOUD_RUN_RUNTIME_SERVICE_ACCOUNT` | `cloud-run-api@project.iam.gserviceaccount.com` |
| `CORS_ALLOWED_ORIGINS` | `https://YOUR_FRONTEND_DOMAIN` |
| `QUEUE_QUESTION_GENERATION` | `projects/project/locations/asia-northeast1/queues/question-generation` |
| `QUEUE_HIGHLIGHT_IMPORT` | `projects/project/locations/asia-northeast1/queues/highlight-import` |
| `TASK_HANDLER_BASE_URL` | `https://api.example.com` |
| `INTERNAL_TASK_INVOKER_SERVICE_ACCOUNT` | `cloud-tasks-invoker@project.iam.gserviceaccount.com` |
| `SHUTDOWN_TIMEOUT_SECONDS` | `90` |
| `ALLOW_DESTRUCTIVE_MIGRATIONS` | `false` |
| `CLOUD_RUN_SMOKE_TEST_URL` | `https://api.example.com` |
| `STRIPE_PRICE_ID_MONTHLY` | `price_...` |
| `STRIPE_SUCCESS_URL` | `https://YOUR_FRONTEND_DOMAIN/billing/success` |
| `STRIPE_CANCEL_URL` | `https://YOUR_FRONTEND_DOMAIN/billing/cancel` |

`INTERNAL_TASK_INVOKER_SERVICE_ACCOUNT` enables Cloud Tasks OIDC authentication
for `/internal/tasks/*`. If it is empty, the backend falls back to
`INTERNAL_TASK_SECRET`.

## 6. Run migrations

The deploy workflow runs migrations automatically before building and deploying
the new image. It reads `MIGRATION_DATABASE_URL` from Secret Manager, falling
back to `DATABASE_URL` if the migration secret is absent, and records applied
files in `schema_migrations`.

Changed migration files are checked for obviously backward-incompatible DDL
such as `DROP TABLE`, `DROP COLUMN`, column rename, and `SET NOT NULL`.
Keep production migrations forward-only and backward-compatible by default.
For a planned multi-phase destructive migration, set
`ALLOW_DESTRUCTIVE_MIGRATIONS=true` only for the controlled deploy that needs it.

For manual runs, use the same script against the Neon direct connection string:

```sh
MIGRATION_DATABASE_URL="postgresql://USER:PASSWORD@HOST.neon.tech/DB_NAME?sslmode=require"
bash backend/scripts/apply_migrations.sh "${MIGRATION_DATABASE_URL}" backend/db/migrations
```

If production was migrated manually before `schema_migrations` existed, baseline
the current files once after confirming the schema is already up to date:

```sh
BASELINE_MIGRATIONS=true \
  bash backend/scripts/apply_migrations.sh "${MIGRATION_DATABASE_URL}" backend/db/migrations
```

The production schema must include `030_add_user_daily_generation_count.sql`
before enabling question sync in production.

## 7. First deploy

Push to `main` or run the `Deploy API to Cloud Run` workflow manually.

The workflow injects `DATABASE_URL`, `GEMINI_API_KEY`, and optionally
`INTERNAL_TASK_SECRET` from Secret Manager and does not attach a Cloud SQL
instance. The workflow deploys with
`--ingress=internal-and-cloud-load-balancing`. For Cloud Tasks, set
`TASK_HANDLER_BASE_URL` to the default `run.app` service URL in the same
project, or to the external Application Load Balancer hostname if all task
traffic is routed through the load balancer. The backend still requires
Firebase ID tokens on `/api/*`.

```sh
gcloud run services add-iam-policy-binding "${SERVICE}" \
  --region="${REGION}" \
  --member="allUsers" \
  --role="roles/run.invoker" \
  --project="${PROJECT_ID}"
```

`allUsers` is acceptable only with `--ingress=internal-and-cloud-load-balancing`
and a controlled public entrypoint such as the load balancer. Do not change the
service ingress to `all` without replacing this with authenticated invoker
access.

Add the deployed frontend domain to:

- Firebase Auth authorized domains
- `CORS_ALLOWED_ORIGINS`

## 8. Monitoring

Create or update the Cloud Monitoring dashboard and alert policies after the
first deploy:

```sh
PROJECT_ID="${PROJECT_ID}" \
SERVICE_NAME="${SERVICE}" \
QUEUE_QUESTION_GENERATION="${QUEUE_QUESTION_GENERATION}" \
QUEUE_HIGHLIGHT_IMPORT="${QUEUE_HIGHLIGHT_IMPORT}" \
ALERT_EMAIL="ops@example.com" \
bash deploy/monitoring/setup-monitoring.sh
```

The script is idempotent: it reuses existing email notification channels,
updates an existing dashboard with the same display name, and updates alert
policies with matching display names instead of creating duplicates.

## Notes

- Locally, `FIREBASE_CREDENTIALS_PATH` can point at a service account JSON.
- On Cloud Run, Firebase Admin uses the runtime service account through Application Default Credentials.
- Cloud Run runtime `DATABASE_URL` should use Neon's pooled `-pooler` endpoint;
  migration commands should use Neon's direct endpoint.
- For staging without Gemini cost, set `USE_GEMINI_MOCK=true`.
- To test tighter generation budgets, set `QUESTION_SYNC_DAILY_LIMIT` and `QUESTION_SYNC_PER_TRIGGER_LIMIT` to small values.
- Security deployment guardrails live under `docs/deployment/`:
  - `cloud-run.md`
  - `cloudflare.md`
  - `budget-alert.md`
  - `incident-runbook.md`
  - `rollback-runbook.md`
