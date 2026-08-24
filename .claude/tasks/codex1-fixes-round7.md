# Codex 1 — Round 7: episodes have no artwork API

Found by Codex 3 while fixing the episode-row layout, and it explains why the layout bug went unnoticed: **not one of the 68 seeded episodes has artwork, and there is no way to give an episode any.**

```sql
select count(*) as episodes,
       count(thumbnail_url) filter (where coalesce(thumbnail_url,'') <> '') as with_thumb
  from episodes;
--  68 | 0
```

- `POST /api/v1/admin/titles/{id}/thumbnail` uploads artwork for a **title** only.
- `PATCH /api/v1/admin/episodes/{id}` does not accept `thumbnail_url`.

So an admin can create an episode, upload its video, watch it transcode and publish it — but cannot give it a still image. `INTEGRATION.md` requires the admin panel to manage episodes including "Images/posters where applicable", and the episode row in the UI has a dedicated artwork cell that currently renders a "No episode artwork" placeholder for every episode in the catalog.

To build the mixed-artwork test fixture, Codex 3 had to write to the database directly (`episode 2259`), because no API could do it. That is the tell.

## What to add

- An episode artwork upload endpoint, consistent with the existing title thumbnail route (same validation, same storage layout, same sniffing and size limits — do not invent a second convention).
- Accept `thumbnail_url` on episode patch if that fits the existing handler shape, so artwork can also be cleared or repointed.
- Serve it through the same `/api/v1/stream/thumbnails/...` path the rest of the artwork uses.
- Apply the same authorization as the other admin routes, and make sure the new route is covered by the admin-authorization table test — Codex 2 enumerates every admin route and will notice a new one that is not in the table.

I already checked the seed data so you do not have to: `seed/series/seed.json` carries 10 `thumbnail_url` fields, all at the **series** level, and every episode object holds only `episode_number`, `title`, `description` and `duration`. There is no episode-level image in the seed, so the importer is not dropping anything and the "No episode artwork" placeholder is the honest rendering for seeded content. This task is purely about closing the API gap so uploaded episodes *can* have artwork.

Report the endpoint shape so Codex 3 can wire the admin UI.

## Definition of done

- `go build ./... && go vet ./... && go test -race ./...` clean in every module; `go test -tags=integration ./...` passes twice.
- Upload artwork to an episode through the API, then confirm it comes back on the series detail read and renders through the streaming path.
- The new route added to the admin-authorization test table.
- A clear statement about whether the seed JSON carries episode images.

Do not commit. Do not touch `frontend/**`.

Work autonomously; do not stop to ask for confirmation.
