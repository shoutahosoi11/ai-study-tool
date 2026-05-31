# Architecture Summary

## 全体構成

```txt
            ┌──────────────────┐
            │ React Web / Admin │
            └────────┬─────────┘
                     │ Web Session Cookie + CSRF
┌──────────────────┐ │
│ Expo Mobile App  ├─┤ Bearer ID Token + App Check
└──────────────────┘ │
                     ▼
            ┌──────────────────┐
            │ Go + Echo API     │
            └───────┬──────────┘
                    │
     ┌──────────────┼────────────────┐
     ▼              ▼                ▼
PostgreSQL     Cloud Tasks      External Services
users          question jobs    Firebase Auth
highlights     import jobs      LLM provider
questions                      Stripe
budgets                        AdMob SSV
audit logs
     ▲
     │ scoped extension token
┌────┴─────────────────┐
│ Chrome Extension MV3 │
│ Kindle Notebook import│
└──────────────────────┘
```

## ユーザー操作の流れ

1. ログイン: WebまたはMobileでFirebase Authを使ってログインする。
2. Extension接続: Extensionがpairing codeを発行し、Webでログイン済みユーザーが承認する。
3. Kindleハイライト取り込み: Extension tokenでハイライトimport APIを呼ぶ。
4. 問題生成: backendが条件を見てjobを作り、Cloud Tasks workerがLLMで問題を生成する。
5. 回答・復習: Web/Mobileで問題に回答し、間違えた問題や保存問題を復習する。
6. 課金/広告: Stripe subscriptionやAdMob SSVで利用量やreward tokenを管理する。
7. 管理/運用: Admin DashboardでLLM budget、job、billing、extension token、audit logを見る。

## 各技術の役割

- React + Vite: Web UI、Extension接続画面、Admin Dashboard。
- Go + Echo: API、認証middleware、usecase、task handler。
- PostgreSQL: 強い整合性が必要なユーザー、課金、ジョブ、問題、audit log。
- Cloud Run: API/worker実行基盤。
- Cloud Tasks: LLM生成やimport処理の非同期化。
- Firebase Auth: ユーザー認証。
- Chrome Extension MV3: Kindle Notebook連携。
- Stripe: subscription checkoutとWebhook。
- AdMob SSV: 広告報酬のサーバー検証。
- LLM provider abstraction: LLM provider差し替えとmock化。

## 選択理由と比較

### Firebase Auth vs 自前認証

Firebase Authを使うことで、パスワード管理、メールログイン、token検証、revokeなどを自前実装しなくて済みます。自前認証より実装量と事故リスクを減らせます。

### Session Cookie vs localStorage

WebではHttpOnly Session Cookieを使い、JavaScriptからsession credentialを読めないようにしました。localStorageにtokenを置く方式よりXSS時の被害を抑えやすいです。

### Bearer Token vs Cookie for Mobile

MobileではCookie/CSRFよりBearer ID Tokenのほうがnative clientと相性が良いです。App Checkとversion gateを組み合わせて、mobile clientの入口をWebと分けています。

### Extension Token vs Firebase Token

Extensionには専用scoped tokenを使いました。Firebase tokenをExtensionに置くより、漏洩時の権限を小さくできます。Extension tokenはhighlight import/checkに限定し、LLM生成や課金APIには使えません。

### PostgreSQL vs Firestore

PostgreSQLは複数テーブルの整合性、transaction、unique constraint、CAS更新、reporting queryに向いています。Firestoreよりjob制御、課金状態、audit logを扱いやすい判断です。

### Cloud Tasks vs goroutine

Cloud Tasksはプロセス再起動やスケールアウトに強く、retryやqueue制御を外部化できます。goroutineだけだとCloud Run instance終了時の処理保証や重複制御が弱くなります。

### DB CAS vs memory lock

DB CASは複数instanceで同時にworkerが動いても一貫して排他できます。memory lockは単一プロセス内でしか効かないため、Cloud Runの水平スケールに向きません。

### Stripe Webhook vs success_url

subscription状態はsuccess_urlではなくWebhookで確定します。success_urlはユーザーの画面遷移であり、支払い完了の信頼できる証拠ではありません。

### AdMob SSV vs client notification

広告報酬はclient通知ではなくAdMob SSVで検証します。client申告だけだと改ざんされやすいため、署名検証とtransaction id idempotencyを使います。
