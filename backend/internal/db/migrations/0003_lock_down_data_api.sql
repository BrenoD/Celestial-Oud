-- Lock the schema away from Supabase's auto-generated Data API (PostgREST).
--
-- The Go API connects as the table owner and is the only client of this
-- database; it does not use Supabase Auth, the anon/publishable key or RLS
-- policies. Without this, anyone holding the public key could read or write
-- tables such as users (password hashes) through https://<project>.supabase.co/rest/v1.
--
-- 1. RLS on every table with no policies => PostgREST roles see nothing.
--    The owner role bypasses RLS, so the API is unaffected.
-- 2. Revoke the grants Supabase gives anon/authenticated, now and for future
--    tables. Guarded so the migration also runs on plain Postgres (CI, local).

do $$
declare
  t record;
begin
  for t in select tablename from pg_tables where schemaname = 'public' loop
    execute format('alter table public.%I enable row level security', t.tablename);
  end loop;

  if exists (select 1 from pg_roles where rolname = 'anon') then
    execute 'revoke all on all tables in schema public from anon';
    execute 'revoke all on all sequences in schema public from anon';
    execute 'revoke all on all functions in schema public from anon';
    execute 'alter default privileges in schema public revoke all on tables from anon';
    execute 'alter default privileges in schema public revoke all on sequences from anon';
    execute 'alter default privileges in schema public revoke all on functions from anon';
  end if;

  if exists (select 1 from pg_roles where rolname = 'authenticated') then
    execute 'revoke all on all tables in schema public from authenticated';
    execute 'revoke all on all sequences in schema public from authenticated';
    execute 'revoke all on all functions in schema public from authenticated';
    execute 'alter default privileges in schema public revoke all on tables from authenticated';
    execute 'alter default privileges in schema public revoke all on sequences from authenticated';
    execute 'alter default privileges in schema public revoke all on functions from authenticated';
  end if;
end $$;
