# Codex 1 — Phase 2 (M5 + M6 + M7: catalog, worker, streaming)

This is the video pipeline: content in, HLS out. It is the highest-risk part of the project.
Repo root: `/Users/froot/Documents/workspace/fernandovmedina/web/netflix_clone`

## Read first

1. `docs/ARCHITECTURE.md` §5 (video pipeline), §6 (API contract), §10 (hard rules) — binding.
2. Your own Phase 1 output: `microservices/shared/**` and `database/migrations/**`.
3. `docker-compose.yaml` + `microservices/*/Dockerfile` — Codex 3 built these; your services must match the entrypoints, health paths and env var names already wired there. **Read them before writing main.go.**

## Files you own

```
microservices/catalog/**
microservices/worker/**
microservices/streaming/**
microservices/shared/**   (extend as needed)
```

Do not touch `frontend/**`, `docker-compose.yaml`, `microservices/nginx/**`, or `microservices/auth/**` beyond adding proxy routes if genuinely required.

---

## M5 — catalog service

Implements every catalog + admin route in §6. Reads come from Postgres via the shared pgxpool.

- **Visibility rule**: a non-admin only ever sees titles that are `published = true` **and** whose `video_assets.status = 'ready'`. Admins see everything. Enforce this in SQL, not in Go after the fact — an unpublished title must never be in the result set.
- Trust `X-User-Id` / `X-User-Role` from the proxy, but only because auth strips client copies. Do not re-verify the JWT here.
- `GET /api/v1/home` returns the browse rows the frontend renders: `Continue watching` (joins watch_progress via the user service's tables — same DB, so query directly), `Trending`, `Movies`, `Series`, and one row per major genre. Shape it to what `frontend/app/home/page.tsx` currently fakes with `api_data_example` so Codex 3 can swap it in cleanly: `[{id, title, items:[{id, content_type, thumbnail_url}]}]` — but include `title_id` and `asset_id` on each item so the player can be opened.
- **Upload intake** (`POST /api/v1/admin/movies/:id/video`, `.../episodes/:id/video`):
  - multipart, admin only
  - validate: extension in {mp4, mov, mkv, webm}, **sniff the real MIME from the first 512 bytes** (`http.DetectContentType`) — do not trust the filename or the client's Content-Type
  - enforce `MAX_UPLOAD_BYTES` with `http.MaxBytesReader` **before** reading
  - stream to `${MEDIA_ROOT}/sources/<asset-uuid>/source.<ext>` with `io.Copy`. Never `ReadAll`. Never buffer the file in memory.
  - insert `video_assets` (`status='pending'`) + `video_jobs` (`status='queued'`) in one transaction
  - respond **202 immediately**. The HTTP request must not wait for ffmpeg.
  - re-uploading over an existing asset supersedes it: mark the old asset row superseded, create a new one, and leave the old HLS on disk for a later cleanup pass (do not delete files inside the request).
- Thumbnail upload: same validation discipline, images only, into `${MEDIA_ROOT}/thumbnails/`.
- Admin CRUD for movies/series/seasons/episodes/genres. Creating a series with `number_of_seasons` must keep `seasons` consistent.
- `GET /api/v1/admin/assets/:id` returns `{status, qualities, error, duration, source_width, source_height}` for the admin UI to poll.

## M6 — worker

No HTTP server (except a tiny `/health` if the compose healthcheck needs one — check what Codex 3 wired).

**Claim loop** exactly as in §5: `FOR UPDATE SKIP LOCKED` with a lease and `attempts`. Two workers must never process the same job — write a test that proves it (spawn N goroutines against a real test DB, assert each job is claimed exactly once). Reclaim jobs whose `lease_expires_at` has passed so a killed container does not strand work. Heartbeat the lease while ffmpeg runs (long transcodes must not lose their lease mid-encode).

**Per job:**
1. `ffprobe -v error -print_format json -show_streams -show_format` the source. Extract width, height, fps, duration, audio presence.
2. Pick the ladder with the pure function from `shared` (Phase 1). **Never upscale** — only heights ≤ source height. 480p source → 144/240/360/480. If source < 144, emit one rendition at the source height.
3. Even-round every dimension (`scale=-2:H`).
4. Transcode each rendition with the bitrate profile in §5's table. Aligned keyframes are mandatory: `-g $(2*fps) -keyint_min $(2*fps) -sc_threshold 0 -force_key_frames "expr:gte(t,n_forced*6)"`. `-hls_time 6 -hls_playlist_type vod`. `libx264 -preset veryfast -profile:v main`, `aac`. Handle a source with **no audio stream** without crashing.
5. Write everything to `${MEDIA_ROOT}/hls/.tmp-<job>/`, then generate `master.m3u8` with correct `BANDWIDTH`, `AVERAGE-BANDWIDTH`, `RESOLUTION` and `CODECS` per variant, then **atomically `os.Rename`** the temp dir to `${MEDIA_ROOT}/hls/<asset-uuid>/`. Streaming must never see a partial manifest.
6. Update `video_assets`: `status='ready'`, `manifest_path`, `qualities`, `duration_seconds`, `source_width/height`, `size_bytes`. Mark the job `done`.
7. On failure: capture ffmpeg stderr (truncated) into `video_jobs.last_error` and `video_assets.error`, increment attempts, retry with backoff up to `max_attempts`, then `status='failed'` on both. Clean up the temp dir.

Run renditions sequentially per job (parallel ffmpeg on 2 workers × N renditions will saturate the host). Make concurrency a config knob `WORKER_CONCURRENCY`, default 1 job at a time per worker.

## M7 — streaming service

```
GET /api/v1/stream/:asset_id/master.m3u8
GET /api/v1/stream/:asset_id/:quality/playlist.m3u8
GET /api/v1/stream/:asset_id/:quality/:segment.ts
GET /api/v1/stream/thumbnails/:file
```

- **`http.ServeContent` only.** Never read a video into memory. Range requests, `206`, and seeking come free from it — do not hand-roll them.
- **Path traversal defense**: `asset_id` must `uuid.Parse`, `quality` must match `^\d{3,4}p$`, `segment` must match `^seg_\d{5}\.ts$`, thumbnail must match `^[a-zA-Z0-9_.-]+$` with no `..`. Reject with 400. Then build the path with `filepath.Join` on the validated components and verify the result is still inside `MEDIA_ROOT` before opening. Validate first, join second, re-check third.
- Refuse assets whose `status != 'ready'` (404). Cache that lookup briefly in-process — but the cache must be a pure optimization; correctness cannot depend on it.
- Cache-Control: manifests `public, max-age=10`; segments `public, max-age=31536000, immutable`; thumbnails `public, max-age=86400`.
- Content-Type: `application/vnd.apple.mpegurl` / `video/mp2t`.
- Its `/media` mount is **read-only**. If you find yourself needing to write, the design is wrong.

---

## Known facts about the real seed source (measured, not assumed)

`seed/video/video.mp4` is **1366×768, h264, 24 fps, 5.875 s, 451 KB, and has NO audio stream.**

Consequences you must handle:
- Its ladder is 144/240/360/480/720 — **not** 1080p or 1440p. 768 is not a ladder height; do not emit a "768p" rung and do not upscale to 1080.
- 1366 is an odd-ish width: at 720p the scaled width is 1280.6 and at 480p it is 853.75. Even-rounding is not theoretical here — it is exercised by the only source we have. Verify the generated `RESOLUTION` values are all even.
- **No audio stream.** Any ffmpeg command with a hardcoded `-c:a aac` or an audio-bitrate flag will fail on this file. Detect audio presence from ffprobe and build the arg list accordingly. This is the single most likely cause of a failed seed transcode — test it explicitly.
- At 5.875 s with 6 s segments every rendition yields exactly one segment, which does not exercise seeking or ABR switching.

**Therefore also generate synthetic test sources** with ffmpeg (`testsrc2` + `sine`) at 360p, 720p, 1080p and 1440p, each ~60 s, *with* audio, and run them through the pipeline. That is how we satisfy "test multiple video source resolutions" and get a manifest long enough to seek in. Put the generator in `microservices/worker/testdata/` as a script; **do not commit the generated media** — it is git-ignored.

## Tests (required)

- ladder selection: 90p, 480p, 720p, 768p, 1080p, 1440p, 2160p sources → expected rendition sets, no upscaling ever
- a source with **no audio stream** transcodes successfully (regression test for the real seed video)
- even-dimension rounding for odd aspect ratios (e.g. 1919×1079)
- `SKIP LOCKED` claim: N concurrent claimers, each job claimed exactly once
- lease reclaim: an expired lease is re-claimable; a live one is not
- path traversal: `../`, `..%2f`, absolute paths, null bytes, `%00`, unicode dot variants all rejected on every streaming route
- upload rejects a `.mp4` whose bytes are actually a PHP script / ELF binary
- visibility: an unpublished or not-ready title is absent from a non-admin list response
- an end-to-end pipeline test against the real `seed/video/video.mp4`: enqueue → worker → `master.m3u8` exists, parses, and lists the expected variants

## Definition of done

- `go build ./... && go vet ./... && go test ./...` clean across all your modules.
- `docker compose up -d --build` brings catalog, worker and streaming up healthy.
- You have actually played a generated `master.m3u8` — at minimum `curl` the master, then a rendition playlist, then a segment, and confirm a `206` on a ranged segment request.

## Report back

Concise report: what you built, the ladder actually produced for the seed video, paste the generated `master.m3u8`, test output, and anything Codex 3 needs (exact asset URL shape for the player). Do not commit.

Work autonomously. Do not stop to ask for confirmation.
