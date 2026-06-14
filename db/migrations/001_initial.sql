create extension if not exists pgcrypto;

create table users (
  id uuid primary key default gen_random_uuid(),
  name text not null,
  email text not null unique,
  password_hash text not null,
  created_at timestamptz not null default now()
);

create table groups (
  id uuid primary key default gen_random_uuid(),
  name text not null,
  created_by uuid references users(id),
  created_at timestamptz not null default now()
);

create table people (
  id uuid primary key default gen_random_uuid(),
  display_name text not null,
  canonical_key text not null unique,
  created_at timestamptz not null default now()
);

create table group_memberships (
  id uuid primary key default gen_random_uuid(),
  group_id uuid not null references groups(id) on delete cascade,
  person_id uuid not null references people(id),
  joined_on date not null,
  left_on date,
  role text not null default 'member',
  check (left_on is null or left_on >= joined_on)
);

create index group_memberships_group_id_idx on group_memberships(group_id);
create index group_memberships_person_id_idx on group_memberships(person_id);

create table imports (
  id uuid primary key default gen_random_uuid(),
  group_id uuid not null references groups(id) on delete cascade,
  file_name text not null,
  status text not null default 'completed',
  rows_read integer not null default 0,
  created_by uuid references users(id),
  created_at timestamptz not null default now()
);

create table import_rows (
  id uuid primary key default gen_random_uuid(),
  import_id uuid not null references imports(id) on delete cascade,
  row_number integer not null,
  raw_payload jsonb not null,
  status text not null,
  unique(import_id, row_number)
);

create table import_anomalies (
  id uuid primary key default gen_random_uuid(),
  import_id uuid not null references imports(id) on delete cascade,
  row_number integer not null,
  code text not null,
  severity text not null,
  message text not null,
  policy text not null,
  action text not null,
  reviewed_by uuid references users(id),
  reviewed_at timestamptz
);

create table expenses (
  id uuid primary key default gen_random_uuid(),
  group_id uuid not null references groups(id) on delete cascade,
  import_id uuid references imports(id),
  source_row_number integer,
  expense_date date not null,
  description text not null,
  paid_by uuid not null references people(id),
  original_amount_paise bigint not null,
  original_currency char(3) not null,
  base_amount_paise bigint not null,
  base_currency char(3) not null default 'INR',
  split_type text not null,
  notes text,
  created_at timestamptz not null default now()
);

create index expenses_group_date_idx on expenses(group_id, expense_date);

create table expense_shares (
  id uuid primary key default gen_random_uuid(),
  expense_id uuid not null references expenses(id) on delete cascade,
  person_id uuid not null references people(id),
  share_amount_paise bigint not null,
  source_weight numeric,
  unique(expense_id, person_id)
);

create table settlements (
  id uuid primary key default gen_random_uuid(),
  group_id uuid not null references groups(id) on delete cascade,
  import_id uuid references imports(id),
  source_row_number integer,
  paid_on date not null,
  from_person_id uuid not null references people(id),
  to_person_id uuid not null references people(id),
  amount_paise bigint not null,
  currency char(3) not null default 'INR',
  notes text,
  created_at timestamptz not null default now()
);

create table exchange_rates (
  id uuid primary key default gen_random_uuid(),
  from_currency char(3) not null,
  to_currency char(3) not null,
  rate numeric(16, 6) not null,
  effective_on date not null,
  source text not null,
  unique(from_currency, to_currency, effective_on, source)
);
