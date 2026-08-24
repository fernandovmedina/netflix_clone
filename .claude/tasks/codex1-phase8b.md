# Codex 1 — Phase 8B: per-title seed sources (supersedes Phase 8 Part B)

**Stop the current re-seed approach.** I have stopped the worker containers; the queue you were waiting on is abandoned and will be wiped by the next reset. Phase 8 Part A (episode artwork API) still stands — finish or keep whatever you completed there.

## Why

Giving all 78 assets the owner's new 171-second video is not viable on this machine. Measured, not estimated:

```
throughput:     7 segments/minute (5 workers, 8 cores, ~1080% CPU against 800% available)
work required:  78 assets x 6 renditions x ~29 segments = ~13,570 segments
projected:      well over a day of pinned CPU
```

It is CPU-bound, so more workers made it worse through contention, not better.

The owner's reason for the longer video was to make **adaptive bitrate switching** demonstrable — the old 5.888 s clip produced exactly one segment per rendition, so a mid-playback quality switch could never occur. That goal needs a handful of multi-segment assets, not all 78. The owner has chosen: **long source for a few titles, short source for the rest.**

## What to build

Make the seed source **per-title** instead of one global video.

- Keep the owner's `seed/video/video.mp4` (171 s, 1920x1080, AAC) as the **long** source. Do not modify or trim their file.
- Generate a **short** source from it — around 6 seconds, re-encoded so it starts on a keyframe and keeps its audio track. Do this with a documented command in the seed tooling, the same way `exec.sql` is generated, so it is reproducible from the owner's file rather than being a mystery binary. Commit the generated clip; at ~1 MB it is reasonable, and it keeps `docker compose up seed` working from a clean checkout.
- Express the choice **in the seed data**, defaulting to the short clip, with a small number of titles opting into the long one. Pick roughly **five** titles for the long source, spread across movies and series so both playback paths get a multi-segment asset — remember a series multiplies by its episode count, so account for that when choosing.
- Document the field and the regeneration command in `docs/ARCHITECTURE.md`.

Expected cost after this change: roughly 5 x 171 s plus ~73 x 6 s across six renditions — on the order of an hour rather than a day. Report the actual drain time.

## Then re-seed

Same hard scope limit as before: **catalog and media only**. `users`, `sessions`, `profiles`, `payments`, `subscriptions`, `discounts`, `discount_redemptions` untouched — verify afterwards, do not assume. Use the atomic reset you built.

Run **3 worker replicas** (`docker compose up -d --no-deps --scale worker=3 worker`), not 5 — on 8 cores, five concurrent ffmpeg processes contend badly. I have stopped the workers entirely, so bring them back yourself as part of the reset.

## Verify, with real output

On a **long-source** title:
- ~29 segments per rendition, `#EXT-X-TARGETDURATION` 6, playlist duration ~171 s with no drift.
- **Segment boundaries aligned across the ladder** — the same segment index covers the same time range in 144p as in 1080p. Compare the `#EXTINF` sequences across renditions; this is what makes seamless switching possible and it has never been testable until now.
- A range request on a **mid-playlist** segment returns 206 with a correct `Content-Range`.

On a **short-source** title: the ladder is intact, audio present, playback works.

Across both: every asset `ready`, 144p–1080p, AAC on every rendition, masters advertising `mp4a.40.2`.

Also report the media volume size — the previous full-length attempt was heading for ~19 GB of HLS output, and the owner should know the real number.

## Definition of done

- Per-title source selection working, documented, with the short clip regenerable by a stated command.
- Re-seed complete, with the segment-alignment evidence pasted for a long-source title.
- User/payment table preservation verified.
- `go build ./... && go vet ./... && go test -race ./...` clean everywhere; `go test -tags=integration ./...` passes twice.
- Drain time and media volume size reported.

Do not commit. Do not touch `frontend/**` or `microservices/nginx/**`.

Work autonomously; do not stop to ask for confirmation.
