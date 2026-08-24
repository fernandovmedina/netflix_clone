# Codex 1 — Phase 5: end-to-end integration test suite (M16)

Your session was restarted, so assume no memory of earlier rounds. Read first:

- `docs/ARCHITECTURE.md` — the build contract. §6 is the API surface, §4 the schema, §5 the video pipeline.
- `INTEGRATION.md` — project rules. Note the definition of done and the hard rules at the end.
- `.claude/tasks/codex1-phase3.md` and `.claude/tasks/codex1-fixes-round2.md` — the behaviour you previously implemented and fixed, so the tests assert the *intended* semantics rather than whatever the code happens to do today.

The unit tests already pass (`go test -race ./...` is clean in `auth`, `catalog`, `user`, `streaming`, `worker`, `shared`). What is missing is the **integration** layer: nothing exercises a real request path through nginx → auth → owning service → Postgres against the running stack.

## Scope

Create a new Go module `microservices/integration` (module path `github.com/fernandovmedina/netflix-clone/microservices/integration`), tests only — it must not become a service. Guard everything behind a build tag so it never runs in the normal unit-test sweep:

```go
//go:build integration
```

Configuration comes from the environment, with working defaults for the local stack:

```
BASE_URL      default http://localhost:8080     (always go through nginx — never talk to a service directly)
DATABASE_URL  default the value from /.env, rewritten to localhost:5433
ADMIN_EMAIL / ADMIN_PASSWORD  from /.env
```

Talking to Postgres directly is fine for *arranging* fixtures and *asserting* stored state. Every user-facing assertion must go through `BASE_URL`.

## What to cover

**1. Auth across instances.** Signup, login, `/auth/me`, refresh rotation, logout. Because nginx load-balances, a loop of N requests naturally lands on different instances — assert every one succeeds, and assert that a session minted by one instance is accepted after a refresh served by another. Assert logout invalidates the session for subsequent requests. Assert an expired/garbage token is rejected with 401.

**2. Refresh-token rotation under concurrency.** Two simultaneous refreshes with the same token: exactly one succeeds, and reuse of the rotated token is refused.

**3. Catalog visibility.** A normal user must never see, in any read path (list, single title, search, home rows), a title that is unpublished or whose asset is not `ready`. Arrange both cases directly in the database, then assert through the API. Assert the admin projection *does* show in-flight asset state.

**4. Admin authorization.** Enumerate every `/api/v1/admin/*` route and assert unauthenticated → 401 and normal-user → 403. This is a table-driven test; when someone adds a route, the table should be the obvious place to add it.

**5. Payments and discounts** — the highest-value tests here:
   - Plan prices come from the backend; a request that includes `price`/`total`/`discount_amount` in the body must not affect the charged total. Assert against the stored `payments` row, not the response alone.
   - A single-use discount code redeemed twice **concurrently** yields exactly one redemption row. Drive it with real parallel requests, and assert on the row count.
   - Expired, exhausted, inactive and unknown codes are refused with distinguishable errors.
   - `POST /discounts/validate` creates no redemption.
   - OXXO: create a payment, assert the reference and amount, simulate payment, assert the subscription becomes active. Assert one user cannot simulate another user's reference.
   - Assert no full PAN is stored anywhere in `payments`.

**6. Streaming.** Master playlist, rendition playlist and a segment: 200 for a ready asset on a published title; `Range` request returns 206 with a correct `Content-Range`; segments carry immutable caching; an asset belonging to an unpublished title returns 404 on all three routes; unauthenticated requests are refused.

**7. Upload → transcode → ready.** The slowest test — mark it so it can be skipped with `-short`. Upload a small video as admin (generate a few seconds with ffmpeg at a low resolution, or reuse `/seed/video/video.mp4`), assert the response returns immediately with a `pending`/`processing` asset rather than blocking on the transcode, then poll until the asset reaches `ready` with a sensible timeout. Assert the generated ladder does not upscale beyond the source resolution, that `master.m3u8` advertises exactly the renditions that exist on disk, and that the asset is not playable by a normal user until it is `ready`.

**8. Watch progress and favorites.** Round-trip through the API, plus an IDOR assertion: user A cannot read or write user B's progress or favorites by supplying B's ids.

## How the tests must behave

- **Self-contained.** Each test creates the users, titles, discounts and payments it needs, with identifiers unique per run (a random suffix), and cleans up after itself. Running the suite twice in a row must pass both times.
- **No cross-test coupling.** They must pass when run with `-shuffle=on`, and the ones that can run in parallel should.
- **Never assert on which instance served a request** — only that every instance can serve it.
- **Honest failure messages.** On failure, print the request, the status and the response body. A bare `expected 200, got 500` wastes the reader's time.
- Do not weaken a test to make it pass. If the suite uncovers a real defect, **fix the defect** in the owning service (that is your area: `microservices/**` except `nginx`), and say so in your report. If a failure lands in `frontend/**` or `microservices/nginx/**`, do not touch it — report it and tag `Owner: codex3`.

## Also

Add a short `microservices/integration/README.md`: how to run the suite (`go test -tags=integration ./... -v`), what must be running first, which environment variables it reads, and how to skip the slow transcode test. Wire it into whatever task runner the repo already uses if there is one — do not introduce a new one.

## Definition of done

- `go build ./... && go vet ./... && go test -race ./...` still clean in every existing module.
- `go test -tags=integration ./...` passes against the running stack, twice in a row, and with `-shuffle=on`.
- Paste the full test output in your report.
- Every defect the suite uncovered is either fixed by you or reported with an owner.

Heads-up on concurrency with the other workers: Codex 3 is reworking `microservices/nginx/nginx.conf` and `docker-compose.yaml` onto a replica-based topology and may restart services under you — if something fails in a way that looks like a restart rather than a real defect, retry before concluding. Codex 2 is running a security review against the same stack and is creating its own test users and payments; do not assume you are alone in the database.

Do not commit. Work autonomously; do not stop to ask for confirmation.
