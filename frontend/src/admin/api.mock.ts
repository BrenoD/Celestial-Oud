/**
 * In-browser admin API for the design preview (no Go API configured).
 * State lives in localStorage so edits survive reloads on the same device.
 */
import { collections as seedCollections, deliveryMethods as seedDelivery, faqs, products as seedProducts } from '@/data/catalogue';
import type { Order, Product } from '@/types';
import type {
  AdminApi, AdminOrder, AdminProduct, AdminProductSummary, AdminReview, BlogPost, ContentBlock, Customer, DeliveryMethodRow,
  DiscountCode, Message, Promotion, SectionRow, Settings, Staff, StockAlert, Subscriber, AuditRow,
} from './types';

const KEY = 'co.admin.mock.v1';
const latency = () => new Promise((r) => setTimeout(r, 80 + Math.random() * 120));
const nowISO = () => new Date().toISOString();
const daysAgo = (n: number) => new Date(Date.now() - n * 86400000).toISOString();

interface State {
  products: AdminProduct[];
  collections: Array<{ id: number; slug: string; name: string; description: string; sort: number; visible: boolean }>;
  orders: AdminOrder[];
  customers: Customer[];
  staff: Staff[];
  discounts: DiscountCode[];
  promotions: Promotion[];
  reviews: AdminReview[];
  subscribers: Subscriber[];
  stockAlerts: StockAlert[];
  messages: Message[];
  delivery: DeliveryMethodRow[];
  content: ContentBlock[];
  blog: BlogPost[];
  sections: Record<string, SectionRow[]>;
  settings: Settings;
  audit: AuditRow[];
  seq: number;
}

function seed(): State {
  const products: AdminProduct[] = seedProducts.map((p, i) => ({
    ...p,
    id: String(i + 1), // numeric-looking ids so admin lookups by Number(id) work
    status: 'active',
    sort: i + 1,
    collectionSlugs: [p.collection.slug],
    allergens: [],
    concentration: 'Eau de Parfum',
    sizes: p.sizes.map((s, j) => ({ ...s, id: i * 10 + j, stock: s.inStock ? 40 : 0, isSample: false, sort: j })),
  }));
  const addr = { firstName: 'Ada', lastName: 'Lovelace', line1: '1 Example Street', city: 'London', postcode: 'SW1A 1AA', country: 'GB' as const };
  const line = (p: AdminProduct, sizeIdx: number, qty: number) => ({
    sku: p.sizes[sizeIdx].sku, productId: String(p.id), slug: p.slug, name: p.name, variantLabel: `Eau de Parfum · ${p.sizes[sizeIdx].label}`,
    image: '', unitPrice: p.sizes[sizeIdx].price, qty,
  });
  const mkOrder = (id: string, status: Order['status'] | 'paid' | 'processing', lines: ReturnType<typeof line>[], created: string, extra: Partial<AdminOrder> = {}): AdminOrder => {
    const subtotal = lines.reduce((n, l) => n + l.unitPrice * l.qty, 0);
    const delivery = subtotal >= 5000 ? 0 : 395;
    return {
      id, status: status as Order['status'], email: 'ada@example.com', customerName: 'Ada Lovelace', createdAt: created, estimatedDelivery: created,
      lines, subtotal, delivery, discount: 0, total: subtotal + delivery, shippingAddress: addr, deliveryMethodId: 'standard', deliveryMethodName: 'Standard delivery',
      ...extra,
    } as AdminOrder;
  };
  const orders = [
    mkOrder('CO-104201', 'paid', [line(products[0], 1, 1), line(products[7], 0, 1)], daysAgo(0)),
    mkOrder('CO-104188', 'processing', [line(products[1], 2, 1)], daysAgo(1)),
    mkOrder('CO-104150', 'dispatched', [line(products[3], 1, 2)], daysAgo(3), { tracking: { carrier: 'Royal Mail', number: 'RM123456789GB' } }),
    mkOrder('CO-104097', 'delivered', [line(products[4], 1, 1)], daysAgo(9), { tracking: { carrier: 'DPD', number: '15501234567890' } }),
    mkOrder('CO-104010', 'refunded', [line(products[6], 0, 1)], daysAgo(20)),
  ];
  return {
    products,
    collections: seedCollections.filter((c) => !['all', 'bestsellers', 'new-in', 'offers'].includes(c.slug)).map((c, i) => ({ id: i + 1, slug: c.slug, name: c.name, description: c.description, sort: i + 1, visible: true })),
    orders,
    customers: [
      { id: 'u1', email: 'ada@example.com', firstName: 'Ada', lastName: 'Lovelace', role: 'customer', tier: 'Silver', points: 412, createdAt: daysAgo(90), orderCount: 4, totalSpent: 24400 },
      { id: 'u2', email: 'james@example.com', firstName: 'James', lastName: 'Turner', role: 'customer', tier: 'Bronze', points: 55, createdAt: daysAgo(30), orderCount: 1, totalSpent: 5500 },
      { id: 'u3', email: 'amira@example.com', firstName: 'Amira', lastName: 'Khan', role: 'customer', tier: 'Gold', points: 1210, createdAt: daysAgo(200), orderCount: 9, totalSpent: 68900 },
    ],
    staff: [
      { id: 's1', email: 'admin@celestialoud.co.uk', firstName: 'Breno', lastName: 'D.', role: 'admin', createdAt: daysAgo(120) },
      { id: 's2', email: 'ops@celestialoud.co.uk', firstName: 'Sam', lastName: 'Ops', role: 'support', createdAt: daysAgo(40) },
    ],
    discounts: [
      { code: 'WELCOME10', type: 'percent', value: 10, minSubtotal: 0, maxUses: null, uses: 38, startsAt: null, endsAt: null, active: true },
      { code: 'FREESHIP', type: 'free_delivery', value: 0, minSubtotal: 3000, maxUses: 200, uses: 12, startsAt: null, endsAt: null, active: true },
    ],
    promotions: [],
    reviews: [
      { id: 1, productId: 1, productName: 'Royal Oud', productSlug: 'royal-oud', author: 'Amira K.', rating: 5, title: 'Lasts all day', body: 'Two sprays in the morning and I can still smell it at night.', verified: true, approved: true, createdAt: daysAgo(12) },
      { id: 2, productId: 1, productName: 'Royal Oud', productSlug: 'royal-oud', author: 'James T.', rating: 4, title: 'Very close to the expensive stuff', body: 'Rich oud with a lovely rose.', verified: true, approved: true, createdAt: daysAgo(30) },
      { id: 3, productId: 2, productName: 'Amber Noir', productSlug: 'amber-noir', author: 'Lena P.', rating: 5, title: 'Cosy', body: 'Perfect for winter evenings.', verified: false, approved: false, createdAt: daysAgo(1) },
    ],
    subscribers: [
      { email: 'ada@example.com', name: 'Ada', source: 'checkout', confirmedAt: daysAgo(90), unsubscribedAt: null, createdAt: daysAgo(90), orderCount: 4, tier: 'Silver' },
      { email: 'new@example.com', name: null, source: 'footer', confirmedAt: daysAgo(2), unsubscribedAt: null, createdAt: daysAgo(2), orderCount: 0, tier: 'Bronze' },
    ],
    stockAlerts: [{ id: 1, sizeId: 62, sku: 'velvet-vanilla-100ml', productName: 'Velvet Vanilla', email: 'wait@example.com', sentAt: null, createdAt: daysAgo(4) }],
    messages: [
      { id: 1, name: 'Bob Smith', email: 'bob@example.com', subject: 'Where is my order?', body: 'Hi, CO-104150 has not arrived yet.', status: 'open', orderId: 'CO-104150', createdAt: daysAgo(0) },
      { id: 2, name: 'Clara', email: 'clara@example.com', subject: 'Wholesale', body: 'Do you supply boutiques?', status: 'answered', reply: 'Yes — details sent.', repliedAt: daysAgo(5), createdAt: daysAgo(6) },
    ],
    delivery: seedDelivery.map((m, i) => ({ ...m, freeOver: m.id === 'standard' ? 5000 : null, carrier: m.id === 'express' ? 'DPD' : 'Royal Mail', active: true, sort: i + 1 })),
    content: [
      { key: 'announcement', type: 'block', title: 'Announcement bar', body: { items: ['Free UK delivery over £50', 'Order by 2pm — dispatched today', 'Free sample with every order'] }, status: 'published', publishAt: null },
      { key: 'hero', type: 'block', title: 'Home hero', body: { eyebrow: 'Eau de Parfum · Long-lasting', title: 'Rich oud perfumes. Made to last all day.', lead: 'Luxury oud, amber and musk fragrances — without the designer price tag.' }, status: 'published', publishAt: null },
      { key: 'faq', type: 'faq', title: 'FAQ', body: { items: faqs }, status: 'published', publishAt: null },
      { key: 'legal.terms', type: 'page', title: 'Terms & Conditions', body: { html: '<p>[Terms]</p>' }, status: 'draft', publishAt: null },
    ],
    blog: [{ id: 1, slug: 'what-is-oud', title: 'What is oud?', excerpt: 'The resinous heart of agarwood.', body: '…', category: 'Guides', author: 'Celestial Oud', imageUrl: null, published: true, publishedAt: daysAgo(15) }],
    sections: {
      taxes: [{ id: 1, region: 'UK VAT (included in prices)', rate: 20, status: 'active' }],
      emails: ['welcome', 'order_confirmation', 'order_dispatched', 'order_delivered', 'order_refunded', 'password_reset', 'back_in_stock', 'newsletter_welcome', 'review_request', 'message_reply']
        .map((k) => ({ key: k, name: k.replace(/_/g, ' ').replace(/^./, (c) => c.toUpperCase()), subject: `[${k}]`, enabled: k !== 'review_request' })),
      notifications: [
        { id: 1, key: 'new_order', name: 'New order', description: 'Email the team when an order is paid', enabled: true },
        { id: 2, key: 'low_stock', name: 'Low stock', description: 'Alert when a size drops to 5 units or fewer', enabled: true },
        { id: 3, key: 'new_review', name: 'New review', description: 'Alert when a review awaits moderation', enabled: true },
        { id: 4, key: 'new_message', name: 'New message', description: 'Alert when a customer sends a message', enabled: true },
        { id: 5, key: 'payment_failed', name: 'Payment failed', description: 'Alert on failed or disputed payments', enabled: false },
      ],
      integrations: [
        { id: 1, key: 'stripe', name: 'Stripe', description: 'Card, Apple Pay, Google Pay, Klarna', connected: false },
        { id: 2, key: 'resend', name: 'Resend', description: 'Transactional email', connected: false },
        { id: 3, key: 'royal_mail', name: 'Royal Mail', description: 'Labels and tracking (Click & Drop)', connected: false },
        { id: 4, key: 'dpd', name: 'DPD', description: 'Express tracked delivery', connected: false },
        { id: 5, key: 'instagram', name: 'Instagram', description: 'Shop tagging for @celestial_oud', connected: true },
        { id: 6, key: 'klaviyo', name: 'Klaviyo', description: 'Marketing automation', connected: false },
      ],
    },
    settings: { storeName: 'Celestial Oud', supportEmail: 'hello@celestialoud.co.uk', currency: 'GBP', freeDeliveryThreshold: 5000, dispatchCutoff: '14:00', returnsDays: 14, maintenanceMode: false, companyName: null, companyNumber: null, registeredAddress: null, vatNumber: null },
    audit: [],
    seq: 1000,
  };
}

let state: State | null = null;
function load(): State {
  if (state) return state;
  try {
    const raw = localStorage.getItem(KEY);
    state = raw ? (JSON.parse(raw) as State) : seed();
  } catch {
    state = seed();
  }
  return state;
}
function save(s: State) {
  state = s;
  try {
    localStorage.setItem(KEY, JSON.stringify(s));
  } catch {
    /* quota */
  }
}
function audit(action: string, entity: string, entityId: string, diff?: unknown) {
  const s = load();
  s.audit.unshift({ id: ++s.seq, action, entity, entity_id: entityId, diff: diff ?? null, created_at: nowISO(), actor: 'preview@celestialoud.co.uk' });
  s.audit = s.audit.slice(0, 200);
  save(s);
}
const fail = (status: number, message: string) => Object.assign(new Error(message), { status });

const summary = (p: AdminProduct): AdminProductSummary => ({
  id: Number(p.id), slug: p.slug, name: p.name, notesLine: p.notesLine, status: p.status, tags: p.tags, rating: p.rating, reviewCount: p.reviewCount,
  fromPrice: Math.min(...p.sizes.filter((s) => !s.isSample).map((s) => s.price)), totalStock: p.sizes.reduce((n, s) => n + (s.stock ?? 0), 0), images: p.images,
});

const transitions: Record<string, string[]> = {
  pending_payment: ['paid', 'cancelled'], paid: ['processing', 'dispatched', 'cancelled', 'refunded'], processing: ['dispatched', 'cancelled', 'refunded'],
  dispatched: ['delivered', 'refunded'], delivered: ['refunded'],
};

export const mockAdminApi: AdminApi = {
  async dashboard() {
    await latency();
    const s = load();
    const paid = s.orders.filter((o) => !['pending_payment', 'cancelled', 'refunded'].includes(o.status));
    const days = Array.from({ length: 14 }, (_, i) => {
      const d = new Date(Date.now() - (13 - i) * 86400000).toISOString().slice(0, 10);
      const dayOrders = paid.filter((o) => o.createdAt.slice(0, 10) === d);
      return { day: d, revenue: dayOrders.reduce((n, o) => n + o.total, 0) + (i % 3 === 0 ? 5500 : i % 4 === 0 ? 9800 : 3200), orders: dayOrders.length + (i % 2) };
    });
    const low = s.products.flatMap((p) => p.sizes.filter((z) => !z.isSample && (z.stock ?? 0) <= 5).map((z) => ({ name: p.name, sku: z.sku, label: z.label, stock: z.stock ?? 0 })));
    const units: Record<string, { units: number; revenue: number }> = {};
    paid.forEach((o) => o.lines.forEach((l) => { const u = (units[l.name] ??= { units: 0, revenue: 0 }); u.units += l.qty; u.revenue += l.qty * l.unitPrice; }));
    return {
      totalRevenue: paid.reduce((n, o) => n + o.total, 0), totalOrders: paid.length, totalCustomers: s.customers.length,
      pendingOrders: s.orders.filter((o) => ['paid', 'processing'].includes(o.status)).length,
      pendingReviews: s.reviews.filter((r) => !r.approved).length, openMessages: s.messages.filter((m) => m.status === 'open').length,
      lowStock: low, recentOrders: s.orders.slice(0, 8), revenueByDay: days,
      topProducts: Object.entries(units).map(([name, v]) => ({ name, ...v })).sort((a, b) => b.units - a.units).slice(0, 5),
    };
  },
  async analytics(period) {
    await latency();
    const d = await mockAdminApi.dashboard();
    const series = d.revenueByDay.slice(-Math.min(period, 14));
    const revenue = series.reduce((n, x) => n + x.revenue, 0);
    const orders = series.reduce((n, x) => n + x.orders, 0);
    return { period, orders, revenue, newCustomers: Math.round(period / 10), ticket: orders ? Math.round(revenue / orders) : 0, ltv: 18200, repeatRate: 0.31, series };
  },
  async audit() { await latency(); return load().audit; },
  products: {
    async list(q) { await latency(); const needle = (q ?? '').toLowerCase(); return load().products.filter((p) => !needle || `${p.name} ${p.notesLine}`.toLowerCase().includes(needle)).map(summary); },
    async get(id) { await latency(); const p = load().products.find((x) => Number(x.id) === id); if (!p) throw fail(404, 'Product not found'); return p; },
    async save(id, input) {
      await latency();
      const s = load();
      if (!input.name) throw fail(400, 'A name is required.');
      const slug = (input.slug || input.name).toLowerCase().trim().replace(/\s+/g, '-');
      const sizes = input.sizes.map((z, i) => ({ id: z.id ?? ++s.seq, sku: z.sku || `${slug}-${z.label.toLowerCase().replace(/\s+/g, '-')}`, label: z.label, ml: z.ml ?? 0, price: z.price, compareAtPrice: z.compareAtPrice ?? undefined, stock: z.stock, inStock: z.stock > 0, isSample: !!z.isSample, sort: i }));
      const sample = sizes.find((z) => z.isSample);
      const base: AdminProduct = {
        ...(id ? s.products.find((x) => Number(x.id) === id)! : ({ id: String(++s.seq), rating: 0, reviewCount: 0 } as unknown as AdminProduct)),
        ...input, slug, badge: input.badge ?? undefined, sizes: sizes.filter((z) => !z.isSample), samplePrice: sample?.price,
        collection: { slug: input.collectionSlugs[0] ?? 'all', name: s.collections.find((c) => c.slug === input.collectionSlugs[0])?.name ?? 'All Fragrances' },
      } as AdminProduct;
      if (id) s.products = s.products.map((x) => (Number(x.id) === id ? base : x));
      else s.products.push(base);
      save(s);
      audit('save', 'product', slug, input);
      return base;
    },
    async remove(id) { await latency(); const s = load(); s.products = s.products.filter((x) => Number(x.id) !== id); save(s); audit('delete', 'product', String(id)); },
    async adjustStock(sku, delta) {
      await latency();
      const s = load();
      let stock = 0;
      s.products.forEach((p) => p.sizes.forEach((z) => { if (z.sku === sku) { z.stock = Math.max(0, (z.stock ?? 0) + delta); z.inStock = z.stock > 0; stock = z.stock; } }));
      save(s);
      audit('stock', 'size', sku, { delta });
      return { sku, stock };
    },
  },
  collections: {
    async list() { await latency(); return load().collections; },
    async save(c) {
      await latency();
      const s = load();
      if (!c.name) throw fail(400, 'A name is required.');
      const row = { id: c.id ?? ++s.seq, slug: (c.slug || c.name).toLowerCase().trim().replace(/\s+/g, '-'), name: c.name, description: c.description ?? '', sort: Number(c.sort ?? s.collections.length + 1), visible: c.visible ?? true };
      s.collections = c.id ? s.collections.map((x) => (x.id === c.id ? row : x)) : [...s.collections, row];
      save(s); audit('save', 'collection', row.slug, c);
      return row;
    },
    async remove(id) { await latency(); const s = load(); s.collections = s.collections.filter((x) => x.id !== id); save(s); },
  },
  async stockAlerts() { await latency(); return load().stockAlerts; },
  orders: {
    async list(f = {}) {
      await latency();
      let items = load().orders;
      if (f.status) { const set = f.status.split(','); items = items.filter((o) => set.includes(o.status)); }
      if (f.q) { const n = f.q.toLowerCase(); items = items.filter((o) => `${o.id} ${o.email} ${o.customerName}`.toLowerCase().includes(n)); }
      return { items, total: items.length };
    },
    async setStatus(id, status) {
      await latency();
      const s = load();
      const o = s.orders.find((x) => x.id === id);
      if (!o) throw fail(404, 'Order not found');
      if (!transitions[o.status]?.includes(status)) throw fail(422, `An order that is ${o.status} cannot become ${status}.`);
      o.status = status as Order['status'];
      save(s); audit('status', 'order', id, { status });
      return o;
    },
    async setTracking(id, t) {
      await latency();
      const s = load();
      const o = s.orders.find((x) => x.id === id);
      if (!o) throw fail(404, 'Order not found');
      if (!t.number.trim()) throw fail(400, 'Invalid tracking code.');
      o.tracking = { carrier: t.carrier, number: t.number, url: t.url };
      if (['paid', 'processing'].includes(o.status)) o.status = 'dispatched';
      save(s); audit('tracking', 'order', id, t);
      return o;
    },
    async refund(id) { await latency(); return mockAdminApi.orders.setStatus(id, 'refunded'); },
  },
  async customers(q) { await latency(); const n = (q ?? '').toLowerCase(); return load().customers.filter((c) => !n || `${c.email} ${c.firstName} ${c.lastName}`.toLowerCase().includes(n)); },
  async payments() {
    await latency();
    const payments = load().orders.filter((o) => o.status !== 'pending_payment').map((o) => ({
      ref: `pi_${o.id.replace('CO-', '')}`, stripeId: `pi_${o.id}`, orderId: o.id, method: 'card', email: o.email, amount: o.total,
      status: (o.status === 'refunded' ? 'refunded' : o.status === 'cancelled' ? 'canceled' : 'paid') as 'paid' | 'refunded' | 'canceled', createdAt: Math.floor(new Date(o.createdAt).getTime() / 1000),
    }));
    return { payments, stripeEnabled: false };
  },
  discounts: {
    async list() { await latency(); return load().discounts; },
    async save(d) {
      await latency();
      const s = load();
      const code = (d.code ?? '').toUpperCase().trim();
      if (!code) throw fail(400, 'A code is required.');
      const existing = s.discounts.find((x) => x.code === code);
      const row: DiscountCode = { code, type: d.type ?? 'percent', value: Number(d.value ?? 0), minSubtotal: Number(d.minSubtotal ?? 0), maxUses: d.maxUses ?? null, uses: existing?.uses ?? 0, startsAt: d.startsAt ?? null, endsAt: d.endsAt ?? null, active: d.active ?? true };
      s.discounts = existing ? s.discounts.map((x) => (x.code === code ? row : x)) : [row, ...s.discounts];
      save(s); audit('save', 'discount', code, d);
      return row;
    },
    async remove(code) { await latency(); const s = load(); s.discounts = s.discounts.filter((x) => x.code !== code); save(s); },
  },
  promotions: {
    async list() { await latency(); return load().promotions; },
    async create(p) {
      await latency();
      const s = load();
      if (!p.productId && !p.collectionId) throw fail(400, 'Choose a product or a collection.');
      const product = s.products.find((x) => Number(x.id) === Number(p.productId));
      const row: Promotion = { id: ++s.seq, productId: p.productId ?? null, productName: product?.name ?? null, collectionId: p.collectionId ?? null, percentOff: Number(p.percentOff ?? 10), startsAt: p.startsAt ?? nowISO(), endsAt: p.endsAt ?? daysAgo(-7), appliedAt: nowISO(), revertedAt: null };
      if (product) product.sizes.forEach((z) => { z.compareAtPrice = z.compareAtPrice ?? z.price; z.price = Math.round(z.compareAtPrice * (100 - row.percentOff) / 100); });
      s.promotions.unshift(row);
      save(s); audit('create', 'promotion', String(row.id), p);
      return row;
    },
    async remove(id) {
      await latency();
      const s = load();
      const row = s.promotions.find((x) => x.id === id);
      const product = row && s.products.find((x) => Number(x.id) === Number(row.productId));
      if (product) product.sizes.forEach((z) => { if (z.compareAtPrice) { z.price = z.compareAtPrice; z.compareAtPrice = undefined; } });
      s.promotions = s.promotions.filter((x) => x.id !== id);
      save(s);
    },
  },
  reviews: {
    async list(pending) { await latency(); return load().reviews.filter((r) => !pending || !r.approved); },
    async update(id, patch) { await latency(); const s = load(); s.reviews = s.reviews.map((r) => (r.id === id ? { ...r, ...patch } : r)); save(s); audit('update', 'review', String(id), patch); },
    async remove(id) { await latency(); const s = load(); s.reviews = s.reviews.filter((r) => r.id !== id); save(s); },
  },
  newsletter: {
    async list() { await latency(); return load().subscribers; },
    async send(emails, subject) { await latency(); audit('newsletter', 'campaign', subject, { recipients: emails.length }); return { sent: emails.length, total: emails.length }; },
  },
  messages: {
    async list(status) { await latency(); return load().messages.filter((m) => !status || m.status === status); },
    async reply(id, reply) {
      await latency();
      const s = load();
      const m = s.messages.find((x) => x.id === id);
      if (!m) throw fail(404, 'Message not found');
      Object.assign(m, { reply, repliedAt: nowISO(), status: 'answered' });
      save(s);
      return m;
    },
    async setStatus(id, status) { await latency(); const s = load(); s.messages = s.messages.map((m) => (m.id === id ? { ...m, status: status as Message['status'] } : m)); save(s); },
  },
  delivery: {
    async list() { await latency(); return load().delivery; },
    async save(m) {
      await latency();
      const s = load();
      if (!m.name) throw fail(400, 'A name is required.');
      const id = m.id || m.name.toLowerCase().trim().replace(/\s+/g, '-');
      const row: DeliveryMethodRow = { id, name: m.name, eta: m.eta ?? '', price: Number(m.price ?? 0), freeOver: m.freeOver ?? null, carrier: m.carrier ?? null, serviceCode: m.serviceCode ?? null, active: m.active ?? true, sort: Number(m.sort ?? s.delivery.length + 1) };
      s.delivery = s.delivery.some((x) => x.id === id) ? s.delivery.map((x) => (x.id === id ? row : x)) : [...s.delivery, row];
      save(s); audit('save', 'delivery_method', id, m);
      return row;
    },
    async remove(id) { await latency(); const s = load(); s.delivery = s.delivery.filter((x) => x.id !== id); save(s); },
  },
  content: {
    async list() { await latency(); return load().content; },
    async save(key, block) {
      await latency();
      const s = load();
      const row: ContentBlock = { key, type: block.type ?? 'block', title: block.title ?? null, body: block.body ?? {}, status: block.status ?? 'published', publishAt: block.publishAt ?? null, updatedAt: nowISO() };
      s.content = s.content.some((c) => c.key === key) ? s.content.map((c) => (c.key === key ? row : c)) : [...s.content, row];
      save(s); audit('save', 'content', key);
    },
    async remove(key) { await latency(); const s = load(); s.content = s.content.filter((c) => c.key !== key); save(s); },
  },
  blog: {
    async list() { await latency(); return load().blog; },
    async save(p) {
      await latency();
      const s = load();
      if (!p.title) throw fail(400, 'A title is required.');
      const row: BlogPost = { id: p.id ?? ++s.seq, slug: p.slug || p.title.toLowerCase().trim().replace(/\s+/g, '-'), title: p.title, excerpt: p.excerpt ?? null, body: p.body ?? null, category: p.category ?? null, author: p.author || 'Celestial Oud', imageUrl: p.imageUrl ?? null, published: !!p.published, publishedAt: p.published ? (p.publishedAt ?? nowISO()) : null, createdAt: p.createdAt ?? nowISO() };
      s.blog = p.id ? s.blog.map((x) => (x.id === p.id ? row : x)) : [row, ...s.blog];
      save(s);
      return row;
    },
    async remove(id) { await latency(); const s = load(); s.blog = s.blog.filter((x) => x.id !== id); save(s); },
  },
  section: (key) => ({
    async list() { await latency(); return load().sections[key] ?? []; },
    async create(row) { await latency(); const s = load(); const r = { id: ++s.seq, ...row }; (s.sections[key] ??= []).push(r); save(s); audit('create', key, String(r.id ?? r.key)); return r; },
    async update(id, patch) {
      await latency();
      const s = load();
      const rows = s.sections[key] ?? [];
      const r = rows.find((x) => String(x.id ?? x.key) === String(id));
      if (!r) throw fail(404, 'Record not found.');
      Object.assign(r, patch);
      save(s); audit('update', key, String(id), patch);
      return r;
    },
    async remove(id) { await latency(); const s = load(); s.sections[key] = (s.sections[key] ?? []).filter((x) => String(x.id ?? x.key) !== String(id)); save(s); },
  }),
  settings: {
    async get() { await latency(); return load().settings; },
    async update(patch) { await latency(); const s = load(); s.settings = { ...s.settings, ...patch }; save(s); audit('update', 'settings', '1', patch); return s.settings; },
  },
  users: {
    async list() { await latency(); return load().staff; },
    async invite(input) {
      await latency();
      const s = load();
      if (!input.email.includes('@')) throw fail(400, 'A valid email is required.');
      const row: Staff = { id: `s${++s.seq}`, email: input.email, firstName: input.firstName, lastName: input.lastName ?? '', role: input.role, createdAt: nowISO() };
      s.staff.push(row); save(s); audit('invite', 'user', row.id, { role: input.role });
      return row;
    },
    async setRole(id, role) { await latency(); const s = load(); const u = s.staff.find((x) => x.id === id); if (!u) throw fail(404, 'User not found'); u.role = role; save(s); return u; },
    async resetPassword(id) { await latency(); audit('reset_password', 'user', id); },
  },
};

/**
 * Live catalogue for the storefront mock: whatever the admin preview has saved
 * (active products only), falling back to the static seed.
 */
export function liveCatalogue(): Product[] {
  try {
    return load().products.filter((p) => p.status === 'active').map((p) => ({ ...p, id: String(p.id) }));
  } catch {
    return seedProducts;
  }
}

export function liveSettings(): Settings | null {
  try {
    return load().settings;
  } catch {
    return null;
  }
}

/** Preview helper: wipe local admin edits. */
export function resetMockAdmin() {
  try {
    localStorage.removeItem(KEY);
  } catch {
    /* ignore */
  }
  state = null;
}
