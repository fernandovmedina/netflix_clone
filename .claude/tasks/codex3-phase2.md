# Codex 3 — Phase 2 (M10 + M11: frontend off Supabase, wired to the real API)

Repo root: `/Users/froot/Documents/workspace/fernandovmedina/web/netflix_clone`

## Read first

1. `docs/ARCHITECTURE.md` §3 (cookies/tokens), §6 (API contract), §7 (frontend) — binding.
2. Every file under `frontend/app/**` and `frontend/utils/**`. Understand the existing flows before you rewrite them — the landing page → signup → plan → payment chain has real business logic in it that must survive.
3. `frontend/components/Navbar.tsx`, `frontend/components/AlertMessage.tsx`.

## Files you own

```
frontend/**
```

Codex 1 owns all Go and SQL. Do not edit `microservices/**` or `database/**`.

---

## M10 — Remove Supabase, own the session

### Delete
- `frontend/utils/supabase/client.ts`, `server.ts`, `middleware.ts` — the whole directory.
- `@supabase/ssr` and `@supabase/supabase-js` from `package.json`, then refresh `pnpm-lock.yaml`.
- `NEXT_PUBLIC_SUPABASE_URL` / `NEXT_PUBLIC_SUPABASE_PUBLISHABLE_KEY` from `frontend/.env.local`.

After this, `grep -ri supabase frontend/app frontend/utils frontend/components` must return nothing.

### Build
- **`utils/api/client.ts`** — one fetch wrapper used by everything:
  - base URL from `NEXT_PUBLIC_API_URL` (default `http://localhost:8080`)
  - `credentials: "include"` on every request (cookies are HttpOnly; JS cannot and must not read the token)
  - on `401`: call `POST /api/v1/auth/refresh` **once**, then retry the original request. Guard the loop — a failed refresh must reject, not recurse. De-duplicate concurrent refreshes so ten parallel 401s trigger one refresh, not ten.
  - typed helpers per §6, replacing `utils/api/auth.ts`.
- **`middleware.ts`** at the frontend root:
  - protect `/home/*` and `/admin/*` — no access-token cookie → redirect to `/login?next=<path>`
  - bounce authenticated users away from `/login` and `/signup`
  - it can only check cookie **presence**, not validity — the server is the authority. Never put `JWT_SECRET` anywhere near the frontend.
- **`AuthProvider`** (`app/providers.tsx` or `components/AuthProvider.tsx`): context exposing `{user, loading, login, signup, logout, refresh}`, hydrated from `GET /api/v1/auth/me`. Replaces the `useEffect` + `supabase.auth.getSession()` pattern in `app/home/layout.tsx`.
- Rewrite `app/login/page.tsx` to call `POST /api/v1/auth/login`. Keep the existing UI, the "use a sign-in code" toggle and the `AlertMessage` error surface — only the auth call changes.
- Rewrite the signup chain (`app/signup/**`) to hit `POST /api/v1/auth/signup`. Preserve the existing 4-step flow and the `localStorage` `signup_email` handoff from the landing page.
- **Google sign-in button** on both `/login` and `/signup`: a plain link to `${NEXT_PUBLIC_API_URL}/api/v1/auth/google`. No client-side OAuth SDK — the backend owns the flow and sets the cookies, then redirects back.
- `app/home/layout.tsx` uses `AuthProvider` instead of its Supabase session check.

## M11 — Real catalog data

- `app/home/page.tsx` (and its four near-identical siblings `movies/`, `series/`, `my_list/`, `new_arrivals/`) currently each carry ~680 lines with a hard-coded `api_data_example`. **Extract the shared carousel/hero/title-modal into reusable components** (`components/Carousel.tsx`, `components/TitleModal.tsx`, `components/Hero.tsx`) and have all five pages render from `GET /api/v1/home` (and the filtered variants). This removes ~2500 lines of duplication — do it properly, do not copy-paste a sixth time.
- Delete every `occ-0-7553-114.1.nflxso.net` URL. Artwork now comes from `GET /api/v1/stream/thumbnails/:file` via `titles.thumbnail_url`. Once nothing references that host, remove it from `next.config.ts` `remotePatterns` and add the local API host instead.
- Title modal: real title data, real episode list per season, season selector driven by the API.
- Wire `my_list` to `/api/v1/favorites` and the "Continue watching" row to `/api/v1/progress/continue`.
- Only `ready` + `published` titles come back from the API — but still handle a title whose asset is not ready by disabling its play button rather than rendering a broken player.

## Quality bar

- No `any` where a real type is available — the API shapes are specified in §6, type them.
- Loading and error states on every page that fetches. A failed request must not render an empty page with no explanation.
- `pnpm build` must pass with no type errors. `pnpm lint` clean.
- Do not regress the visual design. This is a Netflix clone; it should still look like one. Match the existing Tailwind idiom in the surrounding files.

## Verification

- `pnpm build` clean.
- With the stack up (`docker compose up -d`), manually confirm: signup → redirected to home → home shows real seeded titles with real artwork → logout → `/home` redirects to login.
- Confirm the refresh flow: delete only the `access_token` cookie in devtools, reload, and the app must silently recover via `/auth/refresh` rather than logging you out.

## Report back

Concise report: files added/changed/deleted, lines removed by de-duplication, what you verified in the browser, and anything Codex 1 needs to change in the API contract. Do not commit.

Work autonomously. Do not stop to ask for confirmation.
