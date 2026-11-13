-- Live chat (ported from printforge `chats` / `chat_messages`, extended):
-- guests can chat (secret token, only its sha256 stored), read state is
-- shared by the whole team, conversations can be reopened.
create table chats (
  id                    bigserial primary key,
  user_id               uuid references users(id) on delete set null,
  customer_name         text not null check (char_length(customer_name) between 1 and 60),
  customer_email        text not null check (char_length(customer_email) <= 254),
  guest_token_hash      text,
  status                text not null default 'open' check (status in ('open','closed')),
  customer_last_read_at timestamptz,
  staff_last_read_at    timestamptz,
  last_message_at       timestamptz not null default now(),
  created_at            timestamptz not null default now()
);
create index chats_status_last_idx on chats(status, last_message_at desc);
create index chats_user_idx on chats(user_id);

create table chat_messages (
  id         bigserial primary key,
  chat_id    bigint not null references chats(id) on delete cascade,
  sender     text not null check (sender in ('customer','agent')),
  agent_id   uuid references users(id) on delete set null,
  body       text not null check (char_length(body) between 1 and 2000),
  created_at timestamptz not null default now()
);
create index chat_messages_chat_idx on chat_messages(chat_id, id);

alter table chats enable row level security;
alter table chat_messages enable row level security;
do $$
begin
  if exists (select 1 from pg_roles where rolname = 'anon') then
    execute 'revoke all on chats, chat_messages from anon';
  end if;
  if exists (select 1 from pg_roles where rolname = 'authenticated') then
    execute 'revoke all on chats, chat_messages from authenticated';
  end if;
end $$;

insert into email_templates (key, name, subject) values
  ('chat_reply', 'Live chat reply', 'Celestial Oud replied to your message')
on conflict (key) do nothing;
insert into admin_notifications (key, name, description) values
  ('new_chat_message', 'Live chat', 'Email the team when a customer writes in live chat and nobody has read it yet')
on conflict (key) do nothing;
