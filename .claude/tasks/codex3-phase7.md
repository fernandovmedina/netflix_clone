# Codex 3 — Phase 7: the admin panel is unreachable from the UI

Round 5 accepted — the middleware now validates through the auth API, fails closed on an API outage, and role-gates `/admin` correctly. Two follow-ups, the first urgent.

---

## Part A — An admin has no way to reach the admin panel (urgent)

The project owner logged in with their admin account and **could not find the admin panel**. Here is why:

```
$ grep -oE 'href="[^"]*"' frontend/components/Navbar.tsx | sort -u
href="/home/ManageProfiles"
href="/loginhelp"

$ grep -n "admin|role" frontend/components/Navbar.tsx
(no matches)
```

`/admin` exists and works, but nothing anywhere in the authenticated UI links to it. The only way in is to type the URL. That is the whole bug — the panel was never missing, it was invisible.

Fix: surface an admin entry point for users whose session has `role === "admin"`, and only for them. `GET /api/v1/auth/me` already returns `role`, and `AuthProvider` already holds the session, so no new API is needed.

- Put it where an admin will actually look — the profile/account dropdown in the navbar is the conventional spot, and a top-level nav item is defensible too. Your call, but it must be obvious after login, and it must be present on mobile, not only on the desktop bar.
- It must **not** render for normal users. Verify with both account types.
- While you are in there: from `/admin`, is there a clear way back to the normal browsing app? Check that the admin layout is not a dead end.

## Part B — Verify the whole admin panel as an admin, in the browser

You have `ADMIN_EMAIL`/`ADMIN_PASSWORD` in `/.env`; I do not, so this verification is yours. The owner's complaint was "I could not see the admin panel with all I asked", so check the panel against what `INTEGRATION.md` actually specifies under ADMIN CONTENT MANAGEMENT, and report honestly on each line:

**Movies** — metadata · images/posters · video upload · processing status · available streaming qualities · publishing state
**Series** — series metadata · seasons · episodes · episode video uploads · processing status · available qualities · publishing state

Walk each one in a real browser as a real admin: create a movie, upload a poster and a video, watch it go `Uploading → Processing → Ready`, confirm the available qualities are shown once ready, publish it, then confirm it appears for a normal user. Do the same for a series with a season and an episode. Report what works, what is missing, and what is present but confusing.

## Part C — Metadata editing UI (genres and cast)

The admin panel cannot set genres or cast today, and the owner wants the panel to manage the metadata the schema models. The seeded catalog has 58 `title_genres` and 60 `title_actors` rows, and I confirmed these now come back on the read side — but nothing in the UI assigns them.

**Codex 1 is adding the assignment endpoints right now** (Phase 6 Part C) and will report the exact request/response shapes. Build the UI against those shapes once they land — do not invent your own contract, and do not modify the Go services. If you reach this part before Codex 1 reports, do Parts A and B first and tell me.

---

## Definition of done

- `pnpm build`, `pnpm lint`, `pnpm exec tsc --noEmit` clean.
- The admin entry point verified visible for an admin and absent for a normal user, on desktop **and** at 375px.
- A per-line report against the INTEGRATION.md admin checklist above, based on real browser walkthroughs rather than reading the code.
- Anything you find missing either fixed or reported with a clear owner.

Heads-up: the frontend container on port 3000 is currently a stale pre-round-5 image — I am rebuilding it, so do not be surprised if it restarts under you. Use your own `FRONTEND_PORT` instance for verification as you did before.

Do not commit. Work autonomously; do not stop to ask for confirmation.
