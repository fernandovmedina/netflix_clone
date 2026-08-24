# Seed Video

**This directory is intentionally empty in the repository.** Only code is
committed; video files are not. A fresh clone therefore seeds the catalog with
titles, metadata and artwork but with **no video at all**, and every title shows
a "No video yet" placeholder until someone uploads one.

That is the supported starting state, not a broken one.

## Getting video into a local stack

Sign in as the administrator (`ADMIN_EMAIL` / `ADMIN_PASSWORD` from `.env`), open
<http://localhost:3000/admin>, pick a title and upload a clip:

- a **movie** takes one video under *Movie video*;
- a **series** takes one video per episode, under each episode in the series
  editor, plus an optional per-episode thumbnail.

The upload returns immediately with a `pending` asset; a worker transcodes it
into the HLS ladder in the background and the panel flips to *Ready* on its own.
Publishing stays locked until at least one asset is ready.

## Optional: pre-seeding from local clips

If you drop files here, the seed importer will use them instead of leaving the
catalog empty-handed:

| File | Used by |
| --- | --- |
| `video-short.mp4` | manifest entries with `"video_source": "short"` (the default) |
| `video.mp4` | manifest entries with `"video_source": "long"` |

Both are ignored by git. Nothing else in the pipeline changes — seeded media
goes through exactly the same transcode path as an administrator upload.

Generate the short clip from a long one with the seed tool (needs FFmpeg on
`PATH`):

```sh
cd database/seed
go run . -generate-short-video ../../seed/video/video-short.mp4
```

To build a long clip by looping a shorter source — useful for exercising
seeking, ABR switching and resume-from-progress over a realistic runtime:

```sh
ffmpeg -stream_loop 10 -i input.mp4 \
  -c:v libx264 -preset veryfast -pix_fmt yuv420p -profile:v main \
  -b:v 450k -maxrate 750k -bufsize 1500k \
  -g 150 -keyint_min 150 -sc_threshold 0 \
  -c:a aac -b:a 64k -ac 2 -ar 48000 \
  -movflags +faststart seed/video/video.mp4
```

After adding or replacing a clip, re-run the importer. It fingerprints every
seed source by SHA-256, notices the change and re-imports:

```sh
docker compose up --build seed
```

Assets you uploaded through `/admin` are **not** seed-managed: the importer
leaves them alone and a reset will not destroy them.

## Sizing

Transcoding is real work. A 30-minute 1080p source expands into seven renditions
(144p–1440p, never upscaled beyond the source), so expect minutes of worker time
per asset. Start with something short unless you are specifically testing
long-playback behaviour.
