create table if not exists user_rate_limits (
  user_id uuid not null references users(id) on delete cascade,
  action text not null,
  window_start timestamptz not null,
  request_count integer not null default 1,
  primary key (user_id, action, window_start),
  constraint chk_user_rate_limits_count check (request_count > 0)
);

create index if not exists idx_user_rate_limits_expiry on user_rate_limits(window_start);
