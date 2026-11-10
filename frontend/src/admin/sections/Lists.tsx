/** Sections that are plain config-driven CRUD lists (printforge adminSectionConfigs parity). */
import { useAuthStore } from '@/store/auth';
import { formatPence } from '@/lib/money';
import { adminApi } from '../api';
import { canWrite } from '../rbac';
import type { BlogPost, Collection, DeliveryMethodRow, DiscountCode, SectionRow, Staff } from '../types';
import { ListManager, ToggleList, type ListConfig } from '../components/ListManager';
import { Pill, fmtDate, useToast } from '../components/ui';

const useWritable = (section: string) => canWrite(useAuthStore((s) => s.user?.role), section);

export function Collections() {
  const cfg: ListConfig<Collection & { sort?: number; visible?: boolean }> = {
    noun: 'Collection', idKey: 'id', readOnly: !useWritable('collections'),
    fields: [
      { id: 'name', label: 'Name', required: true },
      { id: 'slug', label: 'Slug', placeholder: 'auto from name' },
      { id: 'description', label: 'Description', span: 2 },
      { id: 'sort', label: 'Sort order', type: 'number' },
      { id: 'visible', label: 'Visible in store', type: 'toggle' },
    ],
    columns: [{ key: 'name', label: 'Name' }, { key: 'slug', label: 'Slug' }, { key: 'description', label: 'Description' }, { key: 'visible', label: 'Visible', render: (c) => <Pill value={!!c.visible} label={c.visible ? 'visible' : 'hidden'} /> }],
    api: { list: adminApi.collections.list, create: (p) => adminApi.collections.save(p), update: (id, p) => adminApi.collections.save({ ...p, id: Number(id) }), remove: (id) => adminApi.collections.remove(Number(id)) },
    defaults: { visible: true },
  };
  return <ListManager cfg={cfg} />;
}

export function Discounts() {
  const cfg: ListConfig<DiscountCode> = {
    noun: 'Discount code', idKey: 'code', readOnly: !useWritable('discounts'),
    fields: [
      { id: 'code', label: 'Code', required: true, placeholder: 'WELCOME10', immutable: true },
      { id: 'type', label: 'Type', type: 'select', options: [{ value: 'percent', label: 'Percent off' }, { value: 'fixed', label: 'Fixed amount (pence)' }, { value: 'free_delivery', label: 'Free delivery' }] },
      { id: 'value', label: 'Value (% or pence)', type: 'number' },
      { id: 'minSubtotal', label: 'Min. subtotal (pence)', type: 'number' },
      { id: 'maxUses', label: 'Max uses (blank = ∞)', type: 'number' },
      { id: 'startsAt', label: 'Starts', type: 'date', modalOnly: true },
      { id: 'endsAt', label: 'Ends', type: 'date', modalOnly: true },
      { id: 'active', label: 'Active', type: 'toggle' },
    ],
    columns: [
      { key: 'code', label: 'Code' },
      { key: 'type', label: 'Discount', render: (d) => (d.type === 'percent' ? `${d.value}% off` : d.type === 'fixed' ? `${formatPence(d.value)} off` : 'Free delivery') },
      { key: 'uses', label: 'Uses', render: (d) => `${d.uses}${d.maxUses ? ` / ${d.maxUses}` : ''}` },
      { key: 'endsAt', label: 'Window', render: (d) => (d.startsAt || d.endsAt ? `${fmtDate(d.startsAt)} → ${fmtDate(d.endsAt)}` : 'Always') },
      { key: 'active', label: 'Status', render: (d) => <Pill value={d.active ? 'active' : 'inactive'} /> },
    ],
    api: { list: adminApi.discounts.list, create: (p) => adminApi.discounts.save(p), update: (code, p) => adminApi.discounts.save({ ...p, code: String(code) }), remove: (code) => adminApi.discounts.remove(String(code)) },
    defaults: { type: 'percent', value: 10, minSubtotal: 0, active: true },
  };
  return <ListManager cfg={cfg} />;
}

export function Delivery() {
  const cfg: ListConfig<DeliveryMethodRow> = {
    noun: 'Delivery method', idKey: 'id', readOnly: !useWritable('delivery'),
    fields: [
      { id: 'name', label: 'Name', required: true, placeholder: 'Next day delivery' },
      { id: 'eta', label: 'ETA shown to customer', placeholder: '2–3 working days' },
      { id: 'price', label: 'Price (pence)', type: 'number' },
      { id: 'freeOver', label: 'Free over (pence, blank = never)', type: 'number' },
      { id: 'carrier', label: 'Carrier', type: 'select', options: ['Royal Mail', 'DPD', 'Evri', 'UPS', 'FedEx'] },
      { id: 'serviceCode', label: 'Carrier service code', modalOnly: true },
      { id: 'sort', label: 'Sort', type: 'number' },
      { id: 'active', label: 'Active', type: 'toggle' },
    ],
    columns: [
      { key: 'name', label: 'Method', render: (m) => <><b>{m.name}</b><span className="sub">{m.eta}</span></> },
      { key: 'carrier', label: 'Carrier' },
      { key: 'price', label: 'Price', render: (m) => formatPence(m.price) },
      { key: 'freeOver', label: 'Free over', render: (m) => (m.freeOver ? formatPence(m.freeOver) : '—') },
      { key: 'active', label: 'Status', render: (m) => <Pill value={m.active ? 'active' : 'inactive'} /> },
    ],
    api: { list: adminApi.delivery.list, create: (p) => adminApi.delivery.save(p), update: (id, p) => adminApi.delivery.save({ ...p, id: String(id) }), remove: (id) => adminApi.delivery.remove(String(id)) },
    defaults: { active: true, carrier: 'Royal Mail', price: 395 },
  };
  return <ListManager cfg={cfg} />;
}

export function Taxes() {
  const s = adminApi.section('taxes');
  const cfg: ListConfig<SectionRow & { region?: string; rate?: number; status?: string }> = {
    noun: 'Tax rule', idKey: 'id', readOnly: !useWritable('taxes'),
    fields: [{ id: 'region', label: 'Region', required: true, placeholder: 'UK VAT' }, { id: 'rate', label: 'Rate %', type: 'number' }, { id: 'status', label: 'Status', type: 'select', options: ['active', 'draft'] }],
    columns: [{ key: 'region', label: 'Region' }, { key: 'rate', label: 'Rate', render: (r) => `${r.rate}%` }, { key: 'status', label: 'Status', render: (r) => <Pill value={String(r.status)} /> }],
    api: s, defaults: { rate: 20, status: 'active' },
  };
  return <><p className="form-hint" style={{ marginBottom: 12 }}>UK prices are shown VAT-inclusive; these rules feed reports and invoices.</p><ListManager cfg={cfg} /></>;
}

export function Emails() {
  const s = adminApi.section('emails');
  const cfg: ListConfig<SectionRow & { key?: string; name?: string; subject?: string; enabled?: boolean }> = {
    noun: 'Email template', idKey: 'key', readOnly: !useWritable('emails'),
    fields: [{ id: 'key', label: 'Key (trigger)', required: true, immutable: true, placeholder: 'order_confirmation' }, { id: 'name', label: 'Name', required: true }, { id: 'subject', label: 'Subject', span: 2, placeholder: 'Your order {{order_id}} is confirmed' }, { id: 'enabled', label: 'Enabled', type: 'toggle' }],
    columns: [{ key: 'name', label: 'Template', render: (r) => <><b>{String(r.name)}</b><span className="sub">{String(r.key)}</span></> }, { key: 'subject', label: 'Subject' }, { key: 'enabled', label: 'Status', render: (r) => <Pill value={!!r.enabled} label={r.enabled ? 'enabled' : 'disabled'} /> }],
    api: s, defaults: { enabled: true },
  };
  return <ListManager cfg={cfg} />;
}

export function Notifications() {
  const s = adminApi.section('notifications');
  return <ToggleList api={{ list: s.list, toggle: (id, v) => s.update(id, { enabled: v }) }} flagKey="enabled" onLabel="Enabled" offLabel="Disabled" readOnly={!useWritable('notifications')} />;
}

export function Integrations() {
  const s = adminApi.section('integrations');
  return (
    <>
      <p className="form-hint" style={{ marginBottom: 12 }}>Toggling marks an integration as connected; API keys live in the API environment (Stripe, Resend) — never in the browser.</p>
      <ToggleList api={{ list: s.list, toggle: (id, v) => s.update(id, { connected: v }) }} flagKey="connected" onLabel="Connected" offLabel="Not connected" readOnly={!useWritable('integrations')} />
    </>
  );
}

export function Blog() {
  const cfg: ListConfig<BlogPost> = {
    noun: 'Post', idKey: 'id', readOnly: !useWritable('blog'),
    fields: [
      { id: 'title', label: 'Title', required: true, span: 2 },
      { id: 'slug', label: 'Slug', placeholder: 'auto' },
      { id: 'category', label: 'Category', placeholder: 'Guides' },
      { id: 'excerpt', label: 'Excerpt', type: 'textarea', span: 2, modalOnly: true },
      { id: 'body', label: 'Body (Markdown/HTML)', type: 'textarea', span: 2, modalOnly: true },
      { id: 'imageUrl', label: 'Image URL', modalOnly: true },
      { id: 'author', label: 'Author', modalOnly: true },
      { id: 'published', label: 'Published', type: 'toggle' },
    ],
    columns: [{ key: 'title', label: 'Title', render: (p) => <><b>{p.title}</b><span className="sub">/blog/{p.slug}</span></> }, { key: 'category', label: 'Category' }, { key: 'publishedAt', label: 'Published', render: (p) => fmtDate(p.publishedAt) }, { key: 'published', label: 'Status', render: (p) => <Pill value={p.published ? 'published' : 'draft'} /> }],
    api: { list: adminApi.blog.list, create: (p) => adminApi.blog.save(p), update: (id, p) => adminApi.blog.save({ ...p, id: Number(id) }), remove: (id) => adminApi.blog.remove(Number(id)) },
    defaults: { author: 'Celestial Oud', published: false },
  };
  return <ListManager cfg={cfg} />;
}

export function Users() {
  const toast = useToast();
  const me = useAuthStore((s) => s.user);
  const cfg: ListConfig<Staff & { password?: string }> = {
    noun: 'Staff member', idKey: 'id', readOnly: !useWritable('users'),
    fields: [
      { id: 'email', label: 'Email', required: true, immutable: true },
      { id: 'firstName', label: 'First name', required: true, immutable: true },
      { id: 'lastName', label: 'Last name', immutable: true },
      { id: 'role', label: 'Role', type: 'select', options: ['viewer', 'support', 'manager', 'admin'] },
      { id: 'password', label: 'Temporary password (blank = email a reset link)', modalOnly: false },
    ],
    columns: [
      { key: 'firstName', label: 'Name', render: (u) => <><b>{u.firstName} {u.lastName}</b><span className="sub">{u.email}</span></> },
      { key: 'role', label: 'Role', render: (u) => <Pill value={u.role === 'admin' ? 'paid' : 'processing'} label={u.role} /> },
      { key: 'createdAt', label: 'Since', render: (u) => fmtDate(u.createdAt) },
    ],
    api: {
      list: adminApi.users.list,
      create: (p) => adminApi.users.invite({ email: p.email!, firstName: p.firstName!, lastName: p.lastName, role: p.role ?? 'viewer', password: p.password || undefined }),
      update: (id, p) => adminApi.users.setRole(String(id), p.role ?? 'viewer'),
    },
    defaults: { role: 'viewer' },
    rowActions: (u) => (
      <button type="button" className="btn btn--outline btn--xs" disabled={u.id === me?.id} onClick={() => adminApi.users.resetPassword(u.id).then(() => toast(`Reset link sent to ${u.email}.`)).catch((e) => toast(e.message, 'err'))}>
        Reset password
      </button>
    ),
  };
  return (
    <>
      <p className="form-hint" style={{ marginBottom: 12 }}>viewer: read-only dashboards · support: orders, shipments, messages · manager: catalogue & marketing · admin: staff, settings, refunds.</p>
      <ListManager cfg={cfg} />
    </>
  );
}
