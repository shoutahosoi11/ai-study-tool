# Cloud Run Deployment Guardrails

This service is deployed to Cloud Run by `.github/workflows/deploy-api.yml`.
The API is an Echo HTTP server on port `8080` with separate lightweight
probes:

- `/health`: process liveness only.
- `/ready`: readiness, including database connectivity.

## Required Limits

Set `--max-instances=10` on the API service. This is the final cost guardrail if
application rate limits, Cloudflare rules, or client-side behavior fail. With a
database-backed app and LLM-triggering workflows, an unbounded Cloud Run service
can turn traffic spikes into database pressure and API spend quickly.

Set `--concurrency=80`. Echo can handle many concurrent requests, and `80` is a
reasonable Cloud Run default for mixed API traffic. It keeps instance count
lower during ordinary bursts while still allowing `--max-instances=10` to cap
total concurrent in-flight requests at about `800`.

Recommended starting resources:

- CPU: `1`
- Memory: `512Mi`
- Request timeout: `120s`
- Min instances: `0` for cost control unless cold starts become a product issue.
- Max instances: `10`
- Concurrency: `80`

Increase memory only after observing Cloud Run memory metrics. Increase timeout
only for endpoints that truly need it; ingest and question sync should return
quickly and let workers do longer-running work.

## Runtime Configuration

Use Secret Manager for sensitive values and plain environment variables only for
non-sensitive routing and tuning values.

Required secrets:

- `DATABASE_URL`: Neon pooled connection string for Cloud Run runtime traffic.
- `GEMINI_API_KEY`: Gemini API key.
- `INTERNAL_TASK_SECRET`: fallback shared secret for `/internal/tasks/*` when
  Cloud Tasks OIDC authentication is not configured.

Optional secrets:

- `STRIPE_SECRET_KEY`
- `STRIPE_WEBHOOK_SECRET`

Required environment variables:

- `CORS_ALLOWED_ORIGINS`
- `APP_ENV=production`
- `QUEUE_QUESTION_GENERATION`
- `QUEUE_HIGHLIGHT_IMPORT`
- `TASK_HANDLER_BASE_URL`
- `INTERNAL_TASK_INVOKER_SERVICE_ACCOUNT`: service account email used by Cloud
  Tasks OIDC tokens. Leave empty only when using `INTERNAL_TASK_SECRET`.
- `SHUTDOWN_TIMEOUT_SECONDS=90`

With `--ingress=internal-and-cloud-load-balancing`, Cloud Tasks can call the
default `run.app` URL from the same project. Use an external Application Load
Balancer hostname only when task traffic is intentionally routed through that
load balancer.

For Neon, use the `-pooler` runtime URL in `DATABASE_URL`; use the direct Neon
URL only for migrations through `MIGRATION_DATABASE_URL`. Keep
`DB_MAX_OPEN_CONNS * max-instances` below the effective Neon/PgBouncer
connection capacity. The current default of
`DB_MAX_OPEN_CONNS=10` with `--max-instances=10` caps application-side database
connections at roughly `100`.

## Runtime Service Account IAM

Grant the Cloud Run runtime service account only the resources the API uses:

- `roles/secretmanager.secretAccessor` on `DATABASE_URL`, `GEMINI_API_KEY`, and
  `INTERNAL_TASK_SECRET`, plus any enabled Stripe secrets.
- `roles/cloudtasks.enqueuer` on the `question-generation` and
  `highlight-import` Cloud Tasks queues.
- `roles/logging.logWriter` if log client libraries are used directly.

Use `deploy/setup-service-accounts.sh` after creating the required secrets and
Cloud Tasks queues.

## Health Check

Use liveness checks for process health:

```text
GET /health
```

Use readiness checks before routing real traffic or during manual verification:

```text
GET /ready
```

`/health` should not require Firebase Auth and should not touch expensive
dependencies. `/ready` may return `503` when the database is unavailable.

## Manual Deploy Snippet

```sh
PROJECT_ID="your-gcp-project"
REGION="asia-northeast1"
SERVICE="api-service"
IMAGE="asia-northeast1-docker.pkg.dev/${PROJECT_ID}/ai-study-tool/${SERVICE}:TAG"
RUNTIME_SA="cloud-run-api@${PROJECT_ID}.iam.gserviceaccount.com"

gcloud run deploy "${SERVICE}" \
  --project="${PROJECT_ID}" \
  --region="${REGION}" \
  --image="${IMAGE}" \
  --port=8080 \
  --service-account="${RUNTIME_SA}" \
  --max-instances=10 \
  --concurrency=80 \
  --cpu=1 \
  --memory=512Mi \
  --timeout=120s \
  --ingress=internal-and-cloud-load-balancing \
  --set-env-vars="CORS_ALLOWED_ORIGINS=https://YOUR_FRONTEND_DOMAIN,APP_ENV=production,QUEUE_QUESTION_GENERATION=projects/${PROJECT_ID}/locations/${REGION}/queues/question-generation,QUEUE_HIGHLIGHT_IMPORT=projects/${PROJECT_ID}/locations/${REGION}/queues/highlight-import,TASK_HANDLER_BASE_URL=https://YOUR_SERVICE_HOST,INTERNAL_TASK_INVOKER_SERVICE_ACCOUNT=cloud-tasks-invoker@${PROJECT_ID}.iam.gserviceaccount.com,SHUTDOWN_TIMEOUT_SECONDS=90" \
  --set-secrets="DATABASE_URL=DATABASE_URL:latest,GEMINI_API_KEY=GEMINI_API_KEY:latest,INTERNAL_TASK_SECRET=INTERNAL_TASK_SECRET:latest"
```

If Cloudflare is the only public entrypoint, deploy with authenticated invoker
instead of public access:

```sh
gcloud run deploy "${SERVICE}" \
  --project="${PROJECT_ID}" \
  --region="${REGION}" \
  --image="${IMAGE}" \
  --no-allow-unauthenticated \
  --max-instances=10 \
  --concurrency=80 \
  --timeout=120s \
  --set-env-vars="SHUTDOWN_TIMEOUT_SECONDS=90"
```

## GitHub Actions Update

The deploy workflow should include the same flags:

```yaml
flags: >-
  --port=8080
  --allow-unauthenticated
  --ingress=internal-and-cloud-load-balancing
  --service-account=${{ vars.CLOUD_RUN_RUNTIME_SERVICE_ACCOUNT }}
  --max-instances=10
  --concurrency=80
  --cpu=1
  --memory=512Mi
  --timeout=120s
env_vars: |-
  SHUTDOWN_TIMEOUT_SECONDS=90
```

When Cloudflare authenticated origin access is ready, replace
`--allow-unauthenticated` with the chosen authenticated invoker setup.

## Verification

```sh
gcloud run services describe "${SERVICE}" \
  --project="${PROJECT_ID}" \
  --region="${REGION}" \
  --format="value(spec.template.spec.containerConcurrency,spec.template.metadata.annotations.autoscaling.knative.dev/maxScale)"

curl --fail "https://YOUR_SERVICE_URL/health"
curl --fail "https://YOUR_SERVICE_URL/ready"
```
