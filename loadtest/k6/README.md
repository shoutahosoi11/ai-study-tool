# k6 負荷テスト

この script 群はローカルまたはステージング環境向けの低レート smoke test です。rate limit、queue control、認証動作、webhook rejection path を確認するためのものであり、攻撃用 tooling ではありません。明示的に許可しない限り本番に向けて実行しないでください。

## k6 のインストール

macOS:

```bash
brew install k6
```

その他の platform: https://k6.io/docs/get-started/installation/

## 安全制御

すべての script は環境変数 `BASE_URL` を読みます。`BASE_URL` が本番 host に見える場合、次が設定されていなければ script は終了します。

```bash
ALLOW_PRODUCTION_LOADTEST=true
```

既定ではステージング / ローカル URL を使います。

```bash
BASE_URL=http://localhost:8080 k6 run loadtest/k6/webhook_admob_invalid.js
```

## script 一覧

| Script | 目的 | 必須環境変数 |
| --- | --- | --- |
| `auth_session.js` | 低レートの `/api/v1/auth/session` session 作成確認 | `BASE_URL`, `ID_TOKEN` |
| `extension_import.js` | extension import rate-limit smoke | `BASE_URL`, `EXTENSION_TOKEN` |
| `question_generation_queue.js` | queue / generation endpoint の guardrail 確認 | `BASE_URL`, `AUTH_TOKEN`, `QUESTION_GENERATION_LOADTEST_ENABLED=true` |
| `webhook_admob_invalid.js` | 不正な AdMob SSV の rejection / rate-limit 確認 | `BASE_URL` |

## 実行例

```bash
BASE_URL=http://localhost:8080 ID_TOKEN=test-id-token \
  k6 run loadtest/k6/auth_session.js

BASE_URL=https://staging-api.example.com EXTENSION_TOKEN=test-extension-token \
  k6 run loadtest/k6/extension_import.js

BASE_URL=https://staging-api.example.com AUTH_TOKEN=test-firebase-token \
  QUESTION_GENERATION_LOADTEST_ENABLED=true \
  k6 run loadtest/k6/question_generation_queue.js

BASE_URL=http://localhost:8080 \
  k6 run loadtest/k6/webhook_admob_invalid.js
```

## 問題生成に関する警告

現時点では、問題生成専用の dry-run endpoint はありません。`question_generation_queue.js` は、環境によって実際の生成、Cloud Tasks enqueue、LLM 呼び出しを起こす可能性があるため、既定では無効です。`USE_GEMINI_MOCK=true` と少数の test user を用意したステージングを優先してください。

## ログに出してはいけないもの

実 token、cookie、signature、生 webhook payload、prompt、ハイライト本文を issue や共有ログに貼らないでください。
