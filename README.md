# AI Study Tool

**同じ教材を学ぶ人の解説や考え方を、次の学習者へ残す。**

AI Study Toolは、KindleのハイライトからAIで問題を作り、解答・復習と、解説の共有ができる学習アプリです。

問題には自分の解説や考察を添えて投稿できます。他の人が同じ内容をどう理解し、どのような周辺知識と結びつけているのかを知ることで、自分だけでは気づかなかった視点から学べます。

ハイライトの取り込み → 問題生成 → 解答・復習 → 解説や考察の投稿・共有

## 画面・デモ

<!-- 公開前に実際のスクリーンショット2〜3枚と、公開URLまたは短い操作動画を追加する。参考READMEの画像は流用しない。 -->

画面画像・操作動画は準備中です。掲載する画面は、次の3つを予定しています。

| 問題を作る | 解答・復習する | 他の人の解説を読む |
| --- | --- | --- |
| Kindleから取り込んだハイライトを選び、問題を生成する。 | 問題に答え、間違えた問題や保存した問題を解き直す。 | 共有された問題を解き、投稿者の解説や考察を読む。 |

## 開発背景と解決したい課題

大学で過去問を使って試験勉強をする中で、同じ本や教科書を使っていても、人によって理解の深さや成績に差があることが気になっていました。答えだけでなく、理解している人がどのように考え、何と結びつけて覚えているのかを知りたいと思ったことが、開発の出発点です。

また、大学の試験対策では、担当者が毎年似た内容の解説を書き込んでいました。先輩が考えた解説や思考過程、周辺知識を蓄積していけば、次の学年はそれを土台に学べるのではないかと考えました。

そこで、問題に自分の解説や考察を添えて共有できる学習アプリを作りました。問題を共通の題材にすることで、同じ内容に対する他の人の着眼点や考え方を読めるようにしています。

Kindleのハイライトを問題にするのは、読書中に重要だと思った箇所を、そのまま理解度の確認に使えるようにするためです。問題作成の負担をAIで減らし、解答や、自分の言葉で解説を残すことに時間を使えるようにしました。

現在は問題と解説の投稿・閲覧を実装しています。教材ごとに知見を整理し、学年を越えて引き継げる学習環境へ育てていきたいと考えています。

## 主な機能と実装状況

| 機能 | 内容 | 状況 |
| --- | --- | --- |
| テキスト取り込み | Chrome拡張機能でのKindleハイライト取り込み、テキスト貼り付け、モバイル共有シートからの取り込み。 | 実装済み |
| AI問題生成 | 取り込んだ本文から選択式問題を生成する。生成処理は非同期ジョブとして実行する。 | 実装済み |
| 解答・復習 | 問題への解答、間違えた問題・保存した問題の復習。 | 実装済み |
| 解説の共有 | 問題に解説や考察を添えて投稿し、他のユーザーの投稿を閲覧・解答する。 | 実装済み |
| ユーザー登録・認証 | メールアドレスとパスワードによる登録・ログイン、プロフィール編集、退会。 | 実装済み |
| 運用管理 | 管理者向けのジョブ状態、LLM利用予算、監査ログの確認。 | 実装済み |
| Web課金・広告報酬 | Stripe購読、AdMobのサーバー側検証による報酬付与。 | 実装あり。実環境での設定・確認が必要 |
| モバイルのアプリ内課金 | Apple / Googleの課金への対応。 | 未完了 |

「実装済み」はコードと画面・APIの実装状況を示します。ストア審査や本番公開、各外部サービスとの接続確認の完了を示すものではありません。

## 開発期間・体制・担当

| 項目 | 内容 |
| --- | --- |
| 開発期間 | 約1か月半 |
| 開発体制 | 個人開発 |
| 担当範囲 | 要件整理、設計、Web・モバイル・Chrome拡張機能・バックエンドの実装、テスト、インフラ設定まで全体を担当。 |
| AIの利用範囲 | 設計の壁打ち、実装、QA観点の壁打ちに使用。 |

## 特に工夫した点

### 1. 問題生成を非同期化し、同じジョブの重複処理を抑える

**課題：** ハイライトをまとめて取り込むと、問題生成の外部API呼び出しに時間がかかります。また、生成ボタンの再操作やタスクの再配信によって、同じ本の生成処理が重なる可能性があります。

**選択した方法：** 生成要求をDBのジョブとして保存し、Cloud TasksからHTTP workerを呼び出します。同じユーザー・同じ本の実行中ジョブはPostgreSQLの部分ユニークインデックスで制限し、workerは`queued`のジョブだけを条件付き更新で取得します。

**理由：** HTTPリクエストの中で生成完了を待つ方法に比べ、画面からの要求と時間のかかる生成処理を分けられます。プロセス内のロックだけに頼らずDBでも制限することで、複数のworkerが動く場合にも同時処理を制御できます。一方、生成完了までの待ち時間や、投入に失敗したジョブの状態管理が必要になります。

**検証：** workerの単体テストでは、取得済みジョブでLLMを呼ばないことを確認します。DB統合テストには、実行中ジョブの重複作成を拒否し、同じジョブを2回取得しても2回目は処理を開始しないケースを用意しています。実行結果は後述の「テストと検証結果」に記載しています。

実装・テスト：[worker](backend/internal/usecase/question_worker_usecase.go) / [workerの単体テスト](backend/internal/usecase/question_worker_usecase_test.go) / [DB統合テスト](backend/internal/infrastructure/persistence/question_generation_job_repository_integration_test.go)

### 2. LLMを呼ぶ前に、利用枠とサービス全体の予算を確保する

**課題：** キューの同時実行数だけを制限しても、処理が続けばLLMの呼び出し回数と費用は増えます。複数のworkerが同時に残枠を確認すると、上限を超える可能性もあります。

**選択した方法：** ユーザーの日次生成枠に加え、サービス全体の日次リクエスト数と推定費用の予算を管理します。LLM呼び出し前にDBで予算を確保し、利用記録を残します。

**理由：** 呼び出し後に集計する方法に比べ、上限に達した時点で次の呼び出しを止められます。キューの流量制御と併用して、負荷と利用量を別々に制御しています。費用は推定値のため、実際の請求額を厳密に保証する仕組みではありません。

**検証：** 予算超過時にLLMを呼ばないケースと、並行した予算確保でもリクエスト数の上限を超えないケースを単体テストで確認します。

実装・テスト：[予算管理](backend/internal/usecase/global_llm_budget_usecase.go) / [予算管理のテスト](backend/internal/usecase/global_llm_budget_usecase_test.go) / [workerのテスト](backend/internal/usecase/question_worker_usecase_test.go)

### 3. 拡張機能には取り込み専用の権限を渡す

**課題：** Web・モバイル・Chrome拡張機能は同じユーザーのデータを扱いますが、必要な権限と認証情報の持ち方が異なります。ハイライトを取り込む拡張機能に、課金やアカウント変更までできる権限は不要です。

**選択した方法：** WebはHttpOnly Session CookieとCSRF検証、モバイルはFirebase ID TokenとApp Checkを使います。拡張機能はWebでのペアリング承認後に専用トークンを発行し、ハイライトの取り込み・照会と自身のトークン失効に権限を限定します。

**理由：** すべてのクライアントに同じ権限を渡す方法に比べ、拡張トークンで実行できる操作を小さくできます。その分、トークンの期限・失効・ペアリングを管理する実装が必要になります。

**検証：** 拡張トークンで取り込みを許可し、問題生成・課金を拒否するケースをテストしています。WebではCookieとBearerが同時に送られてもCookie認証を優先し、更新操作にCSRF検証を適用するケースを確認します。

実装・テスト：[APIルーティング](backend/internal/router/router.go) / [拡張機能の認証テスト](backend/internal/middleware/extension_auth_test.go) / [認証方式の切り替えテスト](backend/internal/middleware/hybrid_auth_test.go)

## 構成図・使用技術

```mermaid
flowchart LR
    Web[Web / Cloudflare] --> API
    Mobile[Mobile / Expo] --> API
    Extension[Chrome Extension] --> API
    Web --> Auth[Firebase Authentication]
    Mobile --> Auth
    subgraph CloudRun[Cloud Run]
        API[Go / Echo API]
        Worker[HTTP worker]
    end
    API --> DB[(Neon PostgreSQL)]
    API --> Queue[Cloud Tasks]
    Queue --> Worker
    Worker --> DB
    Worker --> LLM[Gemini API]
```

| 領域 | 使用技術 |
| --- | --- |
| バックエンド | Go / Echo / database/sql / sqlc |
| Web | React / TypeScript / Vite / Tailwind CSS |
| モバイル | React Native / Expo |
| Chrome拡張機能 | Manifest V3 / TypeScript |
| 認証 | Firebase Authentication / App Check |
| DB・非同期処理 | Neon PostgreSQL / Cloud Tasks |
| AI | Gemini API（OpenAI用アダプターの実装もあり） |
| 課金・広告報酬 | Stripe / AdMob SSV |
| デプロイ・機密情報管理 | Cloud Run / Cloudflare Workers Static Assets / Secret Manager |
| テスト・CI | Go testing / Vitest / Jest / Playwright / GitHub Actions |

バックエンドは`handler → usecase → domain`を中心に、DBや外部サービスの実装を分離しています。上記の構成図はデプロイ設定上の構成です。

問題生成時には、取り込んだ本文をLLMへ送信します。ユーザーが投稿した問題や解説は、他のユーザーから閲覧できる共有情報になります。

## テストと検証結果

2026年10月22日に、mainの`cc291b9`に対応するGitHub Actionsの実行結果を確認しました。対象の実行日は2026年10月15日です。

| 確認項目 | 結果 |
| --- | --- |
| バックエンドの`go test ./...` | 成功。DB統合テストは接続設定がない場合にスキップされます。 |
| Web・モバイル・拡張機能の型検査 | 成功。WebのLintも成功。 |
| Secret Scan | 成功。 |
| Goの脆弱性検査、各クライアントの依存関係監査 | 失敗。CI全体は失敗しています。 |
| DB統合テスト・race test・バックエンドビルド | 脆弱性検査の失敗により後続ステップがスキップ。 |
| Web・モバイル・拡張機能のテスト、拡張機能ビルド | 依存関係監査の失敗により後続ステップがスキップ。 |
| Playwrightによる画面のスモークテスト | 成功。APIスモークテストは別途有効化が必要で、この実行では対象外。 |

実行記録：[CI](https://github.com/shoutahosoi11/ai-study-tool/actions/runs/38676084375) / [E2E](https://github.com/shoutahosoi11/ai-study-tool/actions/runs/38676084365)

テストの定義があることと、実際に実行・成功していることを分けて記載しています。負荷試験の性能値や、学習効果の定量評価は掲載していません。

## 起動手順

以下は新規ローカルDBでWebとAPIを起動する手順です。設定・スクリプトとの照合を行っていますが、このREADME改稿時点では新規環境での通し起動を再実施していません。

### 必要なもの

- Git、Node.js 22系、Go（[go.mod](backend/go.mod)で指定する1.26.5）、Docker Compose v2。
- 開発用Firebaseプロジェクト。Authenticationのメール／パスワード認証を有効にし、Webアプリを登録して、認証済みドメインに`localhost`を設定する。
- 同じFirebaseプロジェクトの開発用サービスアカウント認証情報。ファイルはリポジトリの外に置く。

### 1. 取得と設定

```bash
git clone https://github.com/shoutahosoi11/ai-study-tool.git
cd ai-study-tool
cp backend/.env.example backend/.env
cp frontend/.env.example frontend/.env.local
```

`backend/.env`で次の項目を設定します。

| 変数 | ローカルでの設定 |
| --- | --- |
| `APP_ENV` | `development` |
| `DATABASE_URL` | `postgres://postgres:postgres@localhost:5432/ai_study_tool?sslmode=disable` |
| `FIREBASE_CREDENTIALS_PATH` | 開発用サービスアカウントJSONの絶対パス |
| `CSRF_SECRET` | `openssl rand -hex 32`などで生成した値 |
| `ALLOWED_ORIGINS` | `http://localhost:3000,http://127.0.0.1:3000` |
| `USE_GEMINI_MOCK` | 実LLMを使わず基本画面を確認する場合は`true`。実LLMを使う場合は`false`にして`GEMINI_API_KEY`を設定。 |

`frontend/.env.local`の`VITE_FIREBASE_*`をFirebaseのWebアプリ設定で埋めます。`VITE_API_BASE_URL`は`/api/v1`を使います。ViteがAPIリクエストを`localhost:8080`へ転送します。

### 2. DBの起動と初回マイグレーション

```bash
docker compose up -d postgres
docker compose exec -T postgres pg_isready -U postgres -d ai_study_tool
```

DBが接続を受け付ける状態になったら、リポジトリのルートで実行します。初期化済みDBへの再適用は想定していません。

```bash
bash -eu <<'SH'
for file in backend/db/migrations/*.sql; do
  docker compose exec -T postgres psql \
    -U postgres -d ai_study_tool -v ON_ERROR_STOP=1 < "$file"
done
SH
```

### 3. APIとWebの起動

API用のターミナルで実行します。

```bash
cd backend
go run ./cmd/main.go
```

別のターミナルで、リポジトリのルートから実行します。

```bash
cd frontend
npm ci
npm run dev
```

[Web画面](http://localhost:3000)でユーザー登録・ログインします。[APIのヘルスチェック](http://localhost:8080/health)と[DB接続を含む準備状態](http://localhost:8080/ready)でも起動を確認できます。

### 4. 問題生成まで動かすための追加設定

上記だけではCloud Tasksからのジョブ実行は設定されません。問題生成と非同期取り込みの実演には、次の設定が必要です。`USE_GEMINI_MOCK=true`はLLMの代替実装を使う設定で、キューやworkerの実行を代替するものではありません。

- `QUEUE_QUESTION_GENERATION`、`QUEUE_HIGHLIGHT_IMPORT`：作成済みキューのフルパス。作成例は[setup-queues.sh](deploy/cloudtasks/setup-queues.sh)を参照。
- `TASK_HANDLER_BASE_URL`：Cloud Tasksから到達できるAPIのベースURL。ローカルの`localhost`は指定できません。
- `INTERNAL_TASK_INVOKER_SERVICE_ACCOUNT`とGoogle Cloudの認証・IAM設定。本番ではOIDC認証が必須です。開発環境では`INTERNAL_TASK_SECRET`による経路もあります。
- 実LLMで生成する場合は`GEMINI_API_KEY`。

### Chrome拡張機能

リポジトリのルートから実行します。

```bash
cd extension
npm ci
npm run build
```

ローカルAPIを使う場合は`manifest.development.json`を`manifest.json`として利用します。Chromeの拡張機能ページでデベロッパーモードを有効にし、`extension/`を「パッケージ化されていない拡張機能」として読み込みます。

拡張機能の設定画面でAPIの接続先を設定し、ペアリングコードをWebの`/extension/connect`で承認します。その後、Kindle Notebookからハイライトを取り込みます。

### モバイル

`mobile/.env.example`を`mobile/.env`へコピーし、Firebase設定と`EXPO_PUBLIC_API_BASE_URL`を設定します。

- iOSシミュレーター：`http://localhost:8080/api/v1`
- Androidエミュレーター：`http://10.0.2.2:8080/api/v1`
- 実機：同じネットワーク上の開発PCのLANアドレスを使い、末尾に`/api/v1`を含める。

```bash
cd mobile
npm ci
npm run ios
```

iOSはXcodeなどのネイティブ開発環境が必要です。AndroidはAndroid開発環境を用意し、`npm run android`を使います。共有シートの動作確認には開発ビルドを使います。

## 今後の課題

- 依存関係・脆弱性検査の失敗を解消し、スキップされている統合テスト・race test・クライアントテストの結果を更新する。
- 新規環境で起動手順を通し、ハイライト取り込みから解説共有までの操作動画を追加する。
- 同じ教材の解説を探しやすく整理し、学年を越えて知見を引き継ぐ仕組みを検討する。
- 問題の品質と、解答・復習が学習に役立つかを利用者と検証する。
- モバイルのアプリ内課金、追加のログイン方式、ストア公開、監視・通知の運用設定を進める。
