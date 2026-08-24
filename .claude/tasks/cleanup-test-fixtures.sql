\set ON_ERROR_STOP on

-- REVIEW BEFORE RUNNING. This intentionally matches test-only title prefixes.
-- Current read-only inventory (2026-08-24): 463 database rows and 6 media
-- directories: 122 titles, 120 movies, 1 series, 2 seasons, 3 episodes,
-- 123 assets, 63 jobs, 27 progress rows, 2 favorites, 3 HLS directories,
-- and 3 source directories. Join-table counts are currently zero.

begin;

create temporary table cleanup_fixture_titles on commit drop as
select id_title
from titles
where title like any(array[
  'integration-title-%', 'integration-series-%', 'integration-upload-%',
  'claim-%', 'e2e-%', 'synthetic-%', 'unpublished-stream-%', 'user-test-%',
  'visibility-%', 'series-hierarchy-%', 'movie-id-contract-%',
  'asset-state-%', 'limit-%', 'reupload-%'
]);

create temporary table cleanup_fixture_movies on commit drop as
select id_movie from movies where id_title in (select id_title from cleanup_fixture_titles);

create temporary table cleanup_fixture_series on commit drop as
select id_series from series where id_title in (select id_title from cleanup_fixture_titles);

create temporary table cleanup_fixture_seasons on commit drop as
select id_season from seasons where id_series in (select id_series from cleanup_fixture_series);

create temporary table cleanup_fixture_episodes on commit drop as
select id_episode from episodes where id_season in (select id_season from cleanup_fixture_seasons);

create temporary table cleanup_fixture_assets on commit drop as
select id from video_assets
where id_movie in (select id_movie from cleanup_fixture_movies)
   or id_episode in (select id_episode from cleanup_fixture_episodes);

-- psql writes this client-side list for the explicit media cleanup below.
\copy (select id from cleanup_fixture_assets order by id) to '/tmp/netflix-clone-fixture-assets.txt'

delete from video_jobs where asset_id in (select id from cleanup_fixture_assets);
delete from video_assets where id in (select id from cleanup_fixture_assets);
delete from watch_progress
where id_movie in (select id_movie from cleanup_fixture_movies)
   or id_episode in (select id_episode from cleanup_fixture_episodes);
delete from favorites where id_title in (select id_title from cleanup_fixture_titles);
delete from title_actors where id_title in (select id_title from cleanup_fixture_titles);
delete from title_categories where id_title in (select id_title from cleanup_fixture_titles);
delete from title_genres where id_title in (select id_title from cleanup_fixture_titles);
delete from episodes where id_episode in (select id_episode from cleanup_fixture_episodes);
delete from seasons where id_season in (select id_season from cleanup_fixture_seasons);
delete from series where id_series in (select id_series from cleanup_fixture_series);
delete from movies where id_movie in (select id_movie from cleanup_fixture_movies);
delete from titles where id_title in (select id_title from cleanup_fixture_titles);

commit;

-- Run psql from the repository root so docker compose resolves this stack.
-- Every destructive target is an asset UUID captured before the transaction.
\! set -eu; while IFS= read -r asset_id; do if ! printf '%s\n' "$asset_id" | grep -Eq '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'; then echo "refusing invalid asset id: $asset_id" >&2; exit 1; fi; docker compose exec -T worker rm -rf -- "/media/hls/$asset_id" "/media/sources/$asset_id"; done < /tmp/netflix-clone-fixture-assets.txt
\! rm -f /tmp/netflix-clone-fixture-assets.txt
