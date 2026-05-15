# Cloud Run Rollback Runbook

Use this when a deployment fails readiness checks or a production regression
needs immediate traffic rollback.

## 1. Identify revisions

```sh
PROJECT_ID="your-gcp-project"
REGION="asia-northeast1"
SERVICE="api-service"

gcloud run revisions list \
  --service="${SERVICE}" \
  --region="${REGION}" \
  --project="${PROJECT_ID}" \
  --format="table(metadata.name,status.conditions[0].status,metadata.creationTimestamp)"
```

## 2. Move traffic back

Choose the last known-good revision and move all traffic to it:

```sh
REVISION_NAME="api-service-00042-abc"

gcloud run services update-traffic "${SERVICE}" \
  --to-revisions="${REVISION_NAME}=100" \
  --region="${REGION}" \
  --project="${PROJECT_ID}"
```

## 3. Verify

```sh
SERVICE_URL="$(gcloud run services describe "${SERVICE}" \
  --region="${REGION}" \
  --project="${PROJECT_ID}" \
  --format='value(status.url)')"

curl --fail "${SERVICE_URL}/health"
curl --fail "${SERVICE_URL}/ready"
```

## Notes

- Database migrations are forward-only. Roll back traffic first, then decide
  whether a compensating migration is needed.
- The deploy workflow also attempts this traffic rollback automatically when a
  new revision fails the post-deploy readiness check.
