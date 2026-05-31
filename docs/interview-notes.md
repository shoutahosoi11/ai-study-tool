# Interview Notes

## 1分説明

KindleハイライトをChrome Extensionで取り込み、AIで問題を生成して回答・復習できる学習ツールを作りました。Web、Mobile、Extensionの3 clientを想定し、それぞれ認証方式を分けています。特に、LLMのコスト管理、非同期ジョブ、Stripe/AdMobの冪等処理、CSRFやApp Checkなど、本番運用を意識した設計に力を入れました。

## 3分説明

このアプリは、Kindle Notebookのハイライトを学習問題に変換するツールです。Webではログイン、問題回答、復習、Admin Dashboardを提供し、Chrome Extensionではハイライトを取り込みます。MobileはBearer TokenとApp CheckでAPIにアクセスします。

BackendはGo + Echoで、PostgreSQLを中心にユーザー、ハイライト、問題、ジョブ、budget、audit logを管理します。LLM生成は同期処理にせず、Cloud TasksとDB CASで非同期化しています。Stripe webhookとAdMob SSVは署名検証とidempotencyを入れて、二重処理を防ぎました。

セキュリティ面では、WebはHttpOnly Cookie + Signed CSRF、MobileはBearer ID Token + App Check、Extensionはscoped tokenに分けています。Extension tokenにはLLM生成権限を持たせず、漏洩時の被害範囲を小さくしました。

## 5分説明

一番重視したのは、LLMアプリとしての便利さだけでなく、濫用や運用まで含めて成立する設計にすることです。

ユーザーはWebでログインし、Extension接続画面でChrome Extensionを承認します。Extensionは専用tokenを受け取り、Kindle Notebookからハイライトを取り込みます。取り込まれたハイライトはPostgreSQLに保存され、条件を満たすとquestion generation jobが作られます。Cloud Tasks workerがjobをclaimし、LLM provider adapter経由で問題を生成します。

Web認証はSession CookieでUXを保ちつつ、Signed CSRFで状態変更を守ります。MobileはBearer ID TokenとApp Check、Extensionはscoped tokenです。この分離により、clientごとの攻撃面を小さくしています。

LLM濫用対策として、user budget、global budget、usage log、queue depthを入れています。StripeやAdMobはclient申告ではなくサーバー検証を使い、event idやtransaction idで冪等にしました。Admin Dashboardではbudget、job、extension token、billing event、audit logを確認でき、raw tokenやprompt本文などは表示しません。

## 深掘り質問への回答

### 1. なぜFirebase Authを使ったのか

認証の安全な実装を自前で抱えないためです。パスワード管理、ID token検証、session cookie、revokeなどをFirebaseに任せ、アプリ側は認可と業務ロジックに集中しました。

### 2. なぜWebはCookie認証なのか

WebではHttpOnly Cookieにsessionを置くことで、JavaScriptからcredentialを読めないようにできます。localStorage tokenよりXSS時の被害を抑えやすいです。

### 3. Cookie認証のCSRFはどう防ぐのか

状態変更APIではSigned CSRF tokenを要求します。CookieだけではPOSTできないようにし、CSRF tokenとsessionを組み合わせて検証します。

### 4. MobileはなぜBearer Tokenなのか

Native clientではCookie/CSRFよりBearer ID Tokenのほうが扱いやすいためです。代わりにApp Checkとversion gateを入れて、正規appからのアクセスに寄せています。

### 5. Extensionにはなぜ専用tokenを使うのか

ExtensionにFirebase tokenを持たせると権限が大きくなりすぎます。専用tokenにscopeを持たせ、highlight import/checkだけに限定することで漏洩時の被害範囲を抑えました。

### 6. LLM API濫用をどう防ぐのか

ユーザー単位のbudget、global budget、usage log、rate limit、queue depthで制御します。生成はjob化し、無制限にLLMを呼ばない設計にしています。

### 7. Cloud Tasksを使う理由

LLM生成は時間がかかり失敗もあり得るので、API request内で同期実行しないためです。Cloud Tasksならretryやqueue制御を外部化できます。

### 8. DB CASとは何か

DB上で条件付きUPDATEを使い、あるjobがまだqueuedのときだけprocessingに変更する方式です。複数workerが同時に動いても1つだけがclaimできます。

### 9. Stripe webhookの冪等性とは何か

同じevent idを複数回受け取っても、最初の1回だけ処理することです。Stripe webhookは再送される可能性があるため、event idを保存して二重処理を防ぎます。

### 10. AdMob SSVとは何か

広告報酬をサーバー側で検証する仕組みです。clientが「広告を見た」と申告するだけでは信用せず、AdMobの署名付き通知を検証します。

### 11. IDORをどう防いだか

user idをrequest bodyから信用せず、認証済みcontextから取得します。他人のデータにアクセスする処理ではrepository queryにuser_id条件を入れます。

### 12. XSSをどう防いだか

Reactの通常escapeに加え、secretやtokenを画面に出さない設計にしています。Web sessionはHttpOnly Cookieに置き、security headersとsafe URL検証も入れています。

### 13. PostgreSQLを選んだ理由

transaction、unique constraint、CAS、集計queryが必要だったためです。job処理や課金状態、audit logにはRDBの整合性が合っています。

### 14. Gemini依存を避けた理由

LLM providerは価格、品質、制限が変わりやすいためです。adapterに閉じ込めることで、mockや他providerへの差し替えをしやすくしました。

### 15. 本番運用で何を監視するか

Cloud Runの5xx/latency、Cloud Tasks queue depth、LLM budget使用率、job failure、Stripe webhook失敗、AdMob SSV失敗、rate limit、Admin audit logを監視します。
