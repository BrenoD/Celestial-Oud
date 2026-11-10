-- Celestial Oud — core schema. Money is integer pence. See docs/DATA-MODEL.md.
create extension if not exists pgcrypto;
create extension if not exists pg_trgm;

create or replace function set_updated_at() returns trigger language plpgsql as $$
begin new.updated_at = now(); return new; end $$;

-- ── Identity ─────────────────────────────────────────────────────────────────
create table users (
  id                uuid primary key default gen_random_uuid(),
  email             text not null,
  email_norm        text generated always as (lower(email)) stored,
  password_hash     text,
  first_name        text not null default '',
  last_name         text not null default '',
  phone             text,
  role              text not null default 'customer'
                    check (role in ('customer','viewer','support','manager','admin')),
  tier              text not null default 'Bronze',
  points            int  not null default 0,
  marketing_email   boolean not null default false,
  marketing_sms     boolean not null default false,
  marketing_whatsapp boolean not null default false,
  email_verified_at timestamptz,
  last_login_at     timestamptz,
  created_at        timestamptz not null default now(),
  updated_at        timestamptz not null default now()
);
create unique index users_email_norm_idx on users(email_norm);
create index users_role_idx on users(role);
create trigger users_updated_at before update on users for each row execute function set_updated_at();

create table refresh_tokens (
  id          uuid primary key default gen_random_uuid(),
  user_id     uuid not null references users(id) on delete cascade,
  token_hash  text not null unique,
  user_agent  text,
  expires_at  timestamptz not null,
  revoked_at  timestamptz,
  created_at  timestamptz not null default now()
);
create index refresh_tokens_user_idx on refresh_tokens(user_id);
create index refresh_tokens_expires_idx on refresh_tokens(expires_at);

create table password_resets (
  id          uuid primary key default gen_random_uuid(),
  user_id     uuid not null references users(id) on delete cascade,
  token_hash  text not null unique,
  expires_at  timestamptz not null,
  used_at     timestamptz,
  created_at  timestamptz not null default now()
);

create table addresses (
  id          bigserial primary key,
  user_id     uuid not null references users(id) on delete cascade,
  label       text,
  first_name  text not null,
  last_name   text not null,
  line1       text not null,
  line2       text,
  city        text not null,
  postcode    text not null,
  country     text not null default 'GB',
  phone       text,
  is_default  boolean not null default false,
  created_at  timestamptz not null default now()
);
create index addresses_user_idx on addresses(user_id);

create table audit_log (
  id          bigserial primary key,
  actor_id    uuid references users(id) on delete set null,
  action      text not null,
  entity      text not null,
  entity_id   text,
  diff        jsonb,
  created_at  timestamptz not null default now()
);
create index audit_log_entity_idx on audit_log(entity, entity_id);

-- ── Catalogue ────────────────────────────────────────────────────────────────
create table collections (
  id          bigserial primary key,
  slug        text not null unique,
  name        text not null,
  description text,
  image_url   text,
  sort        int not null default 0,
  visible     boolean not null default true,
  created_at  timestamptz not null default now(),
  updated_at  timestamptz not null default now()
);
create trigger collections_updated_at before update on collections for each row execute function set_updated_at();

create table products (
  id                bigserial primary key,
  slug              text not null unique,
  name              text not null,
  tagline           text not null default '',
  notes_line        text not null default '',
  family            text not null default 'woody-oud'
                    check (family in ('woody-oud','amber-oriental','floral','fresh-musk','gourmand','spicy')),
  concentration     text not null default 'Eau de Parfum',
  wear              text[] not null default '{}',
  tags              text[] not null default '{}',        -- bestseller,new,top-rated,gift,set,offer,oud
  badge             text,
  images            jsonb not null default '[]',
  pyramid           jsonb not null default '{"top":[],"heart":[],"base":[]}',
  profile           jsonb not null default '{"intensity":50,"longevity":50,"sweetness":50,"freshness":50}',
  best_for          text not null default '',
  description       text not null default '',
  how_to_wear       text not null default '',
  ingredients_inci  text not null default '',
  allergens         text[] not null default '{}',
  responsible_person text,
  cpsr_ref          text,
  scpn_ref          text,
  status            text not null default 'draft' check (status in ('draft','active','archived')),
  sort              int not null default 0,
  created_at        timestamptz not null default now(),
  updated_at        timestamptz not null default now()
);
create index products_status_idx on products(status);
create index products_tags_idx on products using gin(tags);
create index products_name_trgm_idx on products using gin(name gin_trgm_ops);
create trigger products_updated_at before update on products for each row execute function set_updated_at();

create table product_sizes (
  id                bigserial primary key,
  product_id        bigint not null references products(id) on delete cascade,
  sku               text not null unique,
  label             text not null,
  ml                int  not null default 0,
  price             int  not null check (price >= 0),
  compare_at_price  int  check (compare_at_price is null or compare_at_price >= 0),
  stock             int  not null default 0 check (stock >= 0),
  is_sample         boolean not null default false,
  weight_g          int,
  sort              int not null default 0,
  created_at        timestamptz not null default now()
);
create index product_sizes_product_idx on product_sizes(product_id);

create table product_collections (
  product_id    bigint not null references products(id) on delete cascade,
  collection_id bigint not null references collections(id) on delete cascade,
  sort          int not null default 0,
  primary key (product_id, collection_id)
);

create table notes (
  id    bigserial primary key,
  slug  text not null unique,
  name  text not null
);

create table product_notes (
  product_id bigint not null references products(id) on delete cascade,
  note_id    bigint not null references notes(id) on delete cascade,
  tier       text not null check (tier in ('top','heart','base','key')),
  primary key (product_id, note_id, tier)
);

create table scheduled_promotions (
  id            bigserial primary key,
  product_id    bigint references products(id) on delete cascade,
  collection_id bigint references collections(id) on delete cascade,
  percent_off   int not null check (percent_off between 1 and 90),
  starts_at     timestamptz not null,
  ends_at       timestamptz not null,
  applied_at    timestamptz,
  reverted_at   timestamptz,
  created_at    timestamptz not null default now(),
  check (product_id is not null or collection_id is not null)
);
create index scheduled_promotions_window_idx on scheduled_promotions(starts_at, ends_at);

create table discount_codes (
  code          text primary key,
  type          text not null default 'percent' check (type in ('percent','fixed','free_delivery')),
  value         int  not null default 0,          -- percent (0-100) or pence
  min_subtotal  int  not null default 0,
  max_uses      int,
  uses          int  not null default 0,
  starts_at     timestamptz,
  ends_at       timestamptz,
  active        boolean not null default true,
  created_at    timestamptz not null default now()
);

-- ── Commerce ─────────────────────────────────────────────────────────────────
create table delivery_methods (
  id            text primary key,                  -- standard, next-day, express
  name          text not null,
  eta           text not null,
  price         int  not null default 0,
  free_over     int,                               -- pence; null = never free
  carrier       text,
  service_code  text,
  active        boolean not null default true,
  sort          int not null default 0
);

create table tax_rules (
  id      bigserial primary key,
  region  text not null,
  rate    numeric(5,2) not null default 20,
  status  text not null default 'active' check (status in ('active','draft'))
);

create table orders (
  id                    text primary key,          -- CO-1042
  user_id               uuid references users(id) on delete set null,
  email                 text not null,
  status                text not null default 'pending_payment'
                        check (status in ('pending_payment','paid','processing','dispatched','delivered','cancelled','refunded')),
  subtotal              int not null,
  discount              int not null default 0,
  delivery              int not null default 0,
  total                 int not null,
  currency              text not null default 'GBP',
  discount_code         text references discount_codes(code) on delete set null,
  delivery_method_id    text not null references delivery_methods(id),
  shipping_address      jsonb not null,
  free_sample_sku       text,
  gift_wrap             boolean not null default false,
  gift_message          text,
  marketing_opt_in      boolean not null default false,
  notes                 text,
  stripe_payment_intent text unique,
  payment_status        text not null default 'pending',
  idempotency_key       text unique,
  tracking_carrier      text,
  tracking_number       text,
  tracking_url          text,
  estimated_delivery    date,
  paid_at               timestamptz,
  dispatched_at         timestamptz,
  delivered_at          timestamptz,
  created_at            timestamptz not null default now(),
  updated_at            timestamptz not null default now()
);
create index orders_user_idx on orders(user_id);
create index orders_status_idx on orders(status);
create index orders_created_idx on orders(created_at desc);
create index orders_email_idx on orders(lower(email));
create trigger orders_updated_at before update on orders for each row execute function set_updated_at();

create table order_items (
  id            bigserial primary key,
  order_id      text not null references orders(id) on delete cascade,
  product_id    bigint references products(id) on delete set null,
  size_id       bigint references product_sizes(id) on delete set null,
  sku           text not null,
  name          text not null,
  variant_label text not null,
  image         text,
  unit_price    int not null,
  qty           int not null check (qty > 0)
);
create index order_items_order_idx on order_items(order_id);
create index order_items_product_idx on order_items(product_id);

create table payment_events (
  id          bigserial primary key,
  stripe_id   text not null unique,              -- event id (idempotent replay)
  type        text not null,
  order_id    text references orders(id) on delete set null,
  payload     jsonb not null,
  created_at  timestamptz not null default now()
);

create table idempotency_keys (
  key         text primary key,
  scope       text not null,
  status_code int  not null,
  response    jsonb not null,
  created_at  timestamptz not null default now()
);
create index idempotency_keys_created_idx on idempotency_keys(created_at);

create table store_credits (
  id          bigserial primary key,
  user_id     uuid references users(id) on delete cascade,
  email       text not null,
  code        text not null unique,
  amount      int not null,
  remaining   int not null,
  origin_order_id text references orders(id) on delete set null,
  expires_at  timestamptz,
  created_at  timestamptz not null default now()
);

create table loyalty_redemptions (
  id        bigserial primary key,
  user_id   uuid not null references users(id) on delete cascade,
  points    int not null check (points > 0),
  amount    int not null,
  order_id  text references orders(id) on delete set null,
  created_at timestamptz not null default now()
);

-- ── Engagement ───────────────────────────────────────────────────────────────
create table product_reviews (
  id          bigserial primary key,
  product_id  bigint not null references products(id) on delete cascade,
  user_id     uuid references users(id) on delete set null,
  order_id    text references orders(id) on delete set null,
  author_name text not null,
  rating      int not null check (rating between 1 and 5),
  title       text,
  body        text,
  longevity   text,
  verified    boolean not null default false,
  approved    boolean not null default false,
  created_at  timestamptz not null default now(),
  updated_at  timestamptz not null default now()
);
create index product_reviews_product_idx on product_reviews(product_id, approved);
create unique index product_reviews_one_per_user on product_reviews(product_id, user_id) where user_id is not null;
create trigger product_reviews_updated_at before update on product_reviews for each row execute function set_updated_at();

create table wishlist (
  user_id    uuid not null references users(id) on delete cascade,
  product_id bigint not null references products(id) on delete cascade,
  created_at timestamptz not null default now(),
  primary key (user_id, product_id)
);

create table recently_viewed (
  user_id    uuid not null references users(id) on delete cascade,
  product_id bigint not null references products(id) on delete cascade,
  viewed_at  timestamptz not null default now(),
  primary key (user_id, product_id)
);

create table newsletter_subs (
  email           text primary key,
  name            text,
  source          text,
  confirmed_at    timestamptz,
  unsubscribed_at timestamptz,
  created_at      timestamptz not null default now()
);

create table stock_notifications (
  id          bigserial primary key,
  size_id     bigint not null references product_sizes(id) on delete cascade,
  email       text not null,
  sent_at     timestamptz,
  created_at  timestamptz not null default now(),
  unique (size_id, email)
);
create index stock_notifications_pending_idx on stock_notifications(size_id) where sent_at is null;

create table messages (
  id          bigserial primary key,
  user_id     uuid references users(id) on delete set null,
  order_id    text references orders(id) on delete set null,
  name        text not null,
  email       text not null,
  subject     text,
  body        text not null,
  status      text not null default 'open' check (status in ('open','answered','closed')),
  reply       text,
  replied_at  timestamptz,
  created_at  timestamptz not null default now()
);
create index messages_status_idx on messages(status);

create table scent_profiles (
  user_id        uuid primary key references users(id) on delete cascade,
  answers        jsonb not null default '{}',
  families       text[] not null default '{}',
  liked_notes    text[] not null default '{}',
  avoided_notes  text[] not null default '{}',
  updated_at     timestamptz not null default now()
);

create table blog_posts (
  id           bigserial primary key,
  slug         text not null unique,
  title        text not null,
  excerpt      text,
  body         text,
  category     text,
  author       text not null default 'Celestial Oud',
  image_url    text,
  published    boolean not null default false,
  published_at timestamptz,
  created_at   timestamptz not null default now(),
  updated_at   timestamptz not null default now()
);
create trigger blog_posts_updated_at before update on blog_posts for each row execute function set_updated_at();

create table content_blocks (
  key         text primary key,                   -- announcement, hero, promo.discovery, faq, legal.terms…
  type        text not null default 'block' check (type in ('block','page','banner','faq','email')),
  title       text,
  body        jsonb not null default '{}',
  status      text not null default 'published' check (status in ('draft','scheduled','published')),
  publish_at  timestamptz,
  updated_at  timestamptz not null default now()
);
create trigger content_blocks_updated_at before update on content_blocks for each row execute function set_updated_at();

-- ── Admin configuration ──────────────────────────────────────────────────────
create table store_settings (
  id                       int primary key default 1 check (id = 1),
  store_name               text not null default 'Celestial Oud',
  support_email            text not null default 'hello@celestialoud.co.uk',
  currency                 text not null default 'GBP',
  free_delivery_threshold  int  not null default 5000,
  dispatch_cutoff          text not null default '14:00',
  returns_days             int  not null default 14,
  maintenance_mode         boolean not null default false,
  company_name             text,
  company_number           text,
  registered_address       text,
  vat_number               text,
  updated_at               timestamptz not null default now()
);
create trigger store_settings_updated_at before update on store_settings for each row execute function set_updated_at();

create table email_templates (
  key       text primary key,                     -- order_confirmation, order_dispatched…
  name      text not null,
  subject   text not null,
  enabled   boolean not null default true,
  updated_at timestamptz not null default now()
);

create table admin_notifications (
  id          bigserial primary key,
  key         text not null unique,
  name        text not null,
  description text,
  enabled     boolean not null default true
);

create table integrations (
  id          bigserial primary key,
  key         text not null unique,
  name        text not null,
  description text,
  connected   boolean not null default false,
  config      jsonb not null default '{}'
);
