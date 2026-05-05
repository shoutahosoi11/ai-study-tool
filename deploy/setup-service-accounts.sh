#!/usr/bin/env bash
# Cloud Run Runtime SA の IAM 権限をセットアップする。
# 既存の権限は上書きされないため冪等に実行可能。
#
# 使用例:
#   PROJECT_ID=gen-lang-client-0677093718 \
#   SA_EMAIL=your-sa@gen-lang-client-0677093718.iam.gserviceaccount.com \
#   bash deploy/setup-service-accounts.sh

set -euo pipefail

PROJECT_ID="${PROJECT_ID:?Please set PROJECT_ID}"
SA_EMAIL="${SA_EMAIL:?Please set SA_EMAIL}"

echo "==> Setting up IAM roles for: ${SA_EMAIL}"
echo "    Project: ${PROJECT_ID}"

# Secret Manager へのアクセス（DATABASE_URL, GEMINI_API_KEY, STRIPE_* 等）
gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:${SA_EMAIL}" \
  --role="roles/secretmanager.secretAccessor" \
  --condition=None
echo "    [OK] roles/secretmanager.secretAccessor"

# Cloud Logging への書き込み（slog + Cloud Logging 連携）
gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:${SA_EMAIL}" \
  --role="roles/logging.logWriter" \
  --condition=None
echo "    [OK] roles/logging.logWriter"

# Cloud Run Job 起動（highlight-importer Job のトリガー）
gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:${SA_EMAIL}" \
  --role="roles/run.developer" \
  --condition=None
echo "    [OK] roles/run.developer"

echo ""
echo "==> Done. Current bindings for ${SA_EMAIL}:"
gcloud projects get-iam-policy "$PROJECT_ID" \
  --flatten="bindings[].members" \
  --filter="bindings.members:serviceAccount:${SA_EMAIL}" \
  --format="table(bindings.role)"
