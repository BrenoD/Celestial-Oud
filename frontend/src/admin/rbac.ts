/** Mirrors backend/internal/httpx HasRole and printforge's section gates. */
export type Role = 'customer' | 'viewer' | 'support' | 'manager' | 'admin';

const RANK: Record<string, number> = { customer: -1, viewer: 0, support: 1, manager: 2, admin: 3 };

export function hasRole(role: string | undefined, min: Role): boolean {
  return (RANK[role ?? ''] ?? -2) >= RANK[min];
}

export const isStaff = (role?: string) => hasRole(role, 'viewer');

/** Minimum role to see a section (anything not listed: viewer). */
export const SECTION_VIEW_ROLE: Record<string, Role> = {
  customers: 'manager',
  payments: 'manager',
  users: 'manager',
  messages: 'support',
  audit: 'manager',
};

/** Minimum role to change a section (anything not listed: viewer = read-only UI). */
export const SECTION_WRITE_ROLE: Record<string, Role> = {
  products: 'manager',
  collections: 'manager',
  'stock-alerts': 'manager',
  orders: 'support',
  shipments: 'support',
  messages: 'support',
  discounts: 'manager',
  promotions: 'manager',
  reviews: 'manager',
  newsletter: 'manager',
  delivery: 'manager',
  taxes: 'manager',
  emails: 'manager',
  notifications: 'manager',
  integrations: 'manager',
  content: 'manager',
  blog: 'manager',
  users: 'admin',
  settings: 'admin',
  payments: 'manager',
};

export const canView = (role: string | undefined, section: string) => hasRole(role, SECTION_VIEW_ROLE[section] ?? 'viewer');
export const canWrite = (role: string | undefined, section: string) => hasRole(role, SECTION_WRITE_ROLE[section] ?? 'viewer');

export interface NavGroup {
  label: string;
  items: Array<{ key: string; label: string }>;
}

/** Same grouping as printforge AdminShell NAV_GROUPS, adapted to the perfume store. */
export const NAV_GROUPS: NavGroup[] = [
  { label: 'Operations', items: [{ key: 'dashboard', label: 'Dashboard' }, { key: 'orders', label: 'Orders' }, { key: 'shipments', label: 'Shipments' }, { key: 'messages', label: 'Messages' }] },
  { label: 'Catalogue', items: [{ key: 'products', label: 'Products' }, { key: 'collections', label: 'Collections' }, { key: 'stock-alerts', label: 'Stock alerts' }] },
  { label: 'Sales', items: [{ key: 'customers', label: 'Customers' }, { key: 'analytics', label: 'Analytics' }, { key: 'payments', label: 'Payments' }] },
  { label: 'Marketing', items: [{ key: 'discounts', label: 'Discount codes' }, { key: 'promotions', label: 'Promotions' }, { key: 'reviews', label: 'Reviews' }, { key: 'newsletter', label: 'Newsletter' }] },
  { label: 'Configuration', items: [{ key: 'delivery', label: 'Delivery' }, { key: 'taxes', label: 'Taxes' }, { key: 'emails', label: 'Emails' }, { key: 'notifications', label: 'Notifications' }, { key: 'integrations', label: 'Integrations' }] },
  { label: 'System', items: [{ key: 'content', label: 'Content' }, { key: 'blog', label: 'Blog' }, { key: 'users', label: 'Users' }, { key: 'settings', label: 'Settings' }, { key: 'audit', label: 'Audit log' }] },
];

export const SECTION_TITLES: Record<string, string> = Object.fromEntries(NAV_GROUPS.flatMap((g) => g.items.map((i) => [i.key, i.label])));
