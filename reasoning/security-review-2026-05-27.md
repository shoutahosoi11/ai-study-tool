# Security Review — 2026-05-27

## Review Scope

`main` ブランチに未コミットの変更とトラックされていない新規ファイル一式に対する、セキュリティ観点のレビュー。
新規に追加・変更されたコードによって導入される **High-confidence (>0.8) な HIGH / MEDIUM 脆弱性のみ** を対象とする。既存のセキュリティ懸念、DoS、レート制限、ディスク上のシークレットは対象外。

## Reviewed Files / Areas

### Backend (Go)

- 認証・認可
  - `backend/internal/domain/extension_token.go`, `extension_pairing.go`, `auth_context.go`
  - `backend/internal/middleware/auth.go`, `session_auth.go`, `hybrid_auth.go`, `extension_auth.go`
  - `backend/internal/middleware/scope.go`, `client_type.go`, `recent_auth.go`
  - `backend/internal/handler/extension_handler.go`, `auth_handler.go`
  - `backend/internal/usecase/extension_usecase.go`
  - `backend/internal/infrastructure/persistence/extension_token_repository.go`
- CSRF
  - `backend/internal/middleware/csrf.go`, `csrf_test.go`
- Stripe Webhook
  - `backend/internal/infrastructure/stripe/webhook_validator.go`
  - `backend/internal/infrastructure/persistence/billing_repository.go`
  - `backend/internal/usecase/billing_usecase.go`
- AdMob SSV
  - `backend/internal/infrastructure/admob/ssv_verifier.go`
  - `backend/internal/usecase/token_usecase.go`, `backend/internal/handler/token_handler.go`
  - `backend/internal/domain/admob.go`
- App Check
  - `backend/internal/middleware/app_check.go`, `app_version.go`
  - `backend/internal/infrastructure/firebase/app_check_verifier.go`
- Internal Task Auth
  - `backend/internal/middleware/internal_task.go`, `backend/internal/router/router.go`
- セキュリティヘッダ・レート制限
  - `backend/internal/middleware/security_headers.go`, `rate_limit.go`
- マイグレーション
  - `backend/db/migrations/046_create_extension_tokens.sql` … `050_add_extension_pairing_user_code.sql`
- LLM コスト上限
  - `backend/internal/usecase/global_llm_budget_usecase.go`
  - `backend/internal/infrastructure/persistence/global_llm_budget_repository.go`

### Frontend / Mobile

- `frontend/src/components/common/Avatar.tsx`, `frontend/src/lib/safeUrl.ts`
- `mobile/src/api/client.ts`, `mobile/src/api/app-check.ts`, `mobile/src/share.ts`

### Scripts / Workflow

- `scripts/secret_scan.py`, `scripts/secret_scan.sh`, `.github/workflows/ci.yml`

---

## Findings by Area

### Authentication & Authorization

- Extension token は raw → `sha256` → hex で保存。検証は hash 値の DB lookup で行い、`revoked_at IS NULL` と `expires_at > now` を SQL 側で必須化。
- `setExtensionAuth` が DB から取得したスコープを `NormalizeExtensionScopes` でフィルタしており、`allowedExtensionScopes` (`highlight:write` / `highlight:check` / `extension:revoke-self`) 以外は **常に剥がれる**。DB 改ざんで権限昇格してもルートに到達できない設計。
- Pairing フローは UUID (サーバ生成) を pairing_id とし、`ApprovePairing` は Session + CSRF + `RequireClientType(Web)` で多重防御。`CreateExtensionTokenForApprovedPairing` は `SELECT ... FOR UPDATE` と `used_at IS NULL` 条件付き UPDATE で claim の二重消費を防止。
- `RequireScope` は Extension クライアントにのみスコープを強制し、Web/Mobile は既存のルートガードに委譲する明示的な設計。
- `RequireRecentAuth` は 5 分以内の `auth_time` を要求。Extension は `BillingWrite` スコープを持てないため、`/checkout/session` への到達が構造的に不可能。

**判定**: 高確度の脆弱性なし。

### CSRF

- `Protect` は GET/HEAD/OPTIONS を除外し、Origin allowlist (`ALLOWED_ORIGINS`) と `Sec-Fetch-Site` の二段確認、`hmac.Equal` での定数時間比較、`firebase_uid` に紐づく HMAC-SHA256 署名検証を実施。
- `NewCSRFMiddlewareWithConfig` は strict environment (`production` / `staging` / `preview`) で `CSRF_SECRET` 空 / `CSRF_SIGNING_DISABLED=true` / `ALLOWED_ORIGINS` 空をすべて起動時に拒否する fail-closed 構成。
- セッション Cookie は `__Host-` プレフィックス + `HttpOnly` + `Secure` + `SameSite=Lax`、CSRF Cookie は同条件で `HttpOnly=false`。SameSite=Lax × Origin + Sec-Fetch-Site + 署名トークンの多重防御で堅牢。

**判定**: 高確度の脆弱性なし。

### Stripe Webhook

- `webhook.ConstructEvent` (公式 SDK) で署名検証 → 定数時間 HMAC 検証。
- `stripe_events.event_id` を PRIMARY KEY とし `ON CONFLICT (event_id) DO NOTHING` でイベント単位の冪等性を担保。
- `customer.subscription.updated` の更新は `customer_id OR subscription_id` を含む WHERE 句で複数ユーザを更新し得るが、Stripe webhook 署名がなければ呼べないため exploit 経路はなし。

**判定**: 高確度の脆弱性なし。

### AdMob SSV

- `gstatic.com/admob/reward/verifier-keys.json` から取得した ECDSA 公開鍵で `ecdsa.VerifyASN1` 検証。
- `splitSignedQuery` は `len(values["signature"]) != 1 || len(values["key_id"]) != 1` で重複パラメータを事前拒否しているため、`LastIndex` ベースのスライス分割は安全。
- ±10 分のクロックスキューチェックと、`admob_ssv_events.transaction_id` を PRIMARY KEY としたリプレイ防止。
- ユーザ ID は `user_id` と `custom_data` の両方を比較し、不一致なら `ErrForbidden` で拒否。

**判定**: 高確度の脆弱性なし。

### Session / Cookies

- `SessionCookieName` は `production` 系で `__Host-session`、開発時のみ `session`。
- `setAuthCookies` で `HttpOnly`、`Secure` (非開発)、`SameSite=Lax` を一貫設定。Domain は `__Host-` のとき空。
- `LogoutAll` で `RevokeRefreshTokens` を呼び全セッションを失効。`Logout` は単一クライアントの Cookie のみクリア。
- `issueSession` は `auth_time` 5 分以内を要求し、古い ID Token によるセッション cookie 取得を拒否。

**判定**: 高確度の脆弱性なし。

### App Check

- `AppCheckEnforcementEnabledFromEnv` は `production` で `APP_CHECK_ENFORCEMENT=false` を起動時に拒否 (fail-closed)。
- `Require` middleware は Mobile client にのみ App Check トークンを要求し、欠落・検証失敗時は 401。
- App Version middleware は production で `X-Platform` / `X-App-Version` 欠落リクエストを 400 で拒否。

**判定**: 高確度の脆弱性なし。

### Internal Task Auth

- `RequireInternalTaskAuthWithSecretFallback` は `TASK_HANDLER_BASE_URL` 設定時に Bearer がある場合 OIDC を検証 (`idtoken.Validate` + `audience = baseURL + path` + 期待する SA email)。
- `allowInternalTaskSecretFallback()` は production で常に false を返し、共有シークレットでの bypass を不可。
- `requireInternalTaskOIDCInProduction` が起動時に `TASK_HANDLER_BASE_URL` / `INTERNAL_TASK_INVOKER_SERVICE_ACCOUNT` の空をログ fatal。

**判定**: 高確度の脆弱性なし。

### SQL Injection

- 新規・変更されたすべてのクエリで `$N` プレースホルダを利用。動的な文字列連結なし。
- スコープ配列は `pq.Array` 経由でバインド。UUID は `uuid.Parse` を通してから渡している。
- `ApproveExtensionPairing` / `CreateExtensionTokenForApprovedPairing` のロジックも完全にパラメータ化。

**判定**: SQL injection の懸念なし。

### Frontend / Mobile XSS

- `frontend/src/lib/safeUrl.ts` は `URL` パースに失敗した値、および `http(s)` 以外のスキームを必ず `undefined` に潰す。
- `frontend/src/components/common/Avatar.tsx` は `safeHttpUrl` 経由でのみ `<img src>` に渡し、`dangerouslySetInnerHTML` は未使用。
- `mobile/src/share.ts` は `URL` パース失敗時に sourceURL を空文字へ正規化。
- React/React Native コンポーネントで `dangerouslySetInnerHTML` / `bypassSecurityTrustHtml` 相当の利用箇所なし。

**判定**: XSS の懸念なし。

---

## Final Conclusion

**本 PR の差分について、信頼度 0.8 以上で HIGH / MEDIUM に該当するセキュリティ脆弱性は確認されなかった。**

新規導入されたセキュリティ機構 (Extension Token / CSRF 二重提出 + 署名 / App Check / AdMob SSV / Stripe webhook / Recent auth / Internal task OIDC / Security headers) は、いずれも fail-closed の構成と多重防御の組み合わせで設計されており、攻撃面の縮小と認可境界の明確化に寄与している。

---

## Residual Risks (フォローアップ候補)

これらは本 PR の脆弱性ではないが、運用上引き続き留意すべきリスク。

1. **Firebase Session Cookie の即時失効**
   `Logout` はローカル Cookie をクリアするのみで、流出済みのセッション Cookie 自体は失効しない。失効には `LogoutAll` (`RevokeRefreshTokens`) が必要。盗難 / 漏洩シナリオでは UX 上 `LogoutAll` を促す導線を検討する。

2. **クライアント IP 取得 (XFF 周り)**
   `c.RealIP()` ベースのレート制限は LB / Cloud Armor 越しでスプーフィングされうる。本 PR では XFF を盲信せず Echo の `RealIP()` 既定動作に従う方針だが、本番運用では信頼できる Proxy レンジ / `IPExtractor` の明示的な設定が望ましい。

3. **シークレットスキャンの網羅性**
   `scripts/secret_scan.py` はパターンベースの軽量チェック。GitHub Secret Scanning や `gitleaks` 等の本格スキャナを CI / GitHub 設定で併用するのが望ましい。

4. **Extension Token 漏洩時の影響範囲**
   スコープを `highlight:write` / `highlight:check` / `extension:revoke-self` に限定しているため最悪ケースのインパクトは限定的だが、リーク後は `revoked_at` を立てるか期限切れになるまで Highlight import は可能。クライアント側での安全な保管 (Manifest V3 storage、暗号化など) とサーバ側の `last_used_at` を活用した検知運用を継続する。
