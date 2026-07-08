# Netflix Clone — Frontend

Next.js 16 (App Router) frontend for the Netflix clone. This is the single entry point for the app; its middleware will also host the load-balancer logic that proxies to the Go microservices. For full-system context see `../CLAUDE.md`.

**Author:** Fernando Vazquez / [@fernandovmedina](https://github.com/fernandovmedina)

---

## Stack

- **Framework:** Next.js 16 (App Router), React 19.2
- **Language:** TypeScript (strict), `@/*` path alias → project root
- **Styling:** Tailwind CSS v4 (via `@tailwindcss/postcss`), global styles in `app/globals.css`
- **UI libs:** MUI v9 (`@mui/material` + Emotion), `@deemlol/next-icons`
- **Auth:** `@supabase/ssr` + `@supabase/supabase-js`
- **Font:** Roboto via `next/font/google`
- **Package manager:** pnpm

---

## Commands

```bash
pnpm install       # install dependencies
pnpm dev           # dev server on http://localhost:3000
pnpm build         # production build
pnpm start         # serve production build
pnpm lint          # eslint (flat config, eslint-config-next)
```

---

## Environment Variables

```
NEXT_PUBLIC_SUPABASE_URL
NEXT_PUBLIC_SUPABASE_PUBLISHABLE_KEY
```

Both are read by all three Supabase client factories in `utils/supabase/`. Set them in `.env.local` (git-ignored).

---

## Directory Structure

```
frontend/
├── app/
│   ├── layout.tsx                    # Root layout ("use client"); Roboto font +
│   │                                 #   client-side session check that redirects
│   │                                 #   authed users from / and /login to /home/browse
│   ├── page.tsx                      # Landing (email capture → localStorage → signup)
│   ├── login/page.tsx                # Sign-in (signInWithPassword)
│   ├── loginhelp/page.tsx            # Forgot password
│   ├── signup/                       # Multi-step sign-up flow
│   │   ├── layout.tsx
│   │   ├── page.tsx
│   │   ├── linkRegistration/         #   email → account link
│   │   ├── linkSent/
│   │   ├── verifyEmail/
│   │   ├── regform/                  #   registration form
│   │   ├── planform/                 #   plan selection
│   │   └── payment/                  #   payment method selection
│   │       ├── page.tsx
│   │       ├── card/
│   │       ├── gift_code/
│   │       └── oxxo/
│   └── home/                         # Authenticated area
│       ├── layout.tsx
│       ├── page.tsx
│       ├── browse/                   #   main browse (carousels + title modal)
│       ├── movies/
│       ├── series/
│       ├── my_list/                  #   favorites
│       ├── new_arrivals/
│       ├── ManageProfiles/
│       └── settings/[uuid]/          #   per-profile settings
├── components/
│   ├── Navbar.tsx
│   └── AlertMessage.tsx
├── utils/supabase/
│   ├── client.ts                     # createBrowserClient (browser)
│   ├── server.ts                     # createServerClient (Server Components / RSC)
│   └── middleware.ts                 # createServerClient for Next middleware; refreshes
│                                     #   session cookies — LOAD BALANCER ENTRY POINT
├── public/                           # static assets (netflix.png favicon, imagery)
├── next.config.ts                    # allows remote images from occ-0-7553-114.1.nflxso.net
├── eslint.config.mjs
├── postcss.config.mjs
└── tsconfig.json
```

---

## Auth Flow

1. **Landing (`/`)** captures email → stores in `localStorage` as `signup_email` → routes to `/signup/linkRegistration`.
2. **Login (`/login`)** calls `supabase.auth.signInWithPassword` client-side, then redirects to `/home/browse`.
3. **Root layout** (`app/layout.tsx`) runs a client-side `getSession()` check on mount and redirects already-authenticated users away from `/` and `/login` to `/home/browse`.
4. **Middleware** (`utils/supabase/middleware.ts`) wraps `createServerClient` to refresh session cookies on requests. This is where the planned load-balancer / route-protection logic will live (validate JWT, proxy to auth/catalog/streaming/user microservices, 401 on unauthenticated protected routes).

### Supabase client rule of thumb
- Browser / client components → `utils/supabase/client.ts`
- Server components → `utils/supabase/server.ts` (pass in the `cookies()` store)
- Middleware → `utils/supabase/middleware.ts`

---

## Conventions

- Path alias `@/*` maps to the frontend root (e.g. `@/utils/supabase/client`).
- Route directories are mostly camelCase/lower_snake; `ManageProfiles` is PascalCase (existing inconsistency — match the neighboring route when adding files).
- Remote images must have their host whitelisted in `next.config.ts` `images.remotePatterns`.
- `TODO.txt` tracks ad-hoc frontend tasks.
