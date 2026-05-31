# E2E Tests

This directory contains the first PR14 smoke E2E suite. It uses Playwright and defaults to dry-run behavior so it can run against a local frontend without real Firebase, Stripe, AdMob, or LLM traffic.

## Safety Defaults

Environment variables:

- `E2E_BASE_URL`: frontend base URL. Default: `http://127.0.0.1:3000`.
- `E2E_API_BASE_URL`: backend base URL. Default: `http://127.0.0.1:8080`.
- `E2E_TEST_EMAIL`: staging test user email. Do not print it in tests.
- `E2E_TEST_PASSWORD`: staging test user password. Do not print it in tests.
- `E2E_ADMIN_EMAIL`: staging admin email. Do not print it in tests.
- `E2E_ADMIN_PASSWORD`: staging admin password. Do not print it in tests.
- `E2E_EXTENSION_TOKEN`: disposable staging extension token. Do not print it in tests.
- `E2E_ALLOW_PRODUCTION`: must be `true` to run against production-like HTTPS hosts.
- `E2E_DRY_RUN`: defaults to `true`. Set `false` only for disposable staging.
- `E2E_RUN_API_TESTS`: set `true` only when a disposable backend is running.
- `E2E_SKIP_WEB_SERVER`: set `true` when `E2E_BASE_URL` is already served.

The Playwright config refuses production-like URLs unless `E2E_ALLOW_PRODUCTION=true` is explicitly set.

## Install

```bash
cd e2e
npm install
npx playwright install chromium
```

## Run

Local frontend smoke:

```bash
cd e2e
npm run test
```

Disposable backend security smoke:

```bash
cd e2e
E2E_RUN_API_TESTS=true E2E_API_BASE_URL=http://127.0.0.1:8080 npm run test
```

Staging with pre-running services:

```bash
cd e2e
E2E_SKIP_WEB_SERVER=true \
E2E_BASE_URL=https://staging.example.com \
E2E_API_BASE_URL=https://staging-api.example.com \
E2E_RUN_API_TESTS=true \
npm run test
```

## Current Coverage

- Web login page smoke.
- Protected route redirect for `/extension/connect`.
- Extension connect page sensitive-text regression.
- Admin route auth gate smoke.
- Optional API smoke for unauthenticated Admin and Extension import rejection.
- Optional backend security-header smoke.

## Deferred Full E2E

Full login, pairing approval, highlight import, question generation, answer/review, Stripe test checkout, AdMob SSV, and Admin mutation flows need a disposable staging environment with seeded Firebase users, test extension tokens, mock LLM provider, and cleanup credentials.

## Test Data Policy

- Use only staging or disposable local projects.
- Prefix created records with `e2e_` or `test_`.
- Do not create real Stripe charges or real AdMob reward traffic.
- Do not log passwords, cookies, raw tokens, raw webhook payloads, raw SSV query strings, prompts, or highlight text.
- Cleanup should delete or revoke data by test prefix and revoke disposable extension tokens after each run.
