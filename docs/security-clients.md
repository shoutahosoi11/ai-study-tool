# Client Security Model

This document explains how Web, Mobile, and Browser Extension clients
authenticate differently and how those differences are normalized into
`currentUser`, `clientType`, `auth_time`, and extension `scopes` for backend
usecases.

## Client Authentication Summary

| Client | Credential | Extra checks | Why this shape |
| --- | --- | --- | --- |
| Web | Firebase Session Cookie | Signed CSRF, Origin, Fetch Metadata, recent auth | Browser cookies can be `HttpOnly`; CSRF is handled server-side. |
| Mobile | Firebase ID Token Bearer | Firebase App Check, app version/platform, recent auth for sensitive operations | Native apps should use SDK-managed tokens and not cookie CSRF. |
| Extension | Dedicated `ext_` token | Scope checks, token hash lookup, expiry/revoke, ingest limits | Extensions should not hold Firebase user credentials or broad account capabilities. |

Web uses cookies because browser sessions benefit from `HttpOnly` cookie
storage. Mobile uses Bearer ID Tokens because the Firebase Native SDK manages
refresh and persistence, and CSRF does not apply to non-cookie authentication.
The browser extension uses a separate scoped token because extension storage is
a higher-exposure environment and should not receive full Firebase account
credentials.

## Allowed Operations By Client Type

| Operation | Web | Mobile | Extension |
| --- | --- | --- | --- |
| Read own profile/questions/highlights | Yes | Yes | No unless explicitly scoped |
| Update profile/settings | Yes | Yes | No |
| Highlight import | Yes | Yes | Yes, `highlight:write` only |
| Highlight existence check | Yes | Yes | Yes, `highlight:check` only |
| LLM question generation | Yes, budgeted | Yes, budgeted + App Check | No |
| Billing changes | Yes, recent auth | Mobile future IAP path only | No |
| Ad rewards | SSV/backend only | SSV/backend only | No |
| Post/social writes | Yes | Yes | No |
| Account deletion/logout-all | Yes, recent auth | Yes, recent auth | No |
| Revoke own extension token | Web server-side admin path / future UI | No | Yes, `extension:revoke-self` |

Mobile version headers are not security proof on their own. They are only useful
after Firebase ID Token authentication and App Check have established that the
request is from a legitimate app instance. Mobile biometrics are local app-lock
UX and are not accepted by the server as recent-auth proof; the server uses
Firebase `auth_time`.

## Web Logout Semantics

`POST /api/v1/auth/logout` clears only the current browser's Session Cookie and
CSRF cookie. It does not individually revoke an already stolen Firebase Session
Cookie or revoke refresh tokens for other devices. Use `POST
/api/v1/auth/logout-all` when complete account-wide invalidation is required;
that path requires recent auth and calls Firebase `RevokeRefreshTokens`.

## IP Extraction And Rate Limits

Backend short-window limits such as extension pairing and AdMob SSV use hashed
client IP identifiers. Cloud Run terminates TCP at Google's frontend, so
`RemoteAddr` is a shared frontend address and is not suitable for per-client
limits. The API reads the rightmost valid `X-Forwarded-For` value appended by
Cloud Run and hashes it before storing rate-limit identifiers. If the service is
moved behind another proxy chain, document the trusted forwarding behavior and
re-test pairing and AdMob SSV limits before relying on forwarded client IPs.

## Browser Extension Tokens

Browser Extension clients use a dedicated `ext_` token instead of Firebase Session Cookies or Mobile Firebase ID Tokens. The backend stores only `SHA-256(raw_token)` in `extension_tokens.token_hash`; the raw token is returned once during pairing and is never persisted.

### Pairing Flow

1. The extension calls `POST /api/v1/extension/pairing/start` without authentication and receives a short-lived `pairing_id` plus a human-entered `user_code`.
2. The user opens `/extension/connect` while signed in on Web and enters the `user_code`; the `pairing_id` is not placed in the browser URL.
3. Web calls `POST /api/v1/extension/pairing/approve` with `{ "user_code": "ABCDE-FGHJK" }`, the existing Session Cookie, and CSRF protection.
4. The extension polls `POST /api/v1/extension/pairing/status` with `{ "pairing_id": "..." }`. This endpoint only returns `pending`, `approved`, or `used`; it does not issue or consume tokens.
5. Once approved, the extension calls `POST /api/v1/extension/pairing/claim` with `{ "pairing_id": "..." }`. The backend creates an extension token and returns the raw `ext_` token exactly once. The pairing row is marked `used_at`; repeat claims cannot retrieve the token again.

`/pairing/start`, `/pairing/approve`, and `/pairing/claim` use DB-backed rate limiting. The `user_code` is safe to show to the user, while the `pairing_id` remains a device-held secret used only by the extension. Rate-limit identifiers hash sensitive inputs before storage.

Pairing claim checks both the hashed `pairing_id` and the hashed client IP identifier. Counters are not rolled back when later validation fails; failed attempts intentionally consume the short-window allowance. On Cloud Run, the client identifier comes from the rightmost valid `X-Forwarded-For` value appended by Google's frontend, with direct `RemoteAddr` only as a fallback.

### Scopes

Extension tokens are limited to:

- `highlight:write`
- `highlight:check`
- `extension:revoke-self`

`RequireScope` is enforced only for `clientType=extension`. Web and Mobile continue to use their existing authorization paths, while extension tokens are denied on routes that require scopes such as `question:generate`, `billing:write`, `post:write`, `user:write`, or token/ad reward scopes.

### Highlight Import

The extension import endpoint is:

```text
POST /api/v1/extension/highlights/import
```

It requires `Authorization: Bearer ext_...` and `highlight:write`. The route applies a body size limit and the existing ingest rate limit. It does not directly create LLM generation jobs; highlight processing follows the backend's normal import path.

### Leakage Impact

If an extension token leaks, the attacker can only use the token until `expires_at` or `revoked_at` to check/import highlights and revoke that same token. The token cannot trigger LLM generation, billing changes, ad reward claims, post creation, account deletion, or profile updates.

### Revoke

An extension can revoke its own token with:

```text
DELETE /api/v1/extension/tokens/self
Authorization: Bearer ext_...
```

Server-side operations should set `extension_tokens.revoked_at` for a compromised token. Existing authentication rejects rows where `revoked_at IS NOT NULL` or `expires_at <= now()`.

### Why LLM Generation Is Not Scoped To Extensions

Extension tokens are long-lived bearer credentials that live in a browser extension environment. Giving them LLM generation or billing capabilities would turn token leakage into direct cost and abuse risk. The extension is intentionally limited to highlight ingestion; any downstream generation must be decided by backend-owned workflows and user/account policy.

## LLM Cost Controls

Question generation is protected by layered controls:

- Per-user question budget remains the first user-visible quota.
- `global_llm_budgets` caps service-wide LLM request count and estimated yen cost per Tokyo calendar day.
- The Cloud Tasks worker reserves global budget immediately before the LLM API call. If the reserve fails, the LLM provider is not called.
- Once a worker passes the reserve and attempts the LLM call, the reserve is treated as consumed even if the provider call or response validation fails.
- `llm_usage_logs` records provider, model, token counts when available, and estimated cost. Prompt text, secrets, cookies, tokens, and signatures are not stored.
- `QUESTION_JOB_MAX_PENDING_PER_USER`, `QUESTION_JOB_MAX_PENDING_PER_BOOK`, and `QUESTION_JOB_MAX_PENDING_GLOBAL` limit pending question-generation jobs before queue insertion.

Extension imports still use the ingest rate limit and can only enqueue generation through normal backend conditions. Even if imported highlights satisfy generation thresholds, the resulting Cloud Tasks job must pass user budget, global budget, and queue-depth checks before any LLM request is made.

## Payments And Ad Rewards

Stripe checkout uses server-side configuration such as `STRIPE_PRICE_ID_MONTHLY`; clients do not provide price IDs or amounts. Stripe webhooks must be verified with the raw request body, `Stripe-Signature`, and `STRIPE_WEBHOOK_SECRET`. Processed event IDs are stored in `stripe_events`, so duplicate webhook deliveries return success without applying subscription updates twice.

Subscription state is stored in the provider-neutral `subscriptions` table with `provider` values such as `stripe`, `apple`, and `google`. The backend database is the source of truth for premium state; clients must not be trusted for `isPremium`.

Ad rewards use AdMob SSV at `/webhooks/admob/ssv`. The server verifies the callback signature with AdMob public keys, rejects stale timestamps, stores `transaction_id` in `admob_ssv_events`, and awards tokens only for a verified, previously unseen transaction. Client-only "watched ad" notifications are rejected in production and do not grant tokens.

The legacy client-notification reward path `/api/v1/tokens/award` is not registered in production; production rewards must come through the AdMob SSV webhook. SSV responses return only an HTTP status to Google and never include user token balance or plan data.

## XSS, IDOR, Secrets, And Headers

User-generated content, highlight text, explanations, and LLM output must be rendered as text, not HTML. The frontend and mobile clients do not use raw HTML rendering for these fields. URL attributes should be normalized to `http` or `https` only; `javascript:`, `data:`, and `vbscript:` URLs are rejected at validation or ignored at display time.

The backend validates ownership in repository queries for user-owned updates and deletes by including the authenticated user ID in the mutation condition. Handlers should pass only the current authenticated user from context into usecases; request body user IDs are not authorization inputs. Social answering remains intentionally broader: users may answer visible questions, but writes to private notes, explanations, saved records, tokens, subscriptions, extension tokens, and highlight-owned data must stay bound to the current user.

Prompt text, raw extension tokens, pairing IDs, cookies, signatures, raw payment payloads, and provider secrets must not be logged or stored unless explicitly redacted. Question generation metadata stores model/source identifiers and redacts prompt text. CI runs `scripts/secret_scan.py` to reject high-confidence secret patterns and backend secrets accidentally placed in `VITE_` variables.

Mobile `X-App-Version` and `X-Platform` headers are client metadata. They are useful for minimum-version gates only together with Firebase ID Token authentication and Firebase App Check; the backend must not treat version headers alone as proof of a genuine app.

The Go API applies security headers through `SecurityHeadersMiddleware`: CSP, HSTS in production, `X-Frame-Options`, `X-Content-Type-Options`, `Referrer-Policy`, and `Permissions-Policy`. The static frontend hosting layer should mirror these headers where it serves HTML assets. CSP should keep scripts restricted to trusted origins and must not use `script-src *`; Stripe.js, Firebase, or other browser SDK origins should be added explicitly only when a client page actually requires them.

These controls reduce stored/reflected XSS, common IDOR mistakes, accidental client bundle secret exposure, and clickjacking/MIME sniffing risks. They do not replace code review for every new data write path, dependency vulnerability scanning, browser-extension store review, or cloud-side controls such as WAF rules and access logging.
