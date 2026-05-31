# Security Runbook

This runbook lists the first operational actions for common security incidents.
Do not paste raw tokens, cookies, signatures, raw webhook payloads, raw query
strings, pairing IDs, prompt text, or private keys into tickets or chat.

## LLM Cost Spike

1. Lower the global budget for today in `global_llm_budgets`.
2. Pause the Cloud Tasks question generation queue if queued work is still
   driving provider calls.
3. Temporarily lower `/api/v1/questions/sync` and manual generation edge/app
   rate limits.
4. Disable or rotate the LLM API key if provider calls continue unexpectedly.
5. If available, turn off question generation with a feature flag or emergency
   config. If no feature flag exists, deploy a temporary route-level deny or
   pause all generation queues.
6. Identify suspicious users and generation jobs.
7. Inspect:
   - `llm_usage_logs`
   - `global_llm_budgets`
   - `question_generation_jobs`
   - Cloud Tasks queue depth and retry logs
8. Decide whether failed jobs should remain failed or be manually retried after budget reset.
9. If extension-import abuse is the trigger, temporarily lower highlight ingest
   limits or revoke the involved extension tokens; generation still must pass
   user/global budget and queue-depth checks.

## Cloud Tasks Backlog

1. Check Cloud Tasks queue depth and oldest task age.
2. Inspect worker error logs for `/internal/tasks/question-generation` and
   `/internal/tasks/highlight-import`.
3. Check retry counts and whether failures are permanent validation errors,
   transient DB/API errors, or provider rate limits.
4. Pause the queue if retry storms are increasing cost or saturating Cloud Run.
5. Confirm Cloud Run max instances, concurrency, DB connection limits, and
   Cloud Tasks dispatch settings are aligned.
6. After the root cause is fixed, resume gradually and watch oldest task age.

## Cloud Run 5xx Surge

1. Check the latest revision, deployment time, and error logs.
2. Roll back to the previous known-good revision if the spike started after a
   deployment.
3. Check DB connection saturation, Neon availability, and connection-string
   Secret Manager values.
4. Check required env/secret values for Firebase, Stripe, AdMob, LLM, CSRF, and
   Cloud Tasks.
5. Reduce max instances or pause task queues only if downstream dependencies are
   being overloaded.
6. Keep request/response samples sanitized. Do not paste tokens, cookies, raw
   webhook bodies, prompts, or highlight text.

## Stripe Webhook Anomaly

1. Confirm `STRIPE_WEBHOOK_SECRET` is current.
2. Check whether the `event_id` already exists in `stripe_events`.
3. Compare `subscriptions` with the Stripe dashboard.
4. If replaying events, rely on event idempotency and do not manually update user plan without recording the reason.
5. Rotate Stripe secrets if webhook signature validation unexpectedly fails for legitimate events.
6. If valid Stripe events are failing, use the Stripe dashboard to retry after
   the backend is healthy.
7. If invalid requests are flooding the endpoint, lower the Cloud Armor webhook
   limit before disabling legitimate Stripe delivery.

## AdMob SSV Anomaly

1. Check `/webhooks/admob/ssv` rate-limit behavior and Cloud Run request volume.
2. Confirm public key fetch health. Stale key fallback is allowed only within the configured fallback window.
3. Check whether `transaction_id` already exists in `admob_ssv_events`.
4. Confirm production does not register legacy `/api/v1/tokens/award`.
5. Inspect `user_id` / `custom_data` mismatch warnings without exposing raw query or signature.
6. If abuse continues, lower SSV rate limits and temporarily disable reward issuance.
7. If public key fetch failures continue, verify outbound access and the AdMob
   public-key endpoint status before extending stale-cache reliance.

## Extension Pairing Failure

1. Check app-side pairing start/approve/claim rate-limit counters.
2. Check Cloud Armor 429 logs for `/api/v1/extension/pairing/*`.
3. Confirm the rightmost trusted `X-Forwarded-For` extractor still sees stable
   client identifiers behind Cloud Run/load balancer.
4. Check whether `user_code` entries are expired, already approved, or already
   claimed.
5. If pairing abuse is active, lower Cloud Armor pairing limits and app-side
   pairing limits before disabling extension import.

## CSRF Spike

1. Confirm `CSRF_SECRET` matches the active deployment.
2. Confirm `CSRF_SIGNING_DISABLED=false` in production.
3. Check allowed origins and cookie domain/path settings.
4. Inspect sanitized `csrf_rejected` log-based metrics when available.
5. If legitimate web users are blocked, roll back the most recent auth/client
   change or temporarily narrow the affected route.

## Mobile App Check Failure

1. Confirm `APP_CHECK_ENFORCEMENT=true` is expected for the environment.
2. Check Firebase App Check provider configuration and app identifiers.
3. Check mobile app version/platform headers and minimum-version gates.
4. If a bad release is blocking users, rollback mobile config or temporarily
   adjust the minimum-version gate while keeping auth required.

## Webhook Endpoint Or API Flood

1. Check Cloud Armor rule matches and 429 counts.
2. Check app-side rate-limit counters.
3. Check signature failure logs for webhooks.
4. Add a temporary Cloud Armor block rule for clear abusive IPs only when it
   will not block provider delivery.
5. Keep Stripe and AdMob signature verification as the primary validity check.

## Secret Leak

1. Rotate the secret. Removing it from GitHub history is not enough.
2. Rotate affected provider credentials:
   - Stripe secret key and webhook secret
   - LLM API key
   - Firebase credentials
   - CSRF secret
   - Ad reward / App Check related secrets
3. Revoke or expire extension tokens if browser-extension credentials may be affected.
4. Search recent deployments and CI logs for accidental exposure.
5. Re-run `python3 scripts/secret_scan.py`.
6. Backfill incident notes with the secret type and rotation time only, not the secret value.

## XSS Report

1. Disable or hide the affected display surface if possible.
2. Confirm whether the payload executes or is rendered as text.
3. Check CSP and static hosting security headers.
4. Review rendering paths for posts, comments, profile fields, highlights, explanations, and LLM output.
5. Revoke affected sessions if token/session compromise is plausible.
6. Add a regression test for the exact vector before re-enabling the surface.

## Account Takeover

1. Call Firebase `RevokeRefreshTokens` for the user.
2. Trigger or instruct `logout-all`.
3. Do not rely on normal `logout`; it only clears cookies in the current browser.
4. Revoke that user's extension tokens.
5. Review recent account changes, generation jobs, billing state, and suspicious IP/client metadata.
6. If email/password was changed, require recent sign-in before further sensitive operations.

## Extension Token Leak

1. Set `extension_tokens.revoked_at` for the affected token.
2. Search by token hash only; never store or paste the raw token.
3. Inspect `last_used_at`, scopes, and recent highlight imports.
4. Confirm the token only had highlight/check/revoke-self scope.
5. Ask the user to re-pair the extension after revocation.
6. If many tokens are affected, bulk revoke extension tokens for the user or
   affected time window and temporarily lower extension import rate limits.

## Cloud Run / Cloud Armor / Load Balancer

Cloud Run terminates TCP at Google's frontend, so `RemoteAddr` is a shared
frontend address and can collapse application rate limits across unrelated
users. The API therefore reads the rightmost valid `X-Forwarded-For` address
appended by Cloud Run and hashes it before storing rate-limit identifiers.

If moving to a different proxy or edge chain:

1. Document how the edge appends or overwrites `X-Forwarded-For`.
2. Configure Echo or a custom extractor for that exact trust model.
3. Confirm rate-limit identifiers remain hashed.
4. Verify Cloud Armor rate limits are aligned with backend DB-backed limits.
5. Re-test pairing and AdMob SSV rate limits behind the proxy.

## Risk Response Matrix

| Risk | Immediate stop | Data to inspect | Recovery |
| --- | --- | --- | --- |
| LLM API abuse | Lower global budget, pause queue, rotate key | `llm_usage_logs`, jobs, Cloud Tasks | reset budget next day, replay selected jobs |
| Secret leak | Rotate secret | CI logs, commit history, Secret Manager audit | redeploy with new secret |
| Stripe fraud | Disable webhook endpoint or rotate secret | `stripe_events`, Stripe dashboard | replay verified events |
| XSS | Disable affected UI | CSP reports, payload source | sanitize/render-as-text fix |
| Account takeover | Revoke Firebase tokens | auth logs, extension tokens | force re-login |
| IDOR | Disable route if needed | route logs, repository SQL | add owner condition and test |
| Multi-account abuse | Lower global budget | signup patterns, usage logs | block users, add risk controls |
| CSRF | Tighten origins, rotate CSRF secret | 403 logs, origins | redeploy CSRF config |
| AdMob spoofing | Lower SSV rate limit, disable rewards | `admob_ssv_events` | rotate/reconfigure AdMob |
| Unauthorized generation | Pause generation queue | job tables, user actions | delete/retry jobs |
| MITM | Enforce HTTPS/HSTS | LB/Cloud Run config | rotate potentially exposed tokens |
| Supply chain | Pin/revert dependency | lockfile diff, CI logs | patch and rebuild |
| Open redirect | Disable redirect path | access logs | add URL allowlist |
| Extension token leak | Revoke token | `extension_tokens` | re-pair extension |
| Mobile loss | Revoke Firebase tokens | device/user logs | force re-auth |

## Production Operation Links

- [Cloud Armor Operations Plan](ops-cloud-armor.md)
- [Monitoring and Alert Plan](ops-monitoring-alerts.md)
- [Production Readiness Checklist](production-readiness-checklist.md)
- [k6 Load Tests](../loadtest/k6/README.md)
