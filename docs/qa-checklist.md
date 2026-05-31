# QA Checklist

Use this checklist before public release. Fill in environment, account type, and notes for every run. Do not paste secrets, cookies, raw tokens, raw webhook payloads, SSV query strings, prompts, or highlight text into the notes.

## Environment

- [ ] Environment:
- [ ] Frontend URL:
- [ ] Backend URL:
- [ ] App version:
- [ ] Extension version:
- [ ] Mobile build:
- [ ] Tester:
- [ ] Date:

## Login

- [ ] Confirm staging test user can log in.
  - Expected: dashboard opens and session-backed API calls succeed.
  - Notes:
- [ ] Confirm invalid credentials show a user-facing error.
  - Expected: no stack trace or raw API error is shown.
  - Notes:

## Logout / Logout-All

- [ ] Confirm logout clears the browser session.
  - Expected: protected pages redirect to login.
  - Notes:
- [ ] Confirm logout-all requires recent auth.
  - Expected: stale auth is rejected or user is asked to re-authenticate.
  - Notes:

## Extension Connect

- [ ] Open `/extension/connect`.
  - Expected: unauthenticated users are sent to login and return after login.
  - Notes:
- [ ] Enter invalid `user_code`.
  - Expected: user-facing validation error.
  - Notes:
- [ ] Approve a staging pairing code.
  - Expected: success message asks user to return to the extension.
  - Notes:
- [ ] Confirm token and pairing id are not displayed.
  - Expected: no raw token, token hash, or pairing id appears.
  - Notes:

## Highlight Import

- [ ] Import sample highlights through a disposable extension token.
  - Expected: import succeeds and dashboard reflects imported book/highlight counts.
  - Notes:
- [ ] Confirm 0-highlight UI.
  - Expected: empty state is clear.
  - Notes:
- [ ] Confirm rate-limit UI.
  - Expected: 429 is shown as a retry-later message.
  - Notes:

## Question Generation

- [ ] Trigger question generation with mock/fake LLM or dry-run staging.
  - Expected: loading then queued/completed state.
  - Notes:
- [ ] Confirm budget shortage UI.
  - Expected: user sees a recoverable budget message.
  - Notes:
- [ ] Confirm global budget exceeded UI.
  - Expected: user sees a temporary service-limit message.
  - Notes:
- [ ] Confirm rate-limit UI.
  - Expected: user sees retry-later messaging.
  - Notes:

## Answer And Review

- [ ] Open a question.
  - Expected: choices render without layout overlap.
  - Notes:
- [ ] Submit an answer.
  - Expected: correct/incorrect state is stored.
  - Notes:
- [ ] Open incorrect questions.
  - Expected: incorrect answer appears.
  - Notes:
- [ ] Save a question.
  - Expected: saved question appears in saved list.
  - Notes:

## Billing

- [ ] Create Stripe checkout in test mode.
  - Expected: loading state appears and checkout URL is returned.
  - Notes:
- [ ] Send mocked webhook event.
  - Expected: subscription state updates.
  - Notes:
- [ ] Send duplicate webhook event.
  - Expected: duplicate is idempotent no-op.
  - Notes:
- [ ] Confirm success URL alone does not upgrade plan.
  - Expected: webhook is required for premium state.
  - Notes:

## Ad Reward

- [ ] Confirm ad reward entry point is visible where expected.
  - Expected: client-only action does not grant tokens.
  - Notes:
- [ ] Confirm mocked or integration SSV grants tokens.
  - Expected: verified SSV increments balance once.
  - Notes:

## Admin

- [ ] Admin user can open `/admin`.
  - Expected: overview renders.
  - Notes:
- [ ] Non-admin user opens `/admin`.
  - Expected: 403 or login gate.
  - Notes:
- [ ] Admin user search works.
  - Expected: minimal user data appears.
  - Notes:
- [ ] Extension token revoke creates audit log.
  - Expected: token is revoked and audit log is visible.
  - Notes:
- [ ] Global budget update requires recent auth.
  - Expected: stale auth is rejected.
  - Notes:
- [ ] Confirm sensitive values are not displayed.
  - Expected: no raw token, prompt text, highlight text, signature, cookie, or secret appears.
  - Notes:

## Security Headers

- [ ] Check top-level frontend response headers.
  - Expected: CSP/frame policy is present through hosting layer.
  - Notes:
- [ ] Check backend response headers.
  - Expected: `X-Content-Type-Options`, frame protection, and CSP or `frame-ancestors` policy where applicable.
  - Notes:

## Mobile Version Gate / App Check

- [ ] Mobile request without App Check.
  - Expected: rejected in enforced environments.
  - Notes:
- [ ] Mobile request below minimum version.
  - Expected: version gate rejects or asks update.
  - Notes:

## Error UX

- [ ] 404 route.
  - Expected: user-facing fallback, no stack trace.
  - Notes:
- [ ] 500 response.
  - Expected: temporary problem message, no raw response body.
  - Notes:
- [ ] Network error.
  - Expected: retry-friendly message.
  - Notes:
