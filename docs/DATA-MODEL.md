# Celestial Oud — data model (re-reading of the printforge3d-store CRUD)

Source: `BrenoD/printforge3d-store` (Supabase, 12 migrations, ~35 tables, Vercel API, 19 admin
sections). Every entity was put through the same three questions:

1. **Uses the database?** — or is it derived / computed / lives in a third party (Stripe) / client-side.
2. **Schedule table or plain table?** — *schedule* = rows carry `starts_at`/`ends_at`/`expires_at`/
   `next_*` and a job must act on them; *plain* = CRUD rows read on demand.
3. **Reused through a foreign key elsewhere?** — which tables point at it, and what happens on delete.

Target stack: Go API (Cloudflare Containers) → Supabase Postgres. Money is **integer pence**.
Migrations live in `backend/internal/db/migrations/` (embedded in the binary and applied at start-up).

## Identity & access

| printforge | Celestial | DB? | Schedule? | FK reuse |
|---|---|---|---|---|
| `users` (Supabase Auth + profile) | `users` — own auth: `password_hash`, `email_verified_at`, `role` (`customer`,`viewer`,`support`,`manager`,`admin`), marketing prefs (email/sms/whatsapp), loyalty `tier`/`points` | yes | plain | **target** of addresses, orders (set null), product_reviews (set null), wishlist, recently_viewed, scent_profiles, refresh_tokens, password_resets, store_credits, loyalty_redemptions, audit_log |
| Supabase session | `refresh_tokens` (hashed, rotating) | yes | **schedule** — `expires_at`, purged by cron | → users (cascade) |
| Supabase reset link | `password_resets` (hashed token) | yes | **schedule** — `expires_at`, single use | → users (cascade) |
| `addresses` | `addresses` | yes | plain | → users (cascade). Orders **snapshot** the address as jsonb instead of FK so history survives edits |
| staff list (`users.role`) | same; no separate table | — | — | — |
| — | `audit_log` (who changed what in admin) | yes | plain, append-only | → users (set null) |

## Catalogue

| printforge | Celestial | DB? | Schedule? | FK reuse |
|---|---|---|---|---|
| `categories` | `collections` (slug, name, description, hero copy, sort) | yes | plain | ← `product_collections` (m:n). Nav collections (bestsellers, new-in, offers) are **computed** from tags/dates, not rows |
| `products` (price, promo, stock, category, tag, 3D model, colours) | `products` — perfume attributes: `tagline`, `notes_line`, `family`, `wear[]`, `concentration`, scent `pyramid` (jsonb top/heart/base), `profile` (intensity/longevity/sweetness/freshness), `description`, `how_to_wear`, **`ingredients_inci`, `allergens[]`, `responsible_person`, `cpsr_ref`, `scpn_ref`** (UK cosmetics law), `images` jsonb, `badge`, `tags[]`, `status`. Price/stock move to sizes | yes | plain | **target** of product_sizes (cascade), product_collections, product_notes, product_reviews (cascade), wishlist, recently_viewed, scheduled_promotions, order_items (set null — line keeps snapshot) |
| `product_variants` (label, colour, price_mod, stock, sku) | `product_sizes` — the sellable unit: `sku` unique, `label` (30ml), `ml`, `price`, `compare_at_price`, `stock`, `is_sample`, `weight_g` | yes | plain | → products (cascade); ← order_items.size_id (set null), stock_notifications (cascade) |
| `products.images` jsonb (+ Storage bucket) | kept as jsonb on products; files in Supabase Storage/R2 via signed upload URL | yes | plain | — |
| `products.color_mode/colors`, `model_url` | **dropped** (3D-printing specific) | — | — | — |
| — | `notes` (olfactory note taxonomy: oud, rose…) + `product_notes` (tier top/heart/base) | yes | plain | product_notes → products, notes (cascade). Powers the “Key notes” filter and the pyramid |
| `scheduled_promotions` | `scheduled_promotions` (product or collection, % off, `starts_at`, `ends_at`, `applied`) | yes | **schedule** — cron applies/reverts `compare_at_price`/`price` on sizes | → products (cascade), collections (cascade) |
| `coupons` | `discount_codes` (code PK, type percent/fixed/free_delivery, value, min_subtotal, max_uses, uses, `starts_at`, `ends_at`, active) | yes | plain with date window (checked on read, no job) | ← orders.discount_code (set null) |

## Commerce

| printforge | Celestial | DB? | Schedule? | FK reuse |
|---|---|---|---|---|
| `orders` (text id, status pt-BR, payment text, shipping text, address jsonb) | `orders` — `id` “CO-xxxx”, `status` (`pending_payment`,`paid`,`processing`,`dispatched`,`delivered`,`cancelled`,`refunded`), money columns in pence, `delivery_method_id`, `shipping_address` jsonb snapshot, `free_sample_sku`, `gift_wrap`, `gift_message`, `stripe_payment_intent` unique, **`idempotency_key` unique**, `tracking_*` | yes | plain | → users (set null), discount_codes (set null), delivery_methods (restrict); **target** of order_items (cascade), payment_events, store_credits, product_reviews.order_id |
| `order_items` | `order_items` — snapshot `name`, `variant_label`, `unit_price`, `qty` + nullable FKs | yes | plain | → orders (cascade), products (set null), product_sizes (set null) |
| Stripe PaymentIntent (external) | Stripe stays source of truth for payments; `payment_events` logs webhook events for audit/idempotent replay | yes | plain, append-only | → orders (set null) |
| — | `idempotency_keys` (key, scope, response jsonb) — safe retries of `POST /v1/orders` | yes | **schedule** — TTL 24 h, purged by cron | — |
| `shipping_carriers` (carrier, SLA %) | `delivery_methods` — sellable: `name`, `eta`, `price`, `free_over`, `carrier`, `service_code`, `active`, `sort` | yes | plain | ← orders (restrict) |
| `tax_rules` | `tax_rules` (UK VAT 20 % inclusive; informational for admin/reports) | yes | plain | — |
| — | `store_credits` (Discovery Set “credit back on full bottle”) — `amount`, `remaining`, `code`, `expires_at` | yes | **schedule** — expiry by cron | → users (cascade), orders (origin, set null) |
| `loyalty_redemptions` + RPC | `loyalty_redemptions`; points accrue on `users.points` in the order service (F2, kept) | yes | plain | → users (cascade), orders (set null) |
| basket / cart | **client-side** (localStorage) — not persisted server-side in MVP | no | — | — |

## Engagement & content

| printforge | Celestial | DB? | Schedule? | FK reuse |
|---|---|---|---|---|
| `product_reviews` (+ legacy `reviews`) | `product_reviews` only (author, rating, title, body, `longevity`, `verified`, `approved`, `order_id`) | yes | plain | → products (cascade), users (set null), orders (set null) |
| `wishlist` | `wishlist` (user, product) | yes | plain | composite PK → users, products (cascade) |
| `recently_viewed` | kept (F2) | yes | plain | → users, products (cascade) |
| `newsletter_subs` | `newsletter_subs` (email PK, `confirmed_at`, `unsubscribed_at`, source) | yes | plain | — |
| `stock_notifications` | `stock_notifications` (per **size**) | yes | **queue** — processed when a size’s stock goes 0 → >0 | → product_sizes (cascade) |
| `contact_messages` + live `chats`/`chat_messages` | `messages` — one inbox (contact form, order queries), `status` open/answered/closed, `order_id` | yes | plain | → users (set null), orders (set null). Live chat **dropped** for MVP |
| — | `scent_profiles` (quiz answers, families, liked/avoided notes) | yes | plain | → users (unique, cascade) |
| `blog_posts` | `blog_posts` | yes | plain (publish date, no job) | — |
| `content_items` | `content_blocks` (keyed CMS blocks: announcement bar, hero, promo banners, FAQ, legal pages) | yes | plain | — |
| `subscriptions`, `affiliates*` | **deferred** to F3 per the handoff page map — not migrated | — | — | — |

## Admin configuration

| printforge | Celestial | DB? | Schedule? | FK reuse |
|---|---|---|---|---|
| `store_settings` (singleton) | `store_settings` (id=1): store name, support email, currency, free-delivery threshold, dispatch cut-off, returns days, maintenance mode, company number/address | yes | plain | — |
| `email_templates` | `email_templates` (key, subject, enabled) — the mailer checks `enabled` before sending | yes | plain | — |
| `admin_notifications` | `admin_notifications` (toggle per event) | yes | plain | — |
| `integrations` | `integrations` (Stripe, Resend, Royal Mail, Instagram… connected flag + config jsonb) | yes | plain | — |

## Jobs (what the “schedule” tables need)

A Cloudflare **Cron Trigger** calls `POST /internal/cron/tick` (secret header) every 5 min. The Go job:
1. applies / reverts `scheduled_promotions` whose window opened or closed;
2. expires `store_credits`, purges `idempotency_keys`, `refresh_tokens`, `password_resets` past expiry;
3. sends queued `stock_notifications` for sizes back in stock;
4. flips `orders.status` `pending_payment` → `cancelled` after 24 h without a successful PaymentIntent.

## Admin sections → tables (printforge parity)

| Section | Backed by |
|---|---|
| dashboard, analytics | aggregates over orders / users / product_sizes |
| orders, shipments | orders + order_items (+ delivery_methods) |
| messages (was chats) | messages |
| products | products + product_sizes + product_notes + images |
| collections (was categories) | collections + product_collections |
| stock alerts | stock_notifications |
| customers | users (role=customer) + order aggregates |
| payments | Stripe PaymentIntents list + refund, cross-referenced with orders |
| discount codes (coupons), promotions | discount_codes, scheduled_promotions |
| product reviews | product_reviews |
| newsletter | newsletter_subs (+ bulk email) |
| delivery (shipping), taxes, emails, notifications, integrations | delivery_methods, tax_rules, email_templates, admin_notifications, integrations |
| content, blog | content_blocks, blog_posts |
| users (staff), settings | users (role ≠ customer), store_settings |
