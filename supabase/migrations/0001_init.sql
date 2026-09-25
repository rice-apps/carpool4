create type ride_status as enum ('active', 'cancelled');

create table users
(
    id             uuid primary key references auth.users (id) on delete cascade,
    first_name     text           not null default '',
    last_name      text           not null default '',
    email          text           not null,
    phone          text           not null default '',
    created_at     timestamptz    not null default now(),
    updated_at     timestamptz    not null default now()
);

create table locations
(
    id      uuid primary key default gen_random_uuid(),
    title   text not null,
    address text not null unique
);

create table rides
(
    id                    uuid primary key     default gen_random_uuid(),
    departure_time        timestamptz not null,
    departure_location_id uuid        not null references locations (id),
    arrival_location_id   uuid        not null references locations (id),
    owner_id              uuid        not null references users (id),
    notes                 text        not null default '',
    capacity              integer     not null check (capacity > 0),
    status                ride_status not null default 'active',
    created_at            timestamptz not null default now(),
    updated_at            timestamptz not null default now(),
    constraint rides_locations_distinct check (departure_location_id <> arrival_location_id)
);

create index rides_departure_time_idx on rides (departure_time);
create index rides_departure_location_idx on rides (departure_location_id);
create index rides_arrival_location_idx on rides (arrival_location_id);
create index rides_status_idx on rides (status);
create index rides_owner_departure_id_idx on rides (owner_id, departure_time desc, id desc);

create table ride_occupants
(
    ride_id   uuid        not null references rides (id) on delete cascade,
    user_id   uuid        not null references users (id) on delete cascade,
    joined_at timestamptz not null default now(),
    primary key (ride_id, user_id)
);

create index ride_occupants_user_ride_idx on ride_occupants (user_id, ride_id);

-- Aggregate rider JSON has one definition so every ride read stays aligned.
create function public.ride_riders(target_ride_id uuid)
returns json
language sql
stable
set search_path = public, pg_temp
as $$
    select coalesce(
        json_agg(
            json_build_object(
                'id', u.id,
                'first_name', u.first_name,
                'last_name', u.last_name,
                'email', u.email,
                'phone', u.phone
            )
            order by ro.joined_at, u.id
        ),
        '[]'::json
    )
    from public.ride_occupants ro
    join public.users u on u.id = ro.user_id
    where ro.ride_id = target_ride_id;
$$;

revoke all on function public.ride_riders(uuid) from public;

-- The Go backend authenticates callers and authorizes access to application data.
alter table public.users enable row level security;
alter table public.locations enable row level security;
alter table public.rides enable row level security;
alter table public.ride_occupants enable row level security;

-- Include views and inherited PUBLIC grants as well as the four base tables.
revoke all on all tables in schema public from public;
alter default privileges revoke all on tables from public;
alter default privileges in schema public revoke all on tables from public;

-- Bare PostgreSQL installations need not contain Supabase's API roles.
do $$
begin
    if exists (select from pg_roles where rolname = 'anon') then
        revoke all on all tables in schema public from anon;
        alter default privileges revoke all on tables from anon;
        alter default privileges in schema public revoke all on tables from anon;
    end if;
    if exists (select from pg_roles where rolname = 'authenticated') then
        revoke all on all tables in schema public from authenticated;
        alter default privileges revoke all on tables from authenticated;
        alter default privileges in schema public revoke all on tables from authenticated;
    end if;
end $$;

-- Supabase Auth is the sole admission authority. Google populates
-- custom_claims.hd from its verified ID token before this hook runs.
create function public.hook_restrict_rice_google_signup(event jsonb)
returns jsonb
language plpgsql
stable
set search_path = ''
as $$
declare
    provider text := event->'user'->'app_metadata'->>'provider';
    hosted_domain text := event->'user'->'user_metadata'->'custom_claims'->>'hd';
begin
    if provider = 'google' and lower(hosted_domain) = 'rice.edu' then
        return '{}'::jsonb;
    end if;

    return jsonb_build_object(
        'error', jsonb_build_object(
            'http_code', 403,
            'message', 'A Rice Google account is required.'
        )
    );
end;
$$;

grant usage on schema public to supabase_auth_admin;
grant execute on function public.hook_restrict_rice_google_signup(jsonb)
    to supabase_auth_admin;
revoke execute on function public.hook_restrict_rice_google_signup(jsonb)
    from public, anon, authenticated;
