#!/usr/bin/env bash
# Cloud Run Runtime SA の IAM 権限をセットアップする。
# 既存の権限は上書きされないため冪等に実行可能。
#
# 使用例:
#   PROJECT_ID=gen-lang-client-0677093718 \
#   SA_EMAIL=your-sa@gen-lang-client-0677093718.iam.gserviceaccount.com \
#   LOCATION=asia-northeast1 \
#   bash deploy/setup-service-accounts.sh

set -euo pipefail

PROJECT_ID="${PROJECT_ID:?Please set PROJECT_ID}"
SA_EMAIL="${SA_EMAIL:?Please set SA_EMAIL}"
LOCATION="${LOCATION:-asia-northeast1}"
QUESTION_QUEUE="${QUESTION_QUEUE:-question-generation}"
HIGHLIGHT_QUEUE="${HIGHLIGHT_QUEUE:-highlight-import}"
INTERNAL_TASK_INVOKER_SERVICE_ACCOUNT="${INTERNAL_TASK_INVOKER_SERVICE_ACCOUNT:-}"

grant_secret_accessor() {
  local secret_name="$1"
  local required="$2"

  if ! gcloud secrets describe "$secret_name" --project="$PROJECT_ID" >/dev/null 2>&1; then
    if [ "$required" = "true" ]; then
      echo "    [ERROR] required secret does not exist: ${secret_name}" >&2
      exit 1
    fi

    echo "    [SKIP] optional secret does not exist: ${secret_name}"
    return
  fi

  gcloud secrets add-iam-policy-binding "$secret_name" \
    --project="$PROJECT_ID" \
    --member="serviceAccount:${SA_EMAIL}" \
    --role="roles/secretmanager.secretAccessor" \
    --condition=None
  echo "    [OK] ${secret_name}: roles/secretmanager.secretAccessor"
}

grant_queue_enqueuer() {
  local queue_name="$1"

  if ! gcloud tasks queues describe "$queue_name" \
    --project="$PROJECT_ID" \
    --location="$LOCATION" >/dev/null 2>&1; then
    echo "    [ERROR] Cloud Tasks queue not found; create it first: ${queue_name}" >&2
    exit 1
  fi

  gcloud tasks queues add-iam-policy-binding "$queue_name" \
    --project="$PROJECT_ID" \
    --location="$LOCATION" \
    --member="serviceAccount:${SA_EMAIL}" \
    --role="roles/cloudtasks.enqueuer" \
    --condition=None
  echo "    [OK] ${queue_name}: roles/cloudtasks.enqueuer"
}

echo "==> Setting up IAM roles for: ${SA_EMAIL}"
echo "    Project: ${PROJECT_ID}"
echo "    Location: ${LOCATION}"

# Secret Manager へのアクセスはシークレット単位に限定する。
grant_secret_accessor "DATABASE_URL" "true"
grant_secret_accessor "GEMINI_API_KEY" "true"
if [ -z "$INTERNAL_TASK_INVOKER_SERVICE_ACCOUNT" ]; then
  grant_secret_accessor "INTERNAL_TASK_SECRET" "true"
else
  grant_secret_accessor "INTERNAL_TASK_SECRET" "false"
fi
grant_secret_accessor "STRIPE_SECRET_KEY" "false"
grant_secret_accessor "STRIPE_WEBHOOK_SECRET" "false"

# Cloud Tasks enqueue 権限は queue 単位に限定する。
grant_queue_enqueuer "$QUESTION_QUEUE"
grant_queue_enqueuer "$HIGHLIGHT_QUEUE"

# Cloud Logging への書き込み（slog + Cloud Logging 連携）
gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:${SA_EMAIL}" \
  --role="roles/logging.logWriter" \
  --condition=None
echo "    [OK] roles/logging.logWriter"

echo ""
echo "==> Done. Current bindings for ${SA_EMAIL}:"
gcloud projects get-iam-policy "$PROJECT_ID" \
  --flatten="bindings[].members" \
  --filter="bindings.members:serviceAccount:${SA_EMAIL}" \
  --format="table(bindings.role)"
