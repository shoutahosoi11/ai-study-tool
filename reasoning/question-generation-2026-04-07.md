# 問題生成API 設計根拠

## 現在の目的

Kindle ハイライトを中心に、ユーザーが学習画面を開いた時点で必要な分だけ AI 問題を用意する。ブラウザ版・モバイル版とも、問題を解く画面では既に準備済みの問題を取得することを基本にする。

## 現在の層構成

```text
handler/dto -> usecase -> domain(interface) <- infrastructure
                     |
                domain/entity
```

- handler は認証ユーザー取得、request/response 変換、HTTP ステータス変換を担当する。
- usecase は問題ストック判定、出題対象選定、回答・保存・投稿などのアプリケーションロジックを担当する。
- domain は entity と repository / LLM client interface を持つ。JSON タグや DB タグは置かない。
- infrastructure は PostgreSQL, Gemini, Firebase, Cloud Storage などの具象実装を持つ。

## 問題生成フロー

### 準備済み問題の取得

通常の学習開始では `GET /api/questions/prepared` を呼ぶ。対象本のハイライトに紐づく既存 questions を返し、未準備なら `409 questions are still preparing`、生成失敗なら `409 question generation failed`、元テキストがない場合は `422 source text is unavailable` を返す。

### ストック同期

`POST /api/questions/sync` が本ごとの stock / preparing を集計し、ユーザーの `default_question_count` を満たしていない本だけをキューに入れる。1回の sync は `QUESTION_SYNC_PER_TRIGGER_LIMIT`、1日の総量は `QUESTION_SYNC_DAILY_LIMIT` で制御する。

### AI 呼び出し

Gemini 呼び出しは `QuestionWorkerUsecase` に閉じる。worker は pending highlights を claim し、`LLMClient` interface 経由で最大 `QUESTION_WORKER_MAX_QUESTIONS_PER_CALL` 問ずつ生成する。`USE_GEMINI_MOCK=true` のときは fake Gemini client を使う。

## モデル選択

現在の Gemini adapter は以下を使う。

- free plan: `gemini-2.5-flash`
- pro plan: `gemini-2.5-pro`
- local/staging mock: `mock-model`

モデル名の決定は `domain.LLMClient.ModelForPlan` 経由にし、usecase が infrastructure package を直接 import しない。

## Perspective と再生成

問題には `perspective` と `version` を持たせる。perspective は AI に任せず、コード側で `definition`, `understanding`, `comparison`, `practical`, `pitfall`, `application` から未使用または使用回数が少ないものを選ぶ。

解答後の別観点生成は `regeneration_queue` を使う。初回生成用の `highlights.status = pending` とは分け、過去問題全文を Gemini に渡さないことでトークン増加を避ける。

## エラー処理方針

- バリデーション: 400
- 認証エラー: 401
- リソース不在: 404
- 準備中・生成失敗: 409
- 元テキストなし: 422
- 外部 AI 障害: 502 または usecase 境界で wrap したエラーを handler が変換
- 想定外: 500

## 却下・古い前提

- 旧資料の「2段階プロンプト + goroutine 並列生成」は現在の主経路ではない。現在はハイライト単位の batch 生成と perspective 指定が中心。
- OCR は現時点の実装対象外。画像アップロードからの OCR 差し替え設計は、将来タスクとして扱う。
- `gemini-1.5-*` は古いモデル名。現在の adapter は `gemini-2.5-*` を使う。
- 全ハイライトを時間/件数だけで自動生成する常時 worker は主経路ではない。10分間隔 worker はフェイルセーフとして残す。

## 検証観点

- `go test ./...` で usecase / handler / repository の回帰を確認する。
- `INTEGRATION_DATABASE_URL` を使う統合テストで Neon 互換の PostgreSQL 挙動を確認する。
- staging では `USE_GEMINI_MOCK=true` と小さい `QUESTION_SYNC_*` 上限で、Gemini コストなしに同期フローを確認する。
