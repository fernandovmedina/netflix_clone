# Codex 1 — Round 4: catalog visibility gap + integration-suite data leakage

Two defects I found while verifying the series browse page in the browser. They are related, and the second is why the first was visible at all.

---

## 1. `[HIGH]` The public catalog lists titles whose asset is not `ready`

A brand-new normal user gets not-ready content in the public listing:

```
$ curl -b user.cookies 'http://localhost:8080/api/v1/titles?limit=200'
total titles returned: 43
integration-* titles exposed publicly: 11
{"id":161,"title":"integration-title-5276be53ca07","published":true,
 "movie_id":375,"asset_id":"199aa89b-7802-4562-9cb7-67492cf1cbdf"}
```

```sql
select t.id_title, t.published, va.status from titles t
  join movies m on m.id_title=t.id_title
  join video_assets va on va.id_movie=m.id_movie
 where t.title like 'integration-title-%';
--  161 | t | pending
--  159 | t | pending      (11 rows, all published with a pending asset)
```

In the UI this is worse than a listing bug: on `/home/series` one of these became the **featured hero**, rendering a raw UUID as the title with a greyed-out `Processing` button where Play should be. The user's brief requires that content not be exposed to normal users as playable until processing completes — playback is correctly gated, so this is not a security hole, but a consumer catalog must not list or feature content that cannot be played.

Fix: public read paths must require the joined asset to be `ready`, in SQL, on **every** path — list, single title, movie/series detail, search, the home rows, and whatever feeds the hero. The admin projection keeps showing `pending`/`processing`/`failed` as it does today.

**Note this contradicts your passing `TestCatalogVisibility`.** That test evidently covers only the unpublished case, not the not-ready case, which is exactly how this survived. Extend it so it fails without the fix — a published title with a `pending` asset must be absent from every public read path.

## 2. `[MEDIUM]` The integration suite leaves its fixtures in the database

The suite's own titles are accumulating in the seeded catalog — 43 titles returned where the seed provides 20. Phase 5 required each test to clean up after itself; media cleanup was added, but the `titles`/`movies`/`video_assets`/`video_jobs` rows are not removed. There are also older `claim-*` fixtures from worker-queue testing.

Fix the suite so every fixture it creates is removed in `t.Cleanup`, including on failure, and so a run leaves the catalog exactly as it found it. Assert that: capture the public title count before and after the suite and fail if it changed.

Then **write me a cleanup statement** for the rows already in the database — a single transactional SQL script, restricted to the fixture patterns (`integration-title-%`, `claim-%` and any other pattern your suites created), that removes the titles and their dependent movies/episodes/assets/jobs rows and the orphaned media directories. **Do not run it.** Put it in `.claude/tasks/cleanup-test-fixtures.sql`, tell me exactly how many rows it would affect, and I will get the user's approval before anything is executed — the seeded catalog is data they care about and I am not going to have it deleted on my own judgement.

---

## Definition of done

- `go build ./... && go vet ./... && go test -race ./...` clean in every module.
- `go test -tags=integration ./...` passes, twice consecutively, with the before/after title-count assertion in place.
- The reproduction above re-run and pasted: a published title with a `pending` asset must not appear in `/api/v1/titles`, the home rows, detail, or the hero feed.
- `.claude/tasks/cleanup-test-fixtures.sql` written but **not executed**, with a row count.

Do not commit. Do not touch `frontend/**` or `microservices/nginx/**`.

Work autonomously; do not stop to ask for confirmation.
