# Netflix Clone — Project Context

## Overview

A full Netflix clone with frontend, backend microservices, load balancer middleware, and a PostgreSQL database hosted on Supabase. The goal is feature parity with real Netflix: browsing, authentication, watch progress, favorites, video playback via HLS.

**Author:** Fernando Vazquez / [@fernandovmedina](https://github.com/fernandovmedina)

---

## Architecture

```
Browser (Next.js 16, localhost:3000)
    │
    ▼
nginx load balancer (localhost:8080) — /microservices/nginx/nginx.conf
    │   least_conn across 5 auth containers, logs which upstream served
    ▼
Auth Microservice ×5 (Go) — /microservices/auth
    │   • signup/login via Supabase Auth (GoTrue REST)
    │   • verifies Supabase JWT locally (JWKS) on every request
    │   • CORS allowlist (cors.go)
    │   • proxies authenticated /api/v1/* to the owning service
    │
    ├── [future] Catalog Microservice  — titles, movies, series, episodes
    ├── [future] Streaming Microservice — HLS manifest serving
    └── [future] User Microservice     — watch progress, favorites
```

The auth service is the single entry point / load balancer of the backend (per `docs/obsidian/backend/INSTRUCTIONS.md` — this supersedes the earlier plan to put the load balancer in Next.js middleware). Orchestration lives at `microservices/docker-compose.yaml`: 5 auth containers (`auth1`–`auth5`) behind nginx published on `localhost:8080`.

---

## Directory Structure

```
netflix_clone/
├── CLAUDE.md
├── database/
│   ├── database.sql     # Full PostgreSQL schema
│   └── exec.sql         # Execution helper
├── frontend/            # Next.js 16 app (React 19, TypeScript, Tailwind 4)
│   ├── app/
│   │   ├── page.tsx              # Landing page (email capture, FAQ, carousel)
│   │   ├── login/page.tsx        # Sign-in page (Supabase auth)
│   │   ├── loginhelp/page.tsx    # Forgot password
│   │   ├── signup/               # Multi-step sign-up flow (4 steps)
│   │   └── home/                 # Authenticated area
│   │       ├── layout.tsx
│   │       └── page.tsx          # Browse page with carousels + title modal
│   ├── components/
│   │   ├── Navbar.tsx
│   │   └── AlertMessage.tsx
│   └── utils/supabase/
│       ├── client.ts     # Browser Supabase client
│       ├── server.ts     # Server-side Supabase client
│       └── middleware.ts # Next.js middleware — session handling + load balancer entry point
├── microservices/
│   ├── docker-compose.yaml  # 5 auth containers + nginx LB on localhost:8080
│   ├── nginx/
│   │   └── nginx.conf   # upstream auth1..auth5, least_conn, upstream logging
│   └── auth/            # Go microservice (see auth/CLAUDE.md + auth/README.md)
│       ├── main.go       # HTTP server, routes
│       ├── handlers.go   # signup/login/user handlers
│       ├── middleware.go # requireAuth (JWT) + request logging
│       ├── jwt.go        # Supabase JWT verification (JWKS, HS256 fallback)
│       ├── cors.go       # manual origin allowlist (localhost:3000)
│       ├── proxy.go      # route → microservice reverse proxy
│       ├── go.mod        # module: github.com/fernandovmedina/netflix-clone/microservices/auth
│       ├── Dockerfile    # multi-stage build → alpine
│       └── database/
│           ├── conndb.go    # pgxpool via DATABASE_URL
│           ├── gotrue.go    # Supabase Auth REST client + Session/User types
│           ├── login.go     # password grant
│           ├── signup.go    # signup with name in user metadata
│           └── getSession.go # GET /auth/v1/user (revocation-aware)
└── supabase/
    └── config.toml      # project_id = "netflix_clone", local API port 54321, DB port 54322
```

---

## Database Schema (`database/database.sql`)

PostgreSQL (Supabase-compatible). All tables use soft-delete (`deleted_at`).

| Table | Purpose |
|---|---|
| `actors` | Actor catalog |
| `categories` | Content categories |
| `genres` | Genre tags |
| `titles` | Parent record for all content (enum: `Movie` / `TV Show`) |
| `movies` | Extends `titles`; has `duration`, `hls_manifest_path` |
| `series` | Extends `titles`; has `number_of_seasons` |
| `seasons` | Belongs to `series` |
| `episodes` | Belongs to `seasons`; has `duration`, `hls_manifest_path` |
| `title_actors` | M2M join |
| `title_categories` | M2M join |
| `title_genres` | M2M join |
| `watch_progress` | Per-user progress (movie XOR episode, in seconds) |
| `favorites` | Per-user title bookmarks |

`user_id` in `watch_progress` and `favorites` is a `uuid` that maps to Supabase Auth users — no explicit FK to `auth.users` in the SQL file.

---

## Frontend Stack

- **Framework:** Next.js 16 (App Router), React 19
- **Language:** TypeScript
- **Styling:** Tailwind CSS v4
- **UI libs:** MUI v9 (`@mui/material`), `@deemlol/next-icons`
- **Auth:** `@supabase/ssr` + `@supabase/supabase-js`
- **Package manager:** Bun (has `bun.lock`); npm also present

### Key env vars (frontend)
```
NEXT_PUBLIC_SUPABASE_URL
NEXT_PUBLIC_SUPABASE_PUBLISHABLE_KEY
```

### Auth flow
1. Landing page (`/`) captures email → stores in `localStorage` as `signup_email` → pushes to `/signup/linkRegistration`
2. Login page (`/login`) calls `supabase.auth.signInWithPassword` directly from the client, then redirects to `/home/browse`
3. `frontend/utils/supabase/middleware.ts` wraps `createServerClient` and refreshes session cookies

---

## Load Balancer (nginx + auth service)

Per `docs/obsidian/backend/INSTRUCTIONS.md`, the backend load balancer is nginx in front of 5 auth-service containers (not Next.js middleware):
1. nginx (`microservices/nginx/nginx.conf`) distributes requests (`least_conn`) across `auth1`–`auth5` and logs which upstream served each request
2. Each auth container validates the Supabase session JWT (401 for unauthenticated access) and proxies `/api/v1/*` to the owning microservice, forwarding `X-User-Id`/`X-User-Email`
3. To scale: add `authN` to `docker-compose.yaml` and register `server authN:8080;` in `nginx.conf`

---

## Auth Microservice (Go)

- Module: `github.com/fernandovmedina/netflix-clone/microservices/auth` — full docs in `microservices/auth/README.md`
- Go version: 1.25.x
- Dependencies: `pgx/v5` (pgxpool), `godotenv`, `golang-jwt/jwt/v5`, `MicahParks/keyfunc/v3` (JWKS)
- Env (`.env.local`, git-ignored): `DATABASE_URL`, `SUPABASE_URL`, `SUPABASE_ANON_KEY`; optional `SUPABASE_JWT_SECRET` (legacy HS256), `PORT`, `CATALOG_SERVICE_URL`, `STREAMING_SERVICE_URL`, `USER_SERVICE_URL`
- Endpoints: `GET /health`, `POST /api/v1/auth/signup`, `POST /api/v1/auth/login`, `GET /api/v1/auth/user` (returns `{name, email, token}`), plus authenticated reverse proxy for all other `/api/v1/*` routes
- Signup/login call the Supabase Auth (GoTrue) REST API so tokens are identical to the ones supabase-js issues to the frontend; JWTs are verified locally against the project JWKS
- Every request logs the serving container: `[auth3] GET /api/v1/... -> 200`

---

## Local Development

### Frontend
```bash
cd frontend
bun dev        # or: npm run dev
```
Runs on http://localhost:3000

### Supabase (local)
```bash
supabase start
```
- API: http://localhost:54321
- DB: postgresql://localhost:54322

### Auth microservice
```bash
# single instance (reads microservices/auth/.env.local)
cd microservices/auth
go run .

# load-balanced stack: 5 containers + nginx on http://localhost:8080
cd microservices
docker compose up -d --build
```

---

## What's Built vs. What's Planned

### Built
- Landing page (hero, FAQ accordion, carousels, footer)
- Login page (email/password + "use a sign-in code" toggle)
- Supabase auth integration (client-side sign-in)
- Home/browse page (carousels, title detail modal with episodes list)
- Full PostgreSQL schema
- Auth microservice: HTTP server, signup/login/user endpoints (Supabase Auth), JWT middleware (JWKS), CORS allowlist, reverse proxy to future services
- nginx load balancer + docker-compose (5 auth containers, `localhost:8080`)
- Supabase middleware cookie handler

### Planned / In Progress
- Frontend: point API calls at the load balancer (`http://localhost:8080`) with the Supabase access token as Bearer
- Catalog microservice (list titles, movies, series, episodes) → set `CATALOG_SERVICE_URL`
- Streaming microservice (HLS manifest serving) → set `STREAMING_SERVICE_URL`
- User microservice (watch progress, favorites) → set `USER_SERVICE_URL`
- Sign-up multi-step flow completion (plan selection, payment, profile)
