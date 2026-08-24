# Netflix Clone — Frontend

Next.js 16 (App Router) frontend. It talks to the Go backend through the nginx load balancer at `http://localhost:8080` — it does **not** talk to any database, and it is not a load balancer itself. For full-system context see `../CLAUDE.md`.

**Supabase is gone.** There is no `utils/supabase/`, no `@supabase/*` dependency, and no `NEXT_PUBLIC_SUPABASE_*` variable. Authentication is this project's own cookie-based session against the auth service.

**Author:** Fernando Vazquez / [@fernandovmedina](https://github.com/fernandovmedina)

---

## Stack

- **Framework:** Next.js 16 (App Router), React 19.2
- **Language:** TypeScript (strict), `@/*` path alias → project root
- **Styling:** Tailwind CSS v4 (`@tailwindcss/postcss`), global styles in `app/globals.css`
- **UI libs:** MUI v9 (`@mui/material` + Emotion), `@deemlol/next-icons`
- **Video:** `hls.js`
- **Font:** Roboto via `next/font/google`
- **Package manager:** pnpm

## Commands

```bash
pnpm install
pnpm dev                  # http://localhost:3000 — conflicts with the frontend container on the same port
pnpm build
pnpm start
pnpm lint
pnpm exec tsc --noEmit
```

## Environment

```
NEXT_PUBLIC_API_URL       # e.g. http://localhost:8080 — baked in at build time
```

That is the only variable the browser bundle may contain. Backend secrets never appear here.

---

## Layout

```
app/
├── page.tsx                     landing (email capture → signup)
├── login/  loginhelp/           sign in, password help
├── signup/                      regform → planform → payment → verifyEmail/linkSent
│   └── payment/{card,oxxo,gift_code}
├── home/                        authenticated area
│   ├── page.tsx  browse/        profile picker, hero + carousels
│   ├── movies/ series/ new_arrivals/ my_list/
│   ├── ManageProfiles/  settings/[uuid]/
├── watch/[assetId]/             player route (?kind=movie|episode&id=&title=)
└── admin/                       dashboard, movies/new, movies/[id], series/new, series/[id]

components/
├── Navbar · SearchBox · Hero · Carousel · CatalogPage · TitleModal · AlertMessage
├── AuthProvider                 session context
├── VideoPlayer                  hls.js ABR player
├── admin/{AdminShell, AdminTitleEditor, AdminTitlePreview, UploadPanel, StatusPill}
└── payments/{PaymentShell, DiscountField, PaymentBreakdown}

utils/
├── api/client.ts                the only place that talks to the backend
└── payments.ts                  integer-cent formatting helpers
```

---

## API client (`utils/api/client.ts`)

- Every request sends `credentials: "include"`. **No token is ever stored in JavaScript** — the access and refresh tokens are HttpOnly cookies set by the auth service. (The backend also accepts `Authorization: Bearer`, but the frontend does not use it.)
- On a 401 the client performs a **single-flight** refresh — concurrent callers share one in-flight refresh promise — and then retries the original request **once**. A persistent 401 cannot loop.
- It also refreshes the session in the background every 10 minutes so a long viewing session does not expire mid-playback.
- `ApiError` carries the HTTP status, so callers can distinguish a real failure from an expected 404 (for example "no watch progress yet").

## Route protection (`middleware.ts`)

Matches `/home/:path*`, `/admin/:path*`, `/watch/:path*`, `/login`, `/signup`. Unauthenticated users hitting a protected route are redirected to `/login?next=<path>`; authenticated users hitting `/login` or `/signup` are sent to `/home`.

The middleware is a **UX guard, not a security boundary** — authorization is enforced by the backend on every request, which is what actually protects data. Treat any client-side check here as advisory.

> Next.js 16 warns that the `middleware.ts` convention is deprecated in favour of `proxy`. Migrating is not urgent, but it is coming.

## Player (`components/VideoPlayer.tsx`)

- Loads `${NEXT_PUBLIC_API_URL}/api/v1/stream/<assetId>/master.m3u8` with `xhrSetup` setting `withCredentials`, so segment requests carry the session cookie.
- **hls.js is preferred wherever MSE exists.** The native `<video>` HLS path is only a fallback for engines without MSE (iOS Safari) — Chrome reports `canPlayType("application/vnd.apple.mpegurl")` as `"maybe"` but gives no level information, so preferring it would cost the quality menu and ABR control.
- Register the `MEDIA_ATTACHED` listener **before** calling `attachMedia()`; hls.js can emit it synchronously, and a listener added afterwards misses it, leaving the source never loaded.
- Quality menu is built from `hls.levels` (`Auto` plus each rendition). `capLevelToPlayerSize` is on.
- Error recovery is bounded: up to two network recoveries (each refreshing the session first, then `startLoad()`) and two media recoveries via `recoverMediaError()`. Beyond that it surfaces an error instead of retrying forever. A successful fragment load resets the counters.
- Resumes from saved watch progress when it is within 95% of the duration, and persists progress every 10 seconds.

## The "no video yet" state

Seed video is not committed, so a fresh clone browses a catalog where **no title
has a video**. The public catalog returns those titles anyway — artwork and
metadata with `asset_id: null` — and the client is responsible for saying so.

`playbackState(item)` in `utils/api/client.ts` collapses `asset_id` and
`asset_status` into one of `ready` / `processing` / `failed` / `missing`, and
`playbackLabel` / `playbackHint` give the copy. Use those rather than
re-deriving the state: `Carousel` badges the artwork, `Hero` and `TitleModal`
label the disabled play button and explain why, and `VideoPlayer` treats a
403/404 on the manifest as "no video yet" rather than a transient network fault
worth retrying.

Publishing is deliberately **not** gated on a ready asset — an admin can publish
a metadata-only title and it appears in browse marked "No video yet". Playback
stays gated in the backend: streaming serves an asset only when it is `ready`
and its title is published.

## Browse search (`components/SearchBox.tsx`)

The navbar search icon expands into an input that queries `GET /api/v1/titles?q=`
after a 300 ms debounce. **It matches on the title name only** — the backend
`q` filter is a case-insensitive `ilike` against `titles.title`, nothing else.
Results open the same `TitleModal` the carousels use, so the box works on every
`/home/*` page without those pages holding any search state. A normal user only
ever sees published titles whose asset is `ready`, because that is what the
public catalog projection returns.

## Admin CRUD (`app/admin/page.tsx`)

Each dashboard row carries icon actions — view (`Eye`), edit (`Edit2`) and
delete (`Trash2`) — with delete behind a confirmation dialog. View opens
`AdminTitlePreview`, a read-only record of the full title including its season
and episode ladder. Delete needs the **entity** id, not the title id:
`movie_id` for a movie and `series_id` for a series, both of which the catalog
list projection returns. The series editor mirrors this for seasons and
episodes, and every episode gets its own thumbnail upload alongside its video.

## Payments UI

The client sends `plan_id` and an optional discount `code` — **never a price, subtotal, discount or total**. Plans come from `GET /api/v1/plans`; the discount field previews via `POST /api/v1/discounts/validate`, but the authoritative numbers always come from the payment response. Money is integer cents; format for display and never do float arithmetic on it. Card details are never persisted to `localStorage`.

## Conventions

- Responsive down to 375px. No page may scroll horizontally at 375 / 768 / 1024 / 1440 — the most common regression. Wide content (tables, carousels) scrolls inside its own container.
- Inputs at least 16px, or iOS Safari zooms on focus.
- Artwork and manifests are served by the backend under `/api/v1/stream/...`; use the helper rather than hand-building URLs.
