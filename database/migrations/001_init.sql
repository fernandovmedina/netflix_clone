create extension if not exists citext;
create extension if not exists pgcrypto;

create table if not exists schema_migrations (
  version text primary key,
  applied_at timestamptz not null default now()
);

do $$ begin create type title_type as enum ('Movie', 'TV Show'); exception when duplicate_object then null; end $$;
do $$ begin create type user_role as enum ('user', 'admin'); exception when duplicate_object then null; end $$;
do $$ begin create type processing_status as enum ('pending', 'processing', 'ready', 'failed'); exception when duplicate_object then null; end $$;
do $$ begin create type job_status as enum ('queued', 'leased', 'done', 'failed'); exception when duplicate_object then null; end $$;
do $$ begin create type discount_kind as enum ('percent', 'fixed'); exception when duplicate_object then null; end $$;
do $$ begin create type payment_method as enum ('card', 'oxxo'); exception when duplicate_object then null; end $$;
do $$ begin create type payment_status as enum ('pending', 'paid', 'expired', 'failed'); exception when duplicate_object then null; end $$;

create table if not exists users (
  id uuid primary key default gen_random_uuid(), email citext not null unique,
  password_hash text, name text not null default '', role user_role not null default 'user',
  email_verified boolean not null default false, google_sub text unique,
  created_at timestamptz not null default now(), updated_at timestamptz, deleted_at timestamptz
);
create table if not exists actors (
  id_actor integer generated always as identity primary key, name text not null unique,
  created_at timestamptz not null default now(), updated_at timestamptz, deleted_at timestamptz
);
create table if not exists categories (
  id_category integer generated always as identity primary key, name text not null unique,
  created_at timestamptz not null default now(), updated_at timestamptz, deleted_at timestamptz
);
create table if not exists genres (
  id_genre integer generated always as identity primary key, name text not null unique,
  created_at timestamptz not null default now(), updated_at timestamptz, deleted_at timestamptz
);
create table if not exists titles (
  id_title integer generated always as identity primary key, type title_type not null, title text not null,
  description text, director text, year_released smallint, thumbnail_url text,
  published boolean not null default false,
  created_at timestamptz not null default now(), updated_at timestamptz, deleted_at timestamptz
);
create table if not exists movies (
  id_movie integer generated always as identity primary key, id_title integer not null unique references titles(id_title),
  duration integer, hls_manifest_path text,
  created_at timestamptz not null default now(), updated_at timestamptz, deleted_at timestamptz
);
create table if not exists series (
  id_series integer generated always as identity primary key, id_title integer not null unique references titles(id_title),
  number_of_seasons integer, created_at timestamptz not null default now(), updated_at timestamptz, deleted_at timestamptz
);
create table if not exists seasons (
  id_season integer generated always as identity primary key, id_series integer not null references series(id_series),
  season_number integer not null, number_of_episodes integer,
  created_at timestamptz not null default now(), updated_at timestamptz, deleted_at timestamptz,
  unique (id_series, season_number)
);
create table if not exists episodes (
  id_episode integer generated always as identity primary key, id_season integer not null references seasons(id_season),
  episode_number integer not null, title text not null, description text, duration integer, thumbnail_url text,
  hls_manifest_path text, created_at timestamptz not null default now(), updated_at timestamptz, deleted_at timestamptz,
  unique (id_season, episode_number)
);
create table if not exists title_actors (
  id_title integer not null references titles(id_title), id_actor integer not null references actors(id_actor), primary key (id_title,id_actor)
);
create table if not exists title_categories (
  id_title integer not null references titles(id_title), id_category integer not null references categories(id_category), primary key (id_title,id_category)
);
create table if not exists title_genres (
  id_title integer not null references titles(id_title), id_genre integer not null references genres(id_genre), primary key (id_title,id_genre)
);
create table if not exists watch_progress (
  id_progress integer generated always as identity primary key, user_id uuid not null references users(id),
  id_movie integer references movies(id_movie), id_episode integer references episodes(id_episode),
  current_time_seconds integer not null, updated_at timestamptz not null default now(),
  constraint chk_only_one_content check (num_nonnulls(id_movie,id_episode)=1)
);
create table if not exists favorites (
  user_id uuid not null references users(id), id_title integer not null references titles(id_title),
  created_at timestamptz not null default now(), primary key (user_id,id_title)
);
create table if not exists sessions (
  id uuid primary key default gen_random_uuid(), user_id uuid not null references users(id) on delete cascade,
  session_family uuid not null, refresh_token_hash text not null unique, rotated_from uuid references sessions(id),
  user_agent text, ip inet, expires_at timestamptz not null, revoked_at timestamptz,
  created_at timestamptz not null default now()
);
create table if not exists oauth_states (
  state text primary key, code_verifier text not null, redirect_to text,
  expires_at timestamptz not null, consumed_at timestamptz
);
create table if not exists video_assets (
  id uuid primary key default gen_random_uuid(), kind text not null,
  id_movie integer references movies(id_movie), id_episode integer references episodes(id_episode),
  status processing_status not null default 'pending', source_path text, manifest_path text,
  duration_seconds numeric(10,3), source_width integer, source_height integer, source_fps numeric(6,3),
  qualities jsonb not null default '[]'::jsonb, size_bytes bigint, error text,
  created_at timestamptz not null default now(), updated_at timestamptz,
  constraint chk_asset_target check (num_nonnulls(id_movie,id_episode)=1),
  constraint chk_asset_kind check ((kind='movie' and id_movie is not null) or (kind='episode' and id_episode is not null))
);
create table if not exists video_jobs (
  id uuid primary key default gen_random_uuid(), asset_id uuid not null references video_assets(id) on delete cascade,
  status job_status not null default 'queued', attempts integer not null default 0, max_attempts integer not null default 3,
  locked_by text, locked_at timestamptz, lease_expires_at timestamptz, last_error text,
  created_at timestamptz not null default now(), updated_at timestamptz
);
create table if not exists plans (
  id integer generated always as identity primary key, code text not null unique, name text not null,
  price numeric(10,2) not null, currency text not null default 'MXN', quality text not null,
  max_streams integer not null, active boolean not null default true
);
create table if not exists discounts (
  id integer generated always as identity primary key, code citext not null unique, kind discount_kind not null,
  value numeric(10,2) not null, max_redemptions integer, redemption_count integer not null default 0,
  per_user_limit integer not null default 1, starts_at timestamptz, expires_at timestamptz,
  active boolean not null default true, created_at timestamptz not null default now()
);
create table if not exists payments (
  id uuid primary key default gen_random_uuid(), user_id uuid not null references users(id), plan_id integer not null references plans(id),
  method payment_method not null, status payment_status not null, subtotal numeric(10,2) not null,
  discount_id integer references discounts(id), discount_amount numeric(10,2) not null default 0,
  total numeric(10,2) not null, currency text not null default 'MXN', reference text unique,
  card_last4 text, card_brand text, expires_at timestamptz, paid_at timestamptz,
  simulated boolean not null default true, created_at timestamptz not null default now(), updated_at timestamptz
);
create table if not exists discount_redemptions (
  id uuid primary key default gen_random_uuid(), discount_id integer not null references discounts(id),
  user_id uuid not null references users(id), payment_id uuid references payments(id),
  created_at timestamptz not null default now(), unique(discount_id,user_id)
);
create table if not exists subscriptions (
  id uuid primary key default gen_random_uuid(), user_id uuid unique references users(id), plan_id integer references plans(id),
  status text, current_period_end timestamptz, created_at timestamptz not null default now(), updated_at timestamptz
);
create table if not exists profiles (
  id uuid primary key default gen_random_uuid(), user_id uuid not null references users(id) on delete cascade,
  name text not null, avatar text, is_kids boolean not null default false, created_at timestamptz not null default now()
);

