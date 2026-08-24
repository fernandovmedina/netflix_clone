# Codex 1 — Fixes from review round 3

Codex 2's final security review found **no critical or high** issues. Four mediums are yours, all confirmed with live reproductions against the running stack. Each needs a regression test that fails without the fix.

Also included at the end is a data-exposure gap I found myself while verifying the UI.

---

## 1. `[MEDIUM]` Watch progress accepts unpublished content, impossible timestamps, and 500s on unknown targets
`microservices/user/progress.go:51`

```
PUT /api/v1/progress/movie/169  {"current_time_seconds":999999}   → 200   (169 is UNPUBLISHED)
PUT /api/v1/progress/movie/999999999  {"current_time_seconds":30}  → 500
```

Three defects in one handler:
- Progress can be recorded against a title the user cannot see. The visibility rule that catalog and streaming both enforce is missing here.
- `current_time_seconds` is accepted far beyond the movie/episode duration. Clamp it to the target's duration, and reject negatives.
- An unknown target produces a 500 instead of a 404.

Fix all three, and enforce visibility **in the SQL** (join to the title and require published + not-deleted), not in Go afterwards.

## 2. `[MEDIUM]` Unlimited profiles and unbounded profile names
`microservices/user/library.go:118`

```
profiles created: 9 (including one with a 300-character name)  → all 201
```

There is an intended maximum profile count and it is not enforced. Count and insert must be atomic — lock the user row and count inside the same transaction, or the check races. Add a name-length limit in the application **and** a database constraint, so the invariant survives a direct insert.

Note Codex 2 explicitly confirmed the XSS-shaped name stayed escaped by React, so this is a resource/data-integrity issue, not an XSS one.

## 3. `[MEDIUM]` Discount definitions allow negative and >100% percentages
`database/migrations/001_init.sql:112`

```
percent = -10   → accepted, redeemed, discount_amount 0
percent = 150   → accepted, redeemed, total 0
```

The runtime clamps the resulting amounts so no negative total is ever charged — but invalid discount rows are storable and consumable, and a redemption is burned on them. Add database `CHECK` constraints (value ≥ 0, and value ≤ 100 when `kind = 'percent'`) in a **new migration** — do not edit the existing one, since it has already been applied. Repeat the validation in `validateDiscount` so the API rejects them too.

## 4. `[MEDIUM]` Out-of-range `plan_id` produces a 500
`microservices/user/payments.go:332`

```
{"plan_id": 9223372036854775807}  → 500 {"error":"internal server error"}
```

A valid Go `int64` outside PostgreSQL's `integer` range reaches the query and errors there. Decode database integer ids as `int32`, or reject anything above `math.MaxInt32` before querying, and return 400/404. Audit sibling handlers that decode ids the same way.

---

## 5. Seeded genres and cast never reach the UI (found by the lead)

The title modal renders "Cast: Not available" and "Genres: Not available" for every title, but the data **is** seeded:

```
$ curl -b user.cookies localhost:8080/api/v1/titles/8
{"id":8,"title":"Scarface",...,"director":"Brian De Palma","year_released":1983,...}
   ← no genres, no cast

$ psql -c "select g.name from title_genres tg join genres g using(id_genre) where tg.id_title=8"
 Crime
 Thriller
 Drama
(3 rows)   -- and 60 title_actors rows exist across the catalog
```

`title_genres` (58 rows), `title_actors` (60 rows) and `title_categories` are populated by the seed importer, and the frontend already has the UI slots waiting — the catalog API simply never projects them.

Expose genres and cast on the title-detail response (and on movie/series detail if they have their own projections). Do it without an N+1: one aggregate join, not a query per title. Keep list endpoints lean — detail responses are where this belongs, unless the list already needs genres for filtering.

Do not change the frontend; if the field names you choose differ from what `TitleModal` expects, say so in your report and I will have Codex 3 align it.

---

## Definition of done

- `go build ./... && go vet ./... && go test -race ./...` clean in every module.
- A regression test per finding, each failing without its fix.
- Re-run Codex 2's reproductions above and paste the corrected output.
- Your integration suite from Phase 5 still passes, and gains coverage for these cases where it fits.

Do not commit. Do not touch `frontend/**` or `microservices/nginx/**`.

Work autonomously; do not stop to ask for confirmation.
