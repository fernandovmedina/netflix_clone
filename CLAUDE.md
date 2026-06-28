# Netflix Clone — Project Context

## Overview

A full Netflix clone with frontend, backend microservices, load balancer middleware, and a PostgreSQL database hosted on Supabase. The goal is feature parity with real Netflix: browsing, authentication, watch progress, favorites, video playback via HLS.

**Author:** Fernando Vazquez / [@fernandovmedina](https://github.com/fernandovmedina)

---

## Architecture

```
Browser (Next.js 16)
    │
    ├── Next.js Middleware (load balancer)
    │       • Validates Supabase session/JWT
    │       • Routes authenticated requests to correct microservice
    │
    ├── Auth Microservice (Go)         — /microservices/auth
    ├── [future] Catalog Microservice  — titles, movies, series, episodes
    ├── [future] Streaming Microservice — HLS manifest serving
    └── [future] User Microservice     — watch progress, favorites
```

Microservices are containerized via Docker. Orchestration file lives at `microservices/docker-compose.yaml` (currently minimal — only `version: "3"` defined, services TBD).

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
│   ├── docker-compose.yaml
│   └── auth/            # Go microservice
│       ├── main.go
│       ├── go.mod        # module: github.com/fernandovmedina/netflix-clone/microservices/auth
│       ├── Dockerfile
│       └── supabase/
│           ├── conndb.go    # pgx connection via DATABASE_URL
│           ├── login.go     # (stub)
│           ├── signup.go    # (stub)
│           └── getSession.go # (stub)
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
3. `frontend/utils/supabase/middleware.ts` wraps `createServerClient` and refreshes cookies — **this is where the load balancer logic will live**

---

## Load Balancer (Planned — Next.js Middleware)

The plan is to extend `frontend/utils/supabase/middleware.ts` (or a root `middleware.ts`) to:
1. Validate the Supabase session JWT on every request
2. Based on the route/path, proxy the request to the appropriate backend microservice (auth, catalog, streaming, user)
3. Return 401 for unauthenticated access to protected routes

This keeps the frontend as the single entry point and avoids exposing microservices directly.

---

## Auth Microservice (Go)

- Module: `github.com/fernandovmedina/netflix-clone/microservices/auth`
- Go version: 1.25.3
- Dependencies: `pgx/v5` (PostgreSQL driver), `godotenv`
- Reads `DATABASE_URL` from `.env.local`
- `conndb.go` connects via `pgx.Connect` — **note:** currently defers `conn.Close` before returning, which closes the connection immediately (bug to fix)
- `login.go`, `signup.go`, `getSession.go` are stubs

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
cd microservices/auth
# create .env.local with DATABASE_URL
go run main.go
```

---

## What's Built vs. What's Planned

### Built
- Landing page (hero, FAQ accordion, carousels, footer)
- Login page (email/password + "use a sign-in code" toggle)
- Supabase auth integration (client-side sign-in)
- Home/browse page (carousels, title detail modal with episodes list)
- Full PostgreSQL schema
- Auth microservice skeleton (DB connection)
- Supabase middleware cookie handler

### Planned / In Progress
- Load balancer logic in Next.js middleware
- Auth microservice: login, signup, getSession handlers + HTTP server
- Catalog microservice (list titles, movies, series, episodes)
- Streaming microservice (HLS manifest serving)
- User microservice (watch progress, favorites)
- Docker Compose service definitions
- Sign-up multi-step flow completion (plan selection, payment, profile)
- `/home/browse` route (currently `/home/page.tsx`)
