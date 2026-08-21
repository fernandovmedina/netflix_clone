alter table video_assets add column if not exists superseded_at timestamptz;

alter type processing_status add value if not exists 'superseded';

drop index if exists uq_video_assets_movie;
drop index if exists uq_video_assets_episode;

-- Listing the original live states avoids using the newly-added enum value in
-- this migration transaction while still making superseded rows historical.
create unique index if not exists uq_video_assets_movie
  on video_assets(id_movie)
  where id_movie is not null and status in ('pending', 'processing', 'ready', 'failed');

create unique index if not exists uq_video_assets_episode
  on video_assets(id_episode)
  where id_episode is not null and status in ('pending', 'processing', 'ready', 'failed');
