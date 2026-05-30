# Auth Phase 2: Extension Token

## Background

Browser extensions run in a higher-exposure storage environment than the Web app
and should not hold Firebase Session Cookies or broad Firebase credentials.

## Design

- Pairing uses a device-held `pairing_id` and a user-entered `user_code`.
- `pairing_id` is never placed in Web URLs and is used only by the extension.
- `POST /pairing/status` is read-only; `POST /pairing/claim` issues the token
  once and marks the pairing used.
- Raw `ext_` tokens are returned only at issue time. The database stores only
  `SHA-256(raw_token)`.
- Grantable scopes are limited to highlight import/check and self-revoke.

## Alternatives

- Reusing Firebase ID Tokens in the extension was rejected because it would
  expand compromise impact.
- A single pairing ID in the browser URL was rejected because logs, history, and
  screen sharing could leak the claim secret.

## Tradeoffs

The device-code style flow is more complex than a single code, but it prevents a
visible user code from being sufficient to claim a token. DB-backed rate limits
cost an extra write, but they survive Cloud Run instance scaling.

## Prevents

- Broad account access from extension token theft.
- Accidental token consumption by GET prefetching.
- Fast pairing claim brute force across a single pairing ID.

## Does Not Prevent

- Malicious extension code acting within granted highlight scopes.
- A compromised user account approving a malicious pairing.

## Future Work

- Web UI polling to show when the extension has completed claim.
- Token rotation UX.
- Batch rate-limit repository calls for lower DB write overhead.
