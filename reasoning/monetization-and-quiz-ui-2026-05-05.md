# Monetization and Quiz UI

## 何を作ったか

- 手動問題生成の受け口を追加した。
  - `POST /api/questions/generate/manual`
  - `POST /api/v1/questions/generate/manual`
  - 5件未満のハイライトは 400 とし、`question_generation_jobs` に `manual_selection` job を作成して in-process dispatcher に渡す。
- 課金・トークン管理の基盤を追加した。
  - `POST /api/v1/checkout/session`
  - `POST /webhooks/stripe`
  - `POST /api/v1/tokens/award`
  - `GET /api/v1/tokens/balance`
- DB マイグレーションを追加した。
  - `036_drop_answer_grading_columns.sql`
  - `037_add_highlight_book_order_index.sql`
  - `038_add_users_subscription_columns.sql`
  - `039_add_question_budget_and_tokens.sql`
- `answers.score` / `answers.feedback` / `answers.grader_model` は、選択式固定に合わせてテーブルと書き込みコードから外した。
- `highlights.book_order_index` を追加し、既存データは本ごとの作成順で採番する。新規取り込み後も未採番レコードを補完する。
- モバイルにトークン残高、広告トークン付与、手動生成、Stripe Checkout 起動の最小導線を追加した。

## なぜこの設計か

- 手動生成は既存の `question_generation_jobs` / `question_generation_job_highlights` / dispatcher を使い回した。生成経路を増やしても worker の冪等性、job status、失敗処理を共通化できるため。
- `QuestionGenerationTaskEnqueuer` は維持した。現状は in-process dispatcher だが、将来 Cloud Tasks に戻す場合も usecase 層を変更しないで差し替えられる。
- 日次無料枠と広告トークンは DB で管理した。個人開発の規模では Redis より運用コストが低く、Postgres transaction で無料枠とトークン消費を一貫させられる。
- Stripe は infrastructure に閉じ込め、handler/usecase は interface 経由にした。Stripe SDK の型や署名検証を HTTP 層・usecase 層へ漏らさないため。
- migration 番号は既存 `033` から `035` が Phase 1 で使用済みだったため、衝突を避けて `036` 以降にした。

## 他の選択肢との比較

- AdMob SDK をこの PR で導入する案は見送った。現在の dev client 構成にネイティブ依存を増やすと確認コストが大きいため、まず API と UI 導線を置き、SDK 組み込みは別 PR で行う。
- Stripe の顧客・サブスクリプション情報を users とは別テーブルにする案もあったが、現時点ではユーザー単位の単一プランだけなので users に保持する方が単純。
- `answers` の採点カラムを残す案もあったが、完了条件で DB からの削除が明示されたため migration で drop した。将来記述式採点を戻す場合は新しい migration で復元する。

## 残りの注意点

- モバイルの広告表示はまだ本物の AdMob SDK ではなく、トークン付与 API を直接呼ぶ最小導線。RewardedAd / NativeAdView の実装はネイティブ依存追加を伴う別 PR に分ける。
- ハイライトピッカーは `book_order_index` と manual generation API の土台まで。範囲選択 UI と縦スワイプカードデッキは、既存 `App.tsx` の規模が大きいため分割実装が安全。
- Stripe webhook は Checkout Session の `client_reference_id` / metadata `user_id` を信頼して users を更新する。署名検証は必須で、`STRIPE_WEBHOOK_SECRET` 未設定時は拒否する。
