# Phase 1 Deploy: Cloud Run + Neon PostgreSQL + Firebase Auth + Cloud Storage

This guide sets up the first deployable production-like environment:

- Cloud Run: `api-service`
- Neon PostgreSQL: managed Postgres connection through `DATABASE_URL`
- Firebase Auth: frontend issues ID tokens, backend verifies them
- Cloud Storage: upload/download signed URLs
- GitHub Actions: build, test, push image, deploy Cloud Run

Neon works with a standard PostgreSQL connection string. For the Cloud Run app,
use the pooled Neon connection string when possible. For migrations, use the
direct/unpooled connection string.

## 1. Create Google Cloud resources

Choose values:

```sh
PROJECT_ID="your-gcp-project"
REGION="asia-northeast1"
SERVICE="api-service"
GAR_REPOSITORY="ai-study-tool"
BUCKET_NAME="${PROJECT_ID}-ai-study-tool-uploads"
```

Enable APIs:

```sh
gcloud services enable \
  run.googleapis.com \
  artifactregistry.googleapis.com \
  secretmanager.googleapis.com \
  iamcredentials.googleapis.com \
  storage.googleapis.com \
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

Create Cloud Storage bucket:

```sh
gcloud storage buckets create "gs://${BUCKET_NAME}" \
  --location="${REGION}" \
  --uniform-bucket-level-access \
  --project="${PROJECT_ID}"
```

Apply CORS for browser uploads through signed URLs:

```sh
cp deploy/storage-cors.example.json /tmp/storage-cors.json
# Replace YOUR_FRONTEND_DOMAIN in /tmp/storage-cors.json.
gcloud storage buckets update "gs://${BUCKET_NAME}" \
  --cors-file=/tmp/storage-cors.json \
  --project="${PROJECT_ID}"
```

## 2. Create Neon PostgreSQL

In the Neon console:

1. Create a project.
2. Open the project dashboard.
3. Click **Connect**.
4. Copy the pooled connection string for the app runtime.
5. Copy the direct/unpooled connection string for migrations.

The runtime string should look like this:

```text
postgresql://USER:PASSWORD@HOST-pooler.REGION.aws.neon.tech/DB?sslmode=require&channel_binding=require
```

The migration string should not contain `-pooler` in the hostname.

## 3. Store secrets

Create Secret Manager secrets. Use the pooled Neon connection string for
`DATABASE_URL`.

```sh
printf '%s' 'PASTE_NEON_POOLED_DATABASE_URL_HERE' | gcloud secrets create DATABASE_URL \
  --data-file=- \
  --project="${PROJECT_ID}"

printf '%s' 'PASTE_GEMINI_API_KEY_HERE' | gcloud secrets create GEMINI_API_KEY \
  --data-file=- \
  --project="${PROJECT_ID}"
```

If a secret already exists, add a new version:

```sh
printf '%s' 'PASTE_NEON_POOLED_DATABASE_URL_HERE' | gcloud secrets versions add DATABASE_URL \
  --data-file=- \
  --project="${PROJECT_ID}"

printf '%s' 'PASTE_GEMINI_API_KEY_HERE' | gcloud secrets versions add GEMINI_API_KEY \
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
gcloud projects add-iam-policy-binding "${PROJECT_ID}" \
  --member="serviceAccount:${RUNTIME_SA}" \
  --role="roles/secretmanager.secretAccessor"

gcloud storage buckets add-iam-policy-binding "gs://${BUCKET_NAME}" \
  --member="serviceAccount:${RUNTIME_SA}" \
  --role="roles/storage.objectAdmin"

gcloud iam service-accounts add-iam-policy-binding "${RUNTIME_SA}" \
  --member="serviceAccount:${RUNTIME_SA}" \
  --role="roles/iam.serviceAccountTokenCreator" \
  --project="${PROJECT_ID}"
```

The final binding lets Cloud Storage generate V4 signed URLs from Cloud Run.

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

gcloud projects add-iam-policy-binding "${PROJECT_ID}" \
  --member="serviceAccount:${DEPLOYER_SA}" \
  --role="roles/secretmanager.secretAccessor"
```

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
| `GCS_BUCKET_NAME` | `your-bucket-name` |
| `GCS_SIGNING_SERVICE_ACCOUNT` | `cloud-run-api@project.iam.gserviceaccount.com` |

## 6. Run migrations

For Phase 1, run migrations manually using the direct Neon connection string.
Avoid using the pooled connection string for migrations.

```sh
DIRECT_DATABASE_URL='PASTE_NEON_DIRECT_DATABASE_URL_HERE'

for file in backend/db/migrations/*.sql; do
  psql "$DIRECT_DATABASE_URL" -f "$file"
done
```

## 7. First deploy

Push to `main` or run the `Deploy API to Cloud Run` workflow manually.

After the service exists, allow browser access to the public API. The backend
still requires Firebase ID tokens on `/api/*`.

```sh
gcloud run services add-iam-policy-binding "${SERVICE}" \
  --region="${REGION}" \
  --member="allUsers" \
  --role="roles/run.invoker" \
  --project="${PROJECT_ID}"
```

Add the deployed frontend domain to:

- Firebase Auth authorized domains
- `CORS_ALLOWED_ORIGINS`
- Cloud Storage bucket CORS

## Notes

- Locally, `FIREBASE_CREDENTIALS_PATH` can point at a service account JSON.
- On Cloud Run, Firebase Admin uses the runtime service account through Application Default Credentials.
- Signed uploads must send the exact `Content-Type` returned by `/api/storage/signed-urls/upload`.
