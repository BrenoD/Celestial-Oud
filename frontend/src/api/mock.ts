/**
 * In-browser mock of the Go API. Pure functions over the placeholder catalogue
 * with a small artificial latency so loading states are visible in dev.
 * Orders are kept in sessionStorage so the confirmation page survives reload.
 */
import type * as T from '@/types';
import { collections, deliveryMethods, faqs, reviews } from '@/data/catalogue';
import { liveCatalogue } from '@/admin/api.mock';
import { ApiError } from './client';

// Products come from the admin preview state so edits made in /admin show in the store.
const products = new Proxy([] as T.Product[], {
  get(_t, prop) {
    const live = liveCatalogue();
    const v = Reflect.get(live, prop);
    return typeof v === 'function' ? v.bind(live) : v;
  },
});
import type { StoreApi } from './index';

const latency = () => new Promise((r) => setTimeout(r, 120 + Math.random() * 180));

const toSummary = (p: T.Product): T.ProductSummary => ({
  id: p.id,
  slug: p.slug,
  name: p.name,
  notesLine: p.notesLine,
  badge: p.badge,
  tags: p.tags,
  rating: p.rating,
  reviewCount: p.reviewCount,
  images: p.images,
  fromPrice: Math.min(...p.sizes.map((s) => s.price)),
});

const collectionFilter: Record<string, (p: T.Product) => boolean> = {
  all: () => true,
  bestsellers: (p) => p.tags.includes('bestseller'),
  'new-in': (p) => p.tags.includes('new'),
  'top-rated': (p) => p.tags.includes('top-rated'),
  oud: (p) => p.collection.slug === 'oud' || p.tags.includes('oud'),
  amber: (p) => p.collection.slug === 'amber',
  musk: (p) => p.collection.slug === 'musk',
  'discovery-sets': (p) => p.tags.includes('set'),
  gifts: (p) => p.tags.includes('gift') || p.tags.includes('set'),
  offers: (p) => p.tags.includes('offer'),
};

function priceBand(band: string, price: number) {
  const [lo, hi] = band.split('-').map((n) => (n === '' ? NaN : Number(n)));
  if (!Number.isNaN(lo) && price < lo) return false;
  if (!Number.isNaN(hi) && price > hi) return false;
  return true;
}

const ORDERS_KEY = 'co.mock.orders';
function readOrders(): T.Order[] {
  try {
    return JSON.parse(sessionStorage.getItem(ORDERS_KEY) ?? '[]');
  } catch {
    return [];
  }
}
function writeOrders(orders: T.Order[]) {
  try {
    sessionStorage.setItem(ORDERS_KEY, JSON.stringify(orders));
  } catch {
    /* storage unavailable */
  }
}

const findSku = (sku: string): { p: T.Product; s: T.ProductSize } | null => {
  for (const p of products) {
    const s = p.sizes.find((x) => x.sku === sku);
    if (s) return { p, s };
    // 2ml samples are virtual SKUs: "<slug>-2ml"
    if (sku === `${p.slug}-2ml` && p.samplePrice !== undefined) {
      return { p, s: { sku, label: '2ml sample', ml: 2, price: p.samplePrice, inStock: true } };
    }
  }
  return null;
};

export const api: StoreApi = {
  async listCollections() {
    await latency();
    return collections;
  },
  async getCollection(slug) {
    await latency();
    const c = collections.find((x) => x.slug === slug);
    if (!c) throw new ApiError(404, 'Collection not found', 'not_found');
    return c;
  },
  async listProducts(query) {
    await latency();
    let list = products.filter(collectionFilter[query.collection ?? 'all'] ?? (() => true));
    if (query.q) {
      const q = query.q.toLowerCase();
      list = list.filter(
        (p) =>
          p.name.toLowerCase().includes(q) ||
          p.notesLine.toLowerCase().includes(q) ||
          p.keyNotes.some((n) => n.includes(q)) ||
          p.tagline.toLowerCase().includes(q),
      );
    }
    if (query.family?.length) list = list.filter((p) => query.family!.includes(p.family));
    if (query.notes?.length) list = list.filter((p) => query.notes!.some((n) => p.keyNotes.includes(n)));
    if (query.size?.length) list = list.filter((p) => p.sizes.some((s) => query.size!.includes(s.ml)));
    if (query.wear?.length) list = list.filter((p) => query.wear!.some((w) => p.wear.includes(w)));
    if (query.price?.length)
      list = list.filter((p) => query.price!.some((band) => p.sizes.some((s) => priceBand(band, s.price))));

    const from = (p: T.Product) => Math.min(...p.sizes.map((s) => s.price));
    switch (query.sort) {
      case 'newest':
        list = [...list].sort((a, b) => Number(b.tags.includes('new')) - Number(a.tags.includes('new')));
        break;
      case 'price-asc':
        list = [...list].sort((a, b) => from(a) - from(b));
        break;
      case 'price-desc':
        list = [...list].sort((a, b) => from(b) - from(a));
        break;
      case 'top-rated':
        list = [...list].sort((a, b) => b.rating - a.rating);
        break;
      default:
        list = [...list].sort((a, b) => b.reviewCount - a.reviewCount);
    }
    const page = query.page ?? 1;
    const pageSize = query.pageSize ?? 9;
    return { items: list.slice((page - 1) * pageSize, page * pageSize).map(toSummary), total: list.length, page, pageSize };
  },
  async getProduct(slug) {
    await latency();
    const p = products.find((x) => x.slug === slug);
    if (!p) throw new ApiError(404, 'Product not found', 'not_found');
    return p;
  },
  async listRelated(slug) {
    await latency();
    const p = products.find((x) => x.slug === slug);
    return products
      .filter((x) => x.slug !== slug)
      .sort((a, b) => Number(b.collection.slug === p?.collection.slug) - Number(a.collection.slug === p?.collection.slug))
      .slice(0, 4)
      .map(toSummary);
  },
  async listReviews(slug) {
    await latency();
    return reviews[slug] ?? [];
  },
  async listDeliveryMethods() {
    await latency();
    return deliveryMethods;
  },
  async lookupPostcode(postcode) {
    await latency();
    const pc = postcode.trim().toUpperCase();
    if (!/^[A-Z]{1,2}\d[A-Z\d]? ?\d[A-Z]{2}$/.test(pc)) throw new ApiError(422, 'Enter a valid UK postcode', 'invalid_postcode');
    return [
      { line1: '1 Example Street', city: 'London', postcode: pc },
      { line1: '2 Example Street', city: 'London', postcode: pc },
      { line1: 'Flat 3, 4 Example Street', city: 'London', postcode: pc },
    ];
  },
  async validateDiscount(code, subtotal) {
    await latency();
    const c = code.trim().toUpperCase();
    if (c === 'WELCOME10') return { code: c, amount: Math.round(subtotal * 0.1) };
    if (c === 'FREESHIP') return { code: c, amount: 0 };
    throw new ApiError(422, 'That code is not valid', 'invalid_discount');
  },
  async createOrder(req, idempotencyKey) {
    await latency();
    const existing = readOrders().find((o) => o.id === `CO-${idempotencyKey.slice(0, 4).toUpperCase()}`);
    if (existing) return existing; // idempotent replay
    const lines: T.BasketLine[] = req.lines.map(({ sku, qty }) => {
      const hit = findSku(sku);
      if (!hit) throw new ApiError(422, `Unknown item ${sku}`, 'unknown_sku');
      return {
        sku,
        productId: hit.p.id,
        slug: hit.p.slug,
        name: hit.p.name,
        variantLabel: hit.p.tags.includes('set') || hit.p.tags.includes('gift') ? hit.s.label : `Eau de Parfum · ${hit.s.label}`,
        image: hit.p.images[0],
        unitPrice: hit.s.price,
        qty,
      };
    });
    const subtotal = lines.reduce((n, l) => n + l.unitPrice * l.qty, 0);
    const method = deliveryMethods.find((m) => m.id === req.deliveryMethodId) ?? deliveryMethods[0];
    const discount = req.discountCode?.toUpperCase() === 'WELCOME10' ? Math.round(subtotal * 0.1) : 0;
    const delivery = subtotal >= 5000 && method.id === 'standard' ? 0 : method.price;
    const eta = new Date();
    eta.setDate(eta.getDate() + (method.id === 'standard' ? 3 : 1));
    const order: T.Order = {
      id: `CO-${idempotencyKey.slice(0, 4).toUpperCase()}`,
      status: 'confirmed',
      email: req.email,
      createdAt: new Date().toISOString(),
      estimatedDelivery: eta.toISOString(),
      lines,
      freeSample: req.freeSampleSku,
      subtotal,
      delivery,
      discount,
      total: subtotal + delivery - discount,
      shippingAddress: req.shippingAddress,
    };
    writeOrders([order, ...readOrders()]);
    return order;
  },
  async getOrder(id) {
    await latency();
    const o = readOrders().find((x) => x.id === id);
    if (!o) throw new ApiError(404, 'Order not found', 'not_found');
    return o;
  },
  async listOrders() {
    await latency();
    const own = readOrders();
    return own.length
      ? own
      : [
          {
            id: 'CO-1042',
            status: 'dispatched',
            email: 'you@example.com',
            createdAt: '2026-11-01T10:00:00Z',
            estimatedDelivery: '2026-11-04T00:00:00Z',
            lines: [],
            subtotal: 7000,
            delivery: 0,
            discount: 0,
            total: 7000,
            shippingAddress: { firstName: 'A', lastName: 'B', line1: '1 Example St', city: 'London', postcode: 'SW1A 1AA', country: 'GB' },
            tracking: { carrier: 'Royal Mail', number: 'RM123456789GB' },
          },
          {
            id: 'CO-0987',
            status: 'delivered',
            email: 'you@example.com',
            createdAt: '2026-09-12T10:00:00Z',
            estimatedDelivery: '2026-09-15T00:00:00Z',
            lines: [],
            subtotal: 5500,
            delivery: 395,
            discount: 0,
            total: 5895,
            shippingAddress: { firstName: 'A', lastName: 'B', line1: '1 Example St', city: 'London', postcode: 'SW1A 1AA', country: 'GB' },
          },
        ];
  },
  async signIn(email) {
    await latency();
    if (!email.includes('@')) throw new ApiError(422, 'Enter a valid email address', 'invalid_email');
    // Preview convenience: the local part of the email picks the staff role
    // (admin@…, manager@…, support@…, viewer@…); anything else is a customer.
    const local = email.split('@')[0].toLowerCase();
    const role = ['admin', 'manager', 'support', 'viewer'].find((r) => local.includes(r)) ?? 'customer';
    return { token: 'mock-jwt', user: { id: 'u1', email, firstName: role === 'customer' ? 'Guest' : local, lastName: 'Customer', role } };
  },
  async register(input) {
    await latency();
    return { token: 'mock-jwt', user: { id: 'u1', email: input.email, firstName: input.firstName, lastName: input.lastName, role: 'customer' } };
  },
  async subscribeNewsletter() {
    await latency();
  },
  async confirmNewsletter() {
    await latency();
  },
  async unsubscribeNewsletter() {
    await latency();
  },
  async contact() {
    await latency();
  },
  async listFaqs() {
    await latency();
    return faqs;
  },
};
