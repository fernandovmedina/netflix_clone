# Netflix Clone — Project Context

## Overview

A full Netflix clone: Next.js frontend, Go microservices behind an nginx load balancer, a PostgreSQL database, and a real adaptive-bitrate video pipeline. Everything runs locally through Docker Compose.

**Supabase has been removed.** Authentication, sessions, user management and all data access are implemented in this codebase against its own PostgreSQL instance. There is no GoTrue, no `@supabase/ssr`, and no hosted dependency of any kind — `supabase/config.toml` is a leftover of the old setup and is not used.

**Author:** Fernando Vazquez / [@fernandovmedina](https://github.com/fernandovmedina)

---

## Architecture

```
Browser (Next.js 16, localhost:3000)
    │  fetch(..., { credentials: "include" }) → localhost:8080
    ▼
nginx (localhost:8080) — microservices/nginx/nginx.conf
    │   Docker-DNS resolver, re-resolves replicas at request time
    │   rate limits on login/signup/refresh, connection cap on streaming
    ▼
auth ×3 — the only tier exposed to the browser
    │   • signup/login/logout/refresh, bcrypt(12), JWT access + rotating refresh
    │   • Google OAuth (PKCE)
    │   • verifies the access token on every request, injects X-User-Id / X-User-Email
    │   • reverse-proxies authenticated /api/v1/* to the owning service via nginx :8081
    │
    ├── catalog ×2    titles, movies, series, seasons, episodes, admin CRUD, upload intake
    ├── streaming ×2  HLS manifests and segments from the shared media volume
    └── user ×2       progress, favorites, profiles, plans, discounts, payments
         │
worker ×2 — claims transcode jobs from PostgreSQL, runs ffprobe/ffmpeg, writes HLS
postgres — single source of truth for data *and* for the job queue
media volume — shared between catalog (uploads), worker (output) and streaming (reads)
```

Every service is stateless. Sessions live in PostgreSQL, media lives on a shared volume, and the job queue is a PostgreSQL table — so any replica can serve any request, and no user is pinned to an instance.

### Scaling

Services are declared once with `deploy.replicas`, and nginx proxies to the **compose service name** through Docker's embedded DNS (`resolver 127.0.0.11 valid=1s ipv6=off` with `resolver_timeout 1s`, plus a variable `proxy_pass http://$backend$request_uri`). Adding capacity therefore needs no config change at all:

```bash
docker compose up -d --scale auth=5 auth
```

nginx picks up the new replicas within the one-second DNS TTL, without a restart or an edit.

> **Why the variable `proxy_pass`:** OSS nginx resolves the hostnames in an `upstream` block **once at worker startup** and caches the IPs forever. Because Docker reassigns container IPs on restart, the old `upstream auth_service { server auth1:8080; ... }` form silently routed traffic to whatever service had inherited that IP — in practice a third of all authenticated requests were being handed to the streaming service, which 404'd them. Do not reintroduce a static `upstream` block for these tiers.

---

## Directory Structure

```
netflix_clone/
├── CLAUDE.md
├── INTEGRATION.md            # the build brief this project is executing
├── docker-compose.yaml       # the whole stack: postgres, migrate, seed, all services, nginx, frontend
├── .env                      # local secrets (git-ignored); .env.example documents the keys
├── database/
│   ├── migrations/           # the real SQL migrations — source of truth
│   ├── seed/main.go          # seed importer implementation
│   └── database.sql          # legacy schema reference, not used
├── docs/ARCHITECTURE.md      # the build contract: topology, schema, pipeline, API, milestones
├── seed/                     # catalog seed: movies/, series/, video/video.mp4, artwork
├── frontend/                 # Next.js 16, React 19, TypeScript, Tailwind 4, pnpm
│   ├── app/                  # landing, login, signup flow, home/browse, watch, admin
│   ├── components/           # Navbar, Carousel, Hero, TitleModal, VideoPlayer, payments/, admin/
│   └── utils/api/client.ts   # cookie-based API client with single-flight 401 refresh
└── microservices/
    ├── nginx/nginx.conf      # :80 public edge, :8081 internal tier
    ├── shared/               # jwtutil, authctx, database, jsonx, migrate, renditions, storage
    ├── migrate/              # Dockerfile only; runner is shared/migrate/main.go, SQL in database/migrations
    ├── seed/                 # Dockerfile only; importer is database/seed/main.go
    ├── auth/                 # entry point: auth, OAuth, JWT middleware, reverse proxy
    ├── catalog/              # public reads + /api/v1/admin/* writes + upload intake
    ├── streaming/            # manifest and segment serving
    ├── user/                 # progress, favorites, profiles, plans, discounts, payments
    └── worker/               # ffprobe → ladder → ffmpeg → HLS
```

---

## Authentication

- **Passwords:** bcrypt, cost 12. Login compares against a constant dummy hash when the user does not exist, so response timing does not leak account existence.
- **Access token:** JWT (`ACCESS_TOKEN_TTL`), delivered as an **HttpOnly** cookie. Not readable from JavaScript.
- **Refresh token:** opaque, stored hashed in `sessions`, rotated on every use (`REFRESH_TOKEN_TTL`, default 720h). Reuse of an already-rotated token invalidates the session family — a stolen refresh token cannot be replayed.
- **Google OAuth:** PKCE (S256) with single-use `oauth_states` rows carrying the verifier and a 10-minute expiry. Start at `GET /api/v1/auth/google`, callback at `/api/v1/auth/google/callback`.
- **Authorization:** `requireAuth` on `/api/v1/*`, `requireAdmin` on `/api/v1/admin` and `/api/v1/admin/`. Downstream services trust `X-User-Id`/`X-User-Email` **only** because auth strips any inbound copy and sets them itself.
- Auth injects `X-User-Id`, `X-User-Email` **and** `X-User-Role` downstream.
- Logout revokes the refresh-token record and clears the cookies, but an access JWT already copied elsewhere stays valid until it expires (`ACCESS_TOKEN_TTL`, 15 minutes locally). Sessions are not checked per-request against the database.
- Any replica can mint, verify, refresh or revoke a session, because all session state is in PostgreSQL.

---

## Video Pipeline

**Upload → ready** (the HTTP request never waits for the transcode):

1. Admin uploads to `POST /api/v1/admin/movies/{id}/video` or `.../episodes/{id}/video`.
2. Catalog validates the target exists, sniffs the content type from the bytes, streams to `media/sources/<asset-id>/source<ext>` (`.mp4`, `.mov`, `.mkv` or `.webm`, matched against the sniffed MIME type) under a size cap, and inserts a `video_assets` row plus a `video_jobs` row.
3. It responds **202** with `{asset_id, status: "pending"}`.
4. A worker claims the job with `FOR UPDATE SKIP LOCKED` and a 30-minute lease, heartbeated while ffmpeg runs. Every job and asset write is fenced on still owning the lease, so a stale worker cannot clobber newer state.
5. ffprobe determines the source resolution; the ladder emits only renditions at or below it (`144, 240, 360, 480, 720, 1080, 1440`) — **never upscaled**.
6. ffmpeg encodes H.264 (main, yuv420p, `scale=-2:H` so dimensions stay even and the aspect ratio is preserved) with per-rendition bitrate/maxrate/bufsize, aligned keyframes (`-force_key_frames expr:gte(t,n_forced*6)`, `sc_threshold 0`) so ABR switching is seamless.
7. Output targets 6-second VOD segments (`-hls_time 6`; the final segment is usually shorter): `hls/<asset-id>/<quality>/playlist.m3u8` plus `seg_%05d.ts`, and a `master.m3u8` advertising every rendition. Its `CODECS` attribute is hardcoded to `avc1.4d401f`, plus `mp4a.40.2` when the source has audio — it is not derived per encoded stream.
8. The asset flips to `ready` only after the whole ladder succeeds; failures record the error and terminate after `max_attempts`.

**Serving:** streaming resolves the path, refuses anything escaping `MEDIA_ROOT` (including via symlink), requires the asset to be `ready` **and** its title published, and sends bytes with `ServeContent` — range requests supported, nothing buffered through memory. Segments are `immutable, max-age=31536000`; every upload gets a fresh asset id, so a re-upload can never be served stale segments.

Storage sits behind a `Store` interface (`Put`/`Open`/`Path`/`Remove`) with a local-volume implementation. **It is not yet S3-swappable:** `Path` hands out a local filesystem path, the worker uses `filepath`/`os.Rename`/`Walk` and temp directories directly, and streaming bypasses `Store` entirely with `os.Open`. Introducing object storage would mean changing the interface, the worker's staging/publish step and the streaming read path — the abstraction is a starting point, not a finished seam.

---

## Payments

Money is calculated **only** on the backend. The client sends `plan_id` and an optional discount `code` — never a price, subtotal, discount or total, and any such field in the request body is ignored.

- **Card:** simulated authorization; only brand and last four digits are persisted, never the PAN.
- **OXXO:** generates a voucher reference and expiry; `POST /api/v1/payments/oxxo/{ref}/simulate-payment` completes it. This is explicitly a local simulation and is kept separate from any real-provider path.
- **Discounts:** fixed or percent, validated server-side against active/starts_at/expires_at/max_redemptions/per_user_limit, with distinct errors per failure. Note `unique(discount_id, user_id)` makes the effective per-user maximum **always 1**, whatever `per_user_limit` says. Redemption is a transactional `SELECT ... FOR UPDATE` plus a unique constraint, so concurrent attempts on a single-use code produce exactly one redemption (the loser gets 409).
- **Subscription activation:** a card payment is marked paid immediately and activates a one-month subscription. An OXXO voucher starts `pending` with a 72-hour expiry and activates the subscription only when simulated.
- A discount redemption is consumed when the OXXO voucher is **created**, not when it is paid, and voucher expiry does not release it.
- The whole simulation path can be disabled with `PAYMENTS_SIMULATION_ENABLED=false`.
- **There is no crypto payment method** and never was one.

---

## Database

PostgreSQL, owned by this project. SQL migrations live in `database/migrations` and are applied at startup by the runner in `microservices/shared/migrate` (the `microservices/migrate` directory holds only its Dockerfile). `database/database.sql` is a historical reference. The runner also bootstraps the admin account from `ADMIN_EMAIL`/`ADMIN_PASSWORD` — changing `ADMIN_PASSWORD` later does **not** rotate an existing password hash.

`actors` · `categories` · `genres` · `titles` · `movies` · `series` · `seasons` · `episodes` · `title_actors` · `title_categories` · `title_genres` · `users` · `sessions` · `oauth_states` · `profiles` · `watch_progress` · `favorites` · `video_assets` · `video_jobs` · `plans` · `subscriptions` · `payments` · `discounts` · `discount_redemptions` · `schema_migrations`

Content tables keep soft-delete (`deleted_at`). `user_id` columns are real foreign keys to `users`. Profiles are capped at five per user, with names constrained to 1–50 characters in both the application and the database.

---

## Local Development

```bash
# whole stack (postgres + migrations + seed + all services + nginx + frontend)
docker compose up -d --build

# frontend only, against the containerized backend
cd frontend && pnpm dev        # localhost:3000
```

- Frontend: http://localhost:3000 · API: http://localhost:8080 · PostgreSQL: localhost:5433
- `migrate` and `seed` are one-shot services that must exit 0 before the app tiers start.
- Note: if you run `pnpm dev` on port 3000, the `frontend` container cannot bind and will exit.

### Environment (`.env`, git-ignored — see `.env.example`)

`POSTGRES_USER` · `POSTGRES_PASSWORD` · `POSTGRES_DB` · `DATABASE_URL` · `JWT_SECRET` · `ACCESS_TOKEN_TTL` · `REFRESH_TOKEN_TTL` · `COOKIE_SECURE` · `GOOGLE_CLIENT_ID` · `GOOGLE_CLIENT_SECRET` · `GOOGLE_REDIRECT_URL` · `FRONTEND_URL` · `CORS_ALLOWED_ORIGINS` · `CATALOG_SERVICE_URL` · `STREAMING_SERVICE_URL` · `USER_SERVICE_URL` · `MEDIA_ROOT` · `MAX_UPLOAD_BYTES` · `ADMIN_EMAIL` · `ADMIN_PASSWORD` · `NEXT_PUBLIC_API_URL` · `FRONTEND_PORT` (default 3000) · `PAYMENTS_SIMULATION_ENABLED` (default true) · `WORKER_CONCURRENCY` (default 1)

Only `NEXT_PUBLIC_*` reaches the browser bundle. Nothing else may.

### Tests

```bash
for module in microservices/{shared,auth,catalog,user,streaming,worker,integration} database/seed; do
  (cd "$module" && go build ./... && go vet ./... && go test -race ./...)
done
cd microservices/integration && go test -tags=integration -count=1 ./...   # end-to-end, needs the stack up
cd frontend && pnpm build && pnpm lint && pnpm exec tsc --noEmit
```

---

## API Surface

All routes are reached through `localhost:8080` and require the session cookie unless noted.

| Area | Routes |
|---|---|
| auth | `POST /api/v1/auth/{signup,login,logout,refresh}` (public) · `GET /api/v1/auth/me` · `GET /api/v1/auth/google[/callback]` (public) |
| catalog | `GET /api/v1/{titles,titles/{id},movies/{id},series/{id},genres,categories,actors,home}` |
| catalog admin | `POST/PATCH/DELETE /api/v1/admin/{movies,series,seasons,episodes,genres}` · `POST /api/v1/admin/{movies,episodes}/{id}/video` · `POST /api/v1/admin/titles/{id}/{thumbnail,publish}` · `POST /api/v1/admin/episodes/{id}/thumbnail` · `GET /api/v1/admin/assets/{id}` |
| streaming | `GET /api/v1/stream/{path...}` — `master.m3u8`, `<quality>/playlist.m3u8`, segments, artwork |
| user | `GET/PUT /api/v1/progress/{kind}/{id}` · `GET /api/v1/progress/continue` · `GET/POST/DELETE /api/v1/favorites` · `GET/POST/GET/PATCH/DELETE /api/v1/profiles[/{id}]` |
| payments | `GET /api/v1/plans` · `POST /api/v1/discounts/validate` · `POST /api/v1/payments/{card,oxxo}` · `POST /api/v1/payments/oxxo/{ref}/simulate-payment` · `GET /api/v1/payments/{id}` |

Public catalog reads return only published titles whose asset is `ready`; the admin projection additionally exposes `pending`/`processing`/`failed` state.

---

## Frontend Notes

- **Player** (`components/VideoPlayer.tsx`): hls.js with ABR, a manual quality menu built from `hls.levels`, resume-from-progress, and bounded recovery — up to two network recoveries (each refreshing the session, then `startLoad()`) and two media recoveries, after which it surfaces an error rather than retrying forever. hls.js is preferred wherever MSE exists; the native `<video>` HLS path is a fallback for engines without it, such as iOS Safari. Register the `MEDIA_ATTACHED` listener **before** `attachMedia()` — hls.js can emit it synchronously, and a listener added afterwards misses it, leaving the source never loaded.
- **API client** (`utils/api/client.ts`): cookies only — no token is ever held in JS. A 401 triggers a single-flight refresh, then one retry; it also refreshes in the background every 10 minutes so long viewing sessions do not expire mid-playback. (The backend accepts `Authorization: Bearer` too, but the frontend uses cookies exclusively.)
- **Middleware** matches `/home`, `/admin`, `/watch`, `/login` and `/signup`. It is a UX guard, not a security boundary — the backend authorizes every request, and that is what actually protects data.
- Money arrives as integer cents and is formatted for display; no float arithmetic.

---

## Hard Rules

- Never trust prices, totals or discounts from the client.
- Never serve media by reading a whole file into memory, and never expose the uploaded sources — only transcoded HLS output.
- Never upscale a rendition above the source resolution.
- Never let a transcode block an HTTP request, and never use an in-memory job queue — it breaks horizontal scaling.
- Never pin a session to an instance.
- Never commit generated media or `.env`.
