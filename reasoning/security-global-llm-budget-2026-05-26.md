# Security: Global LLM Budget

## Background

Per-user limits do not stop many-account abuse or indirect generation triggered
by high-volume highlight imports. The largest operational risk is LLM cost
growth from requests that still look authenticated.

## Design

- User and queue-depth checks run before enqueue.
- Workers reserve global LLM budget immediately before calling the provider.
- Reservation is transaction-backed so concurrent workers cannot exceed request
  or estimated-cost caps.
- Usage logs store provider/model/token counts/estimated cost, not prompt text.
- If the API call is reached, budget is consumed even if the provider later
  fails; pre-call validation failures do not consume budget.

## Alternatives

- Only per-user budgets were rejected because they do not limit service-wide
  spend.
- Provider-side spend caps alone were rejected because they do not explain which
  users/jobs caused pressure.

## Tradeoffs

Reserving before the provider call can overcount failed provider attempts, but it
prevents retry loops from repeatedly reaching paid APIs without accounting.

## Prevents

- Service-wide cost explosion from multiple users.
- Unlimited queue growth.
- Extension-import-triggered generation bypassing normal budget checks.

## Does Not Prevent

- All abuse below configured budget thresholds.
- Provider-side billing bugs.

## Future Work

- CTE-based reserve optimization.
- Batch queue-depth checks.
- User-facing retry messaging for global budget exhaustion.
