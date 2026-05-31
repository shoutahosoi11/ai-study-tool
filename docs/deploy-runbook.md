# Deploy And Rollback Runbook

This runbook uses placeholders and contains no production secret values.

Placeholders:

- `PROJECT_ID`
- `SERVICE_NAME`
- `REGION`
- `DOMAIN`
- `BACKEND_SERVICE`

## Backend Deploy

1. Confirm release commit and CI status.
   - `gh pr checks PR_NUMBER --repo shoutahosoi11/ai-study-tool`
2. Confirm production secrets are configured in Secret Manager.
   - Do not print secret values.
3. Apply database migrations.
   - Prefer a controlled migration job using `MIGRATION_DATABASE_URL` or
     `DATABASE_URL` from Secret Manager.
   - Confirm the latest migration version with a read-only query.
4. Deploy Cloud Run.
   - Use the existing deploy workflow or:
     `gcloud run deploy SERVICE_NAME --region=REGION --project=PROJECT_ID`
5. Confirm health.
   - `curl -fsS https://DOMAIN/health`
   - `curl -fsS https://DOMAIN/ready`
6. Run smoke tests.
   - `BASE_URL=https://DOMAIN CONFIRM_PRODUCTION_SMOKE=true scripts/smoke_test.sh`
7. Run E2E smoke against staging or production only with explicit approval.
   - Staging: `cd e2e && E2E_SKIP_WEB_SERVER=true E2E_BASE_URL=https://STAGING_DOMAIN E2E_API_BASE_URL=https://STAGING_API_DOMAIN E2E_RUN_API_TESTS=true npm run test`
   - Production: set `E2E_ALLOW_PRODUCTION=true` only for read-only smoke with approved test accounts.
   - Do not print test passwords, cookies, raw tokens, webhook payloads, SSV query strings, prompts, or highlight text.
8. Complete [QA Checklist](qa-checklist.md) for release-candidate builds.
9. Watch Cloud Run 5xx, latency, instance count, Cloud Tasks queue depth, Stripe
   webhook status, AdMob SSV status, and LLM budget metrics.

## Frontend Deploy

1. Build locally or in CI.
   - `cd frontend && npm run typecheck`
   - `cd frontend && npm test`
   - `cd frontend && npm run build`
2. Deploy to hosting/CDN using the existing release process.
3. Confirm security headers.
   - `curl -I https://DOMAIN`
4. Confirm the frontend can call the production API without CORS or CSRF errors.
5. Confirm no `VITE_*` value contains a backend secret.

## Extension Build And Store Submission

1. Confirm production manifest.
   - `jq '.minimum_chrome_version, .permissions, .host_permissions' extension/manifest.json`
2. Build.
   - `cd extension && npm run typecheck`
   - `cd extension && npm test`
   - `cd extension && npm run build`
3. Create a ZIP from extension source plus built `dist/`, excluding
   `node_modules`, tests, and development manifest.
4. Complete [Extension Store Readiness](../extension/STORE_READINESS.md).
5. Submit to Chrome Web Store with permission justification and privacy policy.

## Mobile Build

1. Confirm production env in EAS/build profile.
2. Confirm Firebase App Check providers.
   - iOS: App Attest / DeviceCheck
   - Android: Play Integrity
3. Confirm `EXPO_PUBLIC_APP_VERSION`, bundle version, and server minimum version.
4. Run:
   - `cd mobile && npm run typecheck`
   - `cd mobile && npm test`
5. Build with Expo/EAS using production profile.
6. Complete [Mobile Release Readiness](mobile-release-readiness.md).

## Rollback

### Cloud Run

1. Identify the previous known-good revision.
   - `gcloud run revisions list --service=SERVICE_NAME --region=REGION --project=PROJECT_ID`
2. Route traffic back to the previous revision.
   - `gcloud run services update-traffic SERVICE_NAME --region=REGION --project=PROJECT_ID --to-revisions REVISION=100`
3. Watch `/health`, `/ready`, 5xx, latency, and task queue metrics.

### Frontend

1. Roll back hosting/CDN to the previous deploy artifact.
2. Purge CDN cache only if the hosting platform requires it.
3. Confirm API connectivity and security headers.

### Database

Migrations are forward-only by default. Do not attempt destructive rollback in
an incident unless the migration author has documented a safe reverse path.
Prefer:

1. Stop or pause the affected feature path.
2. Roll back application code if compatible.
3. Ship a forward-fix migration.
4. Restore from backup only for data-loss incidents with explicit approval.

### Extension

Chrome Extension rollout cannot be instantly rolled back on all clients. Use
backend controls instead:

- minimum supported extension version, when implemented
- token revocation
- extension import rate-limit reduction
- route-level deny or Cloud Armor controls for severe incidents

### Mobile

Mobile binaries cannot be instantly rolled back on all devices. Use:

- server-side minimum supported versions
- feature flags or route-level controls
- App Store / Play Store phased release controls
- backend compatibility for at least one previous supported version
