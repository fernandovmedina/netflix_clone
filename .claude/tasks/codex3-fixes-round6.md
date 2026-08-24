# Codex 3 — Round 6: episode rows collapse into a sliver in the series modal

Reported by the project owner ("when browsing a series, the chapters look weird"), reproduced and diagnosed by me in the browser. Scope: `frontend/**`.

## What the owner sees

Open any series from `/home/series`. Every episode row renders with the title, duration, description and Play button crammed into a ~160 px column that wraps every few words — "No Rules / Tonight", "The Weight of / Strength", "24 / min" — while roughly **600 px of the modal to the right sits completely empty**.

## Root cause

`frontend/components/TitleModal.tsx:131`

```jsx
<div className="grid grid-cols-[2rem_1fr] gap-3 py-5 sm:grid-cols-[2rem_10rem_1fr] sm:items-center">
  <span className="text-xl font-bold">{episode.episode_number ?? …}</span>
  {episode.thumbnail_url && (                       // ← conditional grid child
    <div className="relative hidden aspect-video overflow-hidden rounded sm:block">…</div>
  )}
  <div>…title, duration, description, Play button…</div>
</div>
```

At `sm:` and above the template declares **three** columns — `2rem 10rem 1fr` — where the middle `10rem` is the episode thumbnail. But the thumbnail is a **conditional** child. When `thumbnail_url` is empty the element is never rendered, so the content div slides up into the 160 px thumbnail column and the `1fr` column is left empty.

Measured live in the DOM:

```
row grid-template-columns: 32px 160px 600px     row width 816px
content div width:         160px                 ← should be in the 600px column
```

And it is not an edge case — **no seeded episode has a thumbnail at all**:

```sql
select count(*) as episodes,
       count(thumbnail_url) filter (where coalesce(thumbnail_url,'') <> '') as with_thumb
  from episodes;
--  68 | 0
```

So every episode row in the app is currently broken. It presumably looked fine when it was written against data that had thumbnails.

## Fix

Make the grid independent of whether the thumbnail exists. Any of these is acceptable — pick the one that fits the component best:

- Always render the thumbnail **cell** at `sm:` and put the conditional `<Image>` inside it, with a neutral placeholder background when there is no artwork. Keeps rows aligned with each other, which matters when only some episodes have artwork.
- Or drop to a two-column template when the season has no episode artwork.
- Or let the content cell span the remaining columns explicitly.

Requirements either way:

- The title must have room to sit on one line at desktop widths; the duration must not wrap to "24 / min".
- Rows must stay aligned with one another whether or not individual episodes have artwork — test a season where **some** episodes have a thumbnail and some do not, since that mixed case is what a real catalog looks like. Set a thumbnail on one episode through the admin API to create it.
- Re-check 375 / 768 / 1024 / 1440. The mobile two-column layout is currently fine, so do not regress it.
- The "This episode is still processing." state and the Play button must keep working.

While you are in the modal, look for the same conditional-grid-child mistake elsewhere in the component — a fixed grid template with an optional child is the kind of thing that repeats.

## Definition of done

- `pnpm build`, `pnpm lint`, `pnpm exec tsc --noEmit` clean.
- Screenshots or measured widths at all four viewport widths showing the row filling the modal, for a season with no artwork **and** a season with mixed artwork.
- Confirm with real DOM measurements, not by reading the classes — that is how this survived the responsive pass.

Note the frontend container on port 3000 belongs to the owner and is on the current build; use your own `FRONTEND_PORT` instance for testing, and tell me when it needs rebuilding.

Do not commit. Work autonomously; do not stop to ask for confirmation.
