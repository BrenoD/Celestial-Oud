/** Admin API shapes — match backend/internal/store/types.go. */
import type { Address, Collection, Order, Product, ProductSize, Review, User } from '@/types';

export type AdminProduct = Product & { status: 'draft' | 'active' | 'archived'; sort: number; collectionSlugs: string[]; allergens: string[]; concentration: string };

export interface ProductInput {
  slug?: string;
  name: string;
  tagline: string;
  notesLine: string;
  badge?: string | null;
  tags: string[];
  family: string;
  concentration: string;
  wear: string[];
  images: string[];
  pyramid: { top: string[]; heart: string[]; base: string[] };
  profile: { intensity: number; longevity: number; sweetness: number; freshness: number };
  bestFor: string;
  description: string;
  howToWear: string;
  ingredients: string;
  allergens: string[];
  responsiblePerson?: string | null;
  cpsrRef?: string | null;
  scpnRef?: string | null;
  status: 'draft' | 'active' | 'archived';
  sort: number;
  keyNotes: string[];
  collectionSlugs: string[];
  sizes: Array<Partial<ProductSize> & { label: string; price: number; stock: number; isSample?: boolean }>;
}

export interface AdminProductSummary {
  id: number;
  slug: string;
  name: string;
  notesLine: string;
  status: string;
  tags: string[];
  rating: number;
  reviewCount: number;
  fromPrice: number;
  totalStock: number;
  images: string[];
}

export type AdminOrder = Order & { customerName: string; deliveryMethodName?: string; notes?: string; stripePaymentIntent?: string };

export interface Dashboard {
  totalRevenue: number;
  totalOrders: number;
  totalCustomers: number;
  pendingOrders: number;
  pendingReviews: number;
  openMessages: number;
  unreadChats?: number;
  lowStock: Array<{ name: string; sku: string; label: string; stock: number }>;
  recentOrders: AdminOrder[];
  revenueByDay: Array<{ day: string; revenue: number; orders: number }>;
  topProducts: Array<{ name: string; units: number; revenue: number }>;
}

export interface Breakdown {
  period: number;
  orders: number;
  revenue: number;
  newCustomers: number;
  ticket: number;
  ltv: number;
  repeatRate: number;
  series: Array<{ day: string; revenue: number; orders: number }>;
}

export type Customer = User & { role: string; tier: string; points: number; createdAt: string; orderCount: number; totalSpent: number };
export type Staff = User & { role: string; createdAt: string };

export interface PaymentRow {
  ref: string;
  stripeId: string;
  orderId: string;
  method: string;
  email: string;
  amount: number;
  status: 'paid' | 'pending' | 'refunded' | 'partial_refund' | 'canceled';
  createdAt: number;
}

export interface DiscountCode {
  code: string;
  type: 'percent' | 'fixed' | 'free_delivery';
  value: number;
  minSubtotal: number;
  maxUses: number | null;
  uses: number;
  startsAt: string | null;
  endsAt: string | null;
  active: boolean;
  createdAt?: string;
}

export interface Promotion {
  id: number;
  productId: number | null;
  productName: string | null;
  collectionId: number | null;
  percentOff: number;
  startsAt: string;
  endsAt: string;
  appliedAt: string | null;
  revertedAt: string | null;
}

export type AdminReview = Omit<Review, 'id'> & { id: number; productId: number; productName: string; productSlug: string; approved: boolean };

export interface Subscriber {
  email: string;
  name: string | null;
  source: string | null;
  confirmedAt: string | null;
  unsubscribedAt: string | null;
  createdAt: string;
  orderCount: number;
  tier: string;
}

export interface StockAlert {
  id: number;
  sizeId: number;
  sku: string;
  productName: string;
  email: string;
  sentAt: string | null;
  createdAt: string;
}

export interface Message {
  id: number;
  userId?: string;
  orderId?: string;
  name: string;
  email: string;
  subject: string | null;
  body: string;
  status: 'open' | 'answered' | 'closed';
  reply?: string;
  repliedAt?: string;
  createdAt: string;
}

export interface DeliveryMethodRow {
  id: string;
  name: string;
  eta: string;
  price: number;
  freeOver?: number | null;
  carrier?: string | null;
  serviceCode?: string | null;
  active: boolean;
  sort: number;
}

export interface ContentBlock {
  key: string;
  type: 'block' | 'page' | 'banner' | 'faq' | 'email';
  title: string | null;
  body: unknown;
  status: 'draft' | 'scheduled' | 'published';
  publishAt: string | null;
  updatedAt?: string;
}

export interface BlogPost {
  id: number;
  slug: string;
  title: string;
  excerpt: string | null;
  body: string | null;
  category: string | null;
  author: string;
  imageUrl: string | null;
  published: boolean;
  publishedAt: string | null;
  createdAt?: string;
}

export interface Settings {
  storeName: string;
  supportEmail: string;
  currency: string;
  freeDeliveryThreshold: number;
  dispatchCutoff: string;
  returnsDays: number;
  maintenanceMode: boolean;
  companyName: string | null;
  companyNumber: string | null;
  registeredAddress: string | null;
  vatNumber: string | null;
}

export type SectionRow = Record<string, unknown> & { id?: number; key?: string };

export interface AuditRow {
  id: number;
  action: string;
  entity: string;
  entity_id: string | null;
  diff: unknown;
  created_at: string;
  actor: string | null;
}

export type { Address, Collection };

export interface AdminChat {
  id: number;
  userId?: string;
  customerName: string;
  customerEmail: string;
  status: 'open' | 'closed';
  lastMessageAt: string;
  createdAt: string;
  unread: number;
  lastMessage?: string;
  messages?: Array<{ id: number; sender: 'customer' | 'agent'; body: string; createdAt: string }>;
}

export interface AdminApi {
  dashboard(): Promise<Dashboard>;
  analytics(period: number): Promise<Breakdown>;
  audit(): Promise<AuditRow[]>;
  products: {
    list(q?: string): Promise<AdminProductSummary[]>;
    get(id: number): Promise<AdminProduct>;
    save(id: number | null, input: ProductInput): Promise<AdminProduct>;
    remove(id: number): Promise<void>;
    adjustStock(sku: string, delta: number): Promise<{ sku: string; stock: number }>;
  };
  collections: { list(): Promise<Collection[]>; save(c: Partial<Collection>): Promise<Collection>; remove(id: number): Promise<void> };
  stockAlerts(): Promise<StockAlert[]>;
  orders: {
    list(filter?: { status?: string; q?: string }): Promise<{ items: AdminOrder[]; total: number }>;
    setStatus(id: string, status: string): Promise<AdminOrder>;
    setTracking(id: string, t: { carrier: string; number: string; url?: string }): Promise<AdminOrder>;
    refund(id: string, amount?: number): Promise<AdminOrder>;
  };
  customers(q?: string): Promise<Customer[]>;
  payments(): Promise<{ payments: PaymentRow[]; stripeEnabled: boolean }>;
  discounts: { list(): Promise<DiscountCode[]>; save(d: Partial<DiscountCode>): Promise<DiscountCode>; remove(code: string): Promise<void> };
  promotions: { list(): Promise<Promotion[]>; create(p: Partial<Promotion>): Promise<Promotion>; remove(id: number): Promise<void> };
  reviews: { list(pending?: boolean): Promise<AdminReview[]>; update(id: number, patch: Partial<AdminReview>): Promise<void>; remove(id: number): Promise<void> };
  newsletter: { list(): Promise<Subscriber[]>; send(emails: string[], subject: string, body: string): Promise<{ sent: number; total: number }> };
  chats: {
    list(status?: string): Promise<AdminChat[]>;
    get(id: number, after?: number): Promise<AdminChat>;
    reply(id: number, message: string): Promise<NonNullable<AdminChat['messages']>[number]>;
    setStatus(id: number, status: 'open' | 'closed'): Promise<void>;
  };
  messages: { list(status?: string): Promise<Message[]>; reply(id: number, reply: string): Promise<Message>; setStatus(id: number, status: string): Promise<void> };
  delivery: { list(): Promise<DeliveryMethodRow[]>; save(m: Partial<DeliveryMethodRow>): Promise<DeliveryMethodRow>; remove(id: string): Promise<void> };
  content: { list(): Promise<ContentBlock[]>; save(key: string, block: Partial<ContentBlock>): Promise<void>; remove(key: string): Promise<void> };
  blog: { list(): Promise<BlogPost[]>; save(p: Partial<BlogPost>): Promise<BlogPost>; remove(id: number): Promise<void> };
  section(key: string): { list(): Promise<SectionRow[]>; create(row: SectionRow): Promise<SectionRow>; update(id: string | number, patch: SectionRow): Promise<SectionRow>; remove(id: string | number): Promise<void> };
  settings: { get(): Promise<Settings>; update(patch: Partial<Settings>): Promise<Settings> };
  users: {
    list(): Promise<Staff[]>;
    invite(input: { email: string; firstName: string; lastName?: string; role: string; password?: string }): Promise<Staff>;
    setRole(id: string, role: string): Promise<Staff>;
    resetPassword(id: string): Promise<void>;
  };
}
