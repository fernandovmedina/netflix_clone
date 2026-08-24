# Series Seed Data

This directory contains the series, season, and episode metadata plus the
artwork used to seed the Netflix clone during development.

## Structure

```text
series/
├── README.md
├── seed.json
└── data/
    ├── baki/
    │   ├── README.md
    │   └── baki.jpg
    ├── breaking_bad/
    │   ├── README.md
    │   └── breaking_bad.webp
    └── ...
```

- `seed.json` contains the records under the top-level `series` array.
- `data/<series_name>/` contains the thumbnail and a small title README.

The current catalog contains 10 series, 20 seasons, and 68 episodes. Its
metadata is intended for local development and may be fictionalized.

## Series schema

Each series includes title metadata and a nested list of seasons and episodes:

```json
{
  "name": "Baki",
  "video_source": "short",
  "year_released": 2018,
  "description": "A short summary of the series.",
  "genres": ["Action", "Martial Arts", "Anime"],
  "cast": ["Actor One", "Actor Two", "Actor Three"],
  "thumbnail_url": "series/data/baki/baki.jpg",
  "number_of_seasons": 1,
  "seasons": [
    {
      "season_number": 1,
      "number_of_episodes": 1,
      "episodes": [
        {
          "episode_number": 1,
          "title": "The Underground Gate",
          "description": "A short summary of the episode.",
          "duration": 24
        }
      ]
    }
  ]
}
```

### Series fields

| Field | Type | Description |
| --- | --- | --- |
| `name` | string | Unique display title. |
| `video_source` | string | Optional `short` (default) or `long`; applies to every episode in the series. |
| `year_released` | integer | Four-digit release year. |
| `description` | string | English catalog summary. |
| `genres` | string[] | Genres used to classify the title. |
| `cast` | string[] | Main cast members. |
| `thumbnail_url` | string | Artwork path relative to `seed/`. |
| `number_of_seasons` | integer | Number of items in `seasons`. |
| `seasons` | object[] | Ordered season records. |

### Season and episode fields

| Field | Type | Description |
| --- | --- | --- |
| `season_number` | integer | One-based season position. |
| `number_of_episodes` | integer | Number of items in `episodes`. |
| `episodes` | object[] | Ordered episode records. |
| `episode_number` | integer | One-based episode position within its season. |
| `title` | string | Episode display title. |
| `description` | string | English episode summary. |
| `duration` | integer | Episode runtime in minutes. |

## Adding a series

1. Create `data/<series_name>/` using lowercase `snake_case`.
2. Add the series thumbnail inside that directory and keep its extension in the
   referenced path.
3. Add one object to the `series` array in `seed.json`.
4. Add every season and episode in numerical order, starting at `1`.
5. Set `number_of_seasons` to the length of `seasons`.
6. Set each `number_of_episodes` to the length of that season's `episodes`.
7. Keep numeric values as JSON numbers rather than quoted strings.

Example artwork path:

```text
series/data/baki_hanma/baki_hanma.jpg
```

## Validation

From the repository root, validate the manifest syntax with:

```bash
node -e "JSON.parse(require('fs').readFileSync('seed/series/seed.json'))"
```

Also verify that series names are unique, episode numbers are sequential,
declared counts match their arrays, and every `thumbnail_url` resolves to an
existing image.

## Content notice

This catalog is development seed data, not a verified production dataset.
Confirm metadata accuracy, artwork licensing, and required attribution before
using it in a public or production environment.
