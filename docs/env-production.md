# Production Environment And Secrets

This document classifies production configuration. It intentionally contains no
real secret values. Use dummy values in examples and store production secrets in
Secret Manager or the relevant build system secret store.

## Backend Production Env

| Name | Class | Where | Notes |
| --- | --- | --- | --- |
| `APP_ENV` | public config | Cloud Run env | Must be `production`. |
| `PORT` | public config | Cloud Run env | Cloud Run usually injects this. |
| `GCP_PROJECT_ID` | public config | Cloud Run env | Google Cloud project identifier. |
| `DATABASE_URL` | secret | Secret Manager | Neon pooled production URL with `sslmode=require`. |
| `MIGRATION_DATABASE_URL` | secret | Secret Manager / deploy env | Optional; use when migration credentials differ. |
| `FIREBASE_CREDENTIALS_PATH` or Admin credentials | secret | Secret Manager / workload identity | Do not bundle into frontend/mobile/extension. |
| `ALLOWED_ORIGINS` | public config | Cloud Run env | Production web/extension origins only. |
| `SESSION_COOKIE_DOMAIN` | public config | Cloud Run env | Leave empty for `__Host-` cookie constraints unless intentionally needed. |
| `CSRF_SECRET` | secret | Secret Manager | Rotate if exposed or after auth incident. |
| `CSRF_SIGNING_DISABLED` | public config | Cloud Run env | Must be `false` in production. |
| `APP_CHECK_ENFORCEMENT` | public config | Cloud Run env | Must be `true` in production. |
| `MIN_SUPPORTED_IOS_VERSION` | public config | Cloud Run env | Controls mobile version gate. |
| `MIN_SUPPORTED_ANDROID_VERSION` | public config | Cloud Run env | Controls mobile version gate. |
| `CSP_REPORT_URI` | public config | Cloud Run env | Optional report endpoint. |
| `GEMINI_API_KEY` | secret | Secret Manager | LLM provider key; rotate on exposure/cost incident. |
| `USE_GEMINI_MOCK` | public config | Cloud Run env | Must be `false` in production. |
| `GLOBAL_LLM_DAILY_MAX_REQUESTS` | public config | Cloud Run env | Initial guardrail; DB budget is source for daily use. |
| `GLOBAL_LLM_DAILY_MAX_ESTIMATED_COST_YEN` | public config | Cloud Run env | Initial cost guardrail. |
| `LLM_ESTIMATED_COST_YEN_PER_REQUEST` | public config | Cloud Run env | Estimate, tune operationally. |
| `QUEUE_QUESTION_GENERATION` | public config | Cloud Run env | Cloud Tasks queue name. |
| `QUEUE_HIGHLIGHT_IMPORT` | public config | Cloud Run env | Cloud Tasks queue name. |
| `TASK_HANDLER_BASE_URL` | public config | Cloud Run env | Internal task target base URL. |
| `INTERNAL_TASK_INVOKER_SERVICE_ACCOUNT` | public config | Cloud Run env | Required in production for Cloud Tasks OIDC. |
| `INTERNAL_TASK_SECRET` | secret | Secret Manager | Compatibility fallback; do not rely on it as primary production auth. |
| `STRIPE_SECRET_KEY` | secret | Secret Manager | Rotate through Stripe dashboard if exposed. |
| `STRIPE_WEBHOOK_SECRET` | secret | Secret Manager | Rotate if webhook endpoint is exposed. |
| `STRIPE_PRICE_ID_MONTHLY` | public config | Cloud Run env / deploy var | Price IDs are not secrets but must be server-owned. |
| `STRIPE_SUCCESS_URL` | public config | Cloud Run env | Production frontend URL. |
| `STRIPE_CANCEL_URL` | public config | Cloud Run env | Production frontend URL. |
| `AD_REWARD_HMAC_SECRET` | secret | Secret Manager | Legacy/dev reward path guard. |
| `ADMOB_SSV_PUBLIC_KEYS_URL` | public config | Cloud Run env | Usually default Google URL; override only intentionally. |

## Frontend Production Env

| Name | Class | Notes |
| --- | --- | --- |
| `VITE_API_BASE_URL` | public | May be `/api/v1` behind same-origin hosting or production API URL. |
| `VITE_FIREBASE_API_KEY` | public but sensitive to misuse | Firebase web config identifier; restrict Firebase rules/App Check. |
| `VITE_FIREBASE_AUTH_DOMAIN` | public | Firebase web config. |
| `VITE_FIREBASE_PROJECT_ID` | public | Firebase web config. |
| `VITE_FIREBASE_MESSAGING_SENDER_ID` | public | Firebase web config. |
| `VITE_FIREBASE_APP_ID` | public | Firebase web config. |

Never put backend secrets in `VITE_*`; Vite embeds them into the browser bundle.

## Mobile Production Env

| Name | Class | Notes |
| --- | --- | --- |
| `EXPO_PUBLIC_API_BASE_URL` | public | Must include production `/api/v1`. |
| `EXPO_PUBLIC_FIREBASE_API_KEY` | public but sensitive to misuse | Firebase mobile config; protect with App Check and Firebase rules. |
| `EXPO_PUBLIC_FIREBASE_AUTH_DOMAIN` | public | Firebase config. |
| `EXPO_PUBLIC_FIREBASE_PROJECT_ID` | public | Firebase config. |
| `EXPO_PUBLIC_FIREBASE_MESSAGING_SENDER_ID` | public | Firebase config. |
| `EXPO_PUBLIC_FIREBASE_APP_ID` | public | Firebase config. |
| `EXPO_PUBLIC_FIREBASE_APPCHECK_SITE_KEY` | public | Public site key/provider config. |
| `EXPO_PUBLIC_FIREBASE_APPCHECK_DEBUG_TOKEN` | secret-like debug value | Local/debug only. Do not ship in production. |
| `EXPO_PUBLIC_APP_VERSION` | public | Sent as `X-App-Version`. |
| `EXPO_PUBLIC_ADMOB_*` | public | Ad unit IDs are public identifiers, not reward authority. |

Never put backend secrets in `EXPO_PUBLIC_*`; Expo embeds them into the app
bundle.

## Extension Production Config

| Item | Class | Notes |
| --- | --- | --- |
| `extension/manifest.json` API host permission | public | Must be final API origin only. |
| Kindle Notebook content script matches | public | Limit to `read.amazon.co.jp/notebook*` and `read.amazon.com/notebook*`. |
| Extension API origin setting | public | User-visible origin used for pairing/import. |
| Extension token | secret user credential | Stored in `chrome.storage.local`; never document or log raw value. |

## Secret Manager Required

Store these in Secret Manager or an equivalent secret store:

- `DATABASE_URL`
- `MIGRATION_DATABASE_URL` if used
- Firebase Admin credentials
- `CSRF_SECRET`
- `GEMINI_API_KEY`
- `STRIPE_SECRET_KEY`
- `STRIPE_WEBHOOK_SECRET`
- `INTERNAL_TASK_SECRET` if fallback is still configured
- `AD_REWARD_HMAC_SECRET`

## Rotation List

Rotate after exposure, suspected misuse, staff access changes, or provider
incident:

- Database credentials
- Firebase Admin credentials
- `CSRF_SECRET`
- LLM API key
- Stripe secret key and webhook secret
- Internal task fallback secret
- Ad reward HMAC secret
- Extension tokens for affected users

## `.env.example` Notes

Example files may contain dummy or local-only values. Do not copy production
secrets into `backend/.env.example`, `frontend/.env.example`, `mobile/.env.example`,
or extension manifests. Keep local debug tokens out of committed files.

