# Codex 3 — Phase 9: page metadata across the app, and the admin metadata UI

Your episode-row fix is accepted — I measured it in a real browser at 375 / 768 / 1024 / 1440 and your calculated geometry was exactly right (content 291 / 424 / 600 / 600 px, no overflow anywhere).

Two parts. Part A is a direct request from the project owner.

---

## Part A — Every page needs proper metadata

The owner asked to "update the metadata for all pages". Today there is essentially none:

```
$ grep -n "metadata" frontend/app/layout.tsx
(no matches — the root layout exports no metadata at all)

$ grep -rln "export const metadata|generateMetadata" frontend/app/
app/signup/layout.tsx          ← 1 of 27 pages

$ find frontend/app -name page.tsx | wc -l
27
```

The consequence is visible in any browser: tabs read `localhost:3000/login` and `localhost:3000/home/browse` instead of real titles, because there is no `<title>` to use.

Add Next.js metadata across the app:

- **Root layout**: a `metadata` export with a `title.default` and `title.template` (so pages compose as `Sign In · Netflix Clone` without repeating the suffix), a real `description`, and the icon. Set `metadataBase` so relative URLs resolve.
- **Every route**: a title and description that describe *that* page. Landing, login, password help, each signup step (plan, payment, card, OXXO, gift code, verify email), home, browse, movies, series, new arrivals, my list, manage profiles, settings, watch, and every admin page.
- **Dynamic routes** (`watch/[assetId]`, `admin/movies/[id]`, `admin/series/[id]`) need `generateMetadata` so the title reflects the actual title being watched or edited — "Watching Scarface" beats "Watch". Handle the not-found and still-loading cases without throwing, and do not make the metadata call a second round trip when the page already fetches the same data.
- **Do not leak private data into metadata.** Admin pages and the watch page are behind auth; titles are fine, but no user emails, no ids that are not already in the URL, and no `robots: index` on authenticated routes — mark those `robots: { index: false }`.
- Open Graph and Twitter cards on the **public** pages only (landing, login, signup) — that is where a shared link actually matters.

Keep it maintainable: if you find yourself pasting the same object 27 times, factor a small helper. And note that client components (`"use client"`) cannot export `metadata` — several of these pages are client components, so the metadata belongs in a colocated `layout.tsx` or the page must be split. Say which approach you took.

## Part B — Admin metadata UI (genres, cast, categories)

Finish what Phase 7 Part C started, now that the backend contract is settled and I have verified it live:

- **Omitted** array → server leaves that association unchanged.
- **`[]`** → clears it.
- **Non-empty** → replaces it.

I confirmed all three against the running API, including that clearing genres leaves cast untouched.

- Wire the genre and cast selectors on the movie and series editors to that contract. Never send an array you did not intend to change — that is the whole point of the tri-state.
- **Categories**: you noted you left them out because title reads do not expose category assignments. Check again — Codex 1 has been extending the catalog read side, and its PATCH response returns `category_ids`. If reads now expose categories, add the selector. If they still do not, tell me and I will task Codex 1 rather than you guessing at it.
- The vocabulary endpoints (`/api/v1/admin/{genres,actors,categories}`) now have full CRUD. Decide whether creating a new genre/actor inline from the editor is worth it, or whether picking from the existing vocabulary is enough for this panel. Your call — say what you chose and why.
- Verify in a real browser as an admin: assign genres and cast to a title, save, reload, confirm they persisted, then change only the description and confirm the assignments survive.

---

## Definition of done

- `pnpm build`, `pnpm lint`, `pnpm exec tsc --noEmit` clean.
- Tab titles correct on a sample of public, authenticated and admin routes — checked in a browser, not inferred from source.
- Authenticated routes marked non-indexable.
- The metadata round-trip in Part B proven by reload, including the "change one field, assignments survive" case.

Heads-up: Codex 1 is re-seeding the catalog from a new 171-second source video, so titles and asset ids will change under you and the transcode queue will be busy. Do not treat a title that vanished mid-test as a bug without re-checking.

Do not commit. Work autonomously; do not stop to ask for confirmation.
