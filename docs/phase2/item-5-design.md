# Item 5: Cloud Monitoring ダッシュボード + アラートポリシー

## 目的

Cloud Run の組み込みメトリクスを使い、エラー率・レイテンシ・QPS を可視化する。
閾値超過時にアラートを飛ばす。コード変更は不要（Cloud Run が自動でメトリクスを送信）。

## 使用する Cloud Run 組み込みメトリクス

| メトリクス | 用途 |
|-----------|------|
| `run.googleapis.com/request_count` | QPS・ステータスコード別件数 |
| `run.googleapis.com/request_latencies` | P50/P95/P99 レイテンシ |
| `run.googleapis.com/container/instance_count` | インスタンス数 |
| `run.googleapis.com/container/cpu/utilizations` | CPU 使用率 |
| `run.googleapis.com/container/memory/utilizations` | メモリ使用率 |

## 変更ファイル

なし（コード変更不要）

## 新規ファイル

| ファイル | 内容 |
|---------|------|
| `deploy/monitoring/dashboard.json` | ダッシュボード定義 |
| `deploy/monitoring/setup-monitoring.sh` | ダッシュボード・アラート作成スクリプト |

## アラートポリシー設計

### ポリシー1: エラー率（5xx）

| 項目 | 値 |
|------|---|
| 条件 | 5xx レスポンス率 > 5%（5分間ウィンドウ） |
| 重大度 | CRITICAL |
| 通知 | メール |
| 根拠 | 正常時の 5xx はほぼゼロ。5% 超えはサービス障害 |

### ポリシー2: P99 レイテンシ

| 項目 | 値 |
|------|---|
| 条件 | P99 レイテンシ > 5000ms（5分間ウィンドウ） |
| 重大度 | WARNING |
| 通知 | メール |
| 根拠 | Cloud Run タイムアウトが 60s。5s 超えは要調査 |

### ポリシー3: インスタンス急増

| 項目 | 値 |
|------|---|
| 条件 | インスタンス数 > 8（5分間ウィンドウ） |
| 重大度 | WARNING |
| 通知 | メール |
| 根拠 | max-instances=10。8 を超えたら上限に近づいている |

## セットアップ手順

### Step 1: Cloud Monitoring API 有効化

```bash
gcloud services enable monitoring.googleapis.com \
  --project=gen-lang-client-0677093718
```

### Step 2: アラート通知チャンネル作成（メール）

```bash
gcloud beta monitoring channels create \
  --project=gen-lang-client-0677093718 \
  --display-name="API Alerts" \
  --type=email \
  --channel-labels=email_address=YOUR_EMAIL
```

### Step 3: スクリプト実行

```bash
PROJECT_ID=gen-lang-client-0677093718 \
SERVICE_NAME=YOUR_CLOUD_RUN_SERVICE \
NOTIFICATION_CHANNEL=projects/gen-lang-client-0677093718/notificationChannels/CHANNEL_ID \
bash deploy/monitoring/setup-monitoring.sh
```

## ハマりポイント

- Cloud Run の request_count は `response_code_class` ラベルで `2xx`/`4xx`/`5xx` に分類される
- アラートの `comparison` は `COMPARISON_GT`（greater than）を使う
- ダッシュボード JSON は Cloud Console からエクスポートしたものをベースにすると楽
- 通知チャンネル ID はスクリプト実行前に取得が必要
