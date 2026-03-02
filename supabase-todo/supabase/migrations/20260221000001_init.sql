create function public.set_current_timestamp_updated_at()
returns trigger
set search_path = ''
language plpgsql
as $$
begin
  new.updated_at = now();
  return new;
end;
$$;

-- ─────────────────────────────────────────────────────────────────────────────
-- PROFILES
-- ─────────────────────────────────────────────────────────────────────────────
create table public.profiles (
  id          uuid references auth.users on delete cascade not null primary key,
  updated_at  timestamp with time zone,
  avatar_url  text
);

create function public.handle_new_user()
returns trigger
set search_path = ''
language plpgsql security definer
as $$
begin
  insert into public.profiles (id)
  values (new.id);
  return new;
end;
$$;

create trigger on_auth_user_created
  after insert on auth.users
  for each row execute procedure public.handle_new_user();

create trigger profiles_set_updated_at
  before update on public.profiles
  for each row execute procedure public.set_current_timestamp_updated_at();

-- RLS

alter table public.profiles enable row level security;

create policy "Users can view their own profile."
  on public.profiles for select
  to authenticated
  using ((select auth.uid()) = id);

create policy "Users can update their own profile."
  on public.profiles for update
  to authenticated
  using ((select auth.uid()) = id)
  with check ((select auth.uid()) = id);

-- CLS

revoke all on public.profiles from authenticated;
grant select on public.profiles to authenticated;
grant update (avatar_url) on public.profiles to authenticated;

-- ─────────────────────────────────────────────────────────────────────────────
-- TODOS
-- ─────────────────────────────────────────────────────────────────────────────
create table public.todos (
  id          bigint generated always as identity primary key,
  user_id     uuid references auth.users on delete cascade not null default auth.uid(),
  title       text not null check (char_length(title) > 0),
  description text,
  is_complete boolean not null default false,
  priority    text not null default 'medium' check (priority in ('low', 'medium', 'high')),
  due_date    date,
  inserted_at timestamp with time zone not null default now(),
  updated_at  timestamp with time zone not null default now()
);

create index todos_user_id_idx on public.todos (user_id);

create trigger todos_set_updated_at
  before update on public.todos
  for each row execute procedure public.set_current_timestamp_updated_at();


-- RLS

alter table public.todos enable row level security;

create policy "Users can view their own todos."
  on public.todos for select
  to authenticated
  using ((select auth.uid()) = user_id);

create policy "Users can insert their own todos."
  on public.todos for insert
  to authenticated
  with check ((select auth.uid()) = user_id);

create policy "Users can update their own todos."
  on public.todos for update
  to authenticated
  using ((select auth.uid()) = user_id)
  with check ((select auth.uid()) = user_id);

create policy "Users can delete their own todos."
  on public.todos for delete
  to authenticated
  using ((select auth.uid()) = user_id);


-- CLS

revoke all on public.todos from authenticated;
grant delete on public.todos to authenticated;
grant select (id, title, description, is_complete, priority, due_date, inserted_at) on public.todos to authenticated;
grant insert (title, description, is_complete, priority, due_date) on public.todos to authenticated;
grant update (title, description, is_complete, priority, due_date) on public.todos to authenticated;

-- ─────────────────────────────────────────────────────────────────────────────
-- STORAGE – avatars bucket
-- ─────────────────────────────────────────────────────────────────────────────
insert into storage.buckets (id, name, public)
  values ('avatars', 'avatars', true);

create policy "Avatar images are publicly accessible."
  on storage.objects for select
  using (bucket_id = 'avatars');

create policy "Authenticated users can upload an avatar."
  on storage.objects for insert
  to authenticated
  with check (bucket_id = 'avatars' and (select auth.uid())::text = (storage.foldername(name))[1]);

create policy "Users can update their own avatar."
  on storage.objects for update
  to authenticated
  using (bucket_id = 'avatars' and (select auth.uid())::text = (storage.foldername(name))[1]);

create policy "Users can delete their own avatar."
  on storage.objects for delete
  to authenticated
  using (bucket_id = 'avatars' and (select auth.uid())::text = (storage.foldername(name))[1]);
