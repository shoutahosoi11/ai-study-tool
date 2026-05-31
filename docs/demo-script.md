# Demo Script

Use a staging or local demo environment. Do not show real user data, real emails, secrets, raw tokens, cookies, signatures, webhook payloads, SSV query strings, prompt text, or highlight text.

## 1. アプリ概要

「Kindleで読書中に引いたハイライトを取り込み、AIで問題化して復習できる学習ツールです。Web、Chrome Extension、Mobileを想定し、認証、LLMコスト制御、課金、広告報酬、Admin運用まで含めて作っています。」

## 2. ログイン

1. Demo用アカウントでログインする。
2. dashboardへ遷移する。
3. WebはHttpOnly Session Cookie + Signed CSRFでAPIを守っていると説明する。

## 3. Extension接続

1. Chrome Extensionの接続画面を開く。
2. Extension側に表示されたdummy pairing codeをWebの `/extension/connect` に入力する。
3. 成功画面で「拡張機能に戻ってください」と表示されることを見せる。
4. raw tokenやpairing idは画面に出さないと説明する。

## 4. Kindle Notebookから取り込み

1. Demo用のKindle NotebookまたはサンプルHTMLを開く。
2. Extensionのimport buttonを押す。
3. import successを表示する。
4. Extension tokenはhighlight import/checkだけにscopeを絞っていると説明する。

## 5. ハイライト一覧確認

1. Web dashboardで取り込まれた本やハイライト数を見る。
2. 個人情報や実ハイライト本文が映らないdemo dataを使う。

## 6. 問題生成

1. 問題生成ボタンを押す。
2. loading、queued、completedまたはmock completedを見せる。
3. LLMは本番課金が走らないmock/dry-run/staging設定で説明する。
4. user budgetとglobal budgetでコスト爆発を防ぐと説明する。

## 7. 回答

1. 生成済み問題を開く。
2. 選択肢を選んで回答する。
3. 正誤状態が保存されることを見せる。

## 8. 間違えた問題の復習

1. incorrect questionsを開く。
2. 間違えた問題が復習導線に入ることを見せる。

## 9. 保存問題

1. 問題を保存する。
2. saved questionsに表示されることを見せる。

## 10. 課金/広告tokenの説明

1. Premium / reward token modelを説明する。
2. Stripeはtest modeかmock webhookで説明する。
3. AdMobはclient通知ではなくSSVで検証するため、本番広告報酬はdemoで叩かない。

## 11. Admin Dashboardで運用確認

1. `/admin` を開く。
2. LLM budget、job status、Cloud Tasks queue estimate、billing/admob status、audit logを説明する。
3. user detailでextension token revokeの導線を説明する。
4. dangerous operationはCSRF、role、recent auth、audit logで守ると説明する。

## 12. セキュリティ設計の説明

最後に以下を短くまとめる。

- Web: HttpOnly Session Cookie + Signed CSRF。
- Mobile: Bearer ID Token + App Check + version gate。
- Extension: scoped token。LLM生成権限なし。
- LLM: user/global budget、usage log、queue depth。
- Billing/Reward: Stripe webhook idempotency、AdMob SSV transaction id idempotency。
- Ops: Admin Dashboard、audit log、runbook、alert plan、QA checklist。
