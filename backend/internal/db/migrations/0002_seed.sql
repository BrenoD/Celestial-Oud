-- Reference data + placeholder catalogue (mirrors frontend/src/data/catalogue.ts).
insert into store_settings (id) values (1) on conflict do nothing;

insert into delivery_methods (id, name, eta, price, free_over, carrier, sort) values
  ('standard', 'Standard delivery',  '2–3 working days',       395, 5000, 'Royal Mail', 1),
  ('next-day', 'Next day delivery',  'Order by 2pm',           595, null, 'Royal Mail', 2),
  ('express',  'Express (tracked)',  '1 working day, tracked', 795, null, 'DPD',        3)
on conflict (id) do nothing;

insert into tax_rules (region, rate, status) values ('UK VAT (included in prices)', 20, 'active') on conflict do nothing;

insert into email_templates (key, name, subject) values
  ('welcome',             'Welcome',               'Welcome to Celestial Oud'),
  ('order_confirmation',  'Order confirmation',    'Your order {{order_id}} is confirmed'),
  ('order_dispatched',    'Order dispatched',      'Your order {{order_id}} is on its way'),
  ('order_delivered',     'Order delivered',       'Your order {{order_id}} has arrived'),
  ('order_refunded',      'Refund processed',      'Your refund for {{order_id}}'),
  ('password_reset',      'Password reset',        'Reset your Celestial Oud password'),
  ('back_in_stock',       'Back in stock',         '{{product}} is back in stock'),
  ('newsletter_welcome',  'Newsletter welcome',    'Your 10% welcome code'),
  ('review_request',      'Review request',        'How are you enjoying {{product}}?'),
  ('message_reply',       'Message reply',         'Re: {{subject}}')
on conflict (key) do nothing;

insert into admin_notifications (key, name, description) values
  ('new_order',       'New order',          'Email the team when an order is paid'),
  ('low_stock',       'Low stock',          'Alert when a size drops to 5 units or fewer'),
  ('new_review',      'New review',         'Alert when a review awaits moderation'),
  ('new_message',     'New message',        'Alert when a customer sends a message'),
  ('payment_failed',  'Payment failed',     'Alert on failed or disputed payments')
on conflict (key) do nothing;

insert into integrations (key, name, description) values
  ('stripe',     'Stripe',       'Card, Apple Pay, Google Pay, Klarna'),
  ('resend',     'Resend',       'Transactional email'),
  ('royal_mail', 'Royal Mail',   'Labels and tracking (Click & Drop)'),
  ('dpd',        'DPD',          'Express tracked delivery'),
  ('instagram',  'Instagram',    'Shop tagging for @celestial_oud'),
  ('klaviyo',    'Klaviyo',      'Marketing automation')
on conflict (key) do nothing;

insert into collections (slug, name, description, sort) values
  ('oud',            'Oud',             'Rich, smoky agarwood at the heart of every blend.', 1),
  ('amber',          'Amber & Resins',  'Warm amber, labdanum and benzoin.',                 2),
  ('musk',           'Musk & Florals',  'Clean musks and soft florals.',                      3),
  ('discovery-sets', 'Discovery Sets',  'Try before you commit to a full bottle.',            4),
  ('gifts',          'Gift Sets',       'Boxed, wrapped and dispatched today.',               5)
on conflict (slug) do nothing;

insert into notes (slug, name) values
  ('oud','Oud'),('saffron','Saffron'),('rose','Rose'),('amber','Amber'),('vanilla','Vanilla'),('musk','Musk'),
  ('bergamot','Bergamot'),('incense','Incense'),('leather','Leather'),('patchouli','Patchouli'),('sandalwood','Sandalwood'),
  ('tonka','Tonka'),('benzoin','Benzoin'),('iris','Iris'),('cedar','Cedar'),('labdanum','Labdanum')
on conflict (slug) do nothing;

insert into content_blocks (key, type, title, body) values
  ('announcement', 'block', 'Announcement bar', '{"items":["Free UK delivery over £50","Order by 2pm — dispatched today","Free sample with every order"]}'),
  ('hero', 'block', 'Home hero', '{"eyebrow":"Eau de Parfum · Long-lasting","title":"Rich oud perfumes. Made to last all day.","lead":"Luxury oud, amber and musk fragrances — without the designer price tag."}'),
  ('faq', 'faq', 'FAQ', '{"items":[{"q":"When will my order arrive?","a":"Orders placed before 2pm on working days are dispatched the same day. Standard UK delivery takes 2–3 working days; next day is available at checkout."},{"q":"How much is delivery?","a":"Free UK delivery on orders over £50. Otherwise from £3.95."},{"q":"Can I return a perfume?","a":"Unopened, sealed items can be returned within 14 days. For hygiene reasons opened fragrances cannot be returned unless faulty."},{"q":"Where can I find ingredients and allergens?","a":"Every product page lists the full ingredients and allergens under “Ingredients & allergens”."}]}')
on conflict (key) do nothing;

-- Placeholder products (prices/ratings are NOT final).
with p as (
  insert into products (slug, name, tagline, notes_line, family, wear, tags, badge, pyramid, profile, best_for, description, how_to_wear, status, sort) values
  ('royal-oud','Royal Oud','Smoky oud with saffron and Damask rose.','Oud · Saffron · Rose','woody-oud','{evening,winter}','{bestseller,top-rated,oud}','Bestseller',
    '{"top":["Saffron","Bergamot"],"heart":["Damask rose","Incense"],"base":["Oud","Amber","Musk"]}','{"intensity":85,"longevity":80,"sweetness":45,"freshness":20}',
    'evenings · autumn & winter · special occasions','[Description to be written by the client.]','Spray 2–3 times on pulse points. Do not rub.','active',1),
  ('amber-noir','Amber Noir','Dark amber wrapped in vanilla and smoke.','Amber · Vanilla · Smoke','amber-oriental','{evening,winter}','{bestseller,top-rated}','Bestseller',
    '{"top":["Pink pepper","Cardamom"],"heart":["Labdanum","Benzoin"],"base":["Amber","Vanilla","Birch tar"]}','{"intensity":75,"longevity":75,"sweetness":65,"freshness":10}',
    'evenings · winter','[Description.]','Spray 2–3 times on pulse points.','active',2),
  ('celestial-musk','Celestial Musk','Clean white musk softened with iris and cedar.','White musk · Iris · Cedar','fresh-musk','{day,summer}','{new,bestseller}','New',
    '{"top":["Bergamot","Aldehydes"],"heart":["Iris","Violet leaf"],"base":["White musk","Cedar"]}','{"intensity":45,"longevity":60,"sweetness":30,"freshness":80}',
    'daytime · spring & summer','[Description.]','Spray 2–3 times on pulse points.','active',3),
  ('saffron-night','Saffron Night','Saffron and supple leather over a dark oud base.','Saffron · Leather · Oud','spicy','{evening,winter}','{top-rated,oud}',null,
    '{"top":["Saffron","Nutmeg"],"heart":["Leather","Rose"],"base":["Oud","Vetiver"]}','{"intensity":90,"longevity":85,"sweetness":35,"freshness":15}',
    'evenings · winter','[Description.]','Spray 2–3 times on pulse points.','active',4),
  ('rose-oud','Rose Oud','Velvety rose on smoky oud and earthy patchouli.','Rose · Oud · Patchouli','floral','{evening,day}','{top-rated,oud}',null,
    '{"top":["Turkish rose","Pink pepper"],"heart":["Damask rose","Geranium"],"base":["Oud","Patchouli"]}','{"intensity":70,"longevity":75,"sweetness":50,"freshness":30}',
    'all year · evenings','[Description.]','Spray 2–3 times on pulse points.','active',5),
  ('golden-sandal','Golden Sandal','Creamy sandalwood, warm amber and soft musk.','Sandalwood · Amber · Musk','woody-oud','{day,winter}','{top-rated}',null,
    '{"top":["Cardamom"],"heart":["Sandalwood","Iris"],"base":["Amber","Musk","Tonka"]}','{"intensity":60,"longevity":70,"sweetness":55,"freshness":25}',
    'daytime · autumn & winter','[Description.]','Spray 2–3 times on pulse points.','active',6),
  ('velvet-vanilla','Velvet Vanilla','Rich vanilla with tonka and resinous benzoin.','Vanilla · Tonka · Benzoin','gourmand','{evening,winter}','{offer}',null,
    '{"top":["Bergamot"],"heart":["Vanilla orchid","Heliotrope"],"base":["Tonka","Benzoin","Musk"]}','{"intensity":65,"longevity":70,"sweetness":90,"freshness":10}',
    'evenings · winter','[Description.]','Spray 2–3 times on pulse points.','active',7),
  ('discovery-set','Discovery Set','Five 2ml samples — the cost credited back on your first full bottle.','5 × 2ml samples','woody-oud','{day,evening}','{set,top-rated,new}','Try first',
    '{"top":["Royal Oud","Amber Noir"],"heart":["Celestial Musk","Rose Oud"],"base":["Saffron Night"]}','{"intensity":70,"longevity":75,"sweetness":50,"freshness":40}',
    'finding your signature scent','Five of our most-loved fragrances in 2ml atomisers, with a credit code for your first full bottle.','—','active',8),
  ('oud-gift-set','Oud Gift Set','Royal Oud 50ml with matching body oil, boxed.','50ml + body oil, boxed','woody-oud','{evening,winter}','{gift,bestseller,oud}','Gift',
    '{"top":["Saffron","Bergamot"],"heart":["Damask rose","Incense"],"base":["Oud","Amber","Musk"]}','{"intensity":85,"longevity":80,"sweetness":45,"freshness":20}',
    'gifting','[Description.]','—','active',9)
  on conflict (slug) do nothing
  returning id, slug
)
insert into product_sizes (product_id, sku, label, ml, price, compare_at_price, stock, is_sample, sort)
select p.id, s.sku, s.label, s.ml, s.price, s.cmp, s.stock, s.sample, s.sort
from p join (values
  ('royal-oud','royal-oud-2ml','2ml sample',2,400,null,500,true,0),('royal-oud','royal-oud-30ml','30ml',30,3500,null,40,false,1),('royal-oud','royal-oud-50ml','50ml',50,5500,null,60,false,2),('royal-oud','royal-oud-100ml','100ml',100,8500,null,25,false,3),
  ('amber-noir','amber-noir-2ml','2ml sample',2,400,null,500,true,0),('amber-noir','amber-noir-30ml','30ml',30,3200,null,40,false,1),('amber-noir','amber-noir-50ml','50ml',50,5000,null,60,false,2),('amber-noir','amber-noir-100ml','100ml',100,7800,null,25,false,3),
  ('celestial-musk','celestial-musk-2ml','2ml sample',2,400,null,500,true,0),('celestial-musk','celestial-musk-30ml','30ml',30,3000,null,40,false,1),('celestial-musk','celestial-musk-50ml','50ml',50,4800,null,60,false,2),('celestial-musk','celestial-musk-100ml','100ml',100,7500,null,25,false,3),
  ('saffron-night','saffron-night-2ml','2ml sample',2,450,null,500,true,0),('saffron-night','saffron-night-30ml','30ml',30,3600,null,40,false,1),('saffron-night','saffron-night-50ml','50ml',50,5800,null,60,false,2),('saffron-night','saffron-night-100ml','100ml',100,8900,null,25,false,3),
  ('rose-oud','rose-oud-2ml','2ml sample',2,400,null,500,true,0),('rose-oud','rose-oud-30ml','30ml',30,3400,null,40,false,1),('rose-oud','rose-oud-50ml','50ml',50,5400,null,60,false,2),('rose-oud','rose-oud-100ml','100ml',100,8400,null,25,false,3),
  ('golden-sandal','golden-sandal-2ml','2ml sample',2,400,null,500,true,0),('golden-sandal','golden-sandal-30ml','30ml',30,3200,null,40,false,1),('golden-sandal','golden-sandal-50ml','50ml',50,5000,null,60,false,2),('golden-sandal','golden-sandal-100ml','100ml',100,7800,null,25,false,3),
  ('velvet-vanilla','velvet-vanilla-2ml','2ml sample',2,400,null,500,true,0),('velvet-vanilla','velvet-vanilla-30ml','30ml',30,2700,3200,40,false,1),('velvet-vanilla','velvet-vanilla-50ml','50ml',50,4200,5000,60,false,2),('velvet-vanilla','velvet-vanilla-100ml','100ml',100,6500,7800,0,false,3),
  ('discovery-set','discovery-set','5 × 2ml',10,1500,null,200,false,1),
  ('oud-gift-set','oud-gift-set','Gift box',50,7500,null,30,false,1)
) as s(slug, sku, label, ml, price, cmp, stock, sample, sort) on s.slug = p.slug
on conflict (sku) do nothing;

insert into product_collections (product_id, collection_id)
select p.id, c.id from products p join collections c on c.slug = case
  when p.slug in ('royal-oud','saffron-night','rose-oud') then 'oud'
  when p.slug in ('amber-noir','golden-sandal','velvet-vanilla') then 'amber'
  when p.slug = 'celestial-musk' then 'musk'
  when p.slug = 'discovery-set' then 'discovery-sets'
  when p.slug = 'oud-gift-set' then 'gifts' end
on conflict do nothing;

insert into product_notes (product_id, note_id, tier)
select p.id, n.id, 'key' from products p join notes n on n.slug = any(case p.slug
  when 'royal-oud' then '{oud,saffron,rose}'::text[]
  when 'amber-noir' then '{amber,vanilla}'
  when 'celestial-musk' then '{musk,iris,cedar}'
  when 'saffron-night' then '{saffron,oud,leather}'
  when 'rose-oud' then '{rose,oud,patchouli}'
  when 'golden-sandal' then '{sandalwood,amber,musk}'
  when 'velvet-vanilla' then '{vanilla,tonka,benzoin}'
  when 'discovery-set' then '{oud,amber,rose,musk,saffron}'
  when 'oud-gift-set' then '{oud,saffron,rose}' else '{}' end)
on conflict do nothing;
