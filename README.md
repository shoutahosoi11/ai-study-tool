# AI Study Tool

AI Study Tool は、Kindle のハイライトや任意のテキストを AI 生成のクイズ問題に変換して問題を解いたり共有出来る学習アプリです。Web アプリ、Chrome 拡張機能、モバイルアプリを中心に構成し、Go 製バックエンドが認証、ハイライト取り込み、問題生成、課金、報酬トークン、運用管理を担当します。

## 主な機能

- Web / Mobile ユーザー向け Firebase Auth ログイン。
- Kindle Notebook のハイライトを取り込む Chrome 拡張機能のペアリングフロー。
- 取り込んだハイライトからの AI 問題生成。
- 解答、復習、間違えた問題、保存済み問題の学習フロー。
- プレミアム購読と報酬トークンモデル。
- 運用確認と限定的なサポート操作のための Admin Dashboard。
- セキュリティと運用の強化: セッション Cookie、署名付き CSRF、App Check、スコープ付き拡張トークン、予算制限、secret scan、セキュリティヘッダ、runbook、E2E / QA チェックリスト。

## 技術選定

### Firebase Authentication

Firebase Auth は Web、Mobile、Extension 共通のログイン基盤です。無料枠が大きく、マルチデバイス対応もしやすいため、初期段階から Web / Mobile / Extension を同じユーザー ID で扱いたいこのアプリに合っています。独自のユーザー管理サービスを作らず、同じ Firebase UID で 3 種類のクライアントを紐付けます。

一方で、各クライアントで安全なトークンの持ち方は違います。Firebase Auth を共通基盤にすると、ログイン基盤は共通化しつつ、Web は Session Cookie + CSRF Cookie、Mobile は Bearer Token + App Check、Extension は Pairing + Scoped Token という分担にしやすくなります。

- **Web**: Firebase ID Token を HttpOnly Session Cookie に交換し、生のトークンを JavaScript から読めるストレージに置きません。
- **Mobile**: Bearer ID Token を使い、App Check とアプリバージョン制限を組み合わせます。トークン更新は Firebase SDK に任せます。
- **Extension**: ペアリングフローで発行する短命の専用スコープ付きトークンを使います。これらのトークンは `highlight:write`、`highlight:check`、`extension:revoke-self` に限定され、LLM 生成、課金、アカウント更新 API には到達できません。

Firebase により、トークン失効、MFA フック、Google / Apple サインイン、メール確認も独自認証サービスなしで扱えます。

### Go + Echo

Go + Echo は、AI 開発アプリのバックエンドをシンプルかつ堅く作るために採用しています。Gemini API は HTTP 呼び出しであり、バックエンドの役割はルーティング、認証、DB トランザクション、ジョブキュー制御です。

具体的な理由:

- **`context.Context` によるキャンセル伝播**。ハイライト取り込みや LLM 問題生成は数秒かかることがあります。上流の HTTP タイムアウトやユーザーキャンセルを、DB クエリ、LLM 呼び出し、Cloud Tasks enqueue へ自然に伝播できます。
- **Echo の middleware が豊富**。認証、CSRF、CORS、rate limit、panic recovery、request logging などを共通化しやすく、エラーレベルに応じたログ出力、監視ツールへの通知、クライアントへの共通レスポンス整形も一括で扱えます。
- **`echo.Context` は interface**。handler テストで mock context を注入し、実 HTTP サーバーなしで認証や rate-limit の分岐を検証できます。
- **Cloud Run 上の単一バイナリ**。Dockerfile は静的バイナリを 1 つだけ作るため、イメージが小さく、起動が速く、メモリ使用量を読みやすくできます。ゼロスケール運用で重要です。
- **`database/sql` の接続プール**。Cloud Run は一気に多数インスタンスを立ち上げることがあります。`MaxOpenConns`、`MaxIdleConns` を明示し、Neon への接続スパイクを抑えます。
- **高速なコンパイルとビルド時エラー**。型や import の問題を早い段階で潰せます。
- **後方互換性が強い**。長期運用でのアップグレード負荷を抑えられます。

### React + React Native + Expo

Kindle は PC、スマホ、タブレットなど複数端末で使われるため、マルチデバイス対応が必須です。React / React Native の組み合わせは、ロジック層や状態管理ライブラリなどのエコシステムを共有しやすく、Flutter などの選択肢よりこのプロジェクトでは開発効率を出しやすいと判断しました。

Web アプリと Mobile アプリでは、Firebase Auth クライアント、API base URL、状態管理の考え方を共有しています。React Native を使うことで、ロジック層を別々のネイティブ実装として二重に書かずに済みます。

Mobile アプリでは Expo を使っています。共有シート、端末 API 連携、ビルドツールを初期段階から深いネイティブ設定なしで扱えるため、ハイライト取り込み、問題フロー、課金の検証可能な状態へ早く到達できます。

### Gemini API

Gemini はコンテキストウィンドウが広大です。複数のハイライトをまとめて送信し、1 回の API 呼び出しで複数問題を生成できます。ハイライトごとに 1 回呼び出す方式より、レイテンシとコストを抑えられます。

また、Google の検索結果を利用する grounding と親和性があり、将来的にリアルタイム情報を取り入れた問題生成や解説生成へ拡張しやすい点も選定理由です。現行の主要フローでは、コストと再現性を優先し、取り込んだハイライト本文を中心に問題を生成します。

バックエンドには provider adapter 層 (`infrastructure/gemini`, `infrastructure/openai`) があり、usecase logic を変えずに LLM provider の差し替えや併用ができます。

### Cloud Run + Cloud Tasks

Cloud Run はアイドル時にゼロスケールできるため、ユーザー数が少ない段階でもコストを低く保てます。負荷が増えた場合も手動のキャパシティ計画なしで自動スケールします。

Cloud Run は HTTP worker を直接叩けるため、通常 API と近い設計のまま非同期処理も組めます。Cloud Tasks は非同期の問題生成ジョブを処理します。キューの rate (`max-dispatches-per-second`) は「何秒に何回 worker を叩くか」を直接指定できます。各 task はおおむね 1 回の Gemini call なので、`1 task ≒ 1 Gemini call` として流量を見積もりやすい構造です。これは同時実行数を制御する Lambda concurrency limit より、この用途では単純です。

Global LLM Budget は PostgreSQL の日次カウンタです。キューが高速に空になっても、各 LLM 呼び出し前の予算チェックにより、ジョブ急増時のコスト超過を防ぎます。

### PostgreSQL + Neon

PostgreSQL は、問題生成 job やハイライト状態のような状態遷移を transaction と DB 制約で守りたいので採用しています。データに明確なリレーションがあり、強い整合性が必要です。

- ジョブ状態遷移 (queued → processing → completed/failed) は atomic である必要があります。ジョブの取りこぼしや二重処理はユーザーに見える不具合です。
- **partial unique index** により、同じユーザー・同じ本に対する active な生成ジョブの重複を防ぎます。
  ```sql
  CREATE UNIQUE INDEX uq_active_job_user_book
    ON question_generation_jobs (user_id, book_key)
    WHERE status IN ('queued', 'processing');
  ```
  この制約はアプリケーションコードだけでなく DB レベルで保証されます。
- ハイライト取り込みキューでは `FOR UPDATE SKIP LOCKED` により、複数の Cloud Run インスタンスが競合せずジョブを取得できます。
- UUID 型、配列、JSONB など型の選択肢が多く、ハイライト metadata や token scope を自然に表現できます。

Neon は PostgreSQL を Cloud Run と組み合わせて軽く運用できる managed PostgreSQL です。Cloud Run の水平スケールによる接続スパイクを connection pooler で吸収しやすく、branch 機能により本番データのコピーに対して staging や実験的な schema 変更を試せます。

### database/sql + sqlc

SQL を明示しながら型安全に扱える構成として、`database/sql` + sqlc を採用しています。DB 設計やクエリ性能を意識しやすく、生の SQL が持つ表現力と Go コンパイラが持つ強力な型安全を組み合わせられる点が利点です。

標準的な CRUD クエリは `backend/db/sqlc/` の SQL ソースから sqlc で生成します。生成コードは型安全で、元 SQL も読みやすく保てます。

複雑なクエリ、たとえば複数テーブル CTE、動的フィルタ、admin analytics は `repository/postgres/` と `infrastructure/persistence/` で `database/sql` を直接使って書いています。この分担により、単純な経路は sqlc、クエリの意図を隠したくない経路は raw SQL で扱えます。

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
npm run dev
```

拡張機能:

```bash
cd extension
npm run build
```

モバイル:

```bash
cd mobile
npx expo start --dev-client --lan --port 8081
```

データベース:

- backend の DB 接続には `DATABASE_URL` を使います。
- migration は `backend/db/migrations/` から適用します。
- `.env` ファイルや service account key は commit しないでください。

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

## ドキュメント一覧

ポートフォリオとデモ:

- [アーキテクチャ概要](docs/architecture-summary.md)
- [ポートフォリオメモ](docs/portfolio.md)
- [面接用メモ](docs/interview-notes.md)
- [デモ台本](docs/demo-script.md)
- [スクリーンショットチェックリスト](docs/screenshots.md)
- [今後のロードマップ](docs/future-roadmap.md)

セキュリティ:

- [セキュリティアーキテクチャ](docs/security-architecture.md)
- [クライアントセキュリティモデル](docs/security-clients.md)
- [セキュリティ runbook](docs/security-runbook.md)
- [Admin Dashboard 運用](docs/admin-dashboard.md)

運用とリリース:

- [Cloud Armor 運用計画](docs/ops-cloud-armor.md)
- [監視とアラート計画](docs/ops-monitoring-alerts.md)
- [本番準備チェックリスト](docs/production-readiness-checklist.md)
- [本番環境と secret](docs/env-production.md)
- [デプロイ / rollback runbook](docs/deploy-runbook.md)
- [smoke test チェックリスト](docs/smoke-test.md)
- [QA チェックリスト](docs/qa-checklist.md)

クライアントとテスト:

- [拡張機能 README](extension/README.md)
- [Extension Store 準備](extension/STORE_READINESS.md)
- [Mobile リリース準備](docs/mobile-release-readiness.md)
- [E2E テストガイド](e2e/README.md)
- [k6 負荷テスト](loadtest/k6/README.md)

## 制限事項 / 今後の作業

- Mobile IAP / Play Billing は未完了です。
- Kindle note field 対応は今後の拡張です。
- 本番 Cloud Armor と監視の例は、実環境への適用がまだ必要です。
- Chrome Web Store の最終スクリーンショット、store copy、review assets がまだ必要です。
- 実運用の alert routing、dashboard、on-call process は環境ごとの設定が必要です。
- Spaced repetition は現在の復習フローからさらに改善できます。

優先順位付きの計画は [今後のロードマップ](docs/future-roadmap.md) を参照してください。
