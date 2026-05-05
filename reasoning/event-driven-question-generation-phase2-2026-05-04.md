# Event-Driven Question Generation Phase 2

## 何を作ったか

- `SyncQuestionStock` を target 補充型から、book_key 単位の条件評価型へ変更した。
  - 条件A: pending highlight が 10 件以上。
  - 条件B: pending highlight が 5 件以上 10 件未満、かつ active な未回答 question が 0 件。
- 条件を満たした book について `question_generation_jobs` を作成し、Cloud Tasks enqueue 抽象経由で `/internal/tasks/question-generation` に流す形へ変更した。
- `enqueue_failed` job の再 enqueue と stale `processing` job の `queued` 復旧を sync 冒頭で実行するようにした。
- 回答完了後に対象 highlight を `pending` に戻し、既存 active question を `superseded_at` で soft delete して、同じ book だけ再評価する入口を追加した。
- Cloud Tasks worker の受け口で `queued -> processing` の CAS claim を行い、claim できた job だけ Gemini 生成を実行するようにした。
- worker claim 時に daily limit 100 問/日を確認し、超過時は job を `queued` に戻して no-op にするようにした。
- `POST /api/questions` の同期生成ルートを削除し、既存の polling / prepared / answer / grade ルートは維持した。

## なぜこの設計にしたか

- `pending` highlight をキューとして扱うことで、ハイライト追加時は INSERT のみにでき、取り込み経路を軽く保てる。
- 条件評価を `/questions/sync` と回答完了後に寄せることで、アプリ利用タイミングに合わせて必要な book だけを scan できる。
- Cloud Tasks SDK は infrastructure に閉じ込め、usecase は `QuestionGenerationTaskEnqueuer` interface だけを見る形にした。これにより local test では mock に差し替えられる。
- job claim は DB status CAS に寄せた。同じ task が複数回届いても `queued` を claim できた 1 回だけ処理される。
- `superseded_at IS NULL` を active question の条件にして、保存済み/不正解履歴は残しながら「1 highlight 1 active question」を守る。
- Phase1 の partial unique index に合わせ、同一 `(user_id, book_key)` の active job は 1 件までにした。22 件 pending があっても active job はまず 1 件作り、完了後の次 sync/answer trigger で次を拾う。重複 worker を避ける安全性を優先した。

## 他の選択肢との比較

- handler で直接 Cloud Tasks に enqueue する案もあったが、HTTP 層に job 判定と enqueue 失敗処理が漏れるため避けた。
- Redis / in-memory queue で job 管理する案もあるが、既に DB migration で partial unique / CAS を設計済みなので Postgres に寄せた。Cloud Run の複数 instance でも同じ整合性を使える。
- daily limit を sync 作成時だけで見る案は、task retry や複数 worker でずれる可能性があるため、worker claim 時の予約を正にした。sync 側にも早期 skip を残し、明らかに上限到達済みの日は余計な enqueue を避ける。
- 旧 worker の polling 経路をすぐ全削除する案もあるが、Phase2 では Cloud Tasks 経路を主経路に切り替えつつ、既存 command のビルド互換は維持した。実運用は internal endpoint + Cloud Tasks に寄せる。

## 既存の挙動が変わる入力・操作

- `POST /api/questions` は削除された。フロントから呼ばれていない前提の同期生成 endpoint のため、問題生成は `/api/questions/sync` と回答完了後の条件評価に移る。
- 回答完了後、対象 highlight は `pending` に戻り、既存 active question は `superseded_at` で非 active 化される。
- `ListPrepared` は `superseded_at IS NULL` の question のみを出題対象にする。
- daily limit 到達時、worker は job を処理せず `queued` に戻す。翌日の sync で再処理対象になる。

## テスト観点

- 条件A/B の job 作成、条件未達時 no-op、active job 重複時 no-op。
- enqueue 失敗時の `enqueue_failed` 化と次回 sync での再 enqueue。
- 回答後の highlight pending 復帰、active question supersede、同 book 再評価。
- worker の CAS no-op、daily limit 超過時 queued 復帰、対象 highlight 0 件時 completed、1 highlight 1 active question 生成。
- `cd backend && go build ./... && go test ./...` が green。
