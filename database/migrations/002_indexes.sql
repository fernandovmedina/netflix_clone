alter table titles add column if not exists published boolean not null default false;

do $$ begin
  alter table watch_progress add constraint fk_wp_user foreign key (user_id) references users(id);
exception when duplicate_object then null; end $$;
do $$ begin
  alter table favorites add constraint fk_fav_user foreign key (user_id) references users(id);
exception when duplicate_object then null; end $$;

create index if not exists idx_titles_active on titles(id_title) where deleted_at is null;
create index if not exists idx_titles_type_active on titles(type) where deleted_at is null;
create index if not exists idx_seasons_series on seasons(id_series);
create index if not exists idx_episodes_season on episodes(id_season);
create index if not exists idx_episodes_season_number on episodes(id_season,episode_number);
create index if not exists idx_title_genres_genre on title_genres(id_genre);
create index if not exists idx_watch_progress_user on watch_progress(user_id);
create unique index if not exists uq_watch_progress_movie on watch_progress(user_id,id_movie) where id_movie is not null;
create unique index if not exists uq_watch_progress_episode on watch_progress(user_id,id_episode) where id_episode is not null;
create index if not exists idx_sessions_user on sessions(user_id);
create index if not exists idx_sessions_refresh_hash on sessions(refresh_token_hash);
create index if not exists idx_sessions_family on sessions(session_family);
create index if not exists idx_video_jobs_claim on video_jobs(status,created_at) where status in ('queued','leased');
create unique index if not exists uq_video_assets_movie on video_assets(id_movie) where id_movie is not null;
create unique index if not exists uq_video_assets_episode on video_assets(id_episode) where id_episode is not null;

