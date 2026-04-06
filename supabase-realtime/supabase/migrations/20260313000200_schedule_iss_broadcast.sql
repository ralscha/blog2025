do $$
begin
  if not exists (
    select 1
    from pg_extension
    where extname = 'pg_cron'
  ) then
    execute 'create extension pg_cron with schema pg_catalog';
  end if;
end;
$$;

do $$
begin
  if not exists (
    select 1
    from pg_extension
    where extname = 'pg_net'
  ) then
    execute 'create extension pg_net';
  end if;
end;
$$;

create or replace function public.iss_broadcast_base_url()
returns text
language plpgsql
security definer
set search_path = public
as $$
declare
  configured_url text;
begin
  begin
    select decrypted_secret
    into configured_url
    from vault.decrypted_secrets
    where name = 'project_url'
    limit 1;
  exception
    when undefined_table or invalid_schema_name then
      configured_url := null;
  end;

  return coalesce(configured_url, 'http://api.supabase.internal:8000');
end;
$$;

create or replace function public.iss_broadcast_auth_header()
returns text
language plpgsql
security definer
set search_path = public
as $$
declare
  configured_anon_key text;
begin
  begin
    select decrypted_secret
    into configured_anon_key
    from vault.decrypted_secrets
    where name = 'anon_key'
    limit 1;
  exception
    when undefined_table or invalid_schema_name then
      configured_anon_key := null;
  end;

  if configured_anon_key is null or configured_anon_key = '' then
    raise exception 'Missing Vault secret anon_key required for iss-broadcast cron auth';
  end if;

  return 'Bearer ' || configured_anon_key;
end;
$$;

do $$
begin
  if exists (
    select 1
    from cron.job
    where jobname = 'iss-broadcast-every-minute'
  ) then
    perform cron.unschedule('iss-broadcast-every-minute');
  end if;
end;
$$;

select cron.schedule(
  'iss-broadcast-every-minute',
  '* * * * *',
  $$
  select net.http_post(
    url := public.iss_broadcast_base_url() || '/functions/v1/iss-broadcast',
    headers := jsonb_build_object(
      'Content-Type', 'application/json',
      'Authorization', public.iss_broadcast_auth_header()
    ),
    body := '{}'::jsonb,
    timeout_milliseconds := 10000
  );
  $$
);