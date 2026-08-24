# Movie Seed Data

This directory contains the movie catalog metadata and artwork used to seed the
Netflix clone during development.

## Structure

```text
movies/
├── README.md
├── seed.json
└── data/
    ├── attack.jpg
    ├── creed.webp
    └── ...
```

- `seed.json` contains the movie records under the top-level `movies` array.
- `data/` contains one landscape thumbnail for each movie.

The current catalog contains 10 movies. Its metadata is intended for local
development and may be fictionalized.

## Movie schema

Each item in `seed.json` follows this structure:

```json
{
  "name": "Attack",
  "video_source": "long",
  "year_released": 2022,
  "description": "A short summary of the movie.",
  "genres": ["Action", "Thriller", "Drama"],
  "cast": ["Actor One", "Actor Two", "Actor Three"],
  "director": "Director Name",
  "duration": 112,
  "thumbnail_url": "movies/data/attack.jpg"
}
```

| Field | Type | Description |
| --- | --- | --- |
| `name` | string | Unique display title. |
| `video_source` | string | Optional `short` (default) or `long`; selects which local clip in `seed/video/` the importer uses. Neither is committed — see [`seed/video/README.md`](../video/README.md). |
| `year_released` | integer | Four-digit release year. |
| `description` | string | English catalog summary. |
| `genres` | string[] | Genres used to classify the title. |
| `cast` | string[] | Main cast members. |
| `director` | string | Director's display name. |
| `duration` | integer | Runtime in minutes. |
| `thumbnail_url` | string | Artwork path relative to `seed/`. |

## Adding a movie

1. Add a landscape image to `data/` using a lowercase `snake_case` filename.
2. Add one object to the `movies` array in `seed.json`.
3. Set `thumbnail_url` to `movies/data/<filename>`, including its extension.
4. Keep numeric values as JSON numbers rather than quoted strings.
5. Ensure the title is unique and the JSON remains valid.

Example artwork path:

```text
movies/data/spiderman.webp
```

## Validation

From the repository root, validate the manifest syntax with:

```bash
node -e "JSON.parse(require('fs').readFileSync('seed/movies/seed.json'))"
```

Every image in `data/` should have exactly one matching movie record, and every
`thumbnail_url` should resolve to an existing file.

## Content notice

This catalog is development seed data, not a verified production dataset.
Confirm metadata accuracy, artwork licensing, and required attribution before
using it in a public or production environment.
