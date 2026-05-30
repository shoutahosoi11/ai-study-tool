# Security: Stripe And AdMob SSV

## Background

Payment and reward flows must be idempotent and must not trust client-provided
price, premium state, or ad-watched notifications.

## Design

- Stripe checkout uses server-side price configuration.
- Stripe webhooks verify raw body signatures and store `event_id` once.
- Subscription state is stored in provider-neutral rows for future Apple/Google
  billing support.
- AdMob rewards are granted only from verified SSV callbacks.
- SSV verification checks signature, key ID, timestamp, user mapping, and
  `transaction_id` idempotency.
- Successful SSV responses return only HTTP status, never user balance.

## Alternatives

- Client POST "watched ad" was rejected for production because it is trivial to
  forge.
- Storing raw webhook/SSV payloads was rejected to avoid retaining signatures or
  payment payloads.

## Tradeoffs

SSV public key cache fallback improves availability during transient Google key
fetch failures, but it is capped to a short stale window and fails closed when no
cache exists.

## Prevents

- Duplicate Stripe event processing.
- Duplicate AdMob reward grants.
- Client-forged ad rewards in production.
- User balance disclosure to Google webhook callers.

## Does Not Prevent

- Provider account compromise.
- Valid but abusive reward traffic under rate limits.

## Future Work

- Native Apple/Google purchase verification.
- More operational dashboards for webhook duplicate and signature failure rates.
