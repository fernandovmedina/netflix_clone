# Codex 3 — Round 4: admin responsive pass, containerized frontend, DNS edge

Your nginx replica/resolver rework and its five proofs are accepted — good work, M18 is signed off. Scope for this round: `frontend/**`, `docker-compose.yaml`, `microservices/nginx/nginx.conf`. Do not touch the Go services.

---

## Part A — Admin UI responsive pass (the gap in M15)

I verified the responsive pass myself at 375 / 768 / 1024 / 1440 on: landing, login, signup plan form, signup payment, home/browse, the title modal and the video player. All clean, no horizontal overflow at any width, mobile nav collapses correctly, the title modal is a proper full-screen sheet on mobile.

**I could not check the admin pages** — my session is a normal user and the harness blocked me from promoting it. You have `ADMIN_EMAIL`/`ADMIN_PASSWORD` in `/.env`, so this part is yours:

- Admin dashboard, the content list/tables, the create/edit forms for movies, series, seasons and episodes, and the upload panel with its `Uploading → Processing → Ready` status pills.
- All four widths. The specific things that break in admin layouts: tables forcing horizontal *page* scroll instead of scrolling inside their own container, form rows that collapse to unusable slivers, modals that cannot be dismissed on mobile, and status pills wrapping badly.
- Assert no horizontal page scroll at any width — that is the one regression that must not survive.

A quick way to test true mobile widths despite Chrome's 500 px minimum window: load a page inside a same-origin iframe of the target width and read `document.documentElement.scrollWidth` vs `clientWidth` — media queries respond to the iframe width. I left a harness at `frontend/public/rwd.html` (`?w=375&path=/admin`); **delete it when you are done**, along with `frontend/public/hlstest.html` and `frontend/public/hls.min.js`, which are my scratch files.

## Part B — The frontend container has never actually run

`docker compose ps` shows no `frontend` service because port 3000 is held by the user's `next dev` server, so the container exits on a port bind conflict. That means **DoD item 1 — "the complete application starts locally through Docker" — is unproven for the frontend tier.**

Verify the containerized frontend genuinely works, without disturbing the user's dev server on 3000:

- Build the image and run it bound to a free port (3001 is fine) against the same stack.
- Confirm the production build serves the landing page, login, browse, the player and admin — not just that the container reports healthy. A Next.js container can pass a healthcheck while every authenticated route 500s, so exercise real pages.
- Confirm `NEXT_PUBLIC_API_URL` is baked correctly at build time and the app talks to `localhost:8080` from the browser.
- Report whether the compose `frontend` service works as written once port 3000 is free. If the fixed `ports: ["3000:3000"]` mapping is the only obstacle, make the host port configurable (e.g. `${FRONTEND_PORT:-3000}:3000`) and document it — do not silently change the default.

## Part C — The transient Docker-DNS NXDOMAIN you observed

You reported one 502 after an nginx restart, caused by Docker DNS returning "Host not found for user", which then recovered. Decide whether that is worth hardening and act on your judgement:

- Establish whether it only occurs when nginx starts before the replicas are resolvable, or whether it can also hit a healthy steady-state stack.
- If it is a cold-start artifact, a `depends_on` condition or a resolver tweak may remove it. If it can hit steady state, a user-visible 502 during a routine restart is not acceptable and needs a retry path.
- Whatever you conclude, write it down in `docs/ARCHITECTURE.md` next to the resolver note, including the case where it is genuinely harmless — the next person will hit this and deserves the answer.

## Part D — Cosmetic: the default profile avatar is invisible

On `/home/browse` the profile tile renders `/gray_profile.png` at 112×112 with `opacity: 1` and `naturalWidth 50` — the image loads correctly, but the artwork is dark on the black background, so the tile reads as empty. "Add profile" next to it shows its dashed box, which makes the real avatar look broken by comparison. Give the avatar a visible treatment (a lighter placeholder tile, a ring, or a brighter default asset). Low priority — do it last, and do not spend long on it.

---

## Definition of done

- `pnpm build`, `pnpm lint`, `pnpm exec tsc --noEmit` clean.
- Admin pages checked at all four widths, with a sentence per page on what you found and changed.
- The containerized frontend proven to serve real pages, with the command you used pasted.
- Part C written up in `docs/ARCHITECTURE.md`.
- My three scratch files deleted.

Note: Codex 1 is writing an integration test suite in a new `microservices/integration` module and Codex 2 is running a security review against the same stack; both may create test users and restart nothing. Avoid `docker compose down`.

Do not commit. Work autonomously; do not stop to ask for confirmation.
