import { qs, request } from '@/api/client';
import type { AdminApi } from './types';

const r = request;

export const realAdminApi: AdminApi = {
  dashboard: () => r('/v1/admin/dashboard'),
  analytics: (period) => r(`/v1/admin/analytics${qs({ period })}`),
  audit: () => r('/v1/admin/audit'),
  products: {
    list: async (q) => (await r<{ items: never[] }>(`/v1/admin/products${qs({ q })}`)).items,
    get: (id) => r(`/v1/admin/products/${id}`),
    save: (id, input) => (id ? r(`/v1/admin/products/${id}`, { method: 'PUT', body: input }) : r('/v1/admin/products', { method: 'POST', body: input })),
    remove: (id) => r(`/v1/admin/products/${id}`, { method: 'DELETE' }),
    adjustStock: (sku, delta) => r('/v1/admin/products/stock', { method: 'POST', body: { sku, delta } }),
  },
  collections: {
    list: () => r('/v1/admin/collections'),
    save: (c) => (c.id ? r(`/v1/admin/collections/${c.id}`, { method: 'PUT', body: c }) : r('/v1/admin/collections', { method: 'POST', body: c })),
    remove: (id) => r(`/v1/admin/collections/${id}`, { method: 'DELETE' }),
  },
  stockAlerts: () => r('/v1/admin/stock-alerts'),
  orders: {
    list: (f = {}) => r(`/v1/admin/orders${qs({ ...f, limit: 200 })}`),
    setStatus: (id, status) => r(`/v1/admin/orders/${id}/status`, { method: 'PATCH', body: { status } }),
    setTracking: (id, t) => r(`/v1/admin/orders/${id}/tracking`, { method: 'POST', body: t }),
    refund: (id, amount = 0) => r(`/v1/admin/orders/${id}/refund`, { method: 'POST', body: { amount } }),
  },
  customers: (q) => r(`/v1/admin/customers${qs({ q })}`),
  payments: () => r('/v1/admin/payments'),
  discounts: {
    list: () => r('/v1/admin/discounts'),
    save: (d) => r('/v1/admin/discounts', { method: 'POST', body: d }),
    remove: (code) => r(`/v1/admin/discounts/${encodeURIComponent(code)}`, { method: 'DELETE' }),
  },
  promotions: {
    list: () => r('/v1/admin/promotions'),
    create: (p) => r('/v1/admin/promotions', { method: 'POST', body: p }),
    remove: (id) => r(`/v1/admin/promotions/${id}`, { method: 'DELETE' }),
  },
  reviews: {
    list: (pending) => r(`/v1/admin/reviews${qs({ pending: pending ? 1 : undefined })}`),
    update: (id, patch) => r(`/v1/admin/reviews/${id}`, { method: 'PATCH', body: patch }),
    remove: (id) => r(`/v1/admin/reviews/${id}`, { method: 'DELETE' }),
  },
  newsletter: {
    list: () => r('/v1/admin/newsletter'),
    send: (emails, subject, body) => r('/v1/admin/newsletter/send', { method: 'POST', body: { emails, subject, body } }),
  },
  chats: {
    list: (status) => r(`/v1/admin/chats${qs({ status })}`),
    get: (id, after) => r(`/v1/admin/chats/${id}${qs({ after })}`),
    reply: (id, message) => r(`/v1/admin/chats/${id}/messages`, { method: 'POST', body: { message } }),
    setStatus: (id, status) => r(`/v1/admin/chats/${id}`, { method: 'PATCH', body: { status } }),
  },
  messages: {
    list: (status) => r(`/v1/admin/messages${qs({ status })}`),
    reply: (id, reply) => r(`/v1/admin/messages/${id}/reply`, { method: 'POST', body: { reply } }),
    setStatus: (id, status) => r(`/v1/admin/messages/${id}`, { method: 'PATCH', body: { status } }),
  },
  delivery: {
    list: () => r('/v1/admin/delivery-methods'),
    save: (m) => r('/v1/admin/delivery-methods', { method: 'POST', body: m }),
    remove: (id) => r(`/v1/admin/delivery-methods/${id}`, { method: 'DELETE' }),
  },
  content: {
    list: () => r('/v1/admin/content'),
    save: (key, block) => r(`/v1/admin/content/${encodeURIComponent(key)}`, { method: 'PUT', body: block }),
    remove: (key) => r(`/v1/admin/content/${encodeURIComponent(key)}`, { method: 'DELETE' }),
  },
  blog: {
    list: () => r('/v1/admin/blog'),
    save: (p) => (p.id ? r(`/v1/admin/blog/${p.id}`, { method: 'PUT', body: p }) : r('/v1/admin/blog', { method: 'POST', body: p })),
    remove: (id) => r(`/v1/admin/blog/${id}`, { method: 'DELETE' }),
  },
  section: (key) => ({
    list: () => r(`/v1/admin/sections/${key}`),
    create: (row) => r(`/v1/admin/sections/${key}`, { method: 'POST', body: row }),
    update: (id, patch) => r(`/v1/admin/sections/${key}/${encodeURIComponent(String(id))}`, { method: 'PATCH', body: patch }),
    remove: (id) => r(`/v1/admin/sections/${key}/${encodeURIComponent(String(id))}`, { method: 'DELETE' }),
  }),
  settings: {
    get: () => r('/v1/admin/settings'),
    update: (patch) => r('/v1/admin/settings', { method: 'PATCH', body: patch }),
  },
  users: {
    list: () => r('/v1/admin/users'),
    invite: (input) => r('/v1/admin/users', { method: 'POST', body: input }),
    setRole: (id, role) => r(`/v1/admin/users/${id}`, { method: 'PATCH', body: { role } }),
    resetPassword: (id) => r(`/v1/admin/users/${id}/reset-password`, { method: 'POST' }),
  },
};
