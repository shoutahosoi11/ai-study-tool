# Smoke Test Checklist

Use this checklist for staging and low-risk production verification. Do not use
it for load testing. Do not print secrets, tokens, cookies, signatures, raw
webhook payloads, prompts, or highlight text.

Script:

```bash
BASE_URL=https://staging-api.example.com scripts/smoke_test.sh
```

Production requires an explicit confirmation:

```bash
BASE_URL=https://api.example.com CONFIRM_PRODUCTION_SMOKE=true scripts/smoke_test.sh
```

## Automated Low-Risk Checks

- `GET /health` returns 200.
- `GET /ready` returns 200.
- An authenticated-only API without credentials returns 401.
- `POST /api/v1/extension/pairing/start` returns a controlled response.
- Security headers are present on `/health`.

## Manual Or Staging-Only Checks

- Session creation with a fresh Firebase ID token.
- CSRF-protected state change with valid cookie and `X-CSRF-Token`.
- Extension import with a staging extension token.
- Question generation with `USE_GEMINI_MOCK=true` or very small staging data.
- Stripe webhook with Stripe CLI test event.
- AdMob SSV with mock/staging callback only.
- Frontend can call the API and complete login/session setup.

## Safety

- Do not run production destructive checks.
- Do not send real Stripe or AdMob production replay traffic from local scripts.
- Keep request rates low; use k6 scripts only from `loadtest/k6` with their
  production guard enabled.

