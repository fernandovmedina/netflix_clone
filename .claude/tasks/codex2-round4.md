# Codex 2 — Review round 4: verify the round-3 fixes and the re-seeded media pipeline

Your round-3 findings were all accepted and fixed, and your documentation audit produced 16 corrections that are now applied. This round has two jobs: confirm the fixes actually hold, and validate a substantial change to the media pipeline.

Background you need: `.claude/tasks/codex2-round3.md` (your own findings), `.claude/tasks/codex1-fixes-round3.md`, `.claude/tasks/codex1-fixes-round4.md`, `.claude/tasks/codex3-fixes-round5.md`, `.claude/tasks/codex1-phase6.md`, `.claude/tasks/codex3-phase7.md`.

**Wait until Codex 1 reports its Phase 6 re-seed complete before starting section 2** — the catalog and the whole media volume are being rebuilt, and testing against a half-reset stack wastes your time. Sections 1 and 3 are safe to start immediately.

---

## 1. Verify every round-3 fix

Re-run your own reproductions and confirm each now behaves correctly. Do not take the fix reports on trust.

1. **Frontend route guard** — `/admin` with (a) no cookie, (b) garbage cookie, (c) an expired cookie, (d) a valid **normal-user** cookie, (e) a valid **admin** cookie. Only (e) may return 200. Also confirm it fails closed when the auth API is unreachable, and that the redirect preserves `?next=`.
2. **Watch progress** — unpublished target, unknown target, and a timestamp far beyond duration.
3. **Profile limits** — the cap, concurrently (fire parallel creates and confirm the cap is not exceeded), and the name-length limit at the boundary.
4. **Discount constraints** — negative and >100% percent values must be refused by the database, and the API must refuse them too.
5. **`plan_id` range** — the int64 overflow case must be 400, not 500.
6. **Rate limits** — discount validation and OXXO simulation return 429 under a burst, and a *normal* checkout is never throttled. Codex 1 has added a per-user server-side limit on top of the IP-based nginx one; verify both layers and confirm the per-user limit works across replicas (hit different instances with the same user).
7. **Catalog visibility** — a published title whose asset is `pending` must not appear in any public read path: list, detail, search, home rows, and the featured hero. This was the defect that let test fixtures become the hero on `/home/series`.

## 2. Validate the re-seeded media pipeline (after Codex 1 reports)

The owner replaced `seed/video/video.mp4`. It is now **1920×1080 H.264 with AAC stereo, 48 kHz, 5.888 s**; the previous file was 1366×768 and silent. The entire catalog has been re-transcoded from it.

- **Audio is the risk.** The encoder previously ran only against a silent source, and the ffmpeg invocation began with `-map 0:v:0`. Verify with ffprobe that **every rendition of several different titles** carries an AAC audio track at the configured bitrate — not just the one title Codex 1 checked. A silently dropped audio track that nobody notices until playback is exactly the kind of thing this round exists to catch.
- Confirm the ladder now reaches **1080p** and still refuses to upscale.
- Confirm `master.m3u8` advertises the audio codec, and that audio/video stay in sync across a rendition switch.
- Confirm playback works through `localhost:8080` end to end, with range requests and immutable caching intact.
- Confirm no stale artifacts survived the reset: no orphaned directories under `hls/` or `sources/` for assets that no longer exist, and no `video_assets`/`video_jobs` rows pointing at missing media.
- Confirm the reset **preserved** `users`, `sessions`, `profiles`, `payments`, `subscriptions`, `discounts` and `discount_redemptions`. This was an explicit constraint; verify it rather than assuming it.

## 3. `database/exec.sql`

Codex 1 has generated it from the seed JSON so the owner can run it directly.

- Run it against the live database. Then run it **again** — it must be idempotent, producing no duplicate rows and no errors.
- Verify it actually matches `seed/movies/seed.json` and `seed/series/seed.json`: counts per table, and spot-check a few titles including one series with multiple seasons and episodes.
- Check it does not contain credentials, and does not silently depend on the Go importer having run first.
- Confirm the documented regeneration command works and reproduces the file byte-for-byte from unchanged JSON.

## 4. Regression sweep

- `go build ./... && go vet ./... && go test -race ./...` in every Go module, and `go test -tags=integration ./...` twice consecutively.
- `pnpm build`, `pnpm lint`, `pnpm exec tsc --noEmit`.
- Confirm the integration suite now leaves the catalog exactly as it found it (public title count before and after).
- A quick pass over anything in your round-3 "confirmed clean" list that these changes could plausibly have broken — in particular payments totals, admin authorization and the auth-across-replicas behaviour, since the user service and catalog both changed.

## Deliverable

The usual ranked list with real reproductions, each tagged `Owner: codex1` or `Owner: codex3`, plus an explicit statement of what you re-verified and found clean. If a previously fixed defect has regressed, say so loudly — that matters more than a new low-severity finding.

Do not fix anything. Do not commit.

Work autonomously; do not stop to ask for confirmation.
