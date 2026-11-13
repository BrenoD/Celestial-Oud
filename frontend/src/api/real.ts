/** Real Go API adapter (backend/). Endpoint shapes documented in frontend/README.md. */
import type * as T from '@/types';
import { qs, request } from './client';
import type { StoreApi } from './index';

export const api: StoreApi = {
  listCollections: () => request('/v1/collections'),
  getCollection: (slug) => request(`/v1/collections/${encodeURIComponent(slug)}`),
  listProducts: (query, signal) => request(`/v1/products${qs({ ...query })}`, { signal }),
  getProduct: (slug) => request(`/v1/products/${encodeURIComponent(slug)}`),
  listRelated: (slug) => request(`/v1/products/${encodeURIComponent(slug)}/related`),
  listReviews: (slug) => request(`/v1/products/${encodeURIComponent(slug)}/reviews`),
  listDeliveryMethods: () => request('/v1/delivery-methods'),
  lookupPostcode: (postcode) => request(`/v1/address/lookup${qs({ postcode })}`),
  validateDiscount: (code, subtotal) => request('/v1/discounts/validate', { method: 'POST', body: { code, subtotal } }),
  createOrder: (req, idempotencyKey) =>
    request('/v1/orders', { method: 'POST', body: req, headers: { 'Idempotency-Key': idempotencyKey }, timeoutMs: 20_000 }),
  getOrder: (id, access) =>
    request(`/v1/orders/${encodeURIComponent(id)}${qs({ token: access?.token ?? undefined, email: access?.email ?? undefined })}`, { retries: 0 }),
  listOrders: () => request('/v1/account/orders'),
  signIn: (email, password) => request('/v1/auth/login', { method: 'POST', body: { email, password } }),
  register: (input) => request('/v1/auth/register', { method: 'POST', body: input }),
  subscribeNewsletter: (email) => request('/v1/newsletter', { method: 'POST', body: { email, source: 'footer' } }),
  confirmNewsletter: (token) => request('/v1/newsletter/confirm', { method: 'POST', body: { token } }),
  unsubscribeNewsletter: (token) => request('/v1/newsletter/unsubscribe', { method: 'POST', body: { token } }),
  contact: (input) => request('/v1/contact', { method: 'POST', body: input }),
  listFaqs: () => request('/v1/content/faqs'),
  startChat: (input) => request('/v1/chat', { method: 'POST', body: input }),
  myChat: () => request('/v1/chat/mine', { retries: 0 }),
  // The guest token travels in a header, never in the URL (no logs, no history).
  getChat: (id, { after = 0, read = false, token }) =>
    request(`/v1/chat/${id}${qs({ after, read: read ? 1 : undefined })}`, { retries: 0, headers: token ? { 'X-Chat-Token': token } : {} }),
  sendChat: (id, message, token) =>
    request(`/v1/chat/${id}/messages`, { method: 'POST', body: { message }, headers: token ? { 'X-Chat-Token': token } : {} }),
};

export type { T };
