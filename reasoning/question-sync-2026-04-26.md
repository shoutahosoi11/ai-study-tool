# 問題生成のオンデマンド化（question sync）— 2026-04-26

## 何を作ったか

Gemini API による問題生成を「時間/件数ベースの自動 worker」から **「アプリ起動時のオンデマンド生成」** に切り替えた。

### バックエンド
- `POST /api/questions/sync` エンドポイント新設
- `usecase/question_sync_usecase.go` — ストック判定 + 生成キュー投入のユースケース
- `domain/highlight_repository.go` に追加メソッド:
  - `ListBookStockByUserID` — 本ごとのストック集計（単一 GROUP BY クエリ）
  - `ListUnusedHighlightsByBook` — 出題実績 0 のハイライト
  - `ListUsedHighlightsWithUncoveredPerspectives` — 既出題だが観点未カバー
  - `QueueHighlightsForGeneration` / `ClaimPendingByIDs` — 既存 pending ステートを再利用
- `usecase/question_worker_usecase.go` に `TriggerQueuedHighlights` を追加し、sync usecase から非同期で呼ぶ
- `cmd/question-worker/` の自動ループは 10 分間隔のフェイルセーフだけ残す
- マイグレーション `030_add_user_daily_generation_count.sql` — 1日生成数カウンタテーブル

### フロントエンド
- `hooks/useQuestionSync.ts` — 起動時 sync + focus/30秒ポーリング
- `components/system/KindleSyncBootstrap.tsx` — 取り込み完了後に sync 呼び出し
- `pages/question/KindleBookSection.tsx` — `isPreparing = stock===0 && preparing>0` で「N問準備中」バッジ表示

### 上限ルール
- 1 Gemini コール: 8 問
- 1 ハイライト: 3 問（既存 `questionCountForHighlight` 流用）
- 1 起動トリガー (sync 1 回): 30 問
- 1 ユーザー 1 日: 100 問（Asia/Tokyo 起算）

### 対象選定ロジック
1. `stock < default_question_count` の本（不足本）を抽出
2. 各不足本を default_question_count に到達するまで埋める
3. ハイライト優先順位:
   - 出題実績 0 の highlight（最優先）
   - 既出題だが観点未カバーの highlight（次点）
4. 不足本処理順: stock=0 → created_at DESC

## なぜこの設計にしたか

### オンデマンド化を選んだ理由
- 旧仕様: 全 highlight に対して時間ベースで自動生成 → ユーザーがアプリを開かなくてもコスト発生
- 新仕様: アプリ起動時に必要分だけ生成 → 無駄なコール削減、ユーザー単位で制御可能

### 「未出題優先」を入れた理由
- 同じハイライトに観点違いで N 問作るより、別ハイライトから 1 問ずつ作る方が**学習の幅**が広がる
- ユーザー要望: "まだ出題したことないハイライトを優先する"

### 既存 worker を「停止」ではなく「フェイルセーフ」として残した理由
- sync が何らかの理由で失敗したケースで pending highlight が永続的に残るリスクを回避
- 10 分間隔で十分（旧 5 分よりは弱め）

### 「準備中」バッジを stock=0 のときだけ出す理由（仕様 A）
- 既存ストックが 1 問でもあれば即出題可能 → 待たせない方が UX 良い
- 仕様 B（設定数揃うまで出題不可）は学習体験を阻害するため不採用

## 他の選択肢と比較してなぜこれを選んだか

| 案 | 採用 | 理由 |
|---|---|---|
| 起動時のみトリガー | ✅ | 問題タブ表示時トリガーは判定が冗長になる、起動時で十分 |
| 本ごとのストック判定 | ✅ | UI が本単位なので自然、ユーザー単位より細かい |
| 単純 GROUP BY | ✅ | 1000冊以下なら数ms。カウンタキャッシュは過剰 |
| 同期生成（待たせる） | ❌ | Gemini レイテンシが数秒〜十数秒、UX 悪い |
| 本またぎ Gemini 混載 | ❌ | コストは下がるが JSON パース複雑化、初版はシンプル優先 |
| カウンタキャッシュ (book_question_stats) | ❌ | 1000冊超の本棚は稀、単純クエリで十分 |
| SSE による準備中プッシュ | ❌ | Cloud Run のコネクション課金注意、focus + 30秒ポーリングで十分 |

## 検証状況

- ✅ `go build ./...` 成功
- ✅ `go test ./...` 全 PASS
- ✅ Sync usecase 単体テスト 4 本（充足/30問上限/1日上限/未出題優先）全 PASS
- ✅ コードレビューで仕様コンプライアンス確認
- ❌ DB 統合 E2E（後述: Docker Desktop 環境問題によりスキップ）

## 後でやらなきゃいけないリスト

### 高優先度
1. **DB 統合 E2E 検証**
   - シナリオ A: ストック充足の本 → queued=0 確認
   - シナリオ B: ストック不足 → 数秒後に target に達することを確認
   - シナリオ C: 30 問上限 → queued=30 で打ち切り、残りが次回に持ち越し
   - シナリオ D: 1日 100 問上限 → skipped_due_to_daily_limit=true
   - シナリオ E: 未出題優先 → unused highlight が先にキュー
   - 前提: Docker 復旧 or Homebrew Postgres 移行 + Firebase ID token 取得
   - **`reasoning/question-sync-2026-04-26.md` のシナリオ表を踏襲**

2. **マイグレーション 030 を本番 DB に適用**
   - `user_daily_generation_counts` テーブル作成
   - 既存データへの影響なし（新規テーブル）

3. **Gemini コール数の本番モニタリング（リリース後 1 週間）**
   - GCP コンソール → Gemini API ダッシュボード
   - 旧仕様と比べてコール数が劇的に減ることを確認
   - 想定: 「不足分 / 8」以下に収まる

### 中優先度（観測性・テスト容易性向上）
4. **環境変数で上限を上書きできるようにする**
   - `QUESTION_SYNC_DAILY_LIMIT`（現状 const 100）
   - `QUESTION_SYNC_PER_TRIGGER_LIMIT`（現状 const 30）
   - 理由: staging で 5 等に下げて即検証したい

5. **Gemini モッククライアント追加**
   - `USE_GEMINI_MOCK=true` で fake client に差し替え
   - 統合テスト時に Gemini API キーなしで回せるようにする

6. **構造化ログ追加**
   - Gemini コール毎: `gemini.call userID=X book=Y questions_count=Z`
   - 1日上限到達: `daily_limit.reached`
   - 現状は sync 開始/完了ログのみ

### 低優先度（軽微な仕様ズレ）
7. **`appendQuestionSyncCandidates` のフォールバックで target を 1 問オーバーする可能性**
   - 例: target=3, ストック=0, 長文ハイライト 1 件のみ → 5 問生成される
   - 30 問 budget は超えないので致命ではないが、無駄トークン抑制のため target ぴったりに修正候補
   - 修正案: fallback 採用時に `softTarget - selection.questionCount` で部分採用するか、長文ハイライトの観点数を制限

### 環境問題
8. **Docker Desktop の復旧 or Homebrew Postgres 移行**
   - 現状: Docker.app UI は起動するが engine (`com.docker.backend`) が起動しない
   - 推奨: `brew install postgresql@16 && brew services start postgresql@16`
   - これがあれば今後の DB 統合テストが楽になる

## 関連ファイル
- バックエンド設計: `backend/internal/usecase/question_sync_usecase.go`
- ワーカー連携: `backend/internal/usecase/question_worker_usecase.go:112` (TriggerQueuedHighlights)
- 上限定数: `question_sync_usecase.go:15-17`
- フロント: `frontend/src/hooks/useQuestionSync.ts`
- 準備中バッジ: `frontend/src/pages/question/KindleBookSection.tsx:404`
- マイグレーション: `backend/db/migrations/030_add_user_daily_generation_count.sql`
