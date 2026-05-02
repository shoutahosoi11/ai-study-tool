# Security Deploy Config

## What Was Documented

- Added `backend/db/migrations/README.md` with the Phase 2
  `031_security_hardening.sql` execution procedure, Neon options, preflight and
  postflight checks, data-impact notes, and manual rollback SQL.
- Added `docs/deployment/cloud-run.md` with Cloud Run scaling, concurrency,
  timeout, resource, health-check, and deploy-command guidance.
- Added `docs/deployment/cloudflare.md` with the intended edge topology,
  origin privacy posture, WAF, rate limiting, DDoS, and bot-control checklist.
- Added `docs/deployment/storage-security-checklist.md` for private Cloud
  Storage bucket operation with signed URLs.
- Added `docs/deployment/budget-alert.md` for budget alerts, emergency stop
  commands, and Gemini quota controls.
- Added `docs/deployment/incident-runbook.md` for Cloud Run, Cloudflare, Neon,
  rate-limit, and question-generation incidents.

## Design Rationale

The repository already uses `backend/db/migrations/*.sql` as sorted,
forward-only migrations. The security hardening migration was therefore
documented as `031_security_hardening.sql` instead of introducing `.up.sql` and
`.down.sql` files that the current `Makefile` would accidentally execute.
Rollback is documented as manual SQL plus Neon PITR guidance, which fits the
current operational model without changing tooling during a security phase.

Cloud Run `--max-instances=10` is documented as a hard cost and database
pressure guardrail. Application rate limits help only after requests reach the
service; max instances caps the platform behavior when traffic volume,
Cloudflare rules, or client behavior are wrong.

Cloudflare is documented as the public edge because it can absorb and classify
traffic before Cloud Run. The backend still keeps Firebase Auth and per-user
Postgres-backed rate limits, so Cloudflare is an outer layer rather than the
only enforcement point.

## Alternatives Considered

- Cloud Armor instead of Cloudflare: Cloud Armor is a strong GCP-native option,
  especially with load balancers, but this project already benefits from a
  simpler edge checklist with DNS proxy, WAF, rate limiting, DDoS controls, and
  bot controls in one place. Cloud Armor remains a reasonable future option if
  the architecture moves toward a Google Cloud HTTP(S) Load Balancer.
- Adding migration tooling now: `golang-migrate` would make up/down migrations
  explicit, but adding it during this phase would change release mechanics. The
  safer immediate move is to document the existing psql flow and provide manual
  rollback SQL.
- Relying only on backend rate limits: backend limits are authenticated and
  user-aware, but they still consume Cloud Run, DB, and network resources.
  Cloudflare and Cloud Run caps provide earlier and broader controls.

## Operational Notes

- The Phase 2 migration backfills `content_hash` for non-empty existing
  highlights and keeps duplicate candidates nullable to preserve the partial
  unique index.
- Existing `source IS NULL` rows remain `NULL`; legacy `mobile_share` and
  `kindle` values are converted to `share` and `extension`.
- The deploy workflow currently uses `--allow-unauthenticated`; the docs keep
  that as the current state and describe the target authenticated-origin setup
  once Cloudflare is ready to be the only public entrypoint.
