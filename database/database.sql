-- =========================================
-- Author: Fernando Vazquez
-- Version: v1.0.0
-- Date: Jul 08, 2026
-- DB: PostgreSQL (Supabase compatible)
-- =========================================

-- =========================
-- ENUMS
-- =========================
drop type if exists title_type cascade;
create type title_type as enum ('Movie', 'TV Show');

-- =========================
-- ACTORS
-- =========================
drop table if exists actors cascade;
create table actors (
  id_actor integer generated always as identity primary key,
  name text not null unique,
  created_at timestamptz not null default now(),
  updated_at timestamptz,
  deleted_at timestamptz
);

-- =========================
-- CATEGORIES
-- =========================
drop table if exists categories cascade;
create table categories (
  id_category integer generated always as identity primary key,
  name text not null unique,
  created_at timestamptz not null default now(),
  updated_at timestamptz,
  deleted_at timestamptz
);

-- =========================
-- GENRES
-- =========================
drop table if exists genres cascade;
create table genres (
  id_genre integer generated always as identity primary key,
  name text not null unique,
  created_at timestamptz not null default now(),
  updated_at timestamptz,
  deleted_at timestamptz
);

-- =========================
-- TITLES (MOVIES + SERIES)
-- =========================
drop table if exists titles cascade;
create table titles (
  id_title integer generated always as identity primary key,
  type title_type not null,
  title text not null,
  description text,
  director text,
  year_released smallint,
  thumbnail_url text,
  created_at timestamptz not null default now(),
  updated_at timestamptz,
  deleted_at timestamptz
);

create index idx_titles_active on titles(id_title)
where deleted_at is null;

-- =========================
-- MOVIES
-- =========================
drop table if exists movies cascade;
create table movies (
  id_movie integer generated always as identity primary key,
  id_title integer not null unique,
  duration integer,
  hls_manifest_path text null,
  created_at timestamptz not null default now(),
  updated_at timestamptz,
  deleted_at timestamptz,
  constraint fk_movies_title foreign key (id_title) references titles(id_title)
);

-- =========================
-- SERIES
-- =========================
drop table if exists series cascade;
create table series (
  id_series integer generated always as identity primary key,
  id_title integer not null unique,
  number_of_seasons integer,
  created_at timestamptz not null default now(),
  updated_at timestamptz,
  deleted_at timestamptz,
  constraint fk_series_title foreign key (id_title) references titles(id_title)
);

-- =========================
-- SEASONS
-- =========================
drop table if exists seasons cascade;
create table seasons (
  id_season integer generated always as identity primary key,
  id_series integer not null,
  season_number integer not null,
  number_of_episodes integer,
  created_at timestamptz not null default now(),
  updated_at timestamptz,
  deleted_at timestamptz,
  constraint fk_seasons_series foreign key (id_series) references series(id_series),
  unique (id_series, season_number)
);

create index idx_seasons_series on seasons(id_series);

-- =========================
-- EPISODES
-- =========================
drop table if exists episodes cascade;
create table episodes (
  id_episode integer generated always as identity primary key,
  id_season integer not null,
  episode_number integer not null,
  title text not null,
  description text,
  duration integer,
  thumbnail_url text,
  hls_manifest_path text null,
  created_at timestamptz not null default now(),
  updated_at timestamptz,
  deleted_at timestamptz,
  constraint fk_episodes_season foreign key (id_season) references seasons(id_season),
  unique (id_season, episode_number)
);

create index idx_episodes_season on episodes(id_season);

-- =========================
-- RELATIONS (MANY TO MANY)
-- =========================
drop table if exists title_actors cascade;
create table title_actors (
  id_title integer not null,
  id_actor integer not null,
  primary key (id_title, id_actor),
  constraint fk_ta_title foreign key (id_title) references titles(id_title),
  constraint fk_ta_actor foreign key (id_actor) references actors(id_actor)
);

drop table if exists title_categories cascade;
create table title_categories (
  id_title integer not null,
  id_category integer not null,
  primary key (id_title, id_category),
  constraint fk_tc_title foreign key (id_title) references titles(id_title),
  constraint fk_tc_category foreign key (id_category) references categories(id_category)
);

drop table if exists title_genres cascade;
create table title_genres (
  id_title integer not null,
  id_genre integer not null,
  primary key (id_title, id_genre),
  constraint fk_tg_title foreign key (id_title) references titles(id_title),
  constraint fk_tg_genre foreign key (id_genre) references genres(id_genre)
);

-- =========================
-- USER WATCH PROGRESS
-- =========================
drop table if exists watch_progress cascade;
create table watch_progress (
  id_progress integer generated always as identity primary key,
  user_id uuid not null,
  id_movie integer,
  id_episode integer,
  current_time_seconds integer not null,
  updated_at timestamptz not null default now(),
  constraint fk_wp_movie foreign key (id_movie) references movies(id_movie),
  constraint fk_wp_episode foreign key (id_episode) references episodes(id_episode),
  constraint chk_only_one_content check (
    (id_movie is not null and id_episode is null) or
    (id_movie is null and id_episode is not null)
  )
);

create index idx_watch_progress_user on watch_progress(user_id);

-- =========================
-- FAVORITES
-- =========================
drop table if exists favorites cascade;
create table favorites (
  user_id uuid not null,
  id_title integer not null,
  created_at timestamptz not null default now(),
  primary key (user_id, id_title),
  constraint fk_fav_title foreign key (id_title) references titles(id_title)
);
