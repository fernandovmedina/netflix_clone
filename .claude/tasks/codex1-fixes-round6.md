# Codex 1 — Fixes from review round 4

Codex 2 re-verified every round-3 fix and the whole re-seeded pipeline and found them clean — including 24 sampled renditions across four assets all carrying AAC, the reset preserving user/payment data by table hash, and `exec.sql` reproducing byte-for-byte. Three defects are yours. The first is serious.

---

## 1. `[HIGH]` A failed catalog reset rolls back PostgreSQL but has already destroyed the media

`database/seed/main.go:145`

```
Setup:  one existing title with a ready asset, marker files under hls/ and sources/
Action: -reset-catalog with a manifest referencing a missing thumbnail
Result: seeder exits 1 ("missing.jpg: no such file or directory")
        PostgreSQL correctly rolls back to 1 title / 1 asset
        BOTH media markers are already gone
```

The database transaction is respected; the filesystem is not. Any failure after the media wipe — a missing artwork file, a bad manifest entry, a full disk — leaves the database advertising `ready` assets whose sources and HLS trees no longer exist, and **uploaded originals are unrecoverable**. This is the path the owner now runs to re-seed, so it is a real data-loss risk, not a theoretical one.

Fix: make the media swap atomic with respect to the database commit. Either stage the new media in a separate tree and swap it in only after the transaction commits, or rename the existing trees aside and restore them if the import or commit fails. Delete the old tree only once the new state is durable.

Test it: force a failure partway through a reset (a missing artwork file is the reproduction Codex 2 used) and assert that both the database **and** the media volume are exactly as they were before the attempt.

## 2. `[LOW]` Rate-limit rows are only cleaned when the same user repeats the same action

`microservices/user/rate_limit.go:16`

An expired bucket for user A is removed only when A performs that same action again. A user who stops using the endpoint keeps up to an hour of minute buckets forever, so the table grows with every historical user. The expiry index does not remove rows by itself.

Fix: periodic global retention cleanup, or opportunistic bounded-batch deletion of expired rows across all users. Keep it cheap and make sure it cannot become a hot spot across replicas.

## 3. `[LOW]` The integration module cannot pass the untagged build/vet/race commands

`microservices/integration/integration_test.go:1`

```
cd microservices/integration
go build ./...      exit 0, "matched no packages"
go vet ./...        exit 1, "no packages to vet"
go test -race ./... exit 1, "no packages to test"
```

Every source file in the module is behind `//go:build integration`, so the "every module" regression command and any straightforward CI matrix go red even though the tagged suite passes.

Fix: add a minimal untagged package file, or define that module's commands as `-tags=integration` and document it. Whichever you pick, `go vet ./...` and `go test -race ./...` must exit 0 in every module directory, and `docs/ARCHITECTURE.md` and the root `CLAUDE.md` test section must state the correct commands.

---

## Definition of done

- `go build ./... && go vet ./... && go test -race ./...` exits 0 in **every** module including `integration`; `go test -tags=integration ./...` passes twice.
- The reset-failure test above proves database and media are both unchanged after a failed reset.
- Codex 2's three reproductions re-run and pasted with corrected output.

Do not commit. Do not touch `frontend/**` or `microservices/nginx/**`.

Work autonomously; do not stop to ask for confirmation.
