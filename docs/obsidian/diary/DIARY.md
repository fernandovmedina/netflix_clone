# Diary

## 2026-07-08 — Auth service, load balancer, frontend hookup

Built the backend entry point described in [[INSTRUCTIONS]] (backend/INSTRUCTIONS.md) and connected the frontend to it.

### Auth service (Go, `microservices/auth`)
- HTTP server with `POST /api/v1/auth/signup`, `POST /api/v1/auth/login`, `GET /api/v1/auth/user` (returns name, email, token) and `GET /health`.
- Signup/login call the Supabase Auth (GoTrue) REST API — same JWTs supabase-js issues, so frontend and backend sessions are identical.
- JWT middleware verifies tokens locally against the project JWKS (HS256 fallback via `SUPABASE_JWT_SECRET`); 401 when not logged in.
- `cors.go`: manual origin allowlist (`http://localhost:3000`).
- `proxy.go`: authenticated `/api/v1/*` requests are redirected to the owning service (catalog / streaming / user via `*_SERVICE_URL` env vars; 503 until they exist). Forwards `X-User-Id` / `X-User-Email`.
- Fixed `conndb.go` (connection was closed before use) → `pgxpool`.
- Fixed Dockerfile (ran the app at build time, never started a server) → multi-stage build.

### Load balancer (`microservices/docker-compose.yaml` + `nginx/nginx.conf`)
- 5 auth containers (`auth1`–`auth5`) behind nginx on `localhost:8080`, `least_conn`.
- Logs show routing on both sides: nginx `client -> upstream`, Go `[auth3] GET /... -> 200`.
- Verified: 40 concurrent requests spread 6/9/8/9/8 across containers; note sequential requests may all hit one container (per-worker balancing state).

### Frontend hookup (login + signup only)
- New `frontend/utils/api/auth.ts` — client for the auth service (`NEXT_PUBLIC_API_URL`, default `localhost:8080`).
- `/login`: backend login → `supabase.auth.setSession(tokens)` so existing cookie/session handling keeps working → `/home/browse`.
- `/signup/regform`: "Next" now actually creates the account via the backend, then goes to the verify-email step; errors shown with AlertMessage. Dropped the plaintext password stored in localStorage.
- Verified through nginx with a browser-identical request: CORS preflight passes, login POST reaches Supabase and relays its errors.

### Findings / loose ends
- The Supabase project (`wibbfjpldesaoaeveqmr`) has **email confirmation ON**: signup returns no access token until the link is clicked; login before that → "Email not confirmed".
- The **`public` schema is empty** — `database/database.sql` has never been applied to the remote project. Needed before catalog/user services.
- Leftover unconfirmed test user `authservice.smoketest.delete.me@gmail.com` in auth.users — delete from the dashboard.
- Added project-scoped Supabase MCP (`.mcp.json`); the claude.ai-level connector points at an unrelated project ("ROCEEL").
- Docs updated: root `CLAUDE.md`, `microservices/auth/CLAUDE.md`, `microservices/auth/README.md`.

### Next
- Apply `database.sql` to the remote project.
- Catalog microservice, then set `CATALOG_SERVICE_URL`.
- Wire the rest of the frontend data fetching through the load balancer with the Bearer token.
