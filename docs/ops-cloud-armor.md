# Cloud Armor Operations Plan

This document is the production edge-control plan for Cloud Run behind an
external HTTPS load balancer and Cloud Armor. Values are initial guardrails and
must be tuned with real traffic. Keep application-level authorization, budget,
signature, App Check, and CSRF checks as the primary controls.

Placeholders:

- `PROJECT_ID`
- `SERVICE_NAME`
- `REGION`
- `BACKEND_SERVICE`
- `DOMAIN`

## Scope

Protect these externally reachable paths:

| Path | Primary app control | Cloud Armor goal | Initial edge limit |
| --- | --- | --- | --- |
| `/api/v1/auth/session` | Firebase ID token, recent auth, signed CSRF issue | slow credential/session abuse | 10 req/min/IP |
| `/api/v1/auth/refresh` | Session cookie, signed CSRF, UID match | limit cookie replay pressure | 10 req/min/IP |
| `/api/v1/extension/pairing/start` | app pairing rate limit | reduce user-code spray | 20 req/min/IP |
| `/api/v1/extension/pairing/approve` | authenticated user, user_code/client rate limit | reduce approve brute force | 20 req/min/IP |
| `/api/v1/extension/pairing/claim` | pairing_id/client rate limit | reduce polling bursts | 20 req/min/IP |
| `/api/v1/extension/highlights/import` | extension token scope, app rate limit | cap import bursts | 60 req/min/IP |
| `/api/v1/questions/generate/manual` | user budget and global budget | reduce generation storms | 30 req/min/IP |
| `/api/v1/questions/sync` | auth, queue depth, daily limit | reduce sync storms | 60 req/min/IP |
| `/webhooks/admob/ssv` | AdMob SSV signature and transaction idempotency | reduce invalid CPU work | 60 req/min/IP |
| `/webhooks/stripe` | Stripe signature and event idempotency | reduce invalid CPU work | 120 req/min/IP |

The values above are deliberately conservative starting points. Adjust them
from Cloud Run request counts, backend 429 rates, user complaints, and provider
webhook delivery behavior.

## Design

1. Put Cloud Run behind an external HTTPS load balancer with a serverless NEG.
2. Attach one Cloud Armor security policy to `BACKEND_SERVICE`.
3. Use path-specific rate-limit rules before broader allow rules.
4. Prefer `throttle` while validating traffic, then switch selected rules to
   `rate_based_ban` only after observing no legitimate breakage.
5. Keep app-side rate limits and budgets. Cloud Armor only sees IP and request
   metadata; it cannot replace user, token, `user_code`, `pairing_id`, budget,
   or signature checks.
6. Enable Cloud Armor verbose logging for early rollout and incident triage.

## Webhook Notes

Stripe webhook IP allowlisting is optional and operationally expensive because
provider ranges can change. Treat Stripe signature verification as the primary
defense, and use Cloud Armor rate limiting to reduce invalid request CPU cost.
If an allowlist is used, document the update owner and review cadence.

AdMob SSV should similarly rely on signature verification and transaction
idempotency as the primary defense. Cloud Armor is a DoS mitigation layer, not
proof that a callback is valid.

## Rollout

1. Deploy the policy in preview/logging mode when possible.
2. Watch Cloud Armor denied/throttled logs, backend 429 logs, and provider
   webhook dashboards for at least one business day.
3. Tighten only one group at a time: auth, extension pairing, generation, then
   webhooks.
4. Keep an emergency broad allow override documented and time boxed.
5. Record final limits in this document after production tuning.

## Emergency Actions

- Auth abuse: temporarily lower auth/session and auth/refresh limits; consider
  blocking abusive IPs at Cloud Armor.
- Extension pairing abuse: lower pairing limits, then check app-side
  `user_code` and `pairing_id` rate-limit counters.
- LLM generation abuse: lower generation/sync limits, lower global LLM budget,
  and pause Cloud Tasks if provider calls continue.
- Webhook flood: lower webhook edge limits and verify signatures are still the
  primary rejection source before disabling provider delivery.

If a future `/api/v1/questions/generate` alias is added, put it under the same
generation rule as `/api/v1/questions/generate/manual`.

See [Security Runbook](security-runbook.md) and
[Monitoring and Alerts](ops-monitoring-alerts.md).
