# Celestial Oud — online perfume store (Kartus client project)

Handoff from a Claude chat (6 Nov 2026). Read this file first; it holds all decisions so far.

## The client
- **Celestial Oud** ("Celestial Oud Fragrances"): a small UK perfume brand, not a famous label. Instagram: @celestial_oud.
- Products: oud / amber / musk perfumes (Eau de Parfum), likely several sizes (30/50/100 ml), samples and gift sets.
- Built by Breno (Kartus agency). Breno is a backend developer (Go, PostgreSQL, JWT, APIs) and will build the full system: storefront + admin + backend.

## Brand
**Palette** (hex values estimated from the moodboard; confirm with the designer's file):

| Name | Hex | Use |
|---|---|---|
| Ivory / Parchment | #F2EDE4 (UI light tint #F7F3EC) | light panels, footer |
| Taupe / Sand | #B39C84 | borders, input outlines |
| Bronze gold | #B98A5A (light #D9B07A, dark text #7A5530, stars #8A6236) | accent buttons, badges, logo |
| Espresso | #2E1D15 | text, primary buttons, announcement bar |
| Warm black | #0F0C0A | deep shadows only |

Rules:
- **Background is WHITE (#FFFFFF)**, like allbeauty.com. The client explicitly did NOT want an all-brown site.
- Brown and gold are accents only:
  - Primary CTA ("Add to basket", "Checkout", "Pay now"): espresso background with white text.
  - Secondary/promo CTA ("Try 5 samples", badges): bronze background with espresso text.
- Gold text on ivory or white fails contrast. Use #7A5530 for small accent text on light backgrounds.

**Logo**: a crown above two interlocking arcs read as C + O (Celestial Oud). The left arc is a crescent moon; the dot in the centre is a star. The strokes carry engraved arabesque detail. The wordmark is "CELESTIAL OUD" in a classical serif with wide letter-spacing, with "Fragrances" below.
- The moodboard versions differ slightly from each other, so we need one master vector.
- Small sizes (favicon) need a simplified symbol.
- Run a trademark check at UKIPO: interlocking-C monograms are a crowded space.

**Fonts used in the mockups**: Cinzel (wordmark), Cormorant Garamond 600 (headings), Jost (UI/body).

## UX principles (from Breno)
- UK-retail style: VERY direct. Short hero, clear CTAs, free-delivery threshold always visible.
- Three-plane hierarchy:
  1. First plane: shopping (hero, collections, bestsellers, top rated).
  2. Second plane: small themed blocks (quiz strip, Discovery Set banner).
  3. Third plane: story, policies, terms, privacy (footer and small links).
- Reference site: https://allbeauty.com (layout: announcement bar → header with search + basket → nav → hero banners → product grids with badges/reviews/price → promo banners → footer).

## Home layout (agreed)
1. Announcement bar
2. Header (logo, search, account, wishlist, basket)
3. Nav
4. Direct hero (one line + 2 CTAs + image)
5. Trust strip
6. Shop by collection (4 tiles)
7. Bestsellers
8. Two promo banners (Discovery Set, Gifts)
9. Small quiz strip
10. Top rated
11. Story + newsletter
12. Footer with Help / About / Legal

## Legal / UK notes
- Perfumes are cosmetics under UK law. Each product needs a Responsible Person, a safety assessment (CPSR) and an SCPN notification. Show INCI ingredients and allergens on every product page.
- Never use other brands' names ("smells like X", "dupe of Y") anywhere: UK trademark risk (L'Oréal v Bellure).
- 14-day cancellation right. Opened fragrances can be excluded for hygiene reasons if this is stated clearly before purchase.
- UK GDPR: cookie banner with reject option, privacy policy, data download/delete in the account area.
- Perfume is flammable: check the courier's rules (Royal Mail / Evri / DPD).

## Full page map (spec)
A detailed spec lives in Breno's Claude doc "Celestial Oud — Mapa completo de páginas da loja", with MVP / F2 / F3 phases. Summary:
- **Shop**: Home, collection/listing (PLP), product page (PDP), search, Discovery Set and samples, gifts, gift card, quiz, compare.
- **Purchase**: mini-cart, basket, checkout, confirmation, payment failed.
- **Account**: login/register/reset, dashboard, orders + tracking, addresses, wishlist, scent profile, subscription, loyalty, communication preferences, privacy & data.
- **Content**: about, what is oud, fragrance guide, how to apply, concentrations, blog.
- **Help**: FAQ, contact, track order, delivery, returns.
- **Business**: wholesale, affiliates, corporate.
- **Legal**: terms, privacy, cookies, product safety & ingredients, accessibility.
- **System**: 404, 500, coming soon, newsletter confirm/unsubscribe, transactional emails.
- **Admin**: dashboard, products (variants, notes pyramid, INCI, SEO), collections, stock/batches, orders, customers, discounts, reviews, CMS, shipping, reports, settings, users/roles.

## What exists in this folder
- `design/*.dc.html`: frontend mockups from the Claude design canvas. These are NOT production code.
  - They use a proprietary "Design Component" format (`<x-dc>`, `{{holes}}`, `<sc-for>`, `<sc-if>`, `class Component extends DCLogic`). Treat them as a visual and markup reference: port the layout, copy and inline styles into the real stack.
  - Pages:
    - `Main` (Home)
    - `Colecao` (collection with filters)
    - `Produto` (PDP: sizes, qty, sample upsell, accordions, scent profile, reviews, related)
    - `Carrinho` (basket: free-delivery bar, free sample picker, coupon)
    - `Checkout` (express pay, postcode lookup, delivery methods, card)
    - `Confirmacao` (order confirmed)
    - `Conta` (login → account dashboard)
    - `Ajuda` (track order, FAQ, contact)
    - `Sobre` (our story)
  - Image paths were rewritten to `../assets/`.
- `assets/`: crops from the moodboard.
  - `logo_dark.png` is a transparent espresso logo for white headers.
  - The other images are placeholders until real product photos exist.
  - `moodboard.png` is the original moodboard.
- All copy is in English (UK store). `[XX]` marks real data still to fill: prices, review counts, company number, policies.

## Not done yet
- Quiz page
- Legal pages (Terms, Privacy, Cookies)
- Wishlist and gift card pages
- 404
- Admin UI
- Mobile-specific review
- Choice of stack: custom Go + PostgreSQL backend vs. Shopify/Medusa. Ask Breno before choosing.

## Open questions for the client
- How many perfumes, and which sizes/concentrations?
- UK only or international?
- Which courier?
- Is there a Discovery Set, and will they accept the "credit back on full bottle" model?
- Do they have a Responsible Person, safety assessments and ingredient lists?
- Are any perfumes "inspired by" famous ones?
- Returns policy for opened bottles?
- Professional photos?
- Blog, and who writes it?
- Instagram/TikTok Shop?
- Wholesale?
- Who runs the admin day to day?
