# Codex 1 — Phase 8: episode artwork API, then re-seed with the longer source video

Two parts. Do Part A first (it is code), then Part B once, so the re-seed picks up any importer change you make.

---

## Part A — Episode artwork API

Execute `.claude/tasks/codex1-fixes-round7.md` in full. Short version: there is no way to give an episode a still image — the thumbnail upload route targets titles only and episode PATCH does not accept `thumbnail_url`. I already confirmed the seed JSON has no episode-level images, so this is purely an API gap. Report the endpoint shape for Codex 3.

## Part B — Re-seed with the new 171-second source

The owner has replaced `seed/video/video.mp4` again. It is now:

```
1920x1080 H.264 + AAC stereo, duration 171.06 s, 32 MB
```

The previous source was 5.888 s, which produced exactly **one** segment per rendition — that is why Codex 2 could prove cross-rendition timestamp compatibility but not an actual mid-playback quality switch. At 6-second segments this one yields roughly 29 segments per rendition, so adaptive switching becomes genuinely testable.

Re-run the catalog reset and re-transcode as in Phase 6, with the same hard scope limit: **catalog and media only**. `users`, `sessions`, `profiles`, `payments`, `subscriptions`, `discounts` and `discount_redemptions` must be untouched — verify it afterwards rather than assuming, as Codex 2 did with table hashes.

**Use the atomic reset you just built.** This is its first real outing; if a failure occurs partway, the media volume must survive.

Then verify, with real output:

- Every asset reaches `ready`, with the full 144p–1080p ladder and AAC on every rendition.
- Each rendition has **~29 segments**, and `#EXT-X-TARGETDURATION` is 6.
- **Segment boundaries are aligned across renditions** — the same segment index covers the same time range in 144p as in 1080p. This is what makes seamless switching possible and it was untestable before; check the `#EXTINF` sequences match across the ladder, not just that the files exist.
- Total duration in each rendition playlist matches the source (~171 s) with no drift.
- Range requests still return 206 with correct `Content-Range` on a mid-playlist segment, not just the first.

### Watch the resource cost

This is a much heavier job than the 5.9-second source: 78 assets × 6 renditions × 171 s. Two things to keep an eye on and report:

- **Disk.** 78 copies of a 32 MB source is ~2.5 GB of originals alone, before HLS output. Report the actual volume usage after the re-seed. If the importer copies the same file 78 times, say whether that is worth addressing (it is the honest storage model for per-asset sources, but the owner should know the number).
- **Time.** Report how long the queue took to drain. If it is slow, say what would help — worker replicas, `WORKER_CONCURRENCY`, or ffmpeg preset — but do not change encoder settings to chase speed at the cost of quality without telling me first.

I may scale worker replicas up while this runs; that is expected and should not disturb you.

## Definition of done

- Part A endpoints working, with the shape reported for Codex 3.
- All assets `ready` from the new source, with the segment-alignment evidence above pasted.
- Preservation of user/payment tables verified, not assumed.
- `go build ./... && go vet ./... && go test -race ./...` clean in every module; `go test -tags=integration ./...` passes twice.
- Disk and duration numbers reported.

Do not commit. Do not touch `frontend/**` or `microservices/nginx/**`.

Work autonomously; do not stop to ask for confirmation.
