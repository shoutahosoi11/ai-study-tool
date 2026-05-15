#!/usr/bin/env bash
# Cloud Monitoring ダッシュボードとアラートポリシーをセットアップする。
# 冪等に実行可能（既存リソースは更新される）。
#
# 使用例:
#   PROJECT_ID=gen-lang-client-0677093718 \
#   SERVICE_NAME=your-cloud-run-service \
#   NOTIFICATION_CHANNEL=projects/PROJ/notificationChannels/CHANNEL_ID \
#   ALERT_EMAIL=your@email.com \
#   bash deploy/monitoring/setup-monitoring.sh

set -euo pipefail

PROJECT_ID="${PROJECT_ID:?Please set PROJECT_ID}"
SERVICE_NAME="${SERVICE_NAME:?Please set SERVICE_NAME}"
ALERT_EMAIL="${ALERT_EMAIL:-}"
NOTIFICATION_CHANNEL="${NOTIFICATION_CHANNEL:-}"
QUESTION_QUEUE_ID="${QUESTION_QUEUE_ID:-${QUEUE_QUESTION_GENERATION:-question-generation}}"
HIGHLIGHT_QUEUE_ID="${HIGHLIGHT_QUEUE_ID:-${QUEUE_HIGHLIGHT_IMPORT:-highlight-import}}"
QUESTION_QUEUE_ID="${QUESTION_QUEUE_ID##*/}"
HIGHLIGHT_QUEUE_ID="${HIGHLIGHT_QUEUE_ID##*/}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WORK_DIR="$(mktemp -d)"
trap 'rm -rf "${WORK_DIR}"' EXIT
DASHBOARD_CONFIG="${WORK_DIR}/dashboard.json"

echo "==> Cloud Monitoring setup: project=${PROJECT_ID} service=${SERVICE_NAME}"
echo "    Question queue: ${QUESTION_QUEUE_ID}"
echo "    Highlight queue: ${HIGHLIGHT_QUEUE_ID}"

# ── Step 1: API 有効化 ─────────────────────────────────────────────
echo "--> Enabling Monitoring API..."
gcloud services enable monitoring.googleapis.com --project="$PROJECT_ID"

# ── Step 2: 通知チャンネル作成（メール指定時） ────────────���────────
if [ -n "$ALERT_EMAIL" ] && [ -z "$NOTIFICATION_CHANNEL" ]; then
  echo "--> Resolving email notification channel for ${ALERT_EMAIL}..."
  NOTIFICATION_CHANNEL=$(gcloud beta monitoring channels list \
    --project="$PROJECT_ID" \
    --filter="type=\"email\" labels.email_address=\"${ALERT_EMAIL}\"" \
    --format="value(name)" \
    --limit=1)
  if [ -n "$NOTIFICATION_CHANNEL" ]; then
    echo "    Reusing channel: ${NOTIFICATION_CHANNEL}"
  else
    NOTIFICATION_CHANNEL=$(gcloud beta monitoring channels create \
      --project="$PROJECT_ID" \
      --display-name="AI Study Tool Alerts" \
      --type=email \
      --channel-labels="email_address=${ALERT_EMAIL}" \
      --format="value(name)")
    echo "    Created channel: ${NOTIFICATION_CHANNEL}"
  fi
fi

# ── Step 3: ダッシュボード作成 ────────────────��────────────────────
echo "--> Creating dashboard..."
sed \
  -e "s/__SERVICE_NAME__/${SERVICE_NAME//\//\\/}/g" \
  -e "s/__QUESTION_QUEUE_ID__/${QUESTION_QUEUE_ID//\//\\/}/g" \
  -e "s/__HIGHLIGHT_QUEUE_ID__/${HIGHLIGHT_QUEUE_ID//\//\\/}/g" \
  "${SCRIPT_DIR}/dashboard.json" > "$DASHBOARD_CONFIG"
if gcloud monitoring dashboards list --project="$PROJECT_ID" \
    --filter="displayName='AI Study Tool API'" --format="value(name)" | grep -q .; then
  DASHBOARD_NAME=$(gcloud monitoring dashboards list --project="$PROJECT_ID" \
    --filter="displayName='AI Study Tool API'" --format="value(name)")
  gcloud monitoring dashboards update "$DASHBOARD_NAME" \
    --project="$PROJECT_ID" \
    --config-from-file="$DASHBOARD_CONFIG"
  echo "    Updated: ${DASHBOARD_NAME}"
else
  gcloud monitoring dashboards create \
    --project="$PROJECT_ID" \
    --config-from-file="$DASHBOARD_CONFIG"
  echo "    Created dashboard."
fi

# ── Step 4: アラートポリシー作成 ──────────────────────���────────────
if [ -z "$NOTIFICATION_CHANNEL" ]; then
  echo "WARN: NOTIFICATION_CHANNEL not set. Alerts will be created without notifications."
fi

# Filter に SERVICE_NAME を埋め込む
RESOURCE_FILTER="resource.type=\"cloud_run_revision\" resource.labels.service_name=\"${SERVICE_NAME}\""

create_alert() {
  local name="$1"
  local display_name="$2"
  local filter="$3"
  local aligner="$4"
  local reducer="$5"
  local comparison="$6"
  local threshold="$7"
  local duration="$8"
  local severity="$9"
  local policy_file="${WORK_DIR}/${name}.json"

  local channels_json="[]"
  if [ -n "$NOTIFICATION_CHANNEL" ]; then
    channels_json="[\"${NOTIFICATION_CHANNEL}\"]"
  fi

  cat > "$policy_file" <<EOF
{
  "displayName": "${display_name}",
  "severity": "${severity}",
  "conditions": [
    {
      "displayName": "${display_name}",
      "conditionThreshold": {
        "filter": "${filter}",
        "aggregations": [
          {
            "alignmentPeriod": "300s",
            "perSeriesAligner": "${aligner}",
            "crossSeriesReducer": "${reducer}"
          }
        ],
        "comparison": "${comparison}",
        "thresholdValue": ${threshold},
        "duration": "${duration}",
        "trigger": { "count": 1 }
      }
    }
  ],
  "notificationChannels": ${channels_json},
  "alertStrategy": {
    "autoClose": "604800s"
  }
}
EOF

  local policy_name
  policy_name=$(gcloud monitoring policies list \
    --project="$PROJECT_ID" \
    --filter="displayName=\"${display_name}\"" \
    --format="value(name)" \
    --limit=1)
  if [ -n "$policy_name" ]; then
    gcloud monitoring policies update "$policy_name" \
      --project="$PROJECT_ID" \
      --policy-from-file="$policy_file"
    echo "    [UPDATED] ${display_name}"
  else
    gcloud monitoring policies create \
      --project="$PROJECT_ID" \
      --policy-from-file="$policy_file"
    echo "    [CREATED] ${display_name}"
  fi
}

echo "--> Creating alert policies..."

# エラー率 > 5% (5分間)
create_alert \
  "error-rate" \
  "[AI Study Tool] High 5xx Error Rate" \
  "${RESOURCE_FILTER} metric.type=\"run.googleapis.com/request_count\" metric.labels.response_code_class=\"5xx\"" \
  "ALIGN_RATE" \
  "REDUCE_SUM" \
  "COMPARISON_GT" \
  "0.05" \
  "300s" \
  "CRITICAL"

# P99 レイテンシ > 5000ms (5分間)
create_alert \
  "latency-p99" \
  "[AI Study Tool] High P99 Latency" \
  "${RESOURCE_FILTER} metric.type=\"run.googleapis.com/request_latencies\"" \
  "ALIGN_PERCENTILE_99" \
  "REDUCE_MEAN" \
  "COMPARISON_GT" \
  "5000" \
  "300s" \
  "WARNING"

# インスタンス数 > 8 (5分間)
create_alert \
  "instance-count" \
  "[AI Study Tool] High Instance Count" \
  "${RESOURCE_FILTER} metric.type=\"run.googleapis.com/container/instance_count\"" \
  "ALIGN_MAX" \
  "REDUCE_MAX" \
  "COMPARISON_GT" \
  "8" \
  "300s" \
  "WARNING"

echo ""
echo "==> Done."
echo ""
echo "Dashboard: https://console.cloud.google.com/monitoring/dashboards?project=${PROJECT_ID}"
echo "Alerts:    https://console.cloud.google.com/monitoring/alerting?project=${PROJECT_ID}"
