# Celestial Oud — storefront page report

State of the storefront on the `frontend` branch. Preview: https://brenod.github.io/Celestial-Oud/
(mock API, placeholder data). Legend — **Real**: final copy/behaviour · **Placeholder**: `[XX]` or
invented data to be replaced by the client / Go API · **Stub**: UI present, no behaviour yet.

Shared on every store page: announcement bar (free-delivery threshold from `VITE_FREE_DELIVERY_THRESHOLD`,
2pm dispatch cut-off, free sample), header (logo, search, account, wishlist, basket count), main nav
(All Fragrances · Bestsellers · New In · Oud · Discovery Sets · Gifts · Offers), footer (Shop / Help / About
columns, legal links, payment methods, company line — **Placeholder**), cookie banner (UK GDPR, equal-weight
reject — **Real**), "added to basket" toast.

---

## 1. Home — `/`  (`pages/Home.tsx`)
| Block | Content | Status |
|---|---|---|
| Hero | "Rich oud perfumes. Made to last all day." + sub-line; CTAs *Shop bestsellers* → `/collections/bestsellers`, *Try 5 samples* → `/products/discovery-set`; brand panel in place of imagery | Copy real · awaiting photography |
| Trust strip | Fast UK delivery · Long-lasting · Free gift wrap · Secure checkout | Real |
| Shop by collection | 4 tiles: Oud, Amber & Resins, Musk & Florals, Gift Sets (brand panel, no photo) | Real links · awaiting photography |
| Bestsellers | 4 products from API `collection=bestsellers` | Data placeholder |
| Promo banners | Discovery Set (credit-back model) · Christmas gift sets — copy only, no image | Copy real (seasonal) |
| Quiz strip | "Find your scent in 60 seconds" → `/quiz` | Real |
| Top rated | 4 products `collection=top-rated&sort=top-rated`, rating badge | Data placeholder |
| Story + newsletter | Story blurb **[Placeholder]**; newsletter "Get 10% off" posts to `/v1/newsletter` | Form real · copy placeholder |

## 2. Collection / listing — `/collections/:slug`  (`pages/Collection.tsx`)
Slugs: `all`, `bestsellers`, `new-in`, `top-rated`, `oud`, `amber`, `musk`, `discovery-sets`, `gifts`, `offers`.
| Attribute | Content | Status |
|---|---|---|
| Header | Collection name + description + product count; breadcrumb | Real |
| Sort | Bestselling · Newest · Price low→high · Price high→low · Top rated | Real (in URL `?sort=`) |
| Filters (sidebar, collapsible on mobile) | Scent family (6) · Key notes (oud, amber, rose, vanilla, saffron, musk) · Size (2ml, 30, 50, 100ml) · Price (4 bands) · Wear (day, evening, summer, winter) | Real, URL-driven, shareable |
| Active-filter chips | Remove individually / Clear all | Real |
| Grid | Product cards (name, notes line, stars + count, "From £", badge, *Choose size*) | Data placeholder |
| Pagination | "Showing X of Y" + *Load more* (`?page=`) | Real |
| Empty state | "No products match" + clear filters | Real |

## 3. Search — `/search?q=`
Same component as Collection; matches name, notes line, key notes and tagline. Title "Results for “q”".

## 4. Product page — `/products/:slug`  (`pages/Product.tsx`)
| Attribute | Content | Status |
|---|---|---|
| Gallery | Main image + up to 4 thumbnails. **Products currently have no photos → neutral "Photo coming soon" placeholder** | Awaiting photography |
| Identity | Eyebrow "Eau de Parfum · {collection}", name, tagline, rating link to #reviews | Copy real for names/taglines · ratings placeholder |
| Price | Per selected size; strike-through compare-at price for offers | Values placeholder |
| Sizes | 30 / 50 / 100 ml (default 50 ml); sets have one size; out-of-stock disabled | Real logic |
| Quantity + Add to basket + wishlist heart | Qty 1–10; status message after adding | Real |
| Sample upsell | "Try a 2ml sample first — £X credited back" → adds virtual SKU `<slug>-2ml` | Real logic · price placeholder |
| Trust lines | In stock / order by 2pm · free gift wrap · returns on unopened items within 14 days | Real |
| Accordions | Description **[Placeholder]** · How to wear it (real) · Ingredients & allergens **[Placeholder INCI]** · Delivery & returns (real, links to policy) | Mixed |
| Scent notes | Top / Heart / Base pyramid per product | Invented, plausible — confirm with client |
| Profile bars | Intensity, Longevity, Sweetness, Freshness (0–100) + "Best for" | Invented |
| Reviews | Score, star summary, "Based on N verified reviews", *Write a review*; list with Verified-buyer pill, longevity | Royal Oud has 2 sample reviews; others empty |
| You may also like | 4 related (same collection first) | Real logic |

## 5. Basket — `/basket`  (`pages/Basket.tsx`)
| Attribute | Content | Status |
|---|---|---|
| Free-delivery bar | "You're £X away…" progress to threshold | Real |
| Lines | Placeholder thumb, name → PDP, variant, qty stepper (1–10), remove, line total | Real |
| Free sample picker | Amber Noir · Celestial Musk · Saffron Night · Rose Oud | Real (list from catalogue) |
| Discount code | Posts to `/v1/discounts/validate` (mock: `WELCOME10` = 10%) | Real |
| Gift wrap | Checkbox + 200-char message | Real |
| Summary | Subtotal, delivery (Free / calculated), sample, discount, total incl. VAT, *Checkout securely* | Real |
| Empty state | "Your basket is empty" → bestsellers | Real |
Persistence: `localStorage` (`co.basket`).

## 6. Checkout — `/checkout`  (`pages/Checkout.tsx`, distraction-free layout)
| Attribute | Content | Status |
|---|---|---|
| Express pay | Apple Pay · Google Pay · PayPal buttons | Stub (disabled until PSP) |
| 1. Contact | Email, marketing opt-in (unticked), sign-in link | Real |
| 2. Delivery address | First/last name, postcode + *Find address* (`/v1/address/lookup`), line 1/2, town, phone | Real; lookup mock returns 3 example addresses |
| 3. Delivery method | Standard 2–3 days (free over threshold) · Next day · Express tracked, from `/v1/delivery-methods` | Prices placeholder |
| 4. Payment | Card number, expiry, CVC, name, billing-same checkbox | UI only — PSP must tokenise in browser |
| Terms | "I agree to T&Cs and Returns policy. Opened fragrances cannot be returned for hygiene reasons." | Real (legal requirement) |
| Validation | Inline per-field errors, focus first invalid | Real |
| Submit | `POST /v1/orders` with `Idempotency-Key`; clears basket; → confirmation | Real |
| Summary | Lines, subtotal, delivery, discount, sample, gift wrap, total | Real |

## 7. Order confirmation — `/checkout/confirmed/:id`  (`pages/Confirmation.tsx`)
Tick, "Thank you — your order is confirmed", order number, email, estimated delivery date; itemised box with
subtotal/delivery/discount/total paid and delivery address; "Create account" prompt (guests only);
*Continue shopping*. Survives reload (reads `/v1/orders/:id`). **Real** (email sending is backend).

## 8. Account — `/account`, `/account/:tab`  (`pages/Account.tsx`)
| State / tab | Content | Status |
|---|---|---|
| Signed out | Sign in (email, password, forgotten link) ↔ Create account (names, email, password); Google/Apple buttons | Forms real (mock accepts anything) · social stub |
| Side panel | "New here?" benefits list (track orders, wishlist & scent profile, 10% off) | Real |
| Orders | List: number, date, total, status pill (Confirmed/Dispatched/Delivered/Cancelled), *Track* / *Buy again* / *Write a review* | Mock sample orders |
| Wishlist | Count + link to `/wishlist` | Real (local) |
| Scent profile | **[Placeholder]** | Stub |
| Addresses | **[Placeholder]** | Stub |
| Communication preferences | Email / SMS / WhatsApp checkboxes | Stub (not persisted) |
| Privacy & data | UK GDPR copy, *Download my data*, *Delete my account* | Stub buttons |
| Sign out | Clears JWT | Real |

## 9. Wishlist — `/wishlist`  (`pages/Wishlist.tsx`)
Saved products (localStorage), product cards + Remove, empty state. **Real** (device-local until API sync).

## 10. Scent quiz — `/quiz`  (`pages/Quiz.tsx`)
5 questions (occasion, season, mood, strength, favourite note) → top-2 scent families → links to
`/collections/all?family=…&wear=…` and Discovery Set. **Real** (client-side scoring; result page copy real).

## 11. Help & contact — `/help`  (`pages/Help.tsx`)
| Section (anchor) | Content | Status |
|---|---|---|
| Track my order (`#track`) | Order number + email → confirmation page | Real (mock) |
| Tiles | Returns · Contact us | Real |
| Delivery (`#delivery`) | Same-day dispatch before 2pm, 2–3 days standard, flammable-goods courier note | Real |
| Returns (`#returns`) | 14-day cancellation, opened fragrances excluded for hygiene, statutory rights | Real |
| FAQ (`#faq`) | 6 questions; 2 answers **[Placeholder]** (longevity, international) | Mixed |
| Contact (`#contact`) | Email hello@celestialoud.co.uk, WhatsApp **[number]**, hours; form name/email/order/message → `/v1/contact` | Form real · details placeholder |

## 12. About / Our story — `/about`  (`pages/About.tsx`)
Hero headline + story paragraph **[Placeholder]** with brand panel (no photo); three cards — *What is oud?* (real,
`#oud`), *Made to last* **[Placeholder]** (`#guide`), *Luxury, fairly priced* **[Placeholder]**; CTA band
*Shop all* / *Try samples*.

## 13. Legal — `/legal/:page`  (`pages/Legal.tsx`)
`terms` · `privacy` · `cookies` · `accessibility` · `product-safety`. Each has correct UK framing (Consumer
Contracts Regulations, UK GDPR rights, PECR cookies, WCAG 2.2 AA, Cosmetics Regulation / Responsible Person /
CPSR / SCPN) with **[Placeholder]** company details and full policy text. "Last updated: [date]".

## 14. 404 — `*`  (`pages/NotFound.tsx`)
"We can't find that page" + *Back to home* / *Shop bestsellers*. **Real**. Also used for unknown product/legal slugs.

---

## Imagery
No photography is used anywhere in the storefront: products show a "Photo coming soon" block and hero / collection tiles / about cards show an ivory brand panel with the emblem. Only `logo_dark.png` ships in `public/images`. The moodboard crops remain in `handoff/assets` for reference.

## Catalogue currently in the mock (`src/data/catalogue.ts`)
9 products — Royal Oud, Amber Noir, Celestial Musk, Saffron Night, Rose Oud, Golden Sandal, Velvet Vanilla
(offer, 100ml out of stock), Discovery Set, Oud Gift Set. Names and notes lines come from the design handoff;
prices (£27–£89), ratings, review counts, pyramids and profiles are invented placeholders. **No product has
images** (placeholder shown) — replace via the `images[]` field once photography exists.

## Not built yet (from the handoff page map)
Gift card, compare, payment-failed page, order tracking detail, scent profile / subscription / loyalty
tabs, blog & content pages (fragrance guide, concentrations, how to apply), wholesale/affiliates/corporate,
coming-soon, newsletter confirm/unsubscribe, transactional emails, admin UI, backend.
