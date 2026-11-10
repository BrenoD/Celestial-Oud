/**
 * Domain types shared with the Go API. Keep in sync with the REST contract
 * documented in frontend/README.md. Money is integer pence.
 */

export type ScentFamily =
  | 'woody-oud'
  | 'amber-oriental'
  | 'floral'
  | 'fresh-musk'
  | 'gourmand'
  | 'spicy';

export type Wear = 'day' | 'evening' | 'summer' | 'winter';

export type ProductTag = 'bestseller' | 'new' | 'top-rated' | 'gift' | 'set' | 'offer' | 'oud';

export interface ProductSize {
  id?: number;
  /** Stable identifier used in basket lines and orders, e.g. "royal-oud-50ml". */
  sku: string;
  label: string; // "50ml"
  ml: number;
  price: number; // pence
  compareAtPrice?: number; // pence, for offers
  inStock: boolean;
  stock?: number;
  isSample?: boolean;
  sort?: number;
}

export interface ScentProfile {
  intensity: number; // 0-100
  longevity: number;
  sweetness: number;
  freshness: number;
}

export interface Product {
  id: string;
  slug: string;
  name: string;
  tagline: string; // "Smoky oud with saffron and Damask rose."
  notesLine: string; // "Oud · Saffron · Rose"
  badge?: string;
  tags: ProductTag[];
  family: ScentFamily;
  keyNotes: string[];
  wear: Wear[];
  rating: number; // 0-5
  reviewCount: number;
  images: string[];
  sizes: ProductSize[];
  samplePrice?: number; // pence; undefined when no 2ml sample exists
  pyramid: { top: string[]; heart: string[]; base: string[] };
  profile: ScentProfile;
  bestFor: string;
  description: string;
  howToWear: string;
  ingredients: string;
  allergens?: string[];
  responsiblePerson?: string | null;
  cpsrRef?: string | null;
  scpnRef?: string | null;
  collection: { slug: string; name: string };
}

export interface ProductSummary
  extends Pick<
    Product,
    'id' | 'slug' | 'name' | 'notesLine' | 'badge' | 'tags' | 'rating' | 'reviewCount' | 'images'
  > {
  fromPrice: number;
}

export interface Collection {
  id?: number;
  slug: string;
  name: string;
  description: string;
  image?: string;
  sort?: number;
  visible?: boolean;
}

export interface ProductQuery {
  collection?: string;
  q?: string;
  family?: ScentFamily[];
  notes?: string[];
  size?: number[]; // ml
  price?: string[]; // "0-3000", "3000-5000", "8000-"
  wear?: Wear[];
  sort?: 'bestselling' | 'newest' | 'price-asc' | 'price-desc' | 'top-rated';
  page?: number;
  pageSize?: number;
}

export interface Page<T> {
  items: T[];
  total: number;
  page: number;
  pageSize: number;
}

export interface Review {
  id: string;
  author: string;
  rating: number;
  title: string;
  body: string;
  verified: boolean;
  longevity?: string;
  createdAt: string;
}

/* ---------- Basket & orders ---------- */

export interface BasketLine {
  sku: string;
  productId: string;
  slug: string;
  name: string;
  variantLabel: string; // "Eau de Parfum · 50ml"
  image: string;
  unitPrice: number;
  qty: number;
}

export interface DeliveryMethod {
  id: string;
  name: string;
  eta: string;
  price: number; // pence
}

export interface Address {
  firstName: string;
  lastName: string;
  line1: string;
  line2?: string;
  city: string;
  postcode: string;
  country: 'GB';
  phone?: string;
}

export interface CreateOrderRequest {
  email: string;
  marketingOptIn: boolean;
  lines: Array<{ sku: string; qty: number }>;
  freeSampleSku?: string;
  giftWrap: boolean;
  giftMessage?: string;
  discountCode?: string;
  deliveryMethodId: string;
  shippingAddress: Address;
  /** Payment is tokenised client-side by the PSP; only the token travels here. */
  paymentToken: string;
}

export type OrderStatus = 'pending_payment' | 'paid' | 'processing' | 'confirmed' | 'dispatched' | 'delivered' | 'cancelled' | 'refunded';

export interface Order {
  id: string; // "CO-1042"
  status: OrderStatus;
  email: string;
  createdAt: string;
  estimatedDelivery: string;
  lines: BasketLine[];
  freeSample?: string;
  subtotal: number;
  delivery: number;
  discount: number;
  total: number;
  shippingAddress: Address;
  tracking?: { carrier: string; number: string; url?: string };
  paymentStatus?: string;
  discountCode?: string;
  deliveryMethodId?: string;
  giftWrap?: boolean;
  giftMessage?: string;
  notes?: string;
}

/* ---------- Account ---------- */

export interface User {
  id: string;
  email: string;
  firstName: string;
  lastName: string;
  /** customer | viewer | support | manager | admin */
  role?: string;
  tier?: string;
  points?: number;
  marketingEmail?: boolean;
  marketingSms?: boolean;
  marketingWhatsapp?: boolean;
}

export interface AuthResponse {
  token: string;
  refreshToken?: string;
  expiresAt?: string;
  user: User;
}

export interface Faq {
  q: string;
  a: string;
}
