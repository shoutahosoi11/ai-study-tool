# 問題生成のオンデマンド化（question sync）— 2026-04-26

## 何を作ったか

Gemini API による問題生成を「時間/件数ベースの自動 worker」から
**アプリ起動時・取り込み後のオンデマンド同期**に切り替えた。

### バックエンド

- `POST /api/questions/sync` エンドポイントを追加。
- `usecase.QuestionSyncUsecase` が本ごとの問題ストックを判定し、不足分だけキュー投入する。
- `QuestionRepository.QueueHighlightsWithinDailyLimit` が日次上限確認、日次カウンタ予約、ハイライトの `pending` 化を同一トランザクションで行う。
- `QuestionWorkerUsecase.TriggerQueuedHighlights` が同期リクエスト後に queued highlights を非同期処理する。
- `QuestionWorkerUsecase.RunOnce` は 10 分間隔のフェイルセーフ worker として残す。
- `user_daily_generation_counts` でユーザー単位の日次生成数を管理する。
- stale な `processing` ハイライトは `QUESTION_SYNC_STALE_PROCESSING_SECONDS` 経過後に再キュー可能にする。

### フロントエンド

- `hooks/useQuestionSync.ts` で起動時 sync、focus 時更新、準備中だけ 30 秒ポーリングを行う。
- `components/system/KindleSyncBootstrap.tsx` で Kindle 取り込み後に sync を呼ぶ。
- `pages/question/KindleBookSection.tsx` で stock と preparing に応じて「準備中」表示を行う。

### 上限ルール

- 1 Gemini コール: 最大 8 問。
- 1 ハイライト: 最大 3 問。
- 1 sync トリガー: デフォルト最大 30 問。
- 1 ユーザー 1 日: デフォルト最大 100 問。
- `QUESTION_SYNC_PER_TRIGGER_LIMIT` と `QUESTION_SYNC_DAILY_LIMIT` で上書き可能。
- `USE_GEMINI_MOCK=true` で fake Gemini client を使える。

### 対象選定ロジック

1. `stock < default_question_count` の本を不足本として抽出する。
2. `stock=0` の本を優先し、その中では新しい本を優先する。
3. 生成元ハイライトは「出題実績 0」→「観点未カバー」の順で選ぶ。
4. 既に使った perspective は重複生成しない。
5. 生成数は target と per-trigger budget を超えないように切り詰める。

## なぜこの設計にしたか

### オンデマンド化を選んだ理由

旧仕様は全 highlight に対して時間ベースで自動生成していたため、ユーザーがアプリを開かない場合でもコストが発生していた。新仕様ではアプリ起動時・取り込み後に必要分だけ生成するため、Gemini コール数をユーザー体験に直結する分へ絞れる。

### キュー投入と日次カウンタを同一トランザクションにした理由

キュー投入だけ成功して日次カウンタ加算に失敗すると、1 日上限を超える生成が起きる可能性がある。逆にカウンタだけ増えると、生成されていないのに quota だけ消費する。`QueueHighlightsWithinDailyLimit` にまとめることでこの不整合を避ける。

### worker を完全停止しない理由

同期リクエスト後の goroutine が落ちた場合でも、queued highlights が永続的に残らないようにするため。10 分間隔の worker は通常経路ではなくフェイルセーフとして扱う。

### 「準備中」バッジを stock=0 のときだけ出す理由

既存ストックが 1 問でもあれば学習は開始できるため、設定数が揃うまで待たせるより体験が良い。完全充足は裏側で進める。

## 実装上の境界

- handler は request/response 変換と認証ユーザー取得だけを担当する。
- usecase はストック判定、選定、キュー投入、worker 起動を調整する。
- repository は SQL、トランザクション、排他制御を担当する。
- Gemini 呼び出しは `LLMClient` interface 経由で worker から実行する。
- domain に JSON タグや DB タグを入れない。

## 観測性

- sync 開始/完了、日次上限到達、Gemini call start/success/failure をログに出す。
- staging では `USE_GEMINI_MOCK=true` と小さい `QUESTION_SYNC_*_LIMIT` で軽く動作確認できる。
- 本番では Gemini API ダッシュボードとアプリログで「不足分 / 8」以下の call 数に収まるか確認する。

## 検証状況

- `go build ./...` 成功。
- `go test ./...` 成功。
- `go test -race ./internal/usecase` 成功。
- `question_sync_repository_integration_test.go` は `INTEGRATION_DATABASE_URL` を使った PostgreSQL 統合テストとして用意している。Neon またはローカル PostgreSQL 16 の検証用 DB を指定して実行する。

## 残タスク

- Neon 本番 DB に `030_add_user_daily_generation_count.sql` を適用する。
- リリース後 1 週間、Gemini call 数と daily limit 到達ログを監視する。
- 必要ならフロントエンド側の question sync hook を Vitest で固定する。

## 関連ファイル

- `backend/internal/usecase/question_sync_usecase.go`
- `backend/internal/usecase/question_worker_usecase.go`
- `backend/internal/infrastructure/persistence/question_repository.go`
- `backend/internal/infrastructure/persistence/highlight_repository.go`
- `backend/db/migrations/030_add_user_daily_generation_count.sql`
- `frontend/src/hooks/useQuestionSync.ts`
- `frontend/src/pages/question/KindleBookSection.tsx`
