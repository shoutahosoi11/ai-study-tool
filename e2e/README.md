# E2E テスト

このディレクトリには、PR14 時点の最初の smoke E2E suite を置いています。Playwright を使い、初期状態では dry-run 動作に寄せています。そのため、実 Firebase、Stripe、AdMob、LLM traffic なしでローカルのフロントエンドに対して実行できます。

## 安全な初期設定

環境変数:

- `E2E_BASE_URL`: フロントエンドの base URL。既定値は `http://127.0.0.1:3000`。
- `E2E_API_BASE_URL`: バックエンドの base URL。既定値は `http://127.0.0.1:8080`。
- `E2E_TEST_EMAIL`: ステージング用テストユーザーのメールアドレス。テストログに出さないでください。
- `E2E_TEST_PASSWORD`: ステージング用テストユーザーのパスワード。テストログに出さないでください。
- `E2E_ADMIN_EMAIL`: ステージング用 admin メールアドレス。テストログに出さないでください。
- `E2E_ADMIN_PASSWORD`: ステージング用 admin パスワード。テストログに出さないでください。
- `E2E_EXTENSION_TOKEN`: 使い捨てのステージング用 extension token。テストログに出さないでください。
- `E2E_ALLOW_PRODUCTION`: 本番に近い HTTPS host に対して実行する場合のみ `true` にします。
- `E2E_DRY_RUN`: 既定値は `true`。使い捨てステージングでのみ `false` にします。
- `E2E_RUN_API_TESTS`: 使い捨てバックエンドが動いている場合のみ `true` にします。
- `E2E_SKIP_WEB_SERVER`: `E2E_BASE_URL` がすでに配信されている場合に `true` にします。

Playwright config は、`E2E_ALLOW_PRODUCTION=true` が明示されていない限り本番に近い URL を拒否します。

## インストール

```bash
cd e2e
npm install
npx playwright install chromium
```

## 実行

ローカルフロントエンドの smoke:

```bash
cd e2e
npm run test
```

使い捨てバックエンドの security smoke:

```bash
cd e2e
E2E_RUN_API_TESTS=true E2E_API_BASE_URL=http://127.0.0.1:8080 npm run test
```

事前起動済み service を使うステージング:

```bash
cd e2e
E2E_SKIP_WEB_SERVER=true \
E2E_BASE_URL=https://staging.example.com \
E2E_API_BASE_URL=https://staging-api.example.com \
E2E_RUN_API_TESTS=true \
npm run test
```

## 現在のカバレッジ

- Web login page smoke。
- `/extension/connect` の protected route redirect。
- Extension connect page の sensitive text regression。
- Admin route auth gate smoke。
- 未認証 Admin と Extension import rejection の optional API smoke。
- backend security header の optional smoke。

## 後回しにしている完全 E2E

完全なログイン、ペアリング承認、ハイライト取り込み、問題生成、解答 / 復習、Stripe test checkout、AdMob SSV、Admin mutation flow には、seed 済み Firebase user、test extension token、mock LLM provider、cleanup credential を持つ使い捨てステージング環境が必要です。

## テストデータ方針

- ステージングまたは使い捨てローカル project のみを使います。
- 作成する record には `e2e_` または `test_` prefix を付けます。
- 実 Stripe charge や実 AdMob reward traffic は発生させません。
- password、cookie、生 token、生 webhook payload、生 SSV query string、prompt、ハイライト本文をログに出さないでください。
- cleanup では test prefix のデータを削除または revoke し、使い捨て extension token を各 run 後に revoke してください。
