# 既存コードベース集中レビュー

## Summary

問題生成の中核は CAS claim、active job の partial unique index、in-process dispatcher など重要な防御が入っている一方、job 実行中の quota 予約、soft delete、新問保存、highlight 状態更新が別々の DB 操作になっており、失敗時に日次枠の過剰消費・active question 消失・重複生成が起き得る状態です。認証ミドルウェアは fail-close で概ね堅牢ですが、回答・保存系は question_id の所有者確認が弱く、他人の question_id を知っている場合に回答記録や統計更新が可能です。Stripe webhook は署名検証自体は SDK に委譲できていますが、body サイズ制限がなく unauthenticated endpoint として DoS 面の穴があります。

## Good

- `QuestionGenerationJobRepository.ClaimQueued` は `status='queued'` 条件付き UPDATE で CAS しており、同一 job_id の二重 worker 実行を DB 側で抑止できている。
- `question_generation_jobs` は `(user_id, book_key)` の active partial unique index を持ち、同一 book の queued / processing job 重複を防ぐ意図が明確。
- Firebase 認証は `VerifyIDTokenAndCheckRevoked` を使い、nil token / 空 UID も fail-close している。
- Stripe webhook 署名検証は `webhook.ConstructEvent` を通しており、独自 HMAC 実装を避けている。
- ingest rate limit は Postgres の `INSERT ... ON CONFLICT DO UPDATE RETURNING count` で atomic increment になっている。

## Needs Improvement

## backend/internal/usecase/question_worker_usecase.go

### 🔴 Critical

- 133-160, 209-218: Gemini 呼び出し前に `ReserveDailyGeneratedCount` で日次生成数を確保しているが、以降の `SaveGeneration` / Gemini 呼び出し / `Save` / `MarkGenerationCompleted` 失敗時に予約が戻らない。
  - 影響: 失敗した生成でも `user_daily_generation_counts` が消費される。リトライのたびに再予約されるため、実際には問題が作られていないのに 100 問/日の枠を使い切る。課金面でも、失敗後の再試行が quota 状態とずれて制御される。
  - 修正案: 「生成数の予約」と「job claim」は同じ transaction で予約状態を持たせ、失敗時に明示 rollback できる設計にする。少なくとも worker 内では、成功保存数を基準にカウントするか、job に reserved_count を持たせて retry 時に二重予約しない。

- 165-197: `SupersedeActiveQuestionsForHighlight`、新 question の `Save`、`MarkGenerationCompleted`、`MarkCompleted` が transaction で束ねられていない。
  - 影響: 旧 active question を `superseded_at` で消した後に保存や completed 更新で失敗すると、ユーザーから既存問題が消える。逆に question 保存後、job completed 前にプロセスが落ちると job が再実行され、同じ highlight で再度 Gemini 呼び出しと soft delete が走る。
  - 修正案: highlight 単位または job 単位で `supersede active -> insert new question -> mark highlight completed -> mark job completed` を transaction 化する。加えて `questions(highlight_id) WHERE superseded_at IS NULL` は現在 index のみなので、active question 1件ルールを守るなら unique index にする。

### 🟠 Major

- 213-225, 431-439, 613-621: Gemini / DB の raw error を `last_error` と構造化ログにそのまま保存・出力している。
  - 影響: Gemini のレスポンス本文や upstream エラー詳細がログ・DB に残る。現状は外部レスポンスには出ていないが、運用ログや DB 参照者に prompt 断片・入力内容・プロバイダ詳細が漏れる可能性がある。
  - 修正案: 永続化する error は分類コードに寄せ、詳細は必要最小限に丸める。Gemini response body はステータス・retryable 種別だけにする。

## backend/internal/usecase/question_sync_usecase.go

### 🟠 Major

- 191-215: `enqueue_failed` 救済で `enqueueJob` を呼んだ後に `MarkQueued` している。
  - 影響: 現在の enqueuer は in-process goroutine なので、goroutine が即座に `ClaimQueued` を実行すると、DB status はまだ `enqueue_failed` のままで claim できず no-op になる。その後 `MarkQueued` されるため、次回 sync まで job が動かない。
  - 修正案: `enqueue_failed` を復旧する場合は、先に `MarkQueued` してから enqueue する。Cloud Tasks に戻す場合も、task が先に届く race を避けるため同じ順序が安全。

- 268-285: job 作成、job_highlights 追加、highlight の processing 化、enqueue が同一 transaction ではない。
  - 影響: `Create` 成功後に `MarkHighlightsProcessing` または enqueue が失敗すると、active unique index により同一 book の次 job 作成が抑止されたまま、中途半端な job 状態が残る。特に enqueue 失敗時は highlights が processing のままになり、通常の pending 条件評価から外れる。
  - 修正案: job 作成と highlight status 更新を repository の transaction メソッドにまとめる。enqueue 失敗時に `enqueue_failed` にするなら、highlight 側も retry 対象として明確に復旧できる状態へ戻す。

## backend/internal/usecase/manual_generation_usecase.go

### 🟠 Major

- 46-63: 選択された highlights がリクエスト `book_key` に属するか検証せず、任意の `bookKey` で job を作成している。
  - 影響: 自分の highlight だけでも、別 book_key の active job を作れてしまう。`uq_question_generation_jobs_active` が `(user_id, book_key)` なので、誤った book_key の job が同じ本の自動生成をブロックする。
  - 修正案: `ListByIDs` で取得した highlight 群の book_key expression がすべてリクエスト `book_key` と一致することを usecase で検証する。可能なら repository で `ListByIDsAndBookKey` にして DB 側で絞る。

## backend/internal/usecase/answer_usecase.go

### 🔴 Critical

- 40-55: `FindByID` は question_id だけで取得し、回答者 userID と question の owner / 出題対象との整合性を確認していない。
  - 影響: 認証済みユーザーが他人の question_id を知っていれば、その question に回答を upsert できる。さらに `answer_repository.go` 96-101 で question 側の `answer_count` / `correct_count` も更新される。
  - 修正案: `FindByIDForUser(ctx, questionID, userID)` のような repository メソッドにし、少なくとも `questions.user_id = userID` または公開出題として許可された question のみ回答可能にする。

## backend/internal/usecase/question_usecase.go

### 🔴 Critical

- 260-268: `SaveQuestion` も `GetByID` が question_id のみで、保存者 userID に対する参照権限を確認していない。
  - 影響: 他人の question_id を知っていれば `saved_questions` に保存できる。saved / incorrect 履歴に他人の問題を混入させられる。
  - 修正案: 回答と同じく userID 付き取得にする。公開問題を保存可能にする仕様があるなら、公開範囲を repository の WHERE で明示する。

## backend/internal/infrastructure/persistence/highlight_repository.go

### 🟠 Major

- 116-147, 1095-1121: `fillMissingBookOrderIndexes` は `book_key IS NOT NULL` の highlight だけ採番するが、`buildHighlightBulkUpsert` の INSERT カラムに `book_key` が含まれていない。
  - 影響: 新規取り込み highlight は `book_key` が NULL のままになり、`book_order_index` も埋まらない。mobile の範囲指定出題 UI は `book_order_index` 前提なので、問題あり highlight でも範囲選択対象から落ちる。
  - 修正案: highlight 保存時に `book_key` を必ず設定する。ASIN がある場合は ASIN、ない場合は metadata key を入れる。あわせて `fillMissingBookOrderIndexes` も `bookKeyExpressionSQL` ベースで既存 NULL を補完できるようにする。

- 748-763, 772-790: `MarkGenerationCompleted` / `MarkGenerationFailed` が user_id や現 status を条件にしていない。
  - 影響: 現状は呼び出し元が userID で絞った ID を渡す前提だが、repository 単体では別 user の highlight ID 混入や、すでに別処理で状態が変わった highlight の上書きを防げない。
  - 修正案: lifecycle 更新系は `user_id` と期待 status を引数に取り、`WHERE user_id=$1 AND status='processing'` のように CAS 条件を付ける。

## backend/internal/infrastructure/persistence/question_repository.go

### 🟠 Major

- 87-100: `SupersedeActiveQuestionsForHighlight` は active questions を無条件に全件 supersede するが、その直後の new question insert と transaction で結合されていない。
  - 影響: worker 側の transaction 不足と組み合わさり、active question が 0 件になる時間・失敗状態が発生する。1 highlight 1 active question ルールが DB 制約で保証されていない。
  - 修正案: supersede と insert を同じ repository transaction に移し、active unique index を追加する。

- 65-82, 122-126: `FindQuestionByID` / `GetQuestionByID` が owner を条件にしない。
  - 影響: handler/usecase 側が userID を持っていても repository API が権限境界を表現できず、回答・保存の認可漏れを誘発している。
  - 修正案: userID 付き API を追加し、既存の ID-only API は内部管理用途に限定する。

## backend/internal/usecase/highlight_import_job_usecase.go

### 🟡 Minor

- 79-83: `RetryCount >= ImportQueueMaxRetry` を判定してから `RequeueWithRetry` で increment するため、`ImportQueueMaxRetry=3` の場合は 3 回失敗後も queued に戻り、次の 4 回目の失敗で failed になる。
  - 影響: 最大リトライ回数より 1 回多く BulkUpsert を試行する。DB upsert なので課金爆発ではないが、壊れた payload では余計な再処理が走る。
  - 修正案: `RetryCount+1 >= ImportQueueMaxRetry` で failed にするか、repository の `RequeueWithRetry` 側で next retry count に応じて status を決める。

## backend/internal/infrastructure/inprocess/question_generation_dispatcher.go

### 🟠 Major

- 45-50: semaphore acquire が goroutine 内なので、`EnqueueQuestionGeneration` は呼ばれた数だけ goroutine を作ってから `d.sem <- struct{}{}` で待たせる。
  - 影響: `/questions/sync` が連打された場合、実行同時数は 3 に抑えられるが、待機 goroutine は上限なく増える。DB CAS で多くは no-op になっても、メモリと scheduler 負荷は増える。
  - 修正案: goroutine を作る前に bounded queue に投入する、または `sem` acquire を Enqueue 側で行い、満杯なら明示的に enqueue 失敗として DB status に残す。

- 65-69, `backend/cmd/main.go` 48: shutdown 時の `Container.Close` が `QuestionDispatcher.Wait()` を無期限に待つ。
  - 影響: Cloud Run の SIGTERM 時に in-flight job が長引くと、shutdown timeout 10 秒とは別に process 終了が遅れる。最終的には Cloud Run 側に kill され、後続復旧は stale processing 頼みになる。
  - 修正案: `Wait(ctx)` にして shutdown timeout と同じ context で bounded wait する。未完了 job は DB stale recovery 前提として明示ログを出す。

## backend/internal/handler/stripe_handler.go

### 🟠 Major

- 55-57: unauthenticated な `/webhooks/stripe` で `io.ReadAll` を直接呼んでおり、body size limit がない。
  - 影響: 攻撃者が巨大 body を送ると、署名検証前にメモリを消費する。Stripe 署名が無効でも ReadAll は完了するため、DoS に弱い。
  - 修正案: `http.MaxBytesReader` または Echo の `BodyLimit` を webhook route に適用する。Stripe webhook の実サイズに合わせて 64KB〜1MB 程度で十分。

## backend/internal/middleware/auth.go

### 指摘なし

- nil verifier を初期化時に拒否し、検証失敗時も 401 / 503 で fail-close している。ここは現状維持でよい。

## backend/internal/middleware/rate_limit.go

### 指摘なし

- 認証済み UID を前提に atomic increment し、超過時に 429 と `Retry-After` を返している。境界条件としては `current > limit` なので 100 件目は通り、101 件目から拒否になっており仕様と合う。

## backend/internal/infrastructure/stripe/webhook_validator.go

### 指摘なし

- secret 未設定時は `ErrInvalidInput`、設定時は Stripe SDK の `ConstructEvent` に署名検証を委譲している。HMAC bypass に直結する問題は見当たらない。

## backend/internal/domain/

### 🟡 Minor

- `question_repository.go` 24-36: `QuestionCatalogReader` が read 系だけでなく `Save`、`SupersedeActiveQuestionsForHighlight`、`UpdateStats`、`SaveForUser` まで持っている。
  - 影響: usecase ごとの権限・責務境界が曖昧になり、ID-only question lookup のような認可漏れを誘発しやすい。
  - 修正案: 回答、保存、生成、一覧の interface をさらに分け、回答・保存系には userID 付き取得を必須にする。

## Questions

- `questions` はユーザーごとの private データなのか、SNS 投稿などを通じて他ユーザーが回答・保存できる public データなのか。現状の API は private 前提に見えるが、repository は public ID 参照を許している。
- manual generation の `book_key` はクライアント入力を信頼する仕様か、選択 highlights から server 側で導出する仕様か。後者なら現在の `ManualGenerationUsecase` は修正が必要。
- 失敗した Gemini 呼び出しを日次生成数にカウントする運用か。現在は失敗でも予約が残るため、ユーザー向けの「生成回数」と実生成数が一致しない。
- `book_key` は `highlights` 保存時に必須化する方針か、`bookKeyExpressionSQL` による動的導出を続ける方針か。`book_order_index` と indexed COUNT を使うなら保存時に必須化した方が整合しやすい。
