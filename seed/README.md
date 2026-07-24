# Netflix Clone Seed Data

This directory contains the initial catalog data and artwork used to start the
development of `netflix_clone`. It is intentionally kept separate from the
application so the same content can be reused by a database seeder, local API,
or frontend mock-data loader.

## Directory structure

```text
seed/
├── README.md
├── movies/
│   ├── README.md
│   ├── seed.json
│   └── data/
│       ├── attack.jpg
│       ├── creed.webp
│       ├── fast_and_furious.webp
│       ├── hell.webp
│       ├── karate_kid.webp
│       ├── like_children.webp
│       ├── norbit.webp
│       ├── scarface.webp
│       ├── spiderman.webp
│       └── war_machine.jpg
└── series/
    ├── README.md
    ├── seed.json
    └── data/
        ├── baki/
        ├── baki_hanma/
        ├── better_call_saul/
        ├── breaking_bad/
        ├── death_note/
        ├── hajime_no_ippo/
        ├── how_to_sell_drugs_online/
        ├── pablo_escobar/
        ├── sex_education/
        └── the_cage/
```

Each directory under `series/data/` contains the artwork for one series and a
small README that identifies the title. Movie artwork is currently stored
directly under `movies/data/`.

## Seed manifests

The catalog manifests live in:

- `movies/seed.json` for movies
- `series/seed.json` for series

The current series manifest uses this shape:

```json
{
  "series": [
    {
      "name": "Baki Hanma",
      "year_released": "2020",
      "number_of_seasons": "3"
    }
  ]
}
```

When adding records, keep the top-level collection name aligned with the
content type (`movies` or `series`) and use the same field names consistently.
The application or database seeder should resolve artwork paths relative to
this directory.

Example:

```text
series/data/baki_hanma/baki_hanma.jpg
movies/data/spiderman.webp
```

## Naming conventions

- Use lowercase `snake_case` for directories and filenames.
- Keep image extensions in the path (`.jpg`, `.webp`, or `.png`).
- Give each title a stable, unique name.
- Store series assets in `series/data/<series_name>/`.
- Store movie assets in `movies/data/`.
- Prefer landscape artwork with a consistent aspect ratio. The initial assets
  use a 16:9 ratio, primarily at `665 × 374` or `341 × 192`.

## Adding seed content

1. Add the title's image to the appropriate `data/` directory.
2. Add its metadata to the matching `seed.json` manifest.
3. Verify that the filename and the path referenced by the seeding code match
   exactly, including the extension.
4. Keep the JSON valid and avoid duplicate titles.
5. Run the application's seed command once it is available.

## Current status

This is starter content for development:

- Artwork is available for 10 movies and 10 series.
- `series/seed.json` currently contains one metadata record.
- `movies/seed.json` is currently empty and still needs its movie records.
- The application-specific import or database seed command has not yet been
  defined in this directory.

Do not treat this catalog as production data. Before production use, verify
metadata accuracy, image licensing, required attribution, and content rights.
