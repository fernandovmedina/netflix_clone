# Codex 2 — Review round 2: catalog, worker, streaming, frontend cutover

Charter: `.claude/tasks/codex2-charter.md` (how you work and report). This file is the scope.

## Scope

Commits `c5b75fb` and `d094fd6`:

```
microservices/catalog/**     microservices/worker/**
microservices/streaming/**   microservices/shared/**
frontend/**
```

`microservices/user` is being written right now — out of scope this round. Codex 1 and 3 are also actively editing `microservices/catalog`, `microservices/auth`, `frontend/` and `microservices/nginx`; review the **committed** state (`git show c5b75fb`, `git show d094fd6`) rather than the working tree, so you are not chasing half-finished edits.

## Priorities for this round

### 1. Path traversal on the streaming service — highest priority
`microservices/streaming` serves files from a shared volume based on URL components. Try hard to escape `MEDIA_ROOT`:
- `../`, `....//`, `..%2f`, `%2e%2e%2f`, `%252e%252e%252f`, backslash variants, absolute paths
- null bytes and `%00`
- unicode dot and slash lookalikes, overlong UTF-8
- a `quality` or `segment` component that passes the regex but still resolves outside the root
- symlink escape: if a symlink were placed under `/media`, does the resolved path check catch it?

Target something real: try to read `/etc/passwd`, `/media/sources/*/source.mp4` (the originals should not be publicly fetchable), and the migration SQL. Report exactly what you achieved.

### 2. Media authorization
- Can an **unauthenticated** request fetch a manifest or segment? Trace whether streaming sits behind `requireAuth` in the auth proxy or is reachable another way.
- Can a user stream a title that is unpublished, or whose asset is `pending`/`failed`? Fetch an asset id straight from the database and try it.
- Are the uploaded **source** files exposed through any route? They should not be — only the transcoded HLS output.

### 3. Memory safety on the media path
Grep every handler in `catalog`, `streaming` and `worker` for `io.ReadAll`, `os.ReadFile`, `ioutil.ReadFile` on anything media-sized. A single `ReadAll` on an upload or a segment is an OOM denial-of-service on a 5 GiB file. Confirm uploads use a streaming copy with `MaxBytesReader` and that segment serving uses `ServeContent`.

### 4. Upload validation
- A `.mp4` whose bytes are actually a shell script, an ELF binary, or HTML — is the MIME really sniffed from content, or is the filename trusted?
- A filename containing `../` or a null byte — where does it land on disk?
- Does the size cap actually stop a large upload, and does it stop it *before* consuming the whole body?
- A polyglot file that sniffs as video but that ffmpeg will choke on — does the job fail cleanly and mark the asset `failed`, or does it hang and hold the lease forever?

### 5. Job queue correctness
- Run two workers against a seeded queue and prove no job is processed twice.
- Kill a worker mid-transcode (`docker kill`) and confirm the lease expires and the job is reclaimed rather than stranded.
- Confirm `attempts`/`max_attempts` terminates a permanently failing job instead of retrying forever.
- Is the lease heartbeat actually extending during a long ffmpeg run? A 20-minute transcode under a 30-minute lease is fine; check what happens when it is not.

### 6. Catalog authorization and IDOR
- Every `/api/v1/admin/*` route called as a normal user must 403. Enumerate them and try all of them.
- Is the visibility filter (`published` + `ready`) applied in SQL on **every** read path, including single-title fetch, search and the home rows — or only on the list endpoint?

### 7. SQL injection
Every query in catalog and worker. Pay attention to anything building a `WHERE` from query parameters — the catalog has filtering by `type`, `genre`, `q`, `limit` and `offset`, which is exactly where a concatenation tends to appear. Also check `limit`/`offset` for integer handling.

### 8. Frontend
- Does any secret reach the bundle? Build it and grep the output, not the source.
- Is the access token readable from JS? It should be HttpOnly — confirm `document.cookie` cannot see it.
- The 401-refresh-retry in `utils/api/client.ts`: can it be driven into an infinite loop by a persistent 401? Is the concurrent-refresh de-duplication actually correct, or does it have a race where a second caller gets a stale promise?
- Does `middleware.ts` actually protect `/admin`, and does it fail closed?
- XSS: any `dangerouslySetInnerHTML`, or catalog text rendered unescaped.

### 9. Caching correctness
Segments are served `immutable` with a one-year max-age. After a re-upload supersedes an asset, could a client be served stale segments for the new content under an old URL? Reason about whether the asset-id-per-upload scheme makes this safe, and say so either way.

## Also run

- `go test -race ./...` in catalog, worker, streaming, shared.
- `docker compose logs` for any secret, token, password or full card number being logged.

## Deliverable

The ranked finding list from the charter, with real reproduction output for every CONFIRMED item. Tag `Owner: codex1` or `Owner: codex3`. If a category is clean, say so explicitly — I want to know what was checked and passed, not just what failed.

Do not fix anything. Do not commit.

Work autonomously; do not stop to ask for confirmation.
