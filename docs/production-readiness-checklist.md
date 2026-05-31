# Production Readiness Checklist

Use this checklist before exposing the production API, web app, mobile clients,
and browser extension to real users. Do not paste secret values into this file,
tickets, PRs, or chat.

Placeholders:

- `PROJECT_ID`
- `SERVICE_NAME`
- `REGION`
- `DOMAIN`
- `BACKEND_SERVICE`

## Backend Runtime

- [ ] `APP_ENV=production`
  - Confirm: `gcloud run services describe SERVICE_NAME --region=REGION --project=PROJECT_ID --format='value(spec.template.spec.containers[0].env)'`
- [ ] `GCP_PROJECT_ID` is set to the production Google Cloud project.
  - Confirm in Cloud Run env. TODO: verify final project ID before launch.
- [ ] `FIREBASE_PROJECT_ID` / Firebase project identity is correct for web/mobile Firebase config and Admin credentials.
  - Confirm in Firebase console and client env. Backend currently uses Firebase Admin credentials rather than a separate `FIREBASE_PROJECT_ID` env.
- [ ] Firebase Admin credentials are configured through Secret Manager or workload identity.
  - Confirm: Cloud Run service account and `FIREBASE_CREDENTIALS_PATH`/secret mount configuration.
- [ ] `CSRF_SECRET` is set from Secret Manager.
  - Confirm: Secret Manager secret exists and Cloud Run references it. Do not print value.
- [ ] `CSRF_SIGNING_DISABLED=false`
  - Confirm in Cloud Run env.
- [ ] `APP_CHECK_ENFORCEMENT=true`
  - Confirm in Cloud Run env and Firebase App Check dashboard.
- [ ] `STRIPE_SECRET_KEY` is set from Secret Manager.
  - Confirm secret reference only; do not print value.
- [ ] `STRIPE_WEBHOOK_SECRET` is set from Secret Manager.
  - Confirm secret reference only; do not print value.
- [ ] `STRIPE_PRICE_ID_MONTHLY` is set to the production price ID.
  - Confirm in Cloud Run env and Stripe dashboard.
- [ ] AdMob SSV is configured.
  - Confirm `ADMOB_SSV_PUBLIC_KEYS_URL` if overridden, reward ad units, and `/webhooks/admob/ssv` registration.
- [ ] LLM API key is set from Secret Manager.
  - Confirm `GEMINI_API_KEY` secret reference only; do not print value.
- [ ] Global LLM budget initial values are set.
  - Confirm `GLOBAL_LLM_DAILY_MAX_REQUESTS`, `GLOBAL_LLM_DAILY_MAX_ESTIMATED_COST_YEN`, and DB row in `global_llm_budgets`.
- [ ] Cloud Tasks queues are configured.
  - Confirm: `gcloud tasks queues describe QUEUE_QUESTION_GENERATION --location=REGION --project=PROJECT_ID`
  - Confirm: `gcloud tasks queues describe QUEUE_HIGHLIGHT_IMPORT --location=REGION --project=PROJECT_ID`
- [ ] `TASK_HANDLER_BASE_URL` and `INTERNAL_TASK_INVOKER_SERVICE_ACCOUNT` are set for production.
  - Confirm Cloud Run env and service account IAM.
- [ ] Cloud Armor policy is attached to `BACKEND_SERVICE`.
  - Confirm: `gcloud compute backend-services describe BACKEND_SERVICE --global --project=PROJECT_ID`
- [ ] Secret Manager is used for credentials and connection strings.
  - Confirm Cloud Run env uses secret refs for `DATABASE_URL`, Stripe, CSRF, Firebase Admin, LLM, and internal task secret fallback if used.

## Frontend

- [ ] `VITE_API_BASE_URL` points to the production API base path or same-origin proxy.
  - Confirm hosting env and deployed bundle config.
- [ ] Firebase web config is production project config.
  - Confirm `VITE_FIREBASE_*` values against Firebase console. These are public identifiers, not backend secrets.
- [ ] No `VITE_` variable contains a backend secret.
  - Confirm: review hosting provider env and run `python3 scripts/secret_scan.py`.
- [ ] Security headers are present.
  - Confirm: `curl -I https://DOMAIN`
- [ ] HSTS preload operation is confirmed.
  - Confirm `Strict-Transport-Security` max-age/includeSubDomains/preload policy and rollback implications.
- [ ] Frontend can call the API successfully.
  - Confirm with browser smoke test and `docs/smoke-test.md`.

## Extension

- [ ] Production manifest host permissions contain only Kindle Notebook and the final API origin.
  - Confirm: `jq '.host_permissions' extension/manifest.json`
- [ ] `localhost`, `*.run.app`, and staging origins are absent from production manifest.
  - Confirm: `rg 'localhost|run\\.app|staging' extension/manifest.json`
- [ ] Extension API origin is final.
  - Confirm `https://api.ai-study-tool.com/*` or replace with final `DOMAIN` before packaging. TODO: confirm final API origin.
- [ ] Chrome 102+ minimum version is set.
  - Confirm `minimum_chrome_version` in `extension/manifest.json`.
- [ ] Store readiness is complete.
  - Confirm [Extension Store Readiness](../extension/STORE_READINESS.md).

## Mobile

- [ ] `EXPO_PUBLIC_API_BASE_URL` points to production API `/api/v1`.
  - Confirm EAS/build profile env.
- [ ] Firebase mobile config is production project config.
  - Confirm `EXPO_PUBLIC_FIREBASE_*` against Firebase console.
- [ ] App Check provider is configured.
  - Confirm iOS App Attest/DeviceCheck and Android Play Integrity.
- [ ] `EXPO_PUBLIC_APP_VERSION` matches the release version.
  - Confirm app binary metadata and mobile env.
- [ ] Minimum supported versions are configured server-side.
  - Confirm `MIN_SUPPORTED_IOS_VERSION` and `MIN_SUPPORTED_ANDROID_VERSION`.
- [ ] Mobile release checklist is complete.
  - Confirm [Mobile Release Readiness](mobile-release-readiness.md).

## Data And Operations

- [ ] DB migrations are applied.
  - Confirm migration table and latest migration version. TODO: use production-safe read-only query.
- [ ] DB backup and restore procedure is documented and tested.
  - Confirm Neon backup/branch restore process and owner.
- [ ] Rollback procedure is documented.
  - Confirm [Deploy Runbook](deploy-runbook.md).
- [ ] Monitoring alerts are configured.
  - Confirm [Monitoring and Alert Plan](ops-monitoring-alerts.md) has been converted into GCP policies.
- [ ] Cloud Armor rules are configured.
  - Confirm [Cloud Armor Operations Plan](ops-cloud-armor.md).
- [ ] On-call/runbook links are attached to alerts.
  - Confirm alert documentation links.
- [ ] Incident owners and escalation channels are known.
  - TODO: fill production owner/channel outside this repository if private.

## Validation Commands

- [ ] `cd backend && go test ./...`
- [ ] `cd backend && go build ./...`
- [ ] `cd frontend && npm run typecheck`
- [ ] `cd frontend && npm test`
- [ ] `cd frontend && npm run build`
- [ ] `cd mobile && npm run typecheck`
- [ ] `cd mobile && npm test`
- [ ] `cd extension && npm run typecheck`
- [ ] `cd extension && npm test`
- [ ] `cd extension && npm run build`
- [ ] `python3 scripts/secret_scan.py`
- [ ] GitHub Actions Secret Scan passes.
- [ ] GitHub Actions Backend passes.
- [ ] GitHub Actions Frontend passes.
- [ ] GitHub Actions Mobile passes.
- [ ] GitHub Actions Browser Extension passes.

