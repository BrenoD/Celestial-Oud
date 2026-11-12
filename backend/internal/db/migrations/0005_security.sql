-- Security hardening (audit 2026-11).

-- Orders: guests read their order with a secret token issued at checkout
-- (only its sha256 is stored) or with the order email (rate limited).
alter table orders add column if not exists access_token_hash text;
create index if not exists orders_email_lower_idx on orders (lower(email));

-- Discounts: no negative values, percent capped, optional per-customer limit.
update discount_codes set value = 0 where value < 0;
update discount_codes set value = 100 where type = 'percent' and value > 100;
alter table discount_codes add column if not exists per_customer_limit int check (per_customer_limit is null or per_customer_limit > 0);
alter table discount_codes drop constraint if exists discount_codes_value_check;
alter table discount_codes add constraint discount_codes_value_check
  check (value >= 0 and (type <> 'percent' or value <= 100));

-- The welcome code promised by the newsletter: one use per customer.
insert into discount_codes (code, type, value, per_customer_limit, active)
values ('WELCOME10', 'percent', 10, 1, true)
on conflict (code) do update set per_customer_limit = coalesce(discount_codes.per_customer_limit, 1);

-- Refresh-token families: reusing a rotated token revokes the whole family.
alter table refresh_tokens add column if not exists family_id uuid;
update refresh_tokens set family_id = id where family_id is null;
alter table refresh_tokens alter column family_id set not null;
create index if not exists refresh_tokens_family_idx on refresh_tokens(family_id);

-- Newsletter double opt-in (PECR) and per-subscriber unsubscribe tokens.
alter table newsletter_subs add column if not exists confirm_token_hash text;
alter table newsletter_subs add column if not exists unsubscribe_token text;
update newsletter_subs set unsubscribe_token = replace(gen_random_uuid()::text, '-', '') || replace(gen_random_uuid()::text, '-', '') where unsubscribe_token is null;
insert into email_templates (key, name, subject) values
  ('newsletter_confirm', 'Newsletter confirmation', 'Please confirm your subscription')
on conflict (key) do nothing;

-- Idempotency keys are bound to the request that created them.
alter table idempotency_keys add column if not exists fingerprint text;

-- Shared rate-limit counters (all API instances see the same numbers).
create unlogged table if not exists rate_limits (
  key          text primary key,
  window_start timestamptz not null,
  hits         int not null
);
alter table rate_limits enable row level security;

do $$
begin
  if exists (select 1 from pg_roles where rolname = 'anon') then
    execute 'revoke all on rate_limits from anon';
  end if;
  if exists (select 1 from pg_roles where rolname = 'authenticated') then
    execute 'revoke all on rate_limits from authenticated';
  end if;
end $$;
