# AI Study Tool

AI Study Tool is a cross-platform learning app that turns Kindle highlights into AI-generated quiz questions. It is designed around a Web app, Chrome Extension, and Mobile app, with a Go backend that handles auth, highlight import, question generation, billing, reward tokens, and operations tooling.

The project is also a portfolio piece focused on production-minded backend design: scoped client auth, CSRF protection, LLM cost controls, async jobs, webhook idempotency, security hardening, and operational runbooks.

## Key Features

- Firebase Auth login for Web and Mobile users.
- Chrome Extension pairing flow for Kindle Notebook highlight import.
- AI question generation from imported highlights.
- Answer, review, incorrect-question, and saved-question workflows.
- Premium subscription and reward-token model.
- Admin Dashboard for operational checks and limited support actions.
- Security and operations hardening: session cookies, signed CSRF, App Check, scoped extension tokens, budget limits, secret scan, security headers, runbooks, and E2E/QA checklists.

## Architecture

```txt
Web React/Vite  ─┐
Mobile Expo     ├─> Go + Echo API ─> PostgreSQL
Chrome MV3 Ext ─┘        │
                         ├─> Cloud Tasks workers
                         ├─> Firebase Auth
                         ├─> LLM provider adapter
                         ├─> Stripe webhooks
                         └─> AdMob SSV
```

Core components:

- `frontend/`: React + TypeScript + Vite Web app.
- `backend/`: Go + Echo API, usecases, repositories, middleware, and task handlers.
- `extension/`: Chrome Extension Manifest V3 implementation.
- `mobile/`: React Native + Expo mobile app and share-sheet flows.
- `e2e/`: Playwright smoke E2E suite.
- `docs/`: architecture, security, operations, release, QA, and portfolio notes.

The backend follows a Clean Architecture style:

```txt
handler -> usecase -> domain <- infrastructure / repository
```

## Security Highlights

- Web uses HttpOnly Session Cookie authentication with signed CSRF tokens.
- Mobile uses Firebase Bearer ID Tokens with App Check and app-version gates.
- Browser Extension uses dedicated scoped tokens. Extension tokens can import/check highlights but cannot call LLM generation routes.
- LLM usage is protected by per-user budget, global daily budget, usage logs, queue depth limits, and async workers.
- Cloud Tasks plus database compare-and-set behavior prevents duplicate job processing.
- Stripe webhooks use signature verification and event idempotency.
- AdMob rewards use SSV signature verification and transaction id idempotency.
- Admin APIs require Web Session auth, role checks, CSRF, recent auth for risky actions, and audit logs.
- CI includes secret scanning and security-focused tests.

## Demo Flow

Use a staging or local demo account with dummy highlight data.

1. Log in to the Web app.
2. Open the Chrome Extension and start pairing.
3. Approve the extension code on `/extension/connect`.
4. Import sample Kindle Notebook highlights.
5. Generate questions from the imported highlights.
6. Answer questions and review incorrect/saved questions.
7. Explain premium/reward-token limits without real billing.
8. Open `/admin` to show operational overview, job state, LLM budget, and audit log.

Do not show real email addresses, secrets, raw tokens, cookies, webhook payloads, SSV query strings, prompts, or highlight text in demos.

## Local Development

Backend:

```bash
cd backend
go run ./cmd/main.go
```

Frontend:

```bash
cd frontend
npm run dev
```

Extension:

```bash
cd extension
npm run build
```

Mobile:

```bash
cd mobile
npx expo start --dev-client --lan --port 8081
```

Database:

- Use `DATABASE_URL` for backend database access.
- Apply migrations from `backend/db/migrations/`.
- Do not commit `.env` files or service account keys.

For physical mobile devices, set `EXPO_PUBLIC_API_BASE_URL` to the Mac LAN address and include the `/api/v1` base path.

## Test Commands

Backend:

```bash
cd backend && go test ./... && go build ./...
```

Frontend:

```bash
cd frontend && npm run typecheck && npm test && npm run build
```

Extension:

```bash
cd extension && npm run typecheck && npm test && npm run build
```

Mobile:

```bash
cd mobile && npm run typecheck && npm test
```

E2E:

```bash
cd e2e && npm run test
```

Security scan:

```bash
python3 scripts/secret_scan.py
```

## Docs Index

Portfolio and demo:

- [Architecture Summary](docs/architecture-summary.md)
- [Portfolio Notes](docs/portfolio.md)
- [Interview Notes](docs/interview-notes.md)
- [Demo Script](docs/demo-script.md)
- [Screenshots Checklist](docs/screenshots.md)
- [Future Roadmap](docs/future-roadmap.md)

Security:

- [Security Architecture](docs/security-architecture.md)
- [Client Security Model](docs/security-clients.md)
- [Security Runbook](docs/security-runbook.md)
- [Admin Dashboard Operations](docs/admin-dashboard.md)

Operations and release:

- [Cloud Armor Operations Plan](docs/ops-cloud-armor.md)
- [Monitoring and Alert Plan](docs/ops-monitoring-alerts.md)
- [Production Readiness Checklist](docs/production-readiness-checklist.md)
- [Production Environment and Secrets](docs/env-production.md)
- [Deploy and Rollback Runbook](docs/deploy-runbook.md)
- [Smoke Test Checklist](docs/smoke-test.md)
- [QA Checklist](docs/qa-checklist.md)

Clients and tests:

- [Extension README](extension/README.md)
- [Extension Store Readiness](extension/STORE_READINESS.md)
- [Mobile Release Readiness](docs/mobile-release-readiness.md)
- [E2E Test Guide](e2e/README.md)
- [k6 Load Tests](loadtest/k6/README.md)

Reasoning:

- [Security Final Review](reasoning/security-final-review-2026-05-30.md)

## Limitations / Future Work

- Mobile IAP / Play Billing is not complete.
- Kindle note-field support is a future enhancement.
- Production Cloud Armor and monitoring examples still need real environment application.
- Chrome Web Store final screenshots, store copy, and review assets are still needed.
- Real production alert routing, dashboards, and on-call processes need environment-specific setup.
- Spaced repetition can be improved beyond the current review flows.

See [Future Roadmap](docs/future-roadmap.md) for a prioritized roadmap.
