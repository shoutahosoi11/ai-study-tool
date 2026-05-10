# Cloud Tasks Migration Design

## Summary

問題生成とハイライトインポートを Cloud Tasks に統一する。
どちらも「1 task = 1 DB row」とし、DB 側の status/CAS を正とする。

## Before

- 問題生成: Cloud Run プロセス内 goroutine + `question_generation_jobs`
- ハイライトインポート: Cloud Run Job + `highlight_import_queue`

## Problems

- in-process goroutine の semaphore は Cloud Run インスタンス単位でしか効かないため、スケールアウト時に Gemini RPM を超えやすい。
- Cloud Run Job は同時リクエストで execution が複数起動し、DB キューの claim 競合が増える。

## After

- 問題生成は `QUEUE_QUESTION_GENERATION` に HTTP task を enqueue する。
- ハイライトインポートは `QUEUE_HIGHLIGHT_IMPORT` に HTTP task を enqueue する。
- Cloud Tasks は `/internal/tasks/question-generation` と `/internal/tasks/highlight-import` を叩く。
- worker 処理は既存 usecase を呼び、DB の queued/processing/completed/failed ステートを維持する。

## Queue Settings

`deploy/cloudtasks/setup-queues.sh` で作成する。

- `question-generation`
  - `--max-dispatches-per-second=0.25`
  - `--max-concurrent-dispatches=3`
  - `--max-attempts=3`
  - Gemini 呼び出しのレートを Cloud Tasks 側で抑える。
- `highlight-import`
  - `--max-dispatches-per-second=5`
  - `--max-concurrent-dispatches=10`
  - import は DB upsert 中心なので問題生成より高めにする。

## Environment Variables

Cloud Run に以下を追加する。

```text
QUEUE_QUESTION_GENERATION=projects/<project>/locations/asia-northeast1/queues/question-generation
QUEUE_HIGHLIGHT_IMPORT=projects/<project>/locations/asia-northeast1/queues/highlight-import
TASK_HANDLER_BASE_URL=https://<api-service>.run.app
```

未設定の場合、enqueuer は nil/no-op として動く。ローカル開発では build/run を壊さない。

## Internal Endpoints

```text
POST /internal/tasks/question-generation
POST /internal/tasks/highlight-import
```

現時点では Cloud Run の `--ingress=internal-and-cloud-load-balancing` を前提に保護する。
将来、Cloud Tasks OIDC token の検証を middleware として追加できる。

既存サービスへ手動適用する場合:

```bash
gcloud run services update SERVICE \
  --region=asia-northeast1 \
  --ingress=internal-and-cloud-load-balancing
```

## Idempotency

- 問題生成は `question_generation_jobs` の `ClaimQueued` が queued -> processing を CAS する。
- ハイライトインポートは `highlight_import_queue` の `ClaimProcessing` が queued -> processing を CAS する。
- Cloud Tasks retry で同じ task が複数回来ても、DB の CAS に失敗した場合は no-op になる。

## Rollback

コードを戻す場合:

1. Cloud Run の環境変数 `QUEUE_QUESTION_GENERATION` / `QUEUE_HIGHLIGHT_IMPORT` / `TASK_HANDLER_BASE_URL` を削除する。
2. このPR以前の revision に traffic を戻す。
3. Cloud Tasks queue は残してよい。不要なら以下で削除する。

```bash
gcloud tasks queues delete question-generation --location=asia-northeast1
gcloud tasks queues delete highlight-import --location=asia-northeast1
```

DB schema は変更していないため DB rollback は不要。
