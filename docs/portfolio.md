# Portfolio Notes

## 1. 作ったもの

Kindleハイライトを取り込み、AIで学習問題を生成し、回答・復習できる学習ツールです。Web、Chrome Extension、Mobileを想定し、ユーザーがKindle Notebookのハイライトから問題を作って学習できる流れを作りました。

単なるLLM呼び出しアプリではなく、認証、権限、課金、広告報酬、非同期ジョブ、管理画面、セキュリティ、運用手順まで含めたプロダクトとして整理しています。

## 2. 技術選定

- React + Vite: Web UIを軽量に作り、管理画面や学習画面をTypeScriptで管理するため。
- Go + Echo: 認証、ジョブ、課金、WebhookなどのAPIをシンプルかつ堅牢に作るため。
- PostgreSQL: ユーザー、ハイライト、問題、ジョブ、課金状態を整合性を持って扱うため。
- Cloud Run: コンテナ化したAPIを運用しやすくするため。
- Cloud Tasks: LLM生成やハイライト取り込みを同期処理から切り離すため。
- Firebase Auth: Web/Mobileの認証基盤を自前実装せず、安全に使うため。
- Chrome Extension MV3: Kindle Notebookからハイライトを取り込むため。
- Stripe: subscription checkoutとWebhook連携のため。
- AdMob SSV: 広告報酬をclient申告ではなくサーバー検証で扱うため。
- LLM provider abstraction: Geminiなど特定providerに閉じず、差し替えやmockをしやすくするため。

## 3. 工夫した点

- Web / Mobile / Extensionで認証方式を分けました。WebはSession Cookie、MobileはBearer ID Token、Extensionは専用scoped tokenです。
- Extension tokenはscopeを最小化し、LLM生成や課金APIには入れない設計にしました。
- LLM API濫用によるコスト爆発を、ユーザー単位budget、global budget、usage log、queue depthで抑えています。
- LLM生成はCloud TasksとDB CASで非同期化し、二重実行や重複ジョブを抑えました。
- StripeとAdMobは署名検証と冪等性を入れ、重複イベントで二重付与されないようにしました。
- prompt本文やhighlight本文を管理画面やログへ出さない方針にしました。
- secret scan、security headers、runbook、alert plan、production readiness、QA checklistまで整備しました。

## 4. 苦労した点

- Cookie認証とCSRF対策を両立させるところです。WebはCookieでUXを保ちつつ、状態変更はSigned CSRFで守る設計にしました。
- MobileとExtensionではWebと同じCookie前提にできないため、clientTypeごとに認証方式を切り替える必要がありました。
- LLM生成を同期処理にすると遅延や失敗の影響が大きいので、job tableとCloud Tasksで切り離しました。
- 複数アカウントやExtension token漏洩による濫用を想定し、scope、rate limit、revoke、audit logを入れました。
- Stripe / AdMobは成功画面やclient通知だけを信用せず、サーバー側の署名検証とidempotencyで処理しました。

## 5. 面接で話すポイント

- 認証と認可を分け、認証済みuser idをrequest bodyではなくcontextから取るようにした。
- clientTypeごとに認証方式を分けた。WebはCookie、MobileはBearer、Extensionは専用token。
- LLM cost controlをuser/global budgetとqueue制御で入れた。
- Cloud Tasks + DB CASで非同期処理の二重実行を防いだ。
- LLM provider abstractionによりmockやprovider差し替えをしやすくした。
- Stripe webhookとAdMob SSVを冪等に処理した。
- Secure-by-defaultなroute設計、CSRF、security headers、secret scanを入れた。
- Production readinessとしてrunbook、alert、load test、QA checklistまで準備した。

## 6. 今後の課題

- Mobile IAP / Play Billing対応。
- Admin dashboardの分析機能拡張。
- Kindle note field対応。
- 実運用環境へのCloud Armor適用。
- 監視alertの実設定と通知先整備。
- Chrome Web Store公開準備と最終審査対応。
