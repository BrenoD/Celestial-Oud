/**
 * API facade. Pages call these functions through TanStack Query hooks; the
 * implementation is either the real Go API (`VITE_API_URL` set) or the
 * in-browser mock. Swapping is a one-line env change — no page code changes.
 */
import { useMockApi } from '@/config';
import type * as T from '@/types';
import * as real from './real';
import * as mock from './mock';

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
  getOrder(id: string, email?: string): Promise<T.Order>;
  listOrders(): Promise<T.Order[]>;
  signIn(email: string, password: string): Promise<T.AuthResponse>;
  register(input: { email: string; password: string; firstName: string; lastName: string }): Promise<T.AuthResponse>;
  subscribeNewsletter(email: string): Promise<void>;
  contact(input: { name: string; email: string; orderId?: string; message: string }): Promise<void>;
  listFaqs(): Promise<T.Faq[]>;
}

export const api: StoreApi = useMockApi ? mock.api : real.api;
