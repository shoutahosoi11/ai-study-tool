# Cloudflare Edge Checklist

Target shape:

```text
Client
  -> Cloudflare DNS / WAF / Rate Limiting
  -> Authenticated Cloud Run origin
  -> Echo API
  -> Neon PostgreSQL / Gemini / Cloud Tasks
```

Cloudflare is the public edge. Cloud Run should eventually stop accepting
unauthenticated public origin traffic directly.

## Origin Privacy

Preferred production posture:

1. Put the API hostname behind Cloudflare proxy mode.
2. Configure Cloud Run with authenticated invoker:

   ```sh
   gcloud run services update api-service \
     --region=asia-northeast1 \
     --no-allow-unauthenticated
   ```

3. Use a Cloudflare Worker, service token, or another controlled identity layer
   to call Cloud Run with authentication.
4. Do not publish the raw Cloud Run service URL in frontend or mobile config
   once the Cloudflare hostname is live.

If authenticated origin access is not ready on day one, keep Firebase Auth and
application rate limits enabled, and treat public Cloud Run URL exposure as a
temporary state with a clear removal date.

## WAF

Enable:

- Cloudflare Managed Rules.
- OWASP Core Ruleset.
- Rules for obvious bad paths such as `/wp-admin`, `/xmlrpc.php`, and generic
  scanner traffic.
- Challenge or block requests with suspicious automated characteristics.

Recommended initial WAF actions:

- Block malformed requests to `/api/*`.
- Challenge high-risk countries only if the product does not need them.
- Log first, then block, for any rule that may affect real users.

## Rate Limiting Rules

Start with conservative edge limits and tune from logs:

- `POST /api/highlights/import`: `30` requests per IP per minute.
- `POST /api/highlights/share`: `30` requests per IP per minute.
- `POST /api/highlights/paste`: `30` requests per IP per minute.
- `POST /api/questions/sync`: `20` requests per IP per minute.
- `POST /api/users/signup`: `10` requests per IP per minute.
- All other `POST /api/*`: `120` requests per IP per minute.

The backend still enforces authenticated per-user ingest limits through
`rate_limit_counters`. Cloudflare limits are an edge pressure valve, not the
source of truth for user quotas.

## DDoS And Bot Controls

Confirm:

- Cloudflare proxy is enabled for the API hostname.
- DDoS protection is on.
- Bot Fight Mode or Super Bot Fight Mode is enabled where plan support allows.
- Security event logging is retained long enough for incident review.
- Alerts are routed to the person operating the release.

## Rollout Checks

- `GET /health` succeeds through the Cloudflare hostname.
- Direct Cloud Run URL is blocked or requires auth once origin privacy is
  enabled.
- Firebase-authenticated API calls work through Cloudflare.
- Large malformed JSON bodies are rejected before stressing Cloud Run.
- Cloudflare logs show the expected origin hostname and no unexpected bypass.

## Workers Builds For Frontend

If the frontend is deployed with Cloudflare Workers Builds, use these settings
in Cloudflare Dashboard:

- Root directory: `frontend`
- Build command: `npm ci && npm run build`
- Deploy command: `npx wrangler deploy`
- Wrangler config: `frontend/wrangler.toml`
- Worker name: must match `name` in `frontend/wrangler.toml`

The repository config serves the Vite `dist` directory as static assets and
uses `single-page-application` routing so React Router paths return
`index.html`.

If the Cloudflare project is still configured with the repository root as its
root directory, use:

- Root directory: repository root
- Build command: `cd frontend && npm ci && npm run build`
- Deploy command: `npx wrangler deploy`
- Wrangler config: `wrangler.toml`

The root config forwards the build to `frontend` and serves
`frontend/dist`.
