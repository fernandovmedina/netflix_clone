# Netflix Clone

A full-stack Netflix clone built with a microservices architecture, featuring authentication, content browsing, HLS video streaming, watch progress tracking, and favorites — all behind a Next.js middleware load balancer.

---

## Table of Contents

- [Architecture](#architecture)
- [Project Structure](#project-structure)
- [Technologies](#technologies)
- [Database Schema](#database-schema)
- [Frontend](#frontend)
- [Microservices](#microservices)
- [Load Balancer](#load-balancer)
- [Configuration](#configuration)
- [Local Development](#local-development)
- [Roadmap](#roadmap)

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        Browser                               │
└─────────────────────────────┬───────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│              Next.js 16  (frontend + middleware)             │
│                                                              │
│   ┌──────────────────────────────────────────────────────┐  │
│   │              Middleware  (Load Balancer)              │  │
│   │   1. Validate Supabase JWT / session cookie          │  │
│   │   2. Route request → correct microservice            │  │
│   │   3. Return 401 for unauthenticated requests         │  │
│   └───────┬──────────┬─────────────┬──────────┬─────────┘  │
└───────────┼──────────┼─────────────┼──────────┼────────────┘
            │          │             │          │
            ▼          ▼             ▼          ▼
      ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐
      │   Auth   │ │ Catalog  │ │Streaming │ │  User    │
      │  (Go)    │ │  (Go)    │ │  (Go)    │ │  (Go)    │
      └────┬─────┘ └────┬─────┘ └────┬─────┘ └────┬─────┘
           │             │            │              │
           └─────────────┴────────────┴──────────────┘
                                  │
                                  ▼
                    ┌─────────────────────────┐
                    │   Supabase (PostgreSQL)  │
                    │   + Supabase Auth        │
                    └─────────────────────────┘
```

The **Next.js middleware** is the single entry point for every request. It validates the user's Supabase session before forwarding to any microservice, meaning no backend service is ever exposed directly to the browser.

---

## Project Structure

```
netflix_clone/
├── CLAUDE.md                    # AI assistant context & architecture notes
├── README.md
├── database/
│   ├── database.sql             # Full PostgreSQL schema
│   └── exec.sql
├── frontend/                    # Next.js 16 application
│   ├── app/
│   │   ├── page.tsx             # Public landing page
│   │   ├── login/               # Sign-in page
│   │   ├── loginhelp/           # Forgot password
│   │   ├── signup/              # Multi-step registration (4 steps)
│   │   └── home/                # Authenticated area
│   │       └── page.tsx         # Browse page with carousels + title modal
│   ├── components/
│   │   ├── Navbar.tsx
│   │   └── AlertMessage.tsx
│   └── utils/supabase/
│       ├── client.ts            # Browser Supabase client
│       ├── server.ts            # Server-side Supabase client
│       └── middleware.ts        # Session handling + load balancer
└── microservices/
    ├── docker-compose.yaml
    └── auth/                    # Go auth microservice
        ├── main.go
        ├── Dockerfile
        ├── go.mod
        └── supabase/
            ├── conndb.go        # PostgreSQL connection (pgx)
            ├── login.go
            ├── signup.go
            └── getSession.go
```

---

## Technologies

### Frontend
| | Technology | Version |
|---|---|---|
| 🖼️ | Next.js | 16 |
| ⚛️ | React | 19 |
| 🔷 | TypeScript | 5 |
| 🎨 | Tailwind CSS | 4 |
| 🧩 | MUI (Material UI) | 9 |
| 🔐 | Supabase SSR | 0.8 |
| 📦 | Bun | latest |

### Backend
| | Technology | Purpose |
|---|---|---|
| 🐹 | Go | Microservices runtime |
| 🐘 | PostgreSQL | Primary database (via Supabase) |
| 🔑 | Supabase Auth | JWT-based authentication |
| 🐳 | Docker / Docker Compose | Containerization & orchestration |
| 📡 | HLS | Video streaming protocol |

### Infrastructure
| | Technology | Purpose |
|---|---|---|
| ☁️ | Supabase | Managed PostgreSQL + Auth + Storage |
| 🔀 | Next.js Middleware | Load balancer / API gateway |

---

## Database Schema

The database is PostgreSQL hosted on Supabase. All tables follow soft-delete conventions using a `deleted_at` timestamp.

```
actors ──────────────────────────────┐
categories ──────────────────────┐   │
genres ──────────────────────┐   │   │
                              │   │   │
titles ◄──── title_genres ───┘   │   │
  │    ◄──── title_categories ───┘   │
  │    ◄──── title_actors ───────────┘
  │
  ├──► movies (hls_manifest_path)
  │
  └──► series
           └──► seasons
                    └──► episodes (hls_manifest_path)

auth.users (Supabase)
  ├──► watch_progress  (movie XOR episode, progress in seconds)
  └──► favorites       (bookmarked titles)
```

### Tables

| Table | Description |
|---|---|
| `actors` | Actor catalog |
| `categories` | Content categories (e.g. Family, Thriller) |
| `genres` | Genre tags |
| `titles` | Parent record — enum `Movie` / `TV Show` |
| `movies` | Extends `titles`; stores duration + HLS path |
| `series` | Extends `titles`; stores number of seasons |
| `seasons` | Belongs to `series` |
| `episodes` | Belongs to `seasons`; stores duration + HLS path |
| `title_actors` | Many-to-many: titles ↔ actors |
| `title_categories` | Many-to-many: titles ↔ categories |
| `title_genres` | Many-to-many: titles ↔ genres |
| `watch_progress` | Per-user playback position (seconds) |
| `favorites` | Per-user bookmarked titles |

---

## Frontend

### Pages

| Route | Description |
|---|---|
| `/` | Public landing page — hero, trending carousel, FAQ, email capture |
| `/login` | Sign-in with email/password or one-time code |
| `/loginhelp` | Forgot password |
| `/signup` | Multi-step registration: plan → payment → profile |
| `/home` | Authenticated browse page — content carousels + title detail modal |

### Authentication Flow

1. User enters email on `/` → stored in `localStorage` → redirected to `/signup/linkRegistration`
2. On `/login`, `supabase.auth.signInWithPassword` is called client-side
3. On success, user is redirected to `/home/browse`
4. The Next.js middleware validates the Supabase session cookie on every request to protected routes and refreshes it automatically

---

## Microservices

Each microservice is a standalone Go binary containerized with Docker.

### Auth service — `microservices/auth`

Handles user registration, login, and session validation by talking directly to Supabase Auth and the PostgreSQL database.

```
Endpoints (planned)
  POST /auth/login
  POST /auth/signup
  GET  /auth/session
```

**Go module:** `github.com/fernandovmedina/netflix-clone/microservices/auth`  
**Key deps:** `pgx/v5`, `godotenv`  
**Config:** reads `DATABASE_URL` from `.env.local`

### Catalog service _(planned)_

Serves titles, movies, series, seasons, and episodes from the database.

### Streaming service _(planned)_

Serves HLS manifests and video segments for playback.

### User service _(planned)_

Manages watch progress and favorites per user.

---

## Load Balancer

The load balancer runs as a **Next.js middleware** (`frontend/utils/supabase/middleware.ts`). It intercepts every request before it reaches a page or API route.

### Responsibilities

1. **Auth validation** — creates a Supabase server client, checks the session JWT from the cookie
2. **Route protection** — redirects unauthenticated users away from `/home/*` to `/login`
3. **Upstream proxying** — forwards API requests to the correct microservice based on the path prefix:

| Path prefix | Microservice |
|---|---|
| `/api/auth/*` | Auth service |
| `/api/catalog/*` | Catalog service |
| `/api/stream/*` | Streaming service |
| `/api/user/*` | User service |

4. **Cookie refresh** — propagates updated Supabase session cookies back to the browser on every response

---

## Configuration

### Frontend environment variables

Create `frontend/.env.local`:

```env
NEXT_PUBLIC_SUPABASE_URL=http://localhost:54321
NEXT_PUBLIC_SUPABASE_PUBLISHABLE_KEY=<your-supabase-anon-key>
```

### Auth microservice environment variables

Create `microservices/auth/.env.local`:

```env
DATABASE_URL=postgresql://postgres:postgres@localhost:54322/postgres
```

### Supabase local config

`supabase/config.toml` is pre-configured:

```toml
project_id = "netflix_clone"

[api]
port = 54321          # Supabase REST API

[db]
port = 54322          # Direct PostgreSQL connection
```

---

## Local Development

### Prerequisites

- [Docker](https://www.docker.com) — runs the whole stack, including PostgreSQL
- [Go 1.25+](https://go.dev) — only to run the microservice tests directly
- [pnpm](https://pnpm.io) — only to run the frontend outside its container

### 1. Configure the environment

```bash
cp .env.example .env
```

Edit `.env` and set at least `JWT_SECRET`, `POSTGRES_PASSWORD`, `ADMIN_EMAIL`
and `ADMIN_PASSWORD`. Everything else has a working local default. Only
`NEXT_PUBLIC_*` values ever reach the browser bundle.

### 2. Start everything

```bash
docker compose up -d --build
```

That brings up PostgreSQL, applies the migrations, seeds the catalog, and starts
every service behind nginx.

- Frontend — <http://localhost:3000>
- API — <http://localhost:8080>
- PostgreSQL — `localhost:5433`

Sign in with the `ADMIN_EMAIL` / `ADMIN_PASSWORD` you set; the migration runner
bootstraps that account on first start.

### 3. Add video

**A fresh clone has no video.** Only code is committed — no `.mp4` is in the
repository — so the catalog seeds with titles, descriptions and artwork, and
every title is marked **"No video yet"** until someone uploads one. This is the
expected starting state.

To add video, sign in as the administrator, open
<http://localhost:3000/admin> and upload a clip to a title:

- a **movie** takes a single video;
- a **series** takes one video per episode, plus an optional episode thumbnail.

Uploads return immediately; a worker transcodes each one into an HLS ladder in
the background and the panel flips to **Ready** on its own. The title becomes
playable at that point.

You can also pre-seed from local clips instead of uploading — see
[`seed/video/README.md`](seed/video/README.md).

### Running pieces outside Docker

Frontend against the containerized backend (stop the `frontend` container first,
or it will hold port 3000):

```bash
cd frontend
pnpm install
pnpm dev
```

### Tests

```bash
for module in microservices/{shared,auth,catalog,user,streaming,worker} database/seed; do
  (cd "$module" && go build ./... && go vet ./... && go test -race ./...)
done

cd frontend && pnpm build && pnpm lint && pnpm exec tsc --noEmit
```

End-to-end tests need the stack running:

```bash
cd microservices/integration && go test -tags=integration -count=1 ./...
```

Tests that need a real video clip skip themselves when none is present.

---

## Roadmap

- [x] Landing page (hero, carousels, FAQ)
- [x] Login page with Supabase auth
- [x] Browse/home page with carousels and title detail modal
- [x] PostgreSQL schema (titles, movies, series, episodes, watch progress, favorites)
- [x] Auth microservice skeleton
- [ ] Load balancer routing in Next.js middleware
- [ ] Auth microservice: HTTP server + login / signup / getSession handlers
- [ ] Catalog microservice: list & search titles, movies, series, episodes
- [ ] Streaming microservice: HLS manifest serving
- [ ] User microservice: watch progress + favorites CRUD
- [ ] Docker Compose: full service definitions
- [ ] Sign-up flow: plan selection, payment step, profile creation
- [ ] Video player page with HLS playback

---

Made by [@fernandovmedina](https://github.com/fernandovmedina) & [@Neurovix](https://neurovix.com.mx)
