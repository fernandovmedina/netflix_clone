# Netflix Clone — Auth Service (Go)

Go microservice that is the **single entry point of the backend**: it handles
signup/login against Supabase Auth, validates the Supabase JWT session on
every request, and redirects authenticated `/api/v1/*` requests to the
microservice that owns the route. In its production shape it runs as 5
containers behind an nginx load balancer (`../docker-compose.yaml` +
`../nginx/nginx.conf`), published on `http://localhost:8080`.

Design spec: `docs/obsidian/backend/INSTRUCTIONS.md` (repo root). This
supersedes the earlier plan of putting the load balancer in Next.js
middleware. For full-system context see `../../CLAUDE.md`.

**Author:** Fernando Vazquez / [@fernandovmedina](https://github.com/fernandovmedina)

---

## Stack

- **Language:** Go 1.25+ (module `github.com/fernandovmedina/netflix-clone/microservices/auth`)
- **HTTP:** stdlib `net/http` with Go 1.22+ method patterns (`GET /health`)
- **Database:** `pgx/v5` (`pgxpool`) → Supabase Postgres via `DATABASE_URL`
- **Auth:** Supabase Auth (GoTrue) REST API for signup/login; `golang-jwt/jwt/v5` + `MicahParks/keyfunc/v3` for local JWT verification via JWKS
- **Config:** `godotenv` loading `.env.local` (optional — docker-compose injects env via `env_file` instead)

---

## File Structure

```
auth/
├── main.go          # entry point: env load, JWT init, DB pool, routes, server
├── handlers.go      # /health, signup, login, user handlers + JSON helpers
├── middleware.go    # requireAuth (Bearer JWT → claims in context) + request logging
├── jwt.go           # Supabase JWT verification: JWKS by default, HS256 via SUPABASE_JWT_SECRET
├── cors.go          # manual allowlist of frontend origins (currently http://localhost:3000)
├── proxy.go         # route-prefix → microservice reverse proxy (the "redirect" logic)
├── database/        # Supabase backend access (package database)
│   ├── conndb.go    # pgxpool connection via DATABASE_URL
│   ├── gotrue.go    # GoTrue REST client, Session/User types, AuthError
│   ├── signup.go    # POST /auth/v1/signup (name stored in user metadata)
│   ├── login.go     # POST /auth/v1/token?grant_type=password
│   └── getSession.go# GET /auth/v1/user (round-trips to Supabase; catches revoked sessions)
├── Dockerfile       # multi-stage: golang:1.25-alpine build → alpine runtime
├── .dockerignore    # keeps .env.local out of images (compose injects env instead)
├── .env.local       # git-ignored credentials (see Environment below)
└── README.md        # endpoint/user-facing docs
```

---

## Endpoints

| Method | Path | Auth | Description |
|---|---|---|---|
| GET | `/health` | — | Liveness; returns `{status, container}` (used by compose healthcheck) |
| POST | `/api/v1/auth/signup` | — | `{name, email, password}` → creates user in Supabase Auth |
| POST | `/api/v1/auth/login` | — | `{email, password}` → full Supabase session (`access_token`, `refresh_token`, `user`) |
| GET | `/api/v1/auth/user` | Bearer | `{name, email, token}`; re-checks token with Supabase (catches revocation) |
| ANY | `/api/v1/*` | Bearer | Verify JWT → reverse-proxy to owning service; 401 if not logged in, 503 if target service not configured |

Errors are JSON `{"error": "..."}`. Supabase Auth failures are relayed with
their original status (e.g. 400 "Invalid login credentials").

---

## Key Design Decisions

- **Signup/login call the GoTrue REST API, not the database.** The frontend
  logs in with `supabase.auth.signInWithPassword`, so only Supabase can issue
  compatible JWTs. Both paths therefore produce identical sessions. Never
  write to `auth.users` directly.
- **JWTs are verified locally** (no network per request) against the
  project's JWKS at `{SUPABASE_URL}/auth/v1/.well-known/jwks.json`
  (ES256/RS256, auto-refreshed by keyfunc). Legacy HS256 projects: set
  `SUPABASE_JWT_SECRET` and it switches to shared-secret verification.
  Claims checked: signature, expiry (required), audience `authenticated`.
- **Proxied requests carry `X-User-Id` / `X-User-Email`** headers so
  downstream services don't re-parse the JWT.
- **Every request logs the serving container** (`[auth3] GET /api/v1/... -> 200`),
  and nginx logs `client -> upstream` per request, so load distribution is
  visible in `docker compose logs`.
- The pgx pool (`database.ConnDB`) is connected at startup but no endpoint
  queries it yet — it is non-fatal if unreachable and is there for future
  handlers.

## Route → Service Map (`proxy.go`)

| Prefix (`/api/v1/…`) | Service | Target env var |
|---|---|---|
| `titles`, `movies`, `movie`, `series`, `serie`, `seasons`, `episodes`, `actors`, `categories`, `genres` | catalog | `CATALOG_SERVICE_URL` |
| `stream` | streaming | `STREAMING_SERVICE_URL` |
| `progress`, `favorites`, `profiles` | user | `USER_SERVICE_URL` |

Adding a new microservice: add its prefixes to `serviceRoutes` in `proxy.go`,
add the `*_SERVICE_URL` env var, and add the service to
`../docker-compose.yaml`.

---

## Load Balancer (nginx + Docker)

- `../docker-compose.yaml`: services `auth1`–`auth5` (shared YAML anchor,
  each with a `hostname` so logs identify the container) + `nginx` publishing
  `localhost:8080 → :80`.
- `../nginx/nginx.conf`: `upstream auth_service` with `least_conn` across
  `auth1:8080`–`auth5:8080`; access log shows `$upstream_addr`.
- **Scaling:** copy an `authN` block in docker-compose, register
  `server authN:8080;` in nginx.conf, `docker compose up -d --build`.
- Note: sequential requests may all land on one container (per-worker
  balancing state); distribution shows under concurrent load — verified
  6/9/8/9/8 across the 5 containers with 40 concurrent requests.

---

## Environment (`.env.local`, git-ignored)

```
DATABASE_URL=            # Supabase Postgres (pgxpool)
DATABASE_HOST/PORT/USER/PASS/NAME    # components (legacy, URL is what's used)
SUPABASE_URL=            # https://<ref>.supabase.co — same project as frontend
SUPABASE_ANON_KEY=       # publishable/anon key (same as frontend)
SUPABASE_JWT_SECRET=     # optional — only for legacy HS256 projects
PORT=                    # optional, default 8080
CATALOG_SERVICE_URL=     # unset until the catalog service exists → 503
STREAMING_SERVICE_URL=   # unset until the streaming service exists → 503
USER_SERVICE_URL=        # unset until the user service exists → 503
```

`.dockerignore` keeps `.env.local` out of images; docker-compose injects it
via `env_file`.

---

## Commands

```bash
# single instance (reads .env.local, defaults to :8080)
go run .

# build + vet
go build ./... && go vet ./...

# full load-balanced stack (from ../)
docker compose up -d --build     # entry point: http://localhost:8080
docker compose logs -f nginx     # client -> upstream routing
docker compose logs -f auth3     # one container's request log
docker compose down
```

---

## Gotchas / Current State

- **Email confirmation is ON** in the Supabase project: signup returns 201
  with the user but an empty `access_token` until the email is confirmed —
  login only works after confirmation.
- **The remote `public` schema is empty** (verified 2026-07-08):
  `database/database.sql` has not been applied to the Supabase project yet.
  Auth works regardless (it only touches `auth.*`), but catalog/user services
  will need the schema applied first.
- The Supabase project ref used by the app is `wibbfjpldesaoaeveqmr`; the
  project-scoped Supabase MCP in `.mcp.json` points at the same project.
- CORS allowlist lives in `cors.go` (`allowedOrigins`) — add new frontend
  origins there manually.
- Frontend integration is still pending: the frontend should call
  `http://localhost:8080` with `Authorization: Bearer <supabase access token>`.
