# AI Study Tool

AI Study Tool は、Kindle のハイライトや任意のテキストから AI でクイズを生成し、解答・復習・共有できる学習アプリです。解いた問題に自分の解説や感想を添えて投稿でき、他のユーザーも投稿を読んだり問題に解答したりできます。

テキストの取り込み方法は次の3つです。

1. Chrome 拡張機能で Kindle Notebook (`read.amazon.co.jp/notebook` / `read.amazon.com/notebook`) からハイライトを取り込む。
2. モバイルの共有シートからテキストを送る。
3. アプリ内にテキストを貼り付ける。

Web・Mobile・Extension を Go 製バックエンドへ接続し、認証、取り込み、問題生成、課金、報酬トークン、運用管理をまとめて扱います。

## 主な機能

- Web / Mobile ユーザー向け Firebase Auth ログイン。
- Kindle Notebook のハイライトを取り込む Chrome 拡張機能のペアリングフロー。
- 取り込んだハイライトからの AI 問題生成。
- 解答、復習、間違えた問題、保存済み問題の学習フロー。
- プレミアム購読と報酬トークンモデル。
- 運用確認と限定的なサポート操作のための Admin Dashboard。
- セキュリティと運用の強化: セッション Cookie、署名付き CSRF、App Check、スコープ付き拡張トークン、予算制限、secret scan、セキュリティヘッダ、E2E smoke test。

## 技術構成

| 領域 | 採用技術 |
| --- | --- |
| Web Frontend | React / TypeScript / Vite / Tailwind CSS |
| Frontend Deploy | Cloudflare Workers Static Assets |
| Mobile | React Native / Expo |
| Chrome Extension | Manifest V3 / TypeScript |
| Backend | Go / Echo |
| Auth | Firebase Authentication |
| DB | Neon PostgreSQL |
| DB Access | database/sql / sqlc |
| Async Job | Cloud Tasks / Cloud Run HTTP worker |
| AI | Gemini API / OpenAI provider adapter |
| Payment | Stripe |
| Ad Reward Verification | AdMob Server-Side Verification |
| Backend Deploy | Cloud Run |
| Secret Management | Secret Manager |
| CI/CD | GitHub Actions |

Cloudflare・Cloud Run などはデプロイ構成の採用先です。設定ファイルや実装があることと、本番での設定・審査・公開が完了していることは区別しています。

## アーキテクチャ

バックエンドは、業務ルールを HTTP・DB・外部 SDK から分離するため、クリーンアーキテクチャの依存方向を採用しています。

```text
handler -> usecase -> domain <- infrastructure / repository
```

- `handler`: HTTP の入出力、認証済みユーザーの取得、リクエスト検証、DTO 変換。
- `usecase`: 業務ルール、トランザクション方針、外部サービスを呼ぶ順序の制御。
- `domain`: 業務上の型・定数・エラーと、永続化や外部サービスの interface。
- `infrastructure` / `repository`: PostgreSQL、Firebase、LLM、Stripe、Cloud Tasks などの外部 I/O。
- `di`: interface と具体的な実装の組み立て。

この分離により、usecase にテスト用実装を渡し、DB やネットワークに依存せず業務ルールを検証できます。AI を使った開発でも変更箇所の責務を絞りやすく、外部サービスの変更を業務ロジックへ広げにくくすることが狙いです。一方、interface・DTO・依存の組み立てが必要になるため、小さな機能にも一定の実装量がかかります。

現時点では単一の Go バックエンドとして運用する構成です。マイクロサービスの分散通信や個別デプロイを導入するより、モジュール境界を保ちながら開発・検証できることを優先しています。

問題生成は、取り込んだテキストを保存し、生成ジョブを作成して Cloud Tasks から HTTP worker を呼び、生成結果を保存する流れです。API リクエストから重い LLM 処理を切り離し、ジョブ状態と予算を確認しながら処理します。

## 技術選定

### Firebase Authentication

Firebase Auth は Web、Mobile、Extension 共通のユーザー識別基盤です。パスワード検証やトークン発行を自前で実装せず、同じ Firebase UID で複数のクライアントを紐付けます。Extension は Web で承認するペアリングを通じてユーザーに接続します。

一方で、各クライアントで安全なトークンの持ち方は違います。Firebase Auth を共通基盤にすると、ログイン基盤は共通化しつつ、Web は Session Cookie + CSRF Cookie、Mobile は Bearer Token + App Check、Extension は Pairing + Scoped Token という分担にしやすくなります。

- **Web**: Firebase ID Token で Web Session を作成し、HttpOnly Session Cookie と署名付き CSRF token で Cookie 認証のリクエストを保護します。Cookie と Bearer が同時にある場合も、バックエンドは Cookie を優先して CSRF を検証します。Firebase SDK のログイン状態管理と、サーバー側の Session Cookie は別の役割です。
- **Mobile**: Bearer ID Token を使い、App Check とアプリバージョン制限を組み合わせます。トークン更新は Firebase SDK に任せます。
- **Extension**: ペアリングフローで発行する短命の専用スコープ付きトークンを使います。これらのトークンは `highlight:write`、`highlight:check`、`extension:revoke-self` に限定され、LLM 生成、課金、アカウント更新 API には到達できません。

現在のログイン画面はメールアドレスとパスワードに対応しています。Web には Google ログイン用の関数がありますが、現在の画面には入口を設けていません。Apple ログイン・MFA・メール確認の導線も、Firebase 側の機能を利用できることと、このアプリへの実装が完了していることを区別しています。

### Go + Echo

Go + Echo は、AI 開発アプリのバックエンドをシンプルかつ堅く作るために採用しています。Gemini API は HTTP 呼び出しであり、バックエンドの役割はルーティング、認証、DB トランザクション、ジョブキュー制御です。

具体的な理由:

- **`context.Context` によるキャンセル伝播**。ハイライト取り込みや LLM 問題生成は数秒かかることがあります。リクエストのタイムアウトやキャンセルを、context に対応する DB クエリや外部 API 呼び出しへ伝えられます。enqueue 済みの非同期ジョブは元のリクエストとは別の実行単位です。
- **Echo の middleware が豊富**。認証、CSRF、CORS、rate limit、panic recovery、request logging などを共通化しやすく、エラーレベルに応じたログ出力、監視ツールへの通知、クライアントへの共通レスポンス整形も一括で扱えます。
- **`echo.Context` は interface**。handler テストで mock context を注入し、実 HTTP サーバーなしで認証や rate-limit の分岐を検証できます。
- **Cloud Run 上の単一バイナリ**。Dockerfile は静的バイナリを 1 つだけ作るため、イメージが小さく、起動が速く、メモリ使用量を読みやすくできます。ゼロスケール運用で重要です。
- **`database/sql` の接続プール**。Cloud Run は一気に多数インスタンスを立ち上げることがあります。`MaxOpenConns`、`MaxIdleConns` を明示し、Neon への接続スパイクを抑えます。
- **高速なコンパイルとビルド時エラー**。型や import の問題を早い段階で潰せます。
- **後方互換性が強い**。長期運用でのアップグレード負荷を抑えられます。

### React + React Native + Expo

Kindle は PC、スマホ、タブレットなど複数端末で使われるため、マルチデバイス対応が必須です。React / React Native の組み合わせは、ロジック層や状態管理ライブラリなどのエコシステムを共有しやすく、Flutter などの選択肢よりこのプロジェクトでは開発効率を出しやすいと判断しました。

Web と Mobile は TypeScript、React の状態管理、API 設計の考え方を揃えています。ただし、現在の API クライアントや画面コードはそれぞれのディレクトリにあり、共通パッケージとして全面的に共有しているわけではありません。端末固有の共有シートや認証処理は Mobile 側で扱います。

Mobile アプリでは Expo を使っています。共有シート、端末 API 連携、ビルドツールを初期段階から深いネイティブ設定なしで扱えるため、ハイライト取り込み、問題フロー、課金の検証可能な状態へ早く到達できます。

### Gemini API

Gemini はコンテキストウィンドウが広大です。複数のハイライトをまとめて送信し、1 回の API 呼び出しで複数問題を生成できます。ハイライトごとに 1 回呼び出す方式より、レイテンシとコストを抑えられます。

また、Google の検索結果を利用する grounding と親和性があり、将来的にリアルタイム情報を取り入れた問題生成や解説生成へ拡張しやすい点も選定理由です。現行の主要フローでは、コストと再現性を優先し、取り込んだハイライト本文を中心に問題を生成します。

バックエンドには provider adapter 層 (`infrastructure/gemini`, `infrastructure/openai`) があり、usecase logic を変えずに LLM provider の差し替えや併用ができます。

### Cloud Run + Cloud Tasks

Cloud Run はアイドル時にゼロスケールできるため、ユーザー数が少ない段階でもコストを低く保てます。負荷が増えた場合も手動のキャパシティ計画なしで自動スケールします。

Cloud Run は HTTP worker を直接呼べるため、通常 API と近い設計で非同期処理を実装できます。Cloud Tasks では送出レート (`max-dispatches-per-second`) と同時実行数を制御し、LLM への負荷を調整します。task 数と LLM 呼び出し数は再試行などで一致するとは限らないため、流量制御とは別に API 呼び出し前の予算確認も行います。

Global LLM Budget は PostgreSQL の日次カウンタです。キューが高速に空になっても、各 LLM 呼び出し前の予算チェックにより、ジョブ急増時のコスト超過を防ぎます。

### PostgreSQL + Neon

PostgreSQL は、問題生成 job やハイライト状態のような状態遷移を transaction と DB 制約で守りたいので採用しています。データに明確なリレーションがあり、強い整合性が必要です。

- ジョブ状態遷移 (queued → processing → completed/failed) は atomic である必要があります。ジョブの取りこぼしや二重処理はユーザーに見える不具合です。
- **partial unique index** により、同じユーザー・同じ本に対する active な生成ジョブの重複を防ぎます。
  ```sql
  CREATE UNIQUE INDEX uq_question_generation_jobs_active
    ON question_generation_jobs (user_id, book_key)
    WHERE status IN ('queued', 'processing', 'enqueue_failed');
  ```
  この制約はアプリケーションコードだけでなく DB レベルで保証されます。
- ジョブの取得では、更新条件に現在の状態を含めることで、同じジョブを複数の worker が同時に処理開始することを防ぎます。
- UUID 型、配列、JSONB など型の選択肢が多く、ハイライト metadata や token scope を自然に表現できます。

Neon は PostgreSQL を Cloud Run と組み合わせて軽く運用できる managed PostgreSQL です。Cloud Run の水平スケールによる接続スパイクを connection pooler で吸収しやすく、branch 機能により本番データのコピーに対して staging や実験的な schema 変更を試せます。

### database/sql + sqlc

SQL を明示しながら型安全に扱える構成として、`database/sql` + sqlc を採用しています。DB 設計やクエリ性能を意識しやすく、生の SQL が持つ表現力と Go コンパイラが持つ強力な型安全を組み合わせられる点が利点です。

標準的な CRUD クエリは `backend/db/sqlc/` の SQL ソースから sqlc で生成します。生成コードは型安全で、元 SQL も読みやすく保てます。

複雑なクエリ、たとえば複数テーブル CTE、動的フィルタ、admin analytics は `repository/postgres/` と `infrastructure/persistence/` で `database/sql` を直接使って書いています。この分担により、単純な経路は sqlc、クエリの意図を隠したくない経路は raw SQL で扱えます。

### Cloudflare

Web は Vite で静的ファイルへビルドし、Cloudflare Workers Static Assets で配信する構成です。`wrangler.toml` で `frontend/dist` と SPA のフォールバックを指定しています。Web の配信と Go API の実行を分け、フロントエンドの配信のために API サーバーを増やさずに済む構成を選びました。

### Stripe / AdMob

Web のプレミアム購読には Stripe Checkout と webhook を使います。カード情報の入力を Stripe に任せ、購読状態は署名検証済みの webhook から反映します。イベント ID の記録により重複通知を扱います。

広告報酬には AdMob の Server-Side Verification を使い、署名検証と transaction ID の重複判定を行います。クライアントからの自己申告だけで報酬を付与しないための設計です。Mobile IAP / Google Play Billing は別の課題として残っています。

### Secret Manager

本番の接続文字列や API キーは Secret Manager から Cloud Run に注入する構成です。コード・コンテナイメージ・Web や Mobile の公開設定に backend secret を含めず、実行環境側で管理します。

### GitHub Actions

PR で secret scan、backend のテスト・脆弱性検査・ビルド、クライアントの型検査・依存監査・テストなどを実行します。backend では DB integration test と並行処理に関わる race test も定義しています。E2E は別 workflow で扱います。

Cloud Run へのデプロイ workflow は、必要な環境変数・secret 参照を検証してからビルドとデプロイを行います。workflow の定義があることと、各環境の設定や実行結果が正常であることは別途確認が必要です。

## セキュリティ要点

- Web は HttpOnly Session Cookie 認証と署名付き CSRF token を使います。
- Mobile は Firebase Bearer ID Token、App Check、アプリバージョン制限を使います。
- Browser Extension は専用スコープ付きトークンを使います。拡張トークンはハイライトの import / check はできますが、LLM 生成 API は呼べません。
- LLM 利用はユーザー別予算、全体の日次予算、利用ログ、キュー深度制限、非同期 worker で保護します。
- Cloud Tasks と DB compare-and-set により、ジョブの二重処理を防ぎます。
- Stripe webhook は署名検証と event 冪等性を使います。
- AdMob reward は SSV 署名検証と transaction id 冪等性を使います。
- Admin API は Web Session 認証、role check、CSRF、危険操作の recent auth、audit log を必須にします。
- CI には secret scan とセキュリティ観点のテストを含めています。

## デモ手順

ステージングまたはローカルのデモアカウントとダミーハイライトデータを使います。

1. Web アプリにログインする。
2. Chrome 拡張機能を開き、ペアリングを開始する。
3. `/extension/connect` で拡張機能コードを承認する。
4. サンプル Kindle Notebook ハイライトを取り込む。
5. 取り込んだハイライトから問題を生成する。
6. 問題に答え、間違えた問題 / 保存済み問題を復習する。
7. 実課金なしで premium / reward-token の制限を説明する。
8. `/admin` を開き、運用 overview、job state、LLM budget、audit log を見せる。

デモでは、実メールアドレス、secret、生 token、cookie、webhook payload、SSV query string、prompt、ハイライト本文を表示しないでください。

## ローカル開発

バックエンド:

```bash
cd backend
go run ./cmd/main.go
```

フロントエンド:

```bash
cd frontend
npm ci
npm run dev
```

拡張機能:

```bash
cd extension
npm ci
npm run build
```

モバイル:

```bash
cd mobile
npm ci
npx expo start --dev-client --lan --port 8081
```

データベース:

- backend の DB 接続には `DATABASE_URL` を使います。
- migration は `backend/db/migrations/` から適用します。
- `.env` ファイルや service account key は commit しないでください。

初回起動には PostgreSQL、Firebase の開発用設定、各クライアントの API 接続先が必要です。`backend/.env.example`、`frontend/.env.example`、`mobile/.env.example` を参照し、実値は Git 管理外の環境変数やローカル設定に置いてください。Go は `backend/go.mod` に従う toolchain、Node.js は現在の CI に合わせて 22 系を使います。

Cloud Tasks を使う非同期生成は、キュー・worker の URL・呼び出し認証の設定が必要です。アプリを起動しただけで、本番サービスとの連携まで完了するわけではありません。

物理端末の Mobile では、`EXPO_PUBLIC_API_BASE_URL` に Mac の LAN アドレスを設定し、base path として `/api/v1` を含めてください。

## テストコマンド

バックエンド:

```bash
cd backend && go test ./... && go build ./...
```

フロントエンド:

```bash
cd frontend && npm run typecheck && npm test && npm run build
```

拡張機能:

```bash
cd extension && npm run typecheck && npm test && npm run build
```

モバイル:

```bash
cd mobile && npm run typecheck && npm test
```

E2E:

```bash
cd e2e && npm run test
```

セキュリティスキャン:

```bash
python3 scripts/secret_scan.py
```

## ドキュメント

運用・設計資料はリポジトリ外のローカルアーカイブで管理しています。
`docs/` は Git 管理対象外です。

## 制限事項 / 今後の作業

- Mobile IAP / Google Play Billing は未完了です。
- Google ログインの画面導線、Apple ログイン、MFA、メール確認フローは追加対応が必要です。
- Kindle note field 対応は今後の拡張です。
- 本番 Cloud Armor と監視の例は、実環境への適用がまだ必要です。
- Chrome Web Store の最終スクリーンショット、store copy、review assets がまだ必要です。
- 実運用の alert routing、dashboard、on-call process は環境ごとの設定が必要です。
- Spaced repetition は現在の復習フローからさらに改善できます。
