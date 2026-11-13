import { lazy, type ComponentType } from 'react';
import { Navigate, useParams } from 'react-router-dom';

const Dashboard = lazy(() => import('./Dashboard'));
const Products = lazy(() => import('./Products'));
const Orders = lazy(() => import('./Orders'));
const Chats = lazy(() => import('./Chats'));
const Lists = lazy(() => import('./Lists').then((m) => ({ default: m.Collections })));

const pick = <K extends string>(loader: () => Promise<Record<K, ComponentType<Record<string, never>>>>, key: K) =>
  lazy(() => loader().then((m) => ({ default: m[key] })));

const lists = () => import('./Lists');
const misc = () => import('./Misc');

const SECTIONS: Record<string, ComponentType<Record<string, never>>> = {
  dashboard: Dashboard,
  products: Products,
  orders: Orders,
  shipments: () => <Orders mode="shipments" />,
  collections: Lists,
  discounts: pick(lists, 'Discounts'),
  delivery: pick(lists, 'Delivery'),
  taxes: pick(lists, 'Taxes'),
  emails: pick(lists, 'Emails'),
  notifications: pick(lists, 'Notifications'),
  integrations: pick(lists, 'Integrations'),
  blog: pick(lists, 'Blog'),
  users: pick(lists, 'Users'),
  customers: pick(misc, 'Customers'),
  analytics: pick(misc, 'Analytics'),
  payments: pick(misc, 'Payments'),
  promotions: pick(misc, 'Promotions'),
  reviews: pick(misc, 'Reviews'),
  newsletter: pick(misc, 'Newsletter'),
  messages: pick(misc, 'Messages'),
  chats: Chats,
  'stock-alerts': pick(misc, 'StockAlerts'),
  content: pick(misc, 'Content'),
  settings: pick(misc, 'Settings'),
  audit: pick(misc, 'Audit'),
};

export default function Section() {
  const { section = 'dashboard' } = useParams();
  const C = SECTIONS[section];
  if (!C) return <Navigate to="/admin/dashboard" replace />;
  return <C />;
}
