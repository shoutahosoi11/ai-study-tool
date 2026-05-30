# Auth Phase 2: Mobile App Check

## Background

Mobile clients use Firebase ID Token Bearer auth, not cookies. CSRF does not
apply, but unauthenticated copies of the API client should be harder to use at
scale.

## Design

- Mobile requests send Firebase ID Token, App Check token, app version, and
  platform headers.
- Production requires App Check.
- Development/test can explicitly disable App Check enforcement for local work.
- Version gate runs only after authentication and App Check; version headers are
  client metadata, not trust anchors.
- Recent auth for sensitive operations uses Firebase `auth_time`, not biometric
  headers.

## Alternatives

- Requiring CSRF for Mobile was rejected because Mobile does not use cookie auth.
- Trusting a biometric success header was rejected because it is client
  controlled.

## Tradeoffs

App Check adds operational setup and can block clients if provider verification
breaks. The development bypass is therefore explicit and disallowed in strict
environments.

## Prevents

- Basic scripted abuse with copied Firebase ID Tokens but no valid app
  attestation.
- Old app versions after minimum version is configured.

## Does Not Prevent

- Fully compromised legitimate devices.
- Abuse by real authenticated users within their allowed limits.

## Future Work

- Platform-specific upgrade URLs.
- More structured mobile auth telemetry without logging PII or tokens.
