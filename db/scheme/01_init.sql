-- 01_init.sql
CREATE DATABASE mafia_events_db OWNER postgres;

create table if not exists users (
  id bigserial primary key,
  tg_user_id bigint not null unique,
  tg_chat_id bigint not null,
  created_at timestamptz not null default now()
);

create table if not exists events (
  id bigserial primary key,
  title text not null,
  description text,
  starts_at timestamptz not null
);

create index if not exists idx_events_starts_at on events(starts_at);

create table if not exists registrations (
  id bigserial primary key,
  user_id bigint not null references users(id) on delete cascade,
  event_id bigint not null references events(id) on delete cascade,
  name text not null,
  nickname text not null,
  phone text not null,
  status text not null default 'active', -- active | cancelled
  reminder24_sent boolean not null default false,
  reminder1_sent boolean not null default false,
  created_at timestamptz not null default now(),
  unique(user_id, event_id) -- один пользователь = одна активная запись на событие
);

ALTER TABLE registrations
DROP CONSTRAINT registrations_user_id_event_id_key;

CREATE UNIQUE INDEX registrations_user_event_active_idx
ON registrations(user_id, event_id)
WHERE status = 'active';

ALTER TABLE users
    ADD COLUMN telegram_id BIGINT UNIQUE,
    ADD COLUMN chat_id BIGINT,
    ADD COLUMN name TEXT,
    ADD COLUMN nickname TEXT,
    ADD COLUMN phone TEXT,
    ADD COLUMN updated_at TIMESTAMP DEFAULT now();

UPDATE users SET telegram_id = tg_user_id;
UPDATE users SET chat_id = tg_chat_id;

ALTER TABLE users
    DROP COLUMN tg_user_id RESTRICT,
    DROP COLUMN tg_chat_id RESTRICT,
    ALTER COLUMN telegram_id SET NOT NULL,
    ALTER COLUMN chat_id SET NOT NULL;


ALTER TABLE registrations
    DROP COLUMN name,
    DROP COLUMN nickname,
    DROP COLUMN phone;

-- Для демонстрации можете закинуть пару мероприятий:
insert into events(title, description, starts_at)
values
 ('Турнир мясорубка', 'Турнир без правил', now() + interval '2 days'),
 ('Вечер дружеских игр', 'Казуальные матчи', now() + interval '26 hours')
on conflict do nothing;
