/**
 * API facade. Pages call these functions through TanStack Query hooks; the
 * implementation is either the real Go API (`VITE_API_URL` set) or the
 * in-browser mock. Swapping is a one-line env change — no page code changes.
 */
import type * as T from '@/types';
import { api as adapter } from './adapter';

export interface StoreApi {
  listCollections(): Promise<T.Collection[]>;
  getCollection(slug: string): Promise<T.Collection>;
  listProducts(query: T.ProductQuery, signal?: AbortSignal): Promise<T.Page<T.ProductSummary>>;
  getProduct(slug: string): Promise<T.Product>;
  listRelated(slug: string): Promise<T.ProductSummary[]>;
  listReviews(slug: string): Promise<T.Review[]>;
  listDeliveryMethods(): Promise<T.DeliveryMethod[]>;
  lookupPostcode(postcode: string): Promise<Array<Pick<T.Address, 'line1' | 'line2' | 'city' | 'postcode'>>>;
  validateDiscount(code: string, subtotal: number): Promise<{ code: string; amount: number }>;
  createOrder(req: T.CreateOrderRequest, idempotencyKey: string): Promise<T.Order>;
  getOrder(id: string, access?: T.OrderAccess): Promise<T.Order>;
  listOrders(): Promise<T.Order[]>;
  signIn(email: string, password: string): Promise<T.AuthResponse>;
  register(input: { email: string; password: string; firstName: string; lastName: string }): Promise<T.AuthResponse>;
  subscribeNewsletter(email: string): Promise<void>;
  /** Double opt-in: the token comes from the confirmation email link. */
  confirmNewsletter(token: string): Promise<void>;
  /** The token comes from the unsubscribe link in every newsletter. */
  unsubscribeNewsletter(token: string): Promise<void>;
  contact(input: { name: string; email: string; orderId?: string; message: string }): Promise<void>;
  listFaqs(): Promise<T.Faq[]>;
  /** Live chat. Guests pass the token returned by startChat. */
  startChat(input: { name?: string; email?: string; message: string }): Promise<T.ChatView>;
  myChat(): Promise<T.ChatView>;
  getChat(id: number, opts: { after?: number; read?: boolean; token?: string | null }): Promise<T.ChatView>;
  sendChat(id: number, message: string, token?: string | null): Promise<T.ChatMessage>;
}

export const api: StoreApi = adapter;
