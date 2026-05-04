# Event-Driven Question Generation Phase 2.5

## 何を作ったか

- Cloud Tasks SDK と `/internal/tasks/question-generation` endpoint を削除した。
- `domain.QuestionGenerationTaskEnqueuer` interface は維持し、実装だけ `inprocess.QuestionGenerationDispatcher` に差し替えた。
- dispatcher は enqueue 呼び出し時に goroutine を起動し、`QuestionWorkerUsecase.ProcessQuestionGenerationJob` を同一プロセス内で実行する。
- dispatcher に最大同時実行数を追加した。`QUESTION_DISPATCHER_MAX_CONCURRENT` で設定でき、未指定時は 3。
- request context はレスポンス後に cancel されるため、job 実行は timeout 付き background context で行う。
- graceful shutdown 用に dispatcher の `Wait()` を追加し、container close 時に in-flight goroutine を待つ。
- Echo server 起動を signal-aware にし、SIGTERM / Ctrl+C で shutdown するようにした。
- 選択式問題のみ扱う前提に合わせ、回答判定から LLM 採点を削除した。
  - `LLMClient.GradeAnswer` と `QuestionTypeDescriptive` を削除。
  - `AnswerUsecase.SubmitAnswer` は `Question.IsCorrect()` のみで判定。
  - `answers.score` / `answers.feedback` / `answers.grader_model` カラムは残すが、書き込みコードからは外した。
  - 旧 `/questions/:id/grade` route は互換用に残し、`AnswerHandler.SubmitAnswer` へ集約した。

## なぜこの設計にしたか

- 現状は個人開発かつユーザーゼロなので、Cloud Tasks queue を運用するコストと設定量が機能に対して大きい。
- DB の `question_generation_jobs` がすでに queued / processing / completed / failed / enqueue_failed を持っているため、プロセス内 goroutine でも冪等性と復旧方針は維持できる。
- usecase は引き続き `QuestionGenerationTaskEnqueuer` だけを見る。これにより、将来 Cloud Tasks に戻す場合も DI で実装を差し替えればよい。
- 同時実行数を dispatcher 側で制限し、ローカル/Cloud Run の単一 instance 内で Gemini 呼び出しが無制限に増えないようにした。
- job 実行の正しさは worker の CAS claim に寄せている。同じ job が複数 enqueue されても、`queued -> processing` を取れた 1 回だけが処理する。
- LLM 採点は記述式問題を再導入するまでは不要。選択式固定なら DB の正解文字列とユーザー回答の比較だけで十分なので、レイテンシ・コスト・失敗点を減らすため削除した。

## Cloud Tasks との比較

- Cloud Tasks の利点は、プロセス再起動に強く、retry/backoff/queue 制御を外部サービスに任せられること。
- in-process dispatcher の利点は、設定が少なく、ローカル確認が簡単で、現フェーズの規模に対して運用負荷が低いこと。
- 今回は本番ユーザーがいないため、まず in-process で仕様の正しさを固める。ユーザー数や失敗時復旧要件が増えたら、同じ interface のまま Cloud Tasks 実装へ戻す。

## Cloud Run 設定メモ

- goroutine が HTTP response 後も動くため、Cloud Run では CPU always-allocated を推奨する。

```bash
gcloud run services update <service> --no-cpu-throttling
```

- `min-instances=1` も推奨。scale-to-zero でも起動中 request からの goroutine は動くが、deploy / restart / scale down で in-flight job が消える可能性がある。

```bash
gcloud run services update <service> --min-instances=1
```

- `/internal/tasks/question-generation` を廃止したため、この endpoint 保護目的の `ingress=internal` 前提は不要になった。
- ただし API 全体を Cloudflare / LB 前段で守る方針自体は別途維持する。

## 復旧方針

- dispatcher の goroutine 内で worker が error を返した場合、panic せず log だけ出す。job の retry_count / failed 化は worker 側の DB 更新に任せる。
- enqueue 実装自体は通常 nil を返すため `enqueue_failed` は基本発生しない。
- `enqueue_failed` 救済ロジックは残す。将来 Cloud Tasks に戻した時に同じ usecase を使えるようにするため。
- 古い `processing` job は既存の stale recovery で `queued` に戻す。

## テスト観点

- dispatcher が即座に nil を返し、goroutine 内で runner を呼ぶ。
- runner error でも panic しない。
- request context が cancel されていても background context で job が動く。
- max concurrent が 2 の時、5 job enqueue しても同時実行数が 2 を超えない。
- 回答は LLM を呼ばず、生成済みの `correct_answer` と `explanation` を使う。
- `cd backend && go mod tidy && go build ./... && go test ./...` が green。
- `rg -n "GradeAnswer|GradeInput|GradeResult|QuestionTypeDescriptive|llm_error|BuildGraderPrompt|gradeAnswerSchema" backend/internal backend/cmd -g '*.go'` が no hit。
