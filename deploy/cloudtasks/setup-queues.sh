#!/usr/bin/env bash
set -euo pipefail

PROJECT_ID="${PROJECT_ID:?}"
LOCATION="${LOCATION:-asia-northeast1}"

gcloud tasks queues create question-generation \
  --project="$PROJECT_ID" \
  --location="$LOCATION" \
  --max-dispatches-per-second=0.25 \
  --max-concurrent-dispatches=3 \
  --max-attempts=3 \
  --min-backoff=30s \
  --max-backoff=600s

gcloud tasks queues create highlight-import \
  --project="$PROJECT_ID" \
  --location="$LOCATION" \
  --max-dispatches-per-second=5 \
  --max-concurrent-dispatches=10 \
  --max-attempts=3
