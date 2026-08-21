# Codex 3 — Phase 3 (M12 player + M13 admin UI) + nginx fixes

Repo root: `/Users/froot/Documents/workspace/fernandovmedina/web/netflix_clone`

## Files you own

```
frontend/**    microservices/nginx/**
```

Do not touch any `.go` file, `database/**`, or `docker-compose.yaml`. Codex 1 is building `microservices/user` concurrently.

Do Part A first — it is small and it is a confirmed security finding.

---

## Part A — nginx fixes (do first)

Execute `.claude/tasks/codex3-fixes-round1.md`: rate-limiting on the auth endpoints (login/signup strict, refresh loose) and raising the upload body limit so a file at the documented 5 GiB cap is not rejected by the transport layer. Both were confirmed by Codex 2 with live reproductions.

Be careful not to rate-limit manifest and segment requests aggressively — an ABR player legitimately issues a burst of requests and throttling them looks exactly like a broken player.

---

## Part B — M12: the video player

This is the centrepiece feature. Build `frontend/components/VideoPlayer.tsx` and a watch route (`app/watch/[assetId]/page.tsx`), then wire the play buttons in `Hero`, `TitleModal` and the episode list to it.

Backend contract, confirmed working by Codex 1:
```
/api/v1/stream/{asset_id}/master.m3u8
/api/v1/stream/{asset_id}/{quality}/playlist.m3u8
```
`asset_id` arrives on catalog and home items. Segments already return `206 Partial Content` with a correct `Content-Range`, so seeking is supported server-side.

Requirements:
- **hls.js** for browsers without native HLS, and **native playback** where `video.canPlayType('application/vnd.apple.mpegurl')` is supported (Safari, iOS). Detect and branch — do not force hls.js on Safari.
- Automatic quality selection is the default and must actually work — this is the ABR requirement, not optional.
- A manual quality menu (Auto + each available rendition) driven by `hls.levels`, since the UI has room for it.
- **Error recovery**: on `Hls.ErrorTypes.NETWORK_ERROR` call `startLoad()`, on `MEDIA_ERROR` call `recoverMediaError()`, and only tear down on a fatal error that survives recovery. A transient segment 404 or network blip must not kill playback — INTEGRATION.md calls this out explicitly.
- Controls: play/pause, seek scrubber, volume/mute, elapsed/total time, fullscreen, quality menu, back. Keyboard: space, arrows, `f`, `m`.
- Send watch progress to `PUT /api/v1/progress/{kind}/{id}` periodically (every ~10 s and on pause/unload). **Codex 1 is building that endpoint right now** — code against the contract in ARCHITECTURE.md §6 and degrade gracefully if it 404s, rather than blocking on it.
- Resume from saved progress when the title has any.
- Mobile: touch-friendly controls, correct behaviour on iOS (`playsInline`).
- `credentials: "include"` — hls.js needs `xhrSetup` to set `withCredentials` or authenticated segment requests will fail.

Add `hls.js` to `package.json`. That is a justified dependency, not an unnecessary one.

## Part C — M13: admin UI

New area under `frontend/app/admin/`, guarded by the existing middleware plus a role check (`user.role === "admin"` from `AuthProvider`; the backend enforces it too — the UI guard is only to avoid showing a dead page).

Pages:
- `/admin` — dashboard listing all titles with status, including unpublished ones.
- `/admin/movies/new` and `/admin/movies/[id]` — metadata form, poster upload, video upload, publish toggle.
- `/admin/series/new` and `/admin/series/[id]` — series metadata, season management, episode management with per-episode video upload.

The upload experience is the part that matters:
- Real upload **progress percentage**. `fetch` cannot report upload progress — use `XMLHttpRequest` with `upload.onprogress`, or you will end up with a fake spinner.
- After the 202 comes back, **poll `GET /api/v1/admin/assets/{id}`** and show the pipeline state as a clear status pill: `Uploading → Processing → Ready`, or `Failed` with the error text from the backend.
- Stop polling when the asset reaches a terminal state; do not leave a timer running forever on an unmounted component.
- When ready, show the qualities the pipeline actually produced (e.g. `144p 240p 360p 480p 720p`) — that is the visible proof the ladder worked.
- Publishing must be blocked, with an explanation, until processing succeeds. INTEGRATION.md is explicit that unready content must not be exposed as playable.

## Quality bar

- `pnpm build` and `pnpm lint` clean, no type errors, no `any` where the shape is known.
- Loading and error states everywhere that fetches.
- Match the existing Tailwind idiom. It should still look like Netflix.

## Verification

Bring the stack up and actually use it. Do not report success on a build passing alone:
- Play a seeded title. Confirm playback starts, the quality menu lists the real renditions, and seeking works.
- Confirm in devtools Network that the player pulls **segments**, not one large file.
- Upload a video through the admin UI and watch it go Uploading → Processing → Ready.

If `microservices/user` is still unbuildable when you get there, note it and verify everything that does not depend on it.

## Report back

Files added/changed, the dependency you added and why, what you verified in a real browser, and anything you need from Codex 1. Do not commit.

Work autonomously; do not stop to ask for confirmation.
