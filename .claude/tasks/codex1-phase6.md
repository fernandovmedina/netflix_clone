# Codex 1 — Phase 6: seed SQL, re-seed with the new 1080p/audio video, admin metadata API

Three requests straight from the project owner, in priority order. Read `INTEGRATION.md` (SEED DATA and ADMIN CONTENT MANAGEMENT) and `seed/README.md`, `seed/movies/README.md`, `seed/series/README.md` first.

---

## Part A — `database/exec.sql` must contain the real seed data

Today `database/exec.sql` is a 198-byte header comment. The owner wants the catalog seed that lives in `seed/movies/seed.json` and `seed/series/seed.json` materialised as **runnable SQL**, so they can execute it against the database and see the data.

- Generate `database/exec.sql` from the JSON: every title, movie, series, season, episode, genre, actor, category and every join row (`title_genres`, `title_actors`, `title_categories`).
- It must be **idempotent and re-runnable** — `on conflict` upserts, no duplicate rows on a second run, and it must not fight the migrations. Do not include `video_assets`/`video_jobs` rows; media is the pipeline's job, not SQL's.
- It must run standalone: `docker compose exec -T postgres psql -U netflix -d netflix -f /path/exec.sql`, or piped in. State the exact command in a header comment at the top of the file.
- **Generate it, do not hand-write it.** Add the generator to the seed tooling (`database/seed`) behind a flag or a small subcommand, so regenerating after a JSON change is one command and the file never drifts. Say what that command is in your report.
- Keep the existing Go seed importer working — `exec.sql` is an additional, human-runnable artifact, not a replacement.

## Part B — Re-seed with the replaced source video

The owner replaced `seed/video/video.mp4`. It is now **1920×1080 H.264 with AAC stereo audio at 48 kHz, 5.888 s** (the previous file was 1366×768 and silent). Everything currently in the media volume was transcoded from the old file, so the whole catalog is stale.

Re-run the pipeline over the entire seeded catalog so every movie and episode is re-transcoded from the new source. Then verify, with real output:

- The ladder now emits **144p through 1080p** (the old source topped out at 720p) and still does not upscale.
- Every rendition carries an **audio track** — AAC, and the `-c:a`/`-b:a` settings actually applied. The previous encoder path was never exercised with an audio-bearing source, so confirm the audio is really mapped, not dropped by `-map 0:v:0`. If the ffmpeg invocation only maps video, that is a bug — fix it.
- `master.m3u8` advertises `CODECS` including `mp4a.40.2` now that audio exists.
- Playback still works end to end: fetch a master, a rendition playlist and a segment through `localhost:8080` and ffprobe the result.

**Scope of the reset — important.** Purge and rebuild only the **catalog and its media**: titles, movies, series, seasons, episodes, their joins, `video_assets`, `video_jobs`, and the `hls/` and `sources/` trees on the media volume. **Do not touch `users`, `sessions`, `profiles`, `payments`, `subscriptions`, `discounts` or `discount_redemptions`** — the owner has a real admin account and test accounts in there, and I am not having those deleted. This also disposes of the `integration-title-*` and `claim-*` fixtures from `.claude/tasks/cleanup-test-fixtures.sql`; fold that cleanup into this reset and tell me the row counts.

Make the reset a documented, repeatable command (a compose profile, a flag on the seed container, or a script) — not a sequence of ad-hoc SQL the owner has to retype. `docker compose up seed` should end with a catalog whose media matches the current `seed/video/video.mp4`.

## Part C — Admin metadata: genres and cast are not assignable

You exposed genres and cast on the read side in round 3. There is still no way for an admin to **set** them: the admin API has genre-vocabulary CRUD (`POST/PATCH/DELETE /api/v1/admin/genres`) but nothing that attaches a genre or an actor to a title.

Add it — accepting `genre_ids` and cast on movie/series create and update is the natural shape, but pick what fits the existing handlers. Also cover actors and categories if the effort is comparable. The owner explicitly wants the admin panel to manage the metadata the schema models.

Codex 3 is building the admin UI for this, so **report the exact request and response shapes** you settle on. Do not touch `frontend/**`.

## Part D — lower priority: per-user rate limiting

Codex 3 added nginx rate limits for discount validation and OXXO simulation, and correctly notes they key on IP: a distributed attacker evades them and NAT users share a bucket. Add a server-side per-user limit in the user service for discount validation and payment simulation. Keep it cheap — a counter in PostgreSQL keyed by user and window is fine, and it must work across replicas, so no in-memory state.

---

## Definition of done

- `go build ./... && go vet ./... && go test -race ./...` clean in every module; `go test -tags=integration ./...` passes twice.
- `database/exec.sql` regenerated by a documented command, runs cleanly twice in a row against a live database.
- Re-seed done, with ffprobe output proving 1080p renditions **and** audio, and a master playlist showing the audio codec.
- The reset command documented in `docs/ARCHITECTURE.md` and reported to me.
- Genre/cast assignment endpoints working, with their shapes reported for Codex 3.

Do not commit. Work autonomously; do not stop to ask for confirmation.
