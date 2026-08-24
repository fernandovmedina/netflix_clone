# Netflix Clone — Target Architecture (build contract)

> **This file is the contract.** Codex 1, Codex 2 and Codex 3 all build against it.
> If you need to deviate, say so in your output instead of silently diverging.
> Owner of this file: the lead engineer (Claude). Workers do not edit it.

Status legend: `TODO` not started · `WIP` in progress · `DONE` implemented + verified.

---

## 1. Decisions already made (do not re-litigate)

| # | Decision |
|---|---|
| D1 | **Supabase is removed entirely.** Auth, users, sessions, DB access are ours. No `@supabase/*` package in the frontend, no GoTrue calls in Go. |
| D2 | **PostgreSQL 17 in Docker** with a named volume, plus versioned SQL migrations. The old local Supabase stack and the remote Supabase project are NOT touched or migrated in place; they are torn down only after the new stack is verified. |
| D3 | **Services are stateless.** No in-memory sessions, no in-memory job queue, no sticky routing. Any instance must serve any request. |
| D4 | **Job queue is Postgres** (`FOR UPDATE SKIP LOCKED` + lease/heartbeat). No Redis, no RabbitMQ, no new infra. |
| D5 | **Shared media lives on a Docker named volume** behind a `storage.Store` Go interface so S3 can be swapped in later without touching handlers. |
| D6 | **HLS / H.264 / AAC**, 6-second segments, per-rendition playlists under one master playlist. |
| D7 | **No crypto payments.** The frontend never had a crypto method — nothing to remove. Report this, do not invent one to delete. |
| D8 | **Gift Code is the discount system.** Codes live in Postgres; redemption, amount and single-use are validated **server-side only**. Card and OXXO totals both flow through it. |
| D9 | **Layout is `microservices/<service>/`** (already moved). One `docker-compose.yaml` at the repo root runs everything. |
| D10 | **Money is `numeric(10,2)` in Postgres and integer minor units (cents) on the wire.** Never a float, anywhere. |

---

## 2. Service topology

```
                    Browser (Next.js :3000)
                            │
                            ▼
              nginx load balancer  :8080          microservices/nginx/nginx.conf
                            │  Docker DNS round-robin, logs $upstream_addr
                            ▼
                   auth ×3   (:8080)              microservices/auth
                   │  the single entry point:
                   │  · owns signup/login/logout/refresh/Google OAuth
                   │  · verifies the access token on every other route
                   │  · reverse-proxies to the owning service with
                   │    X-User-Id / X-User-Email / X-User-Role
       ┌───────────┼───────────────┬──────────────────┐
       ▼           ▼               ▼                  ▼
  catalog ×2   streaming ×2     user ×2          (worker ×2, no HTTP)
   :8080         :8080           :8080            microservices/worker
  microservices/catalog          microservices/user
       │           │               │                  │
       └───────────┴───────┬───────┴──────────────────┘
                           ▼
                    postgres:17  :5432   volume: pgdata
                           +
                    media volume /media  (shared: catalog W, worker RW, streaming R)
```

Container count: 1 nginx + 3 auth + 2 catalog + 2 streaming + 2 user + 2 worker + 1 postgres + 1 frontend = **14**.

Each tier is one Compose service with `deploy.replicas`; Docker DNS exposes its
active replica addresses and nginx re-resolves them every second. Compose
health conditions gate nginx's initial startup on the HTTP tiers. Scale a tier
without editing or restarting nginx, for example:

```sh
docker compose up -d --scale auth=5
docker compose up -d --scale auth=3
```

Compose waits for healthy HTTP replicas before starting nginx, so nginx cold
start is not the source of the previously observed DNS miss. Repeated requests
against a healthy steady-state stack and immediately after an nginx restart did
not reproduce it. A forced container remove/recreate did: Docker Desktop can
briefly return NXDOMAIN for several Compose service aliases while network
membership changes, even though the other replicas remain healthy. A trial of
nginx 1.27 shared upstream zones with dynamic `resolve` made that edge worse by
emptying the peer groups on NXDOMAIN, so the proven variable `proxy_pass`
configuration remains in place. A normal `docker restart` does not detach the
container from its network; the one-second resolver validity and timeout bound
any Docker DNS pause while the process comes back. Scale/recreate operations can
still expose a transient 502 if a request lands inside Docker Desktop's DNS
control-plane window; it is not stale-IP misrouting and recovery is automatic
on the next resolver refresh. Eliminating that Docker Desktop edge completely
would require a stale-answer DNS cache or service discovery outside nginx, which
is not justified for this local-only stack.

The frontend publishes port 3000 by default. If that host port is occupied by a
development server, run the production container on another port without
changing the Compose file:

```sh
FRONTEND_PORT=3001 docker compose up -d --build frontend
```

The local nginx CORS allowlist includes both `http://localhost:3000` and the
documented alternate `http://localhost:3001`, so authenticated browser requests
still target the baked `http://localhost:8080` API URL in either mode.

Replica hostnames are container IDs, so per-request service logs still identify
the individual container that handled each request.

### Service ownership

| Service | Owns | Never does |
|---|---|---|
| `auth` | users, credentials, sessions, refresh rotation, Google OAuth, RBAC gate, reverse proxy | business queries, media I/O |
| `catalog` | titles/movies/series/seasons/episodes, genres/actors/categories, admin CRUD, upload intake, job creation | transcoding, streaming bytes |
| `worker` | claims video jobs, ffprobe, ffmpeg transcode, writes HLS, updates job + asset rows | serving HTTP |
| `streaming` | serves `master.m3u8`, rendition playlists, `.ts` segments, range requests, cache headers | transcoding, auth decisions beyond the injected headers |
| `user` | profiles, watch progress, favorites, plans, subscriptions, discounts, card + OXXO payment simulation | media, catalog writes |

---

## 3. Authentication (stateless, multi-instance)

### Tokens

| Token | Form | Lifetime | Storage |
|---|---|---|---|
| Access | JWT, **HS256**, secret `JWT_SECRET` shared by all instances via env | 15 min | `access_token` cookie, HttpOnly, SameSite=Lax, Path=/ |
| Refresh | opaque 32-byte random, base64url | 30 days | `refresh_token` cookie, HttpOnly, SameSite=Lax, Path=/api/v1/auth |

Access-token claims: `sub` (user uuid), `email`, `role` (`user`|`admin`), `iat`, `exp`, `jti`, `iss: netflix-clone`, `aud: netflix-clone`.

**Why this survives load balancing:** the access token is verified with a shared secret — no instance-local state, no JWKS fetch, no lookup. The refresh token is only ever checked against Postgres, which every instance shares.

### Refresh tokens

- Stored **hashed** (`sha256`) in `sessions.refresh_token_hash`. Never store the raw token.
- **Rotation on every use.** The old row is marked `revoked_at = now()` and the new row records `rotated_from`.
- **Reuse detection:** presenting an already-revoked token revokes the whole `session_family` and forces re-login. This is what makes rotation safe.
- Rotation must be atomic: `UPDATE ... WHERE revoked_at IS NULL RETURNING` in one statement, so two concurrent refreshes on two instances cannot both win.

### Passwords

`bcrypt` cost 12 (`golang.org/x/crypto/bcrypt`). Compare with `CompareHashAndPassword` only — never `==`. Login must take the same time whether or not the email exists (hash a dummy on the miss path).

### Google OAuth

Authorization Code + PKCE. Credentials already exist in `microservices/auth/.env.local` as `GOOGLE_CLIENT_ID` / `GOOGLE_CLIENT_SECRET`.

```
GET  /api/v1/auth/google           -> 302 to Google; state + code_verifier persisted in oauth_states
GET  /api/v1/auth/google/callback  -> exchange code, upsert user, set cookies, 302 to frontend
```

`oauth_states` is a **Postgres table**, not a cookie and not memory — the instance that starts the flow is not the one that finishes it. Rows are single-use (`consumed_at`) and expire after 10 minutes.

Redirect URI: `http://localhost:8080/api/v1/auth/google/callback`.

### Authorization

- `requireAuth` — valid access token or 401.
- `requireAdmin` — valid access token **and** `role == "admin"` or 403.
- The role is read from the **token**, which is signed. Never from a header or body the client controls.
- The proxy injects `X-User-Id` / `X-User-Email` / `X-User-Role` **after stripping any client-supplied copies of those headers.** A client that sends `X-User-Role: admin` must not get admin.

### Admin bootstrap

A migration seeds one admin from `ADMIN_EMAIL` / `ADMIN_PASSWORD` env vars (defaults `admin@netflix.local` / `admin12345` for local dev only, documented in the README as dev-only).

---

## 4. Database schema

Migrations live in `database/migrations/NNN_name.sql`, applied in order, tracked in a `schema_migrations` table. They must be **idempotent and additive** — never `DROP TABLE` an existing catalog table. The legacy `database/database.sql` stays as historical reference; `001_init.sql` supersedes it.

### New tables

```sql
users (
  id uuid pk default gen_random_uuid(),
  email citext not null unique,
  password_hash text,               -- null for OAuth-only accounts
  name text not null default '',
  role user_role not null default 'user',   -- enum: 'user' | 'admin'
  email_verified boolean not null default false,
  google_sub text unique,
  created_at, updated_at, deleted_at
)

sessions (
  id uuid pk,
  user_id uuid not null references users(id) on delete cascade,
  session_family uuid not null,     -- reuse detection revokes the family
  refresh_token_hash text not null unique,
  rotated_from uuid references sessions(id),
  user_agent text, ip inet,
  expires_at timestamptz not null,
  revoked_at timestamptz,
  created_at
)
-- index: (user_id), (refresh_token_hash), (session_family)

oauth_states (
  state text pk,
  code_verifier text not null,
  redirect_to text,
  expires_at timestamptz not null,
  consumed_at timestamptz
)

video_assets (
  id uuid pk,
  kind text not null,               -- 'movie' | 'episode'
  id_movie int references movies(id_movie),
  id_episode int references episodes(id_episode),
  status processing_status not null default 'pending',  -- pending|processing|ready|failed
  source_path text,                 -- storage key of the uploaded original
  manifest_path text,               -- storage key of master.m3u8
  duration_seconds numeric(10,3),
  source_width int, source_height int,
  source_fps numeric(6,3),
  qualities jsonb not null default '[]',   -- ["144p","240p",...]
  size_bytes bigint,
  error text,
  created_at, updated_at,
  constraint chk_asset_target check (num_nonnulls(id_movie, id_episode) = 1)
)

video_jobs (
  id uuid pk,
  asset_id uuid not null references video_assets(id) on delete cascade,
  status job_status not null default 'queued',  -- queued|leased|done|failed
  attempts int not null default 0,
  max_attempts int not null default 3,
  locked_by text, locked_at timestamptz,
  lease_expires_at timestamptz,
  last_error text,
  created_at, updated_at
)
-- index: (status, created_at) where status in ('queued','leased')

plans (id serial pk, code text unique, name text, price numeric(10,2), currency text default 'MXN',
       quality text, max_streams int, active bool default true)

discounts (
  id serial pk,
  code citext not null unique,
  kind discount_kind not null,      -- 'percent' | 'fixed'
  value numeric(10,2) not null,
  max_redemptions int,              -- null = unlimited
  redemption_count int not null default 0,
  per_user_limit int not null default 1,
  starts_at timestamptz, expires_at timestamptz,
  active boolean not null default true,
  created_at
)

discount_redemptions (
  id uuid pk, discount_id int not null references discounts(id),
  user_id uuid not null references users(id),
  payment_id uuid, created_at,
  unique (discount_id, user_id)     -- enforces per_user_limit = 1 at the DB level
)

payments (
  id uuid pk,
  user_id uuid not null references users(id),
  plan_id int not null references plans(id),
  method payment_method not null,   -- 'card' | 'oxxo'
  status payment_status not null,   -- 'pending' | 'paid' | 'expired' | 'failed'
  subtotal numeric(10,2) not null,
  discount_id int references discounts(id),
  discount_amount numeric(10,2) not null default 0,
  total numeric(10,2) not null,
  currency text not null default 'MXN',
  reference text unique,            -- OXXO barcode reference
  card_last4 text, card_brand text, -- NEVER the PAN, NEVER the CVV
  expires_at timestamptz, paid_at timestamptz,
  simulated boolean not null default true,
  created_at, updated_at
)

subscriptions (id uuid pk, user_id uuid unique references users(id), plan_id int references plans(id),
               status text, current_period_end timestamptz, created_at, updated_at)

profiles (id uuid pk, user_id uuid references users(id) on delete cascade,
          name text, avatar text, is_kids bool default false, created_at)
```

### Changes to existing tables

- `titles` gains `published boolean not null default false` — nothing reaches a normal user until it is both `published` **and** its asset is `ready`.
- `watch_progress.user_id` / `favorites.user_id` now carry a real FK to `users(id)`.
- `watch_progress` gains `unique (user_id, id_movie)` and `unique (user_id, id_episode)` so upserts are atomic across instances.
- Add the indexes the catalog actually reads by: `titles(type) where deleted_at is null`, `title_genres(id_genre)`, `episodes(id_season, episode_number)`.
- `user_rate_limits(user_id, action, window_start, request_count)` is the shared per-user counter for discount-preview and OXXO-simulation limits. Its composite primary key makes increments atomic across all user-service replicas.

---

## 5. Deterministic catalog seed and reset

The Go importer remains the source of the live seeded catalog. Compose runs it
in content-aware reset mode. If the catalog shape, seed-title inventory, or
stored source fingerprint differs, this single command removes and rebuilds
only catalog data, `video_assets`, `video_jobs`, and the exact `/media/hls` and
`/media/sources` trees, then queues the current `seed/video/video.mp4` for every
seeded movie and episode. When catalog and video are already current, it uses
the normal idempotent importer and preserves ready media, so restarting a
dependent worker cannot accidentally trigger another transcode:

```sh
docker compose up --build seed
```

The reset snapshots favorites and watch progress by stable movie or episode
identity and restores entries that still exist in the seed. Fixture-only
library entries naturally disappear. It does not delete or update users,
sessions, profiles, payments, subscriptions, discounts, discount redemptions,
or plans. Stop running workers before an operator-initiated reset and start
them afterward; normal full-stack startup already orders workers after seed.

`database/exec.sql` is the media-free, human-runnable equivalent for catalog
metadata. Regenerate and execute it with:

```sh
cd database/seed && go run . -generate-sql ../exec.sql
cd ../.. && docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U netflix -d netflix < database/exec.sql
```

The generated SQL is idempotent and contains titles, movies, series, seasons,
episodes, vocabulary rows, and all title metadata joins. Media assets and jobs
remain exclusively owned by the importer/pipeline.

---

## 6. Video pipeline

### Upload → ready

```
POST /api/v1/admin/movies/:id/video   (multipart, admin only)
  1. validate: extension, sniffed MIME, size cap (MAX_UPLOAD_BYTES, default 5 GiB)
  2. stream to /media/sources/<asset-uuid>/source.mp4   -- io.Copy, never ReadAll
  3. insert video_assets (status='pending') + video_jobs (status='queued')
  4. 202 Accepted { asset_id, status: "pending" }        -- returns immediately
```

The HTTP request **must not** wait for ffmpeg.

### Worker claim loop

```sql
UPDATE video_jobs SET status='leased', locked_by=$1, locked_at=now(),
       lease_expires_at=now() + interval '30 minutes', attempts=attempts+1
WHERE id = (
  SELECT id FROM video_jobs
  WHERE (status='queued')
     OR (status='leased' AND lease_expires_at < now())   -- reclaim dead workers
  ORDER BY created_at
  FOR UPDATE SKIP LOCKED
  LIMIT 1
)
RETURNING *;
```

`SKIP LOCKED` is what guarantees two workers never process the same job. A worker whose container dies has its lease expire and the job is reclaimed. On `attempts >= max_attempts` the job goes `failed` and the asset records `error`.

### Rendition ladder

Probe with ffprobe, then emit **only renditions at or below the source height** — never upscale.

| Name | Height | v-bitrate | maxrate | bufsize | audio |
|---|---|---|---|---|---|
| 144p | 144 | 200k | 214k | 400k | 64k |
| 240p | 240 | 400k | 428k | 800k | 64k |
| 360p | 360 | 800k | 856k | 1600k | 96k |
| 480p | 480 | 1400k | 1498k | 2800k | 128k |
| 720p | 720 | 2800k | 2996k | 5600k | 128k |
| 1080p | 1080 | 5000k | 5350k | 10000k | 192k |
| 1440p | 1440 | 8000k | 8560k | 16000k | 192k |

Rules:
- Width is derived from the source aspect ratio and **rounded to an even number** (`-vf scale=-2:H`). H.264 4:2:0 requires even dimensions.
- A 480p source produces 144/240/360/480 only. A 1080p source stops at 1080p. 1440p only when the source is ≥1440.
- If the source is smaller than 144p, emit a single rendition at the source height (still even-rounded).
- **Keyframes must align across renditions** or ABR switching breaks: `-g $(2*fps) -keyint_min $(2*fps) -sc_threshold 0 -force_key_frames "expr:gte(t,n_forced*6)"`.
- Codecs: `libx264` (`-profile:v main`, `-preset veryfast`) + `aac`. Audio-bearing sources explicitly map `0:a:0`, apply each profile's `-b:a`, and advertise `mp4a.40.2` in the master playlist. Silent sources use `-an`. Nothing exotic.
- Segments: `-hls_time 6 -hls_playlist_type vod -hls_segment_type mpegts`.

### Output layout on the shared volume

```
/media/hls/<asset-uuid>/
├── master.m3u8
├── 144p/playlist.m3u8 + seg_00000.ts ...
├── 360p/playlist.m3u8 + ...
└── 720p/playlist.m3u8 + ...
```

The master playlist must carry correct `BANDWIDTH`, `RESOLUTION` and `CODECS` per variant, or the player cannot pick a quality.

**Write to a temp dir and `rename()` into place** when the transcode succeeds, so `streaming` can never observe a half-written manifest. Only then flip `video_assets.status = 'ready'`.

### Streaming delivery

```
GET /api/v1/stream/:asset_id/master.m3u8
GET /api/v1/stream/:asset_id/:quality/playlist.m3u8
GET /api/v1/stream/:asset_id/:quality/:segment.ts
```

- `http.ServeContent` / `ServeFile` only — **never** `os.ReadFile` a video into memory. Range requests and seeking come free from `ServeContent`.
- Path safety: the `asset_id` must parse as a UUID, `quality` must match `^\d{3,4}p$`, `segment` must match `^seg_\d{5}\.ts$`. Reject everything else with 400. Never join user input into a path without validating it first — this is the path-traversal control.
- Cache headers: manifests `Cache-Control: public, max-age=10`; segments `public, max-age=31536000, immutable` (they never change once written).
- Content types: `application/vnd.apple.mpegurl` for `.m3u8`, `video/mp2t` for `.ts`.
- Refuse to serve an asset whose status is not `ready`.

---

## 7. API contract

All routes are behind nginx at `http://localhost:8080`. Every route below `/api/v1/auth/*` requires a valid access token.

### auth
```
POST   /api/v1/auth/signup      {name,email,password}     -> 201 {user}      + cookies
POST   /api/v1/auth/login       {email,password}          -> 200 {user}      + cookies
POST   /api/v1/auth/refresh                               -> 200 {user}      + rotated cookies
POST   /api/v1/auth/logout                                -> 204             + cleared cookies
GET    /api/v1/auth/me                                    -> 200 {id,name,email,role}
GET    /api/v1/auth/google                                -> 302
GET    /api/v1/auth/google/callback                       -> 302
```

### catalog (read: any user · write: admin)
```
GET    /api/v1/titles?type=&genre=&q=&limit=&offset=      -> published+ready only for non-admins
GET    /api/v1/titles/:id
GET    /api/v1/movies/:id
GET    /api/v1/series/:id                                  -> includes seasons[].episodes[]
GET    /api/v1/genres | /api/v1/categories | /api/v1/actors
GET    /api/v1/home                                        -> the browse rows the frontend renders

POST   /api/v1/admin/movies            PATCH/DELETE /api/v1/admin/movies/:id
POST   /api/v1/admin/series            PATCH/DELETE /api/v1/admin/series/:id
       create/PATCH body metadata: {genre_ids: int[], actor_ids: int[], category_ids: int[]}
       create -> 201, PATCH -> 200: {id,title_id,genre_ids,actor_ids,category_ids}
POST/PATCH/DELETE /api/v1/admin/genres[/:id]      body: {name}
POST/PATCH/DELETE /api/v1/admin/actors[/:id]      body: {name}
POST/PATCH/DELETE /api/v1/admin/categories[/:id]  body: {name}
POST   /api/v1/admin/series/:id/seasons
POST   /api/v1/admin/seasons/:id/episodes
POST   /api/v1/admin/movies/:id/video      (multipart) -> 202 {asset_id}
POST   /api/v1/admin/episodes/:id/video    (multipart) -> 202 {asset_id}
POST   /api/v1/admin/titles/:id/thumbnail  (multipart)
GET    /api/v1/admin/assets/:id            -> {status, qualities, error, progress}
POST   /api/v1/admin/titles/:id/publish    {published: bool}
```

Metadata arrays use replacement semantics when present (including an explicit
empty array). An omitted array is preserved on PATCH. Movie and series creates
default `category_ids` to the `Movies` or `Series` category when it is omitted.
Duplicate IDs are normalized; a missing, deleted, zero, or negative reference
returns `400 {"error":"one or more metadata IDs do not exist"}` and rolls back
the entire title mutation.

### user
```
GET/PUT    /api/v1/progress/:kind/:id     kind = movie|episode
GET        /api/v1/progress/continue
GET/POST/DELETE /api/v1/favorites[/:title_id]
GET/POST/PATCH/DELETE /api/v1/profiles[/:id]
GET    /api/v1/plans
POST   /api/v1/discounts/validate   {code, plan_id}  -> {valid, discount_amount, total}  (preview only, never authoritative)
POST   /api/v1/payments/card        {plan_id, code?, card{number,exp,cvv,name}}
POST   /api/v1/payments/oxxo        {plan_id, code?}   -> {reference, amount, expires_at}
POST   /api/v1/payments/oxxo/:ref/simulate-payment      -> marks paid (dev-only, clearly separated)
GET    /api/v1/payments/:id
```

**Pricing rule:** the request carries `plan_id` and optionally `code`. It never carries a price, subtotal, discount or total — those are looked up and computed server-side and any client-sent value is ignored. `/discounts/validate` exists only so the UI can show a preview; the real numbers are recomputed at payment time inside the same transaction that increments `redemption_count`.

The user service additionally limits discount validation to 20 requests per
user per minute and OXXO payment simulation to 5 requests per user per minute.
Counters are in PostgreSQL, so limits are consistent across replicas. A denied
request returns `429 {"error":"rate limit exceeded"}` with `Retry-After`.

---

## 8. Frontend

- Delete `utils/supabase/*` and both `@supabase/*` dependencies.
- `utils/api/client.ts` — a single fetch wrapper: `credentials: "include"`, JSON, and on a 401 it calls `/auth/refresh` once and retries the original request (guard against a refresh loop).
- `middleware.ts` at the frontend root protects `/home/*`, `/admin/*` and redirects `/login`+`/signup` when already authenticated. It reads the access-token cookie; it must not attempt signature verification with a secret the browser bundle could see.
- `AuthProvider` context exposes `{user, loading, login, signup, logout}`.
- Player: **hls.js** with native HLS fallback for Safari/iOS (`canPlayType('application/vnd.apple.mpegurl')`). Auto quality by default, manual override in the UI, `ERROR` handler that recovers from `NETWORK_ERROR` / `MEDIA_ERROR` instead of dying.
- Admin at `/admin`: titles list with status pills `Uploading → Processing → Ready | Failed`, create/edit forms, upload with progress, and polling of `/admin/assets/:id` while a job is in flight.
- Replace every hard-coded `api_data_example` array and every `occ-0-7553-114.1.nflxso.net` remote image with real API data and locally served artwork. Drop that host from `next.config.ts` once nothing references it.
- Responsive breakpoints: 375 / 768 / 1024 / 1440. Nothing may use a fixed `w-[900px]` or `w-[30%]` that breaks on mobile — the title modal and the payment column are the two current offenders.

---

## 9. Environment variables

Single root `.env` (git-ignored) consumed by compose; `.env.example` **is** committed with placeholder values.

```
POSTGRES_USER / POSTGRES_PASSWORD / POSTGRES_DB
DATABASE_URL=postgres://user:pass@postgres:5432/netflix?sslmode=disable
JWT_SECRET                 # >= 32 bytes, shared by every auth instance
ACCESS_TOKEN_TTL=15m
REFRESH_TOKEN_TTL=720h
GOOGLE_CLIENT_ID / GOOGLE_CLIENT_SECRET / GOOGLE_REDIRECT_URL
FRONTEND_URL=http://localhost:3000
CORS_ALLOWED_ORIGINS=http://localhost:3000
CATALOG_SERVICE_URL / STREAMING_SERVICE_URL / USER_SERVICE_URL
MEDIA_ROOT=/media
MAX_UPLOAD_BYTES=5368709120
ADMIN_EMAIL / ADMIN_PASSWORD
NEXT_PUBLIC_API_URL=http://localhost:8080
```

Never expose a secret to the browser. Only `NEXT_PUBLIC_*` is allowed to reach the bundle, and none of those may be a credential.

---

## 10. Definition of done per milestone

| M | Milestone | Owner | Status |
|---|---|---|---|
| M1 | Postgres + migrations + seed importer | Codex 1 | DONE — SQL in `database/migrations`, runner in `shared/migrate`, importer in `database/seed` |
| M2 | Auth rewrite: signup/login/logout/refresh/RBAC, Supabase gone | Codex 1 | DONE — bcrypt(12), JWT access + rotating refresh with reuse detection, RBAC; Supabase removed |
| M3 | Google OAuth end to end | Codex 1 | DONE (backend) — PKCE S256, single-use state. Google consent screen is unverified; publishing it is the owner's step |
| M4 | Docker: root compose, all Dockerfiles, nginx, volumes, healthchecks | Codex 3 | DONE — replica-based compose, Docker-DNS resolver in nginx, healthchecks, `FRONTEND_PORT` |
| M5 | catalog service + admin CRUD + upload intake | Codex 1 | DONE — public reads + 20 admin routes + upload intake (202, non-blocking) |
| M6 | worker: ffprobe, ladder, ffmpeg, HLS, SKIP LOCKED queue | Codex 1 | DONE — ffprobe, no-upscale ladder, aligned keyframes, `SKIP LOCKED` + lease fencing |
| M7 | streaming service: manifests, segments, ranges, cache headers | Codex 1 | DONE — manifests, segments, ranges, immutable caching, symlink containment, published+ready gating |
| M8 | user service: progress, favorites, profiles | Codex 1 | DONE — progress, favorites, profiles (capped at 5, names 1–50) |
| M9 | payments: plans, discounts, card, OXXO simulation | Codex 1 | DONE — plans, backend-only totals, discounts with race-safe redemption, card + OXXO simulation |
| M10 | frontend de-Supabase: API client, auth pages, middleware, AuthProvider | Codex 3 | DONE — cookie API client with single-flight refresh, auth pages, AuthProvider; middleware validates via `/auth/me` and fails closed |
| M11 | frontend catalog wired to real API, artwork served locally | Codex 3 | DONE — catalog wired to the real API, artwork served from the media volume |
| M12 | hls.js player: ABR, seek, quality menu, error recovery | Codex 3 | DONE — hls.js ABR, seek, quality menu, bounded error recovery |
| M13 | admin UI: CRUD, upload progress, status pills | Codex 3 | IN PROGRESS — CRUD, upload progress and status pills done; admin entry point and genre/cast editing outstanding (Codex 3 Phase 7) |
| M14 | payments UI wired to backend totals | Codex 3 | DONE — plans and totals from the backend; the client never sends money |
| M15 | responsive pass at 375/768/1024/1440 | Codex 3 | DONE — 375/768/1024/1440 verified on public, authenticated and admin pages; no horizontal overflow |
| M16 | automated tests (Go unit + integration) | Codex 1 + 3 | DONE — unit tests race-clean in every module; tagged integration suite green twice and under `-shuffle=on` |
| M17 | security + QA review loop | Codex 2 | DONE (round 3) — no critical/high; 6 medium + 1 low, all fixed. Docs audit performed separately |
| M18 | horizontal-scaling verification | lead | DONE — replica replacement under a new IP, per-tier recreation, scale 3→5→3 with nginx untouched |
| M19 | Chrome MCP manual verification | lead | DONE — login, browse, playback, seek, quality menu and responsive widths verified in Chrome; admin walkthrough with Codex 3 |
| M20 | CLAUDE.md rewritten to match reality | lead | DONE — root and frontend `CLAUDE.md` rewritten and fact-checked against the code by Codex 2 |

---

## 11. Hard rules

1. Never load a video into memory. `io.Copy` and `http.ServeContent`, always.
2. Never trust a price, total or discount from the client.
3. Never trust `X-User-*` from the client — strip and re-inject at the proxy.
4. Never store a raw refresh token, a PAN or a CVV.
5. Never `SELECT ... + fmt.Sprintf`. Parameterized queries only.
6. Never commit generated media. `/media`, `*.ts`, `*.m3u8` are git-ignored.
7. Never make one instance special. If it breaks with 3 replicas, it is wrong.
8. Never claim something works without running it.
