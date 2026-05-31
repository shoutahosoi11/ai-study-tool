# k6 Load Tests

These scripts are low-rate smoke tests for local or staging environments. They
are meant to verify rate limits, queue controls, auth behavior, and webhook
rejection paths. They are not attack tooling and must not be pointed at
production unless explicitly allowed.

## Install k6

macOS:

```bash
brew install k6
```

Other platforms: https://k6.io/docs/get-started/installation/

## Safety Controls

All scripts read `BASE_URL` from the environment. If `BASE_URL` looks like a
production host, the script exits unless:

```bash
ALLOW_PRODUCTION_LOADTEST=true
```

Use staging/local URLs by default:

```bash
BASE_URL=http://localhost:8080 k6 run loadtest/k6/webhook_admob_invalid.js
```

## Scripts

| Script | Purpose | Required env |
| --- | --- | --- |
| `auth_session.js` | low-rate `/api/v1/auth/session` session creation check | `BASE_URL`, `ID_TOKEN` |
| `extension_import.js` | extension import rate-limit smoke | `BASE_URL`, `EXTENSION_TOKEN` |
| `question_generation_queue.js` | queue/generation endpoint guardrail check | `BASE_URL`, `AUTH_TOKEN`, `QUESTION_GENERATION_LOADTEST_ENABLED=true` |
| `webhook_admob_invalid.js` | invalid AdMob SSV rejection/rate-limit check | `BASE_URL` |

## Examples

```bash
BASE_URL=http://localhost:8080 ID_TOKEN=test-id-token \
  k6 run loadtest/k6/auth_session.js

BASE_URL=https://staging-api.example.com EXTENSION_TOKEN=test-extension-token \
  k6 run loadtest/k6/extension_import.js

BASE_URL=https://staging-api.example.com AUTH_TOKEN=test-firebase-token \
  QUESTION_GENERATION_LOADTEST_ENABLED=true \
  k6 run loadtest/k6/question_generation_queue.js

BASE_URL=http://localhost:8080 \
  k6 run loadtest/k6/webhook_admob_invalid.js
```

## Question Generation Warning

There is no dedicated dry-run question generation endpoint today. The
`question_generation_queue.js` script is disabled by default because it may
trigger real generation, Cloud Tasks enqueueing, or LLM calls depending on the
environment. Prefer staging with `USE_GEMINI_MOCK=true` and a small test user.

## Do Not Log

Do not paste real tokens, cookies, signatures, raw webhook payloads, prompts, or
highlight text into tickets or shared logs.

