# Codex 1 — Fixes from review round 2

All confirmed by Codex 2 with live reproductions against the committed state. Ordered by severity. Each needs a regression test that fails without the fix.

Two of these overlap with work you may have already done in Phase 3: the **series id** bug (Part B) and **asset supersession** (Part C). If those are already fixed, verify against the reproductions below and say so — do not redo them.

---

## 1. `[HIGH]` Streaming serves assets belonging to unpublished titles
`microservices/streaming/main.go:163`

```
curl -b normal-user.cookies http://.../api/v1/stream/<asset-uuid>/master.m3u8
→ 200, manifest returned for an UNPUBLISHED title
```

The `ready()` check validates only `video_assets.status = 'ready'`. Anyone who obtains or guesses an asset UUID streams unreleased content. Catalog correctly hides these titles, so the listing is safe — the media path is not.

Fix: `ready()` must join the asset to its movie/episode → title and require `status='ready'` **and** `titles.published = true` **and** the title, movie/episode rows all non-deleted. Enforce it in the query, not in Go afterwards.

Test: a ready asset on an unpublished title returns 404 for a normal user on master, rendition playlist and segment routes alike.

## 2. `[HIGH]` Symlink escape from MEDIA_ROOT
`microservices/streaming/main.go:201`

The lexical containment check passes, but a symlink inside the media tree is followed out of the root:

```
hls/<ready-asset>/master.m3u8 -> /etc/passwd
→ 200, "## User Database ##..."
```

Every other traversal vector in the corpus was correctly rejected (encoded and double-encoded separators, backslashes, null bytes, unicode and overlong variants). This one is the gap.

Fix: after building the path, resolve it with `filepath.EvalSymlinks` and re-check containment against the resolved `MEDIA_ROOT`; or open with no-follow semantics (`openat` + `O_NOFOLLOW`). Note `EvalSymlinks` fails on a nonexistent path, so order the checks so a missing file still returns 404 rather than 500.

Test: create a symlink under the media root pointing outside it and assert the request is refused.

## 3. `[HIGH]` Stale worker resurrects a superseded job and clobbers newer state
`microservices/worker/jobs.go:71`

```
superseded job resurrected: status=leased locked_by=<nil>
stale worker clobbered replacement: asset_status=failed
```

A worker that lost its lease (or whose job was superseded by a re-upload) can still write `fail()`/completion updates, marking valid replacement content as failed or re-running work.

Fix: **every** job and asset update from a worker must be conditional on still owning the lease — `WHERE status='leased' AND locked_by = $worker AND lease_expires_at > now()`. Check the affected-row count; if it is zero, the lease was lost, so abandon the job quietly and do not touch the asset. Update job and asset in the **same transaction** so they cannot diverge.

Test: claim a job, externally supersede it, then attempt both the fail and the success path from the stale worker — neither may modify the newer rows.

## 4. `[MEDIUM]` Re-upload reuses the same asset UUID under immutable caching
`microservices/catalog/admin.go:443`

Uploading twice returns the same `asset_id`. Segments are served `immutable, max-age=31536000`, so clients and caches can serve the **previous** upload's segments for the new content.

Fix: allocate a **new** asset UUID for every upload, mark the previous asset `superseded`, and never write over `/hls/<old-asset-id>/`. This is exactly what Phase 3 Part C's migration enables — if you have done that, confirm the reproduction now returns two different UUIDs.

## 5. `[MEDIUM]` Size cap is applied to the whole multipart body
`microservices/catalog/admin.go:414`

```
file_bytes=1024 multipart_bytes=1268 → 413
```

A file at exactly `MAX_UPLOAD_BYTES` is always rejected, because boundaries and part headers push the encoded request over the limit. (Codex 3 is fixing the mirror-image problem in nginx; the application cap needs the same correction.)

Fix: allow the outer request a bounded overhead allowance, and enforce the exact limit on the **selected part** with a `LimitedReader(maxUpload + 1)` — if it yields more than `maxUpload`, reject. The file limit must be about the file, not its framing.

## 6. `[MEDIUM]` Failed uploads leave orphaned source files
`microservices/catalog/admin.go:445`

```
POST .../admin/movies/999999/video  → 500
/media/sources/<uuid>/source.mp4    451298 bytes, orphaned
```

Bytes are written to the shared volume before the target is validated, and are not cleaned up when the insert fails. Repeated requests against nonexistent ids fill the media volume with no database record — an unauthenticated-adjacent disk-exhaustion path (admin-only, but still).

Fix: verify the target movie/episode exists **before** storing bytes, and on any failure after the write, remove the stored object. Commit the database transaction first, then treat the file as durable.

## 7. `[MEDIUM]` Series lookup uses the title id where the series id is required
`microservices/catalog/query.go:122`

```
title_id=28 series_id=2 → seasons=0
```

Seasons silently vanish whenever the `titles` and `series` sequences do not coincide — which the seed data happens to mask. This is Phase 3 Part B; confirm it against this reproduction (create a series where `id_title != id_series`, not a seeded one).

## 8. `[MEDIUM]` Invalid season creation returns 200 with an empty body
`microservices/catalog/admin.go:207`

`{"season_number": 0}` → `HTTP 200`, `Content-Length: 0`. The client believes it succeeded.

Fix: return a 400 JSON error for `season_number < 1`. Audit the sibling handlers for the same early-return-without-writing pattern.

## 9. `[MEDIUM]` Malformed JSON produces two concatenated JSON objects
`microservices/catalog/admin.go:286`

```
{"error":"invalid JSON body"}
{"error":"episode_number and title are required"}
```

Not parseable as one JSON value, and it logs a superfluous `WriteHeader`. The decode failure falls through into field validation instead of returning.

Fix: return immediately when decode fails. Apply the same pattern across the season, episode and genre handlers — Codex 2 flagged them as sharing it.

---

## 10. Catalog response gaps blocking the admin UI (from Codex 3)

Two fields the frontend needs and cannot synthesize:

- **Expose `movie_id`** on list and movie-detail responses. Catalog currently returns only the title id, but the movie upload, update and progress endpoints key on the movie id. The frontend is currently declining to send a progress id at all rather than guess wrong — so movie watch progress is inert until this lands.
- **Expose the latest asset id and status on admin responses**, including `pending`, `processing` and `failed`. Today only *ready* assets are joined, so after a page reload the admin dashboard cannot tell that a job is still in flight and shows nothing. Admin views must see every state; the public views keep their `ready`-only filter.

Keep the public and admin projections distinct — do not leak in-flight asset state to non-admin responses.

---

## Definition of done

- `go build ./... && go vet ./... && go test -race ./...` clean in every module.
- A regression test per finding, each failing without its fix.
- Re-run Codex 2's reproductions and paste the corrected output.

Do not commit. Do not touch `frontend/**` or `microservices/nginx/**`.
