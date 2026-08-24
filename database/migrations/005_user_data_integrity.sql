do $$ begin
  alter table profiles add constraint chk_profiles_name_length
    check (char_length(name) between 1 and 50) not valid;
exception when duplicate_object then null;
end $$;

do $$ begin
  alter table discounts add constraint chk_discounts_nonnegative_value
    check (value >= 0) not valid;
exception when duplicate_object then null;
end $$;

do $$ begin
  alter table discounts add constraint chk_discounts_percent_range
    check (kind <> 'percent' or value <= 100) not valid;
exception when duplicate_object then null;
end $$;
