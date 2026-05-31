# Admin Dashboard

The Admin Dashboard provides a Web-only operations surface for checking production state without querying the database directly. It is intentionally narrow: it exposes counts, identifiers, status fields, and safe operational actions only.

## Permission Model

Admin access is controlled by the `user_roles` table, not by client-provided flags.

Roles:

- `viewer`: read-only dashboard access.
- `support`: support operations such as extension token revoke and generation job retry/cancel.
- `admin`: dangerous operations such as global LLM budget updates, revoke-all extension tokens, and logout-all.

Admin APIs require Web Session Cookie authentication. Mobile bearer tokens and Browser Extension scoped tokens are rejected before role lookup. Mutating admin routes require signed CSRF. High-impact operations also require recent auth through the existing recent-auth middleware.

Bootstrap an admin user manually in production after the app user row exists:

```sql
INSERT INTO user_roles (user_id, role)
VALUES ('<user-id>', 'admin')
ON CONFLICT (user_id) DO UPDATE SET role = EXCLUDED.role;
```

## Audit Log

Admin operations are written to `admin_audit_logs`.

Logged actions include:

- user lookup and user detail view
- extension token list, revoke, revoke all
- global LLM budget update
- generation job retry and cancel
- logout-all

Audit metadata is sanitized before storage. Keys containing `token`, `secret`, `cookie`, `signature`, `prompt`, `highlight`, or `raw` are dropped.

## Screens

- `/admin`: overview for LLM usage, global budget, generation job counts, queue estimate, Stripe/AdMob counts, extension imports, rate limit count, and recent audit logs.
- `/admin/users`: user search by email, user id, Firebase UID, Stripe customer id, or Stripe subscription id.
- `/admin/users/:id`: user detail and extension token management.
- `/admin/llm`: global LLM budget and usage summary by provider/model.
- `/admin/jobs`: generation job list with retry for failed/enqueue-failed jobs and cancel for queued/enqueue-failed jobs.
- `/admin/billing`: recent Stripe event ids and event types.
- `/admin/admob`: recent verified AdMob SSV events.

## Data Minimization

The dashboard must not display or log:

- raw extension tokens or token hashes
- cookies, signatures, or secrets
- raw Stripe webhook payloads
- raw AdMob query strings, signatures, or key material
- prompt text
- highlight text

Admin user search displays verified Firebase email only inside the admin surface. `users.email` is stored lowercase for search, but it is not treated as an identity key and may match multiple users. If `users.email` is not populated for older accounts, the dashboard shows `-` and search by email only works for rows with that column filled.

## Emergency Playbooks

Lower LLM budget:

1. Open `/admin/llm`.
2. Set `max_requests` or `max_estimated_cost_yen` near current usage.
3. Submit after recent sign-in.
4. Confirm `global_llm_budget_update` appears in audit logs.

Revoke an extension token:

1. Open `/admin/users`.
2. Search by user id, Firebase UID, email, or Stripe id.
3. Open the user detail.
4. Revoke one token or revoke all after recent sign-in.
5. Confirm audit logs and ask the user to reconnect the extension if needed.

Logout all sessions:

1. Open the user detail page.
2. Use `Logout all`.
3. Recent sign-in is required.
4. Firebase refresh tokens are revoked through the backend session manager.

Check failed jobs:

1. Open `/admin/jobs`.
2. Filter by `failed` or `enqueue_failed`.
3. Retry eligible failed jobs to set the job back to `queued` and re-enqueue the Cloud Tasks handler.
4. Cancel queued/enqueue-failed jobs when they should not run.
5. Queue pause remains a Cloud Tasks operation outside the dashboard.

## Current Limitations

- Stripe webhook failures and AdMob duplicate/stale-key fallback counts are shown only when represented by existing durable tables. Raw webhook/query data is intentionally not stored or displayed.
- Generation job cancel is limited to queued/enqueue-failed jobs. Processing job interruption remains a worker/Cloud Tasks operational concern.
- Admin role assignment is intentionally not exposed in the UI.
