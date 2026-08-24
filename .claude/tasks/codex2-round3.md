# Codex 2 — Review round 3 (final): payments, user service, admin UI, checkout, full-system security

Your session was restarted, so assume no memory of earlier rounds. Read these first:

- `.claude/tasks/codex2-charter.md` — how you work and how you report. Follow it exactly.
- `docs/ARCHITECTURE.md` — the build contract (API surface in §6, schema in §4, pipeline in §5).
- `INTEGRATION.md` — the security checklist is under `# SECURITY`.
- `.claude/tasks/codex2-round1.md` and `codex2-round2.md` — what was already reviewed. **Do not re-review those areas from scratch**; instead verify the round-2 fixes actually hold (list below), then spend your effort on the new scope.

The full stack is already running locally: `docker compose ps` (postgres, migrate/seed done, auth1-3, catalog1-2, streaming1-2, user1-2, worker1-2, nginx on `localhost:8080`, frontend on `localhost:3000`). Seed data is loaded: 20 titles, 10 movies, 68 episodes, 78 `ready` video assets. Admin credentials are `ADMIN_EMAIL`/`ADMIN_PASSWORD` in `/.env`.

## Scope — commits `c754450`, `0c8160b`, `83cde75`

```
microservices/user/**            frontend/app/signup/payment/**
microservices/catalog/admin.go   frontend/app/admin/**
frontend/components/payments/**  frontend/components/admin/**
frontend/utils/payments.ts       frontend/utils/api/client.ts
frontend/components/VideoPlayer.tsx
```

## 1. Payments and discounts — highest priority

The rule the whole design rests on: **the client never sends money**. Attack it.

- Send `price`, `total`, `subtotal`, `discount_amount`, `amount`, `currency` in the card and OXXO request bodies. Does any of it reach the stored payment or the charged total? Try nesting them, try them at the top level, try `plan_id` pointing at one plan with a price field from another.
- Negative and absurd values: negative `plan_id`, a discount that exceeds the plan price (does the total floor at 0 or go negative?), integer overflow on cents, a `plan_id` that does not exist, a deleted plan.
- Discount abuse — the important one:
  - Redeem a single-use code twice **sequentially**, then **concurrently** (fire N parallel requests with the same code and user; exactly one may succeed). Prove it with the redemption row count, not just the HTTP codes.
  - Redeem the same code from two different users when it is global-single-use.
  - An expired code, a code with `max_redemptions` exhausted, a code that is inactive/deleted.
  - Case sensitivity and whitespace: does ` welcome10 ` or `WELCOME10` bypass a redemption check that keys on the exact string? A code that normalizes for lookup but not for the uniqueness constraint is a double-redeem.
  - Does `POST /discounts/validate` (the preview) create a redemption or consume anything? It must not.
- OXXO simulation: can a normal user call the `simulate-payment` endpoint for **someone else's** reference? Can they call it twice and get two subscriptions or a double credit? Can they guess references (are they random enough)? Is the simulation path clearly separated from any real-provider path?
- Card: is the PAN stored anywhere — database column, log line, error message? `docker compose logs` and `select * from payments`. Confirm what is persisted; only a last-4/brand is acceptable.
- Payment → subscription: after a successful payment, is the subscription actually activated, and can a user activate one without paying (call the subscription-granting path directly)? Can a user read another user's payment via `GET /api/v1/payments/:id` (IDOR)?

## 2. User service — `microservices/user`

- IDOR across every route: watch progress, favorites, profiles. Can user A read or write user B's rows by supplying an id? The identity must come from the auth-injected header, never from the body or query.
- Can a client forge `X-User-Id`/`X-User-Email` by sending those headers directly to nginx? Trace whether the auth proxy strips inbound copies before setting its own. This is a full authentication bypass if it works — test it explicitly and report it either way.
- Watch progress: out-of-range values (negative, beyond duration, non-numeric), progress on a title the user cannot see (unpublished), progress referencing both a movie and an episode at once (the XOR constraint), concurrent updates from two devices.
- Profiles: max profile count enforced? Name validation and XSS-in-name rendered back on the frontend?
- SQL injection on every query in the module.

## 3. Admin authorization — full route enumeration

Enumerate **every** `/api/v1/admin/*` route in `microservices/catalog` and call each one as (a) unauthenticated, (b) a normal user, (c) an admin. Every one of (a) and (b) must fail closed. Include the upload routes and any route reachable through the nginx path that does not go through `requireAdmin`.

Then: can a normal user reach an admin route by path tricks that nginx and Go's mux normalize differently — `/api/v1//admin/movies`, `/api/v1/./admin/movies`, `%2e%2e`, a trailing-slash variant, mixed case, or a `X-Forwarded-*`/method-override header? The `requireAdmin` matcher is registered on two exact patterns (`/api/v1/admin` and `/api/v1/admin/`); prove nothing routes around it.

- Can a user escalate their own `role` through the signup body, the profile update, or any user-service write?

## 4. Round-2 regression check

Confirm each of these still holds (reproductions are in `.claude/tasks/codex1-fixes-round2.md`):

1. Streaming refuses assets on unpublished titles (master, playlist and segment routes).
2. Symlink escape from `MEDIA_ROOT` is refused.
3. A stale worker cannot clobber a superseded job or asset.
4. Re-upload allocates a **new** asset UUID; old segments are never overwritten.
5. Upload size cap applies to the file, not the multipart framing.
6. A failed upload leaves no orphaned source file on the volume.

## 5. Frontend

- Build the bundle and **grep the build output** (not the source) for: card numbers or expiry persisted anywhere, `JWT_SECRET`, `GOOGLE_CLIENT_SECRET`, `POSTGRES_PASSWORD`, `DATABASE_URL`, admin credentials.
- Confirm the access token is HttpOnly and unreadable from `document.cookie`.
- `utils/api/client.ts`: the 401-refresh-retry. Persistent 401 must not loop infinitely; concurrent refresh de-duplication must not hand a second caller a stale/rejected promise.
- `middleware.ts` must protect `/admin` and `/home` and fail closed when the API is unreachable.
- XSS: any `dangerouslySetInnerHTML` or catalog/profile text rendered unescaped.

## 6. Horizontal scaling and concurrency

- Log in against one auth instance and use the session against the other two (`docker compose exec` into nginx or hit it repeatedly and watch which upstream serves). No instance affinity is allowed. Same for refresh: refresh on auth2 with a session minted by auth1.
- Refresh-token rotation under concurrency: two simultaneous refreshes with the same token — is exactly one accepted, and does reuse of a rotated token get detected?
- Confirm logout on one instance invalidates the session on all of them.
- Two workers, one queue: prove no double-processing (`FOR UPDATE SKIP LOCKED`), and that killing a worker mid-job releases the lease.

## 7. Automated tests

Run and report: `go build ./... && go vet ./... && go test -race ./...` in every Go module (`microservices/{auth,catalog,streaming,user,worker,shared,migrate,seed}`), and `pnpm build`, `pnpm lint`, `pnpm exec tsc --noEmit` in `frontend`.

## Deliverable

The ranked finding list from the charter, with real reproduction output for every CONFIRMED item, each tagged `Owner: codex1` (backend) or `Owner: codex3` (frontend). State explicitly which categories you checked and found **clean** — I need the passes as much as the failures, because this is the final review before sign-off.

Do not fix anything. Do not commit. Do not restart or rebuild the docker stack without saying so in your report (I am using it concurrently for browser verification) — read-only interaction with the running stack is fine, and creating test users, payments and uploads is fine.

Work autonomously; do not stop to ask for confirmation.
