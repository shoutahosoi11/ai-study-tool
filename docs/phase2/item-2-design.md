# Item 2: Secret Manager 導入（残りシークレット整備）

## 目的

Stripe の機密キーを Secret Manager で管理し、Cloud Run に安全に注入する。
非機密な設定値は GitHub Actions vars に配置して明確に分離する。

## 現状 (Before)

| 変数名 | 機密度 | 現在の場所 |
|--------|--------|-----------|
| DATABASE_URL | 🔴 | Secret Manager ✅ |
| GEMINI_API_KEY | 🔴 | Secret Manager ✅ |
| STRIPE_SECRET_KEY | 🔴 | **未設定** |
| STRIPE_WEBHOOK_SECRET | 🔴 | **未設定** |
| STRIPE_PRICE_ID_MONTHLY | 🟢 | **未設定** |
| STRIPE_SUCCESS_URL | 🟢 | **未設定** |
| STRIPE_CANCEL_URL | 🟢 | **未設定** |
| APP_ENV | 🟢 | **未設定** |
| CORS_ALLOWED_ORIGINS | 🟢 | GitHub Actions vars ✅ |

## 変更後 (After)

| 変数名 | 保管先 |
|--------|--------|
| STRIPE_SECRET_KEY | Secret Manager |
| STRIPE_WEBHOOK_SECRET | Secret Manager |
| STRIPE_PRICE_ID_MONTHLY | GitHub Actions vars |
| STRIPE_SUCCESS_URL | GitHub Actions vars |
| STRIPE_CANCEL_URL | GitHub Actions vars |
| APP_ENV | GitHub Actions vars（値: `production`） |

## 変更ファイル

| ファイル | 変更内容 |
|---------|---------|
| `.github/workflows/deploy-api.yml` | secrets / env_vars に Stripe・APP_ENV を追加 |
| `backend/.env.example` | Stripe 変数のコメント追加 |

## 新規ファイル

なし（インフラ変更のみ）

## 実装手順

### Step 1: Secret Manager にシークレット作成（手動 gcloud コマンド）

Stripe アカウントで実際のキー取得後に実行する:

```bash
# STRIPE_SECRET_KEY
echo -n "sk_live_YOUR_KEY" | gcloud secrets create STRIPE_SECRET_KEY \
  --project=gen-lang-client-0677093718 \
  --replication-policy=automatic \
  --data-file=-

# STRIPE_WEBHOOK_SECRET
echo -n "whsec_YOUR_SECRET" | gcloud secrets create STRIPE_WEBHOOK_SECRET \
  --project=gen-lang-client-0677093718 \
  --replication-policy=automatic \
  --data-file=-
```

既存シークレットの更新は:
```bash
echo -n "NEW_VALUE" | gcloud secrets versions add SECRET_NAME \
  --project=gen-lang-client-0677093718 \
  --data-file=-
```

### Step 2: Cloud Run SA に Secret Accessor 権限付与

```bash
# Cloud Run Runtime SA のメールを確認
SA_EMAIL=$(gcloud run services describe YOUR_SERVICE \
  --project=gen-lang-client-0677093718 \
  --region=asia-northeast1 \
  --format='value(spec.template.spec.serviceAccountName)')

# 各シークレットへのアクセス権限付与
for SECRET in STRIPE_SECRET_KEY STRIPE_WEBHOOK_SECRET; do
  gcloud secrets add-iam-policy-binding $SECRET \
    --project=gen-lang-client-0677093718 \
    --member="serviceAccount:${SA_EMAIL}" \
    --role="roles/secretmanager.secretAccessor"
done
```

### Step 3: GitHub Actions vars に非機密値を追加

GitHub リポジトリの Settings > Secrets and variables > Actions > Variables で追加:

```
STRIPE_PRICE_ID_MONTHLY = price_XXXXXXXXXX
STRIPE_SUCCESS_URL      = https://yourapp.com/billing/success
STRIPE_CANCEL_URL       = https://yourapp.com/billing/cancel
APP_ENV                 = production
```

### Step 4: deploy-api.yml の更新

```yaml
env_vars: |-
  CORS_ALLOWED_ORIGINS=${{ vars.CORS_ALLOWED_ORIGINS }}
  GCS_BUCKET_NAME=${{ vars.GCS_BUCKET_NAME }}
  GCS_SIGNING_SERVICE_ACCOUNT=${{ vars.GCS_SIGNING_SERVICE_ACCOUNT }}
  APP_ENV=${{ vars.APP_ENV }}
  STRIPE_PRICE_ID_MONTHLY=${{ vars.STRIPE_PRICE_ID_MONTHLY }}
  STRIPE_SUCCESS_URL=${{ vars.STRIPE_SUCCESS_URL }}
  STRIPE_CANCEL_URL=${{ vars.STRIPE_CANCEL_URL }}
secrets: |-
  DATABASE_URL=DATABASE_URL:latest
  GEMINI_API_KEY=GEMINI_API_KEY:latest
  STRIPE_SECRET_KEY=STRIPE_SECRET_KEY:latest
  STRIPE_WEBHOOK_SECRET=STRIPE_WEBHOOK_SECRET:latest
```

また Validate ステップに Stripe 変数のチェックを追加する。

## 動作確認方法

```bash
# Secret Manager に登録確認
gcloud secrets list --project=gen-lang-client-0677093718

# バージョン確認
gcloud secrets versions list STRIPE_SECRET_KEY \
  --project=gen-lang-client-0677093718

# deploy 後に Cloud Run の env var 確認
gcloud run services describe SERVICE_NAME \
  --project=gen-lang-client-0677093718 \
  --region=asia-northeast1 \
  --format='value(spec.template.spec.containers[0].env)'
```

## ハマりポイント

- Secret Manager の `:latest` は最新バージョンを参照する。Rotate 時は新バージョン追加後に Cloud Run を再デプロイ
- Stripe の test key (`sk_test_`) と live key (`sk_live_`) を間違えない
- Secret Accessor 権限はシークレットごとに付与が必要（プロジェクトレベルで付与すると過剰権限）

## 代替案

| 案 | 不採用理由 |
|----|----------|
| GitHub Actions Secrets に Stripe キーを入れる | CI に機密情報を置くリスク、Cloud Run から直接参照できない |
| 全変数を env_vars に入れる | 機密値が Cloud Run の環境変数として平文で見える |
