# Codex 1 — Phase 3 (M8 + M9: user service, payments, discounts) + fixes

Repo root: `/Users/froot/Documents/workspace/fernandovmedina/web/netflix_clone`

Do these in order. Parts A and B are small and unblock other people; do them first.

## Files you own

```
microservices/user/**      microservices/catalog/**
microservices/shared/**    microservices/auth/**
database/migrations/**
```

Do not touch `frontend/**` or `microservices/nginx/**` — Codex 3 is in those concurrently.

---

## Part A — Review fixes (do first)

Execute `.claude/tasks/codex1-fixes-round1.md` in full. Four confirmed defects with live reproductions: the login timing leak, missing email validation, the 72-byte bcrypt 500, and the `X_User_Role` underscore bypass. Each needs a regression test that fails without the fix.

**Additionally**, add these to `microservices/shared/jwtutil` — Codex 2 could not complete them and I want them covered by unit tests rather than a live probe:
- A token signed with a different algorithm than the pinned one is rejected. Specifically: build a token with `alg: RS256` in the header whose signature is an HMAC over the HS256 secret, and assert `verify` returns an error. This is the classic algorithm-confusion bypass and the test documents that we are not vulnerable.
- `alg: none` rejected (already passing live — pin it with a test).
- Wrong `aud` rejected. Wrong `iss` rejected. Both are configured but untested.

## Part B — Catalog series bug (do second, Codex 3 is blocked on it)

`GET /api/v1/series/:id` returns the title correctly but its seasons query passes the **title id** where `id_series` is expected. For Baki it returns `"series_id": 1` with `"seasons": null`.

Fix the query to resolve `series.id_series` from the title first, then load seasons and episodes by that. Add a test asserting a seeded series returns its real seasons with their episodes in order — the seed has 10 series and 68 episodes, so a correct response is easy to assert against.

While you are there: confirm the same title-id/entity-id confusion is not present in the movies path.

## Part C — Supersession migration (do third)

Your Phase 2 report flagged that `uq_video_assets_movie` / `uq_video_assets_episode` plus the lack of a `superseded` status forced re-upload to reset the existing asset row rather than keep history. Migrations were out of scope then; they are in scope now.

Add `004_asset_supersession.sql`:
- add `superseded` to the `processing_status` enum
- replace the two unique constraints with **partial** unique indexes that only apply to non-superseded rows, so exactly one live asset exists per movie/episode but history is retained
- add `superseded_at timestamptz`

Then change re-upload to mark the previous asset `superseded` and insert a new row, instead of resetting in place. Keep the existing behaviour of failing the old asset's outstanding jobs and archiving the previous HLS directory. Migrations must stay additive and idempotent — no `DROP TABLE`.

---

## Part D — user service (M8 + M9), the main work

New Go module `microservices/user`, same shape as catalog: reads identity from the injected `X-User-*` headers, connects via the shared pgxpool, exposes `GET /health` on 8080. Codex 3 already wired `user1`/`user2` into compose and the image expects a root buildable main package with a `go.sum` — match that.

### Profiles, progress, favorites
Endpoints per ARCHITECTURE.md §6.
- Watch progress uses an **upsert** against the `(user_id, id_movie)` / `(user_id, id_episode)` partial unique indexes, so two instances writing concurrently converge rather than duplicating. `ON CONFLICT ... DO UPDATE`.
- `GET /api/v1/progress/continue` powers the "Continue watching" row: most recent first, excluding anything essentially finished (say >95% of duration).
- **IDOR is the risk here.** Every query must be scoped by the authenticated `user_id` from the header — never by an id taken from the path or body. A user must not be able to read or modify another user's progress, favorites or profiles by changing an id. Write tests that attempt exactly that and assert 403/404.

### Plans, discounts, payments
Read §6 and §4 carefully. The non-negotiable rules:

- **The client never sends money.** Requests carry `plan_id` and optionally `code`. Any `price`, `subtotal`, `total` or `discount_amount` in the body is ignored — do not even read it. Prices come from `plans`, discounts from `discounts`, and the total is computed server-side.
- `POST /api/v1/discounts/validate` is a **preview only**. It must not increment anything or reserve the code. The authoritative computation happens again inside the payment transaction.
- **Redemption must hold under concurrency.** Incrementing `redemption_count`, checking `max_redemptions`, and inserting into `discount_redemptions` all happen in **one transaction** with the discount row locked (`SELECT ... FOR UPDATE`). The `unique (discount_id, user_id)` index is the backstop for `per_user_limit = 1`. Test this with parallel requests, not sequential ones — Codex 2 will try to double-redeem a code from two sessions at once and it must fail cleanly with a 409, not overspend the code.
- Reject expired, not-yet-started, inactive, and exhausted codes. Reject a computed total below zero — clamp at 0 and never produce a negative charge.
- Money is `numeric(10,2)` in the DB and integer cents on the wire. No floats anywhere in the calculation path.

### Card payments
Finish the existing flow. Simulate authorization — there is no real provider.
- **Never store or log a PAN or CVV.** Persist only `card_last4` and `card_brand`. The CVV must not survive the request handler. Codex 2 will grep for this.
- Luhn-check the number and validate the expiry is in the future, so the simulation rejects obviously bad input the way a real gateway would.
- On success: create the `payments` row `paid`, activate the `subscriptions` row.

### OXXO simulation
- `POST /api/v1/payments/oxxo` creates a `pending` payment with a generated barcode `reference` and an `expires_at` (72h is realistic).
- `POST /api/v1/payments/oxxo/:ref/simulate-payment` marks it paid. This is the **dev-only** stand-in for the store confirming payment.
- Keep simulation clearly separated from anything that would be a real integration: put it behind a `PAYMENTS_SIMULATION_ENABLED` env flag (default true locally), give it its own file, and make the code obvious about what is fake. A `simulated boolean` column already exists — set it.
- Authorization: a user may only pay or view **their own** reference. Confirm a second user cannot mark someone else's reference paid.

---

## Definition of done

- `go build ./... && go vet ./... && go test -race ./...` clean in every module.
- `docker compose up -d --build` now builds **all** services including user1/user2 — this currently fails and you are the one unblocking it. Verify the full stack converges healthy.
- Concurrency tests for discount redemption and watch-progress upsert actually run in parallel.
- Report: what you built, test output, and confirmation that the full compose stack comes up.

Do not commit. Work autonomously; do not stop to ask for confirmation.
