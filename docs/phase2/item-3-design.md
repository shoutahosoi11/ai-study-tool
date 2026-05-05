# Item 3: サービスアカウント整備（最小権限）

## 目的

Cloud Run が動作するために必要な最小限の IAM 権限を定義し、
不要な権限（GCS 関連など）を排除する。

## 現状 (Before)

- `CLOUD_RUN_RUNTIME_SERVICE_ACCOUNT`: GitHub Actions var に設定済みだが権限詳細不明
- `GCS_SIGNING_SERVICE_ACCOUNT`: コード上は不使用（storage_usecase.go 削除済み）なのに deploy に残存
- `GCS_BUCKET_NAME`: 同上
- SA の権限が設計書として存在しない

## 変更後 (After)

### Cloud Run Runtime SA が持つべき権限

| ロール | 理由 |
|--------|------|
| `roles/secretmanager.secretAccessor` | DATABASE_URL 等の Secret Manager 読み取り |
| `roles/logging.logWriter` | Cloud Logging への書き込み（項目4で使用） |
| `roles/run.developer` | highlight-importer Cloud Run Job の起動 |

### Deployer SA（GitHub Actions）が持つべき権限

| ロール | 理由 |
|--------|------|
| `roles/run.admin` | Cloud Run サービスのデプロイ |
| `roles/iam.serviceAccountUser` | Runtime SA として動作するための権限 |
| `roles/artifactregistry.writer` | Docker イメージのプッシュ |

## 変更ファイル

| ファイル | 変更内容 |
|---------|---------|
| `.github/workflows/deploy-api.yml` | GCS 関連の不要な vars と validate チェックを削除 |
| `deploy/setup-service-accounts.sh` | SA 作成・権限付与のセットアップスクリプト（新規） |

## 新規ファイル

| ファイル | 内容 |
|---------|------|
| `deploy/setup-service-accounts.sh` | idempotent なSA セットアップスクリプト |

## 実装手順

### Step 1: deploy-api.yml から GCS 不要設定を削除

削除対象:
- `vars.GCS_BUCKET_NAME` の validate チェックと env_vars への注入
- `vars.GCS_SIGNING_SERVICE_ACCOUNT` の validate チェックと env_vars への注入

### Step 2: deploy/setup-service-accounts.sh 作成

```bash
#!/usr/bin/env bash
# Cloud Run SA と権限のセットアップ
# 使用例: PROJECT_ID=xxx SA_NAME=xxx bash deploy/setup-service-accounts.sh
set -euo pipefail

PROJECT_ID="${PROJECT_ID:?}"
SA_NAME="${SA_NAME:-cloud-run-api}"  # デフォルト名（実際の値に合わせて変更）
SA_EMAIL="${SA_NAME}@${PROJECT_ID}.iam.gserviceaccount.com"

# Secret Accessor
gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:${SA_EMAIL}" \
  --role="roles/secretmanager.secretAccessor" \
  --condition=None

# Log Writer（構造化ログ対応のため）
gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:${SA_EMAIL}" \
  --role="roles/logging.logWriter" \
  --condition=None

# Cloud Run Developer（Job 起動のため）
gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:${SA_EMAIL}" \
  --role="roles/run.developer" \
  --condition=None
```

### Step 3: 現在の権限確認と過剰権限の除去

```bash
# 現在の Runtime SA の権限確認
SA_EMAIL="YOUR_RUNTIME_SA@gen-lang-client-0677093718.iam.gserviceaccount.com"
gcloud projects get-iam-policy gen-lang-client-0677093718 \
  --flatten="bindings[].members" \
  --filter="bindings.members:${SA_EMAIL}" \
  --format="table(bindings.role)"
```

## ハマりポイント

- `roles/run.developer` は Job の `run` だけでなく更新・削除も含む。より厳密にするなら `roles/run.invoker` + Job 実行権限のカスタムロールにするが、運用コストが上がるため今回は `developer` で許容
- `--condition=None` は条件なしバインドの明示的指定。省略すると gcloud が対話的に確認を求める場合がある

## 代替案

| 案 | 不採用理由 |
|----|----------|
| カスタムロール | 管理コスト大、運用初期には不要 |
| プロジェクトレベルの secretAccessor | 全シークレットへのアクセスになり過剰 |
