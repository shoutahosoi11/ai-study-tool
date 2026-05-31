# Monitoring and Alert Plan

This document defines the production monitoring baseline for Cloud Run, Cloud
Tasks, LLM budget controls, Stripe, AdMob SSV, auth, and security controls.
Examples use placeholders only and are not applied automatically.

Runbook: [Security Runbook](security-runbook.md)

## Alert Severity

| Severity | Meaning | Response target |
| --- | --- | --- |
| `warning` | early signal, degradation, or threshold drift | inspect during business hours or active launch watch |
| `critical` | user-impacting, cost-risking, or security-relevant failure | page/on-call response |

## Cloud Run

| Signal | Suggested source | Warning | Critical | Runbook |
| --- | --- | --- | --- | --- |
| 5xx rate | Cloud Run request metric | >1% for 5m | >5% for 5m | Cloud Run 5xx |
| p95 latency | Cloud Run request latency | >1.5s for 10m | >3s for 5m | Cloud Run 5xx |
| request count spike | Cloud Run request metric | 3x 7-day baseline | 5x 7-day baseline | Webhook/request flood |
| instance count spike | Cloud Run instance metric | near configured max | sustained max instances | Cloud Run 5xx |
| CPU/memory | Cloud Run container metrics | >75% for 10m | >90% for 5m | Cloud Run 5xx |

## Cloud Tasks

| Signal | Suggested source | Warning | Critical | Runbook |
| --- | --- | --- | --- | --- |
| queue depth | Cloud Tasks queue metric | >100 queued | >500 queued | Cloud Tasks backlog |
| oldest task age | Cloud Tasks queue metric | >10m | >30m | Cloud Tasks backlog |
| dispatch failure rate | Cloud Tasks attempt metric | >5% for 10m | >20% for 5m | Cloud Tasks backlog |
| retry count spike | Cloud Tasks metric/log | 3x baseline | retry storm | Cloud Tasks backlog |

## LLM Budget

Some LLM budget signals are application/database signals and need custom
metrics. Until direct metrics exist, use scheduled SQL checks or log-based
metrics emitted from budget reservation paths.

| Signal | Current source | Warning | Critical | Runbook |
| --- | --- | --- | --- | --- |
| global request budget | `global_llm_budgets` | 70% used | 90% used | LLM cost spike |
| global request budget exhausted | `global_llm_budgets` | 90% used | 100% used | LLM cost spike |
| global estimated cost | `global_llm_budgets`, `llm_usage_logs` | 70% used | 90% used | LLM cost spike |
| user budget anomaly | app DB query/log metric | 3x user baseline | 10x user baseline | LLM cost spike |
| `llm_usage_logs` spike | DB count or log metric | 3x baseline | 5x baseline | LLM cost spike |
| estimated cost yen spike | DB sum or log metric | 3x baseline | 5x baseline | LLM cost spike |

## Stripe

| Signal | Current source | Warning | Critical | Runbook |
| --- | --- | --- | --- | --- |
| webhook 4xx | Cloud Run path metric/log metric | >2% for 10m | >10% for 5m | Stripe webhook anomaly |
| webhook 5xx | Cloud Run path metric/log metric | >1% for 10m | >5% for 5m | Stripe webhook anomaly |
| duplicate event spike | `stripe_events` conflict/log metric | 3x baseline | replay storm | Stripe webhook anomaly |
| signature verification failure | log-based metric, not fully emitted today | any sustained spike | high sustained spike | Stripe webhook anomaly |
| subscription update failure | error log metric | any | sustained or user-impacting | Stripe webhook anomaly |

## AdMob SSV

| Signal | Current source | Warning | Critical | Runbook |
| --- | --- | --- | --- | --- |
| signature verification failure | log-based metric, partially available through errors | >5/min | sustained spike | AdMob SSV anomaly |
| transaction duplicate spike | `admob_ssv_events` conflict/log metric | 3x baseline | replay storm | AdMob SSV anomaly |
| public key fetch failure | `admob_ssv_public_key_fetch_failed_using_stale_cache` log | any | stale fallback near expiry | AdMob SSV anomaly |
| stale cache fallback | existing slog warning | any | repeated for >15m | AdMob SSV anomaly |

## Auth And Client Integrity

| Signal | Current source | Warning | Critical | Runbook |
| --- | --- | --- | --- | --- |
| CSRF 403 spike | log-based metric needed for `csrf_rejected` | 3x baseline | likely attack or broken client | CSRF |
| App Check failure spike | log-based metric needed for `app_check_rejected` | 3x baseline | mobile clients locked out | Mobile/App Check |
| extension pairing 429 | backend rate-limit logs / Cloud Armor | 3x baseline | pairing outage or attack | Extension pairing |
| login/session failure | Cloud Run path status metric | >2% for 10m | >10% for 5m | Account takeover/auth |

## Security

| Signal | Current source | Warning | Critical | Runbook |
| --- | --- | --- | --- | --- |
| secret scan failure | GitHub Actions | any failure | merge blocked | Secret leak |
| suspicious request count | Cloud Armor logs | 3x baseline | active attack | Cloud Armor |
| rate-limit 429 spike | backend/Cloud Armor logs | 3x baseline | legitimate users impacted or attack | Cloud Armor |

## Structured Logging Inventory

The backend already initializes `slog` with Cloud Logging-compatible fields and
has some structured warnings, including stale AdMob public key cache and
extension token last-used update failures. The following event names should be
used for log-based metrics. Events marked "missing" should be added incrementally
near their existing handler/usecase boundaries without logging secrets, raw
tokens, cookies, signatures, raw query strings, raw bodies, prompts, or
highlight text.

| Event | Status | Notes |
| --- | --- | --- |
| `auth_session_created` | missing | include hashed/short UID and client type only |
| `auth_logout_all` | missing | include hashed/short UID |
| `csrf_rejected` | missing | include route, method, reason class, origin class |
| `app_check_rejected` | missing | include platform/version, not token |
| `extension_pairing_started` | missing | include rate-limit result, not raw user_code |
| `extension_pairing_approved` | missing | include hashed UID and pairing state |
| `extension_pairing_claimed` | missing | include token id, not raw token |
| `extension_token_revoked` | missing | include token id |
| `extension_import_rate_limited` | missing | include token id and route |
| `llm_budget_reserved` | missing | include budget date and counts |
| `llm_budget_exceeded` | missing | include budget type and route |
| `llm_generation_failed` | partial via task logs | avoid prompt/highlight text |
| `stripe_webhook_processed` | missing | include event type/id hash and duplicate flag |
| `stripe_webhook_rejected` | missing | include reason class, not payload |
| `admob_ssv_verified` | missing | include transaction id hash and ad unit |
| `admob_ssv_rejected` | missing | include reason class, not raw query/signature |
| `admob_pubkey_stale_cache_used` | partial | current log name is `admob_ssv_public_key_fetch_failed_using_stale_cache` |

