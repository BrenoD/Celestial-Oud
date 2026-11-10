/** Customers, analytics, payments, promotions, reviews, newsletter, messages, stock alerts, content, settings, audit. */
import { useState, type FormEvent } from 'react';
import { useAuthStore } from '@/store/auth';
import { formatPence } from '@/lib/money';
import { adminApi } from '../api';
import { canWrite } from '../rbac';
import type { ContentBlock, Settings as SettingsT } from '../types';
import { BarChart, LoadState, Metrics, Modal, Pill, errMsg, exportCsv, fmtDate, fmtDateTime, useLoad, useToast } from '../components/ui';

const useWritable = (section: string) => canWrite(useAuthStore((s) => s.user?.role), section);

export function Customers() {
  const [q, setQ] = useState('');
  const { data, error, loading } = useLoad(() => adminApi.customers(q), [q]);
  return (
    <>
      <div className="adm-toolbar">
        <label className="field" htmlFor="cq">Search<input id="cq" className="input" value={q} onChange={(e) => setQ(e.target.value)} placeholder="Name or email…" /></label>
        <button type="button" className="btn btn--outline btn--sm" style={{ alignSelf: 'end' }} onClick={() => exportCsv([['Name', 'Email', 'Tier', 'Points', 'Orders', 'Spent', 'Since'], ...(data ?? []).map((c) => [`${c.firstName} ${c.lastName}`, c.email, c.tier, c.points, c.orderCount, c.totalSpent / 100, c.createdAt])], 'customers.csv')}>CSV</button>
      </div>
      <LoadState loading={loading && !data} error={error} />
      <div className="adm-table">
        <table>
          <thead><tr><th>Customer</th><th>Tier</th><th>Points</th><th>Orders</th><th>Total spent</th><th>Since</th></tr></thead>
          <tbody>
            {(data ?? []).map((c) => (
              <tr key={c.id}>
                <td><b>{c.firstName} {c.lastName}</b><span className="sub">{c.email}</span></td>
                <td><Pill value={c.tier === 'Gold' ? 'paid' : c.tier === 'Silver' ? 'processing' : 'draft'} label={c.tier} /></td>
                <td>{c.points}</td>
                <td>{c.orderCount}</td>
                <td>{formatPence(c.totalSpent)}</td>
                <td>{fmtDate(c.createdAt)}</td>
              </tr>
            ))}
            {!loading && !data?.length && <tr><td colSpan={6} className="form-hint">No customers yet.</td></tr>}
          </tbody>
        </table>
      </div>
    </>
  );
}

export function Analytics() {
  const [period, setPeriod] = useState(30);
  const { data, error, loading } = useLoad(() => adminApi.analytics(period), [period]);
  return (
    <>
      <div className="adm-toolbar">
        <label className="field" htmlFor="ap">Period
          <select id="ap" className="select" value={period} onChange={(e) => setPeriod(Number(e.target.value))}>
            <option value={7}>Last 7 days</option><option value={30}>Last 30 days</option><option value={90}>Last 90 days</option><option value={365}>Last year</option>
          </select>
        </label>
      </div>
      <LoadState loading={loading && !data} error={error} />
      {data && (
        <>
          <Metrics items={[[formatPence(data.revenue), 'Revenue'], [String(data.orders), 'Orders'], [formatPence(data.ticket), 'Average order'], [formatPence(data.ltv), 'Customer LTV'], [`${Math.round(data.repeatRate * 100)}%`, 'Repeat rate'], [String(data.newCustomers), 'New customers']]} />
          <BarChart series={data.series} title={`Revenue by day — last ${data.period} days`} />
        </>
      )}
    </>
  );
}

export function Payments() {
  const toast = useToast();
  const writable = useWritable('payments');
  const [q, setQ] = useState('');
  const { data, error, loading, reload } = useLoad(adminApi.payments, []);
  const rows = (data?.payments ?? []).filter((p) => !q || `${p.ref} ${p.orderId} ${p.email}`.toLowerCase().includes(q.toLowerCase()));
  return (
    <>
      {data && !data.stripeEnabled && <p className="status status--warn" style={{ marginBottom: 12 }}>Stripe is not connected — showing orders as payments. Set STRIPE_SECRET_KEY on the API to list live PaymentIntents.</p>}
      <div className="adm-toolbar"><label className="field" htmlFor="pq2">Search<input id="pq2" className="input" value={q} onChange={(e) => setQ(e.target.value)} placeholder="Reference, order, email…" /></label></div>
      <LoadState loading={loading && !data} error={error} />
      <div className="adm-table">
        <table>
          <thead><tr><th>Reference</th><th>Method</th><th>Customer</th><th>Amount</th><th>Status</th><th>Date</th>{writable && <th>Actions</th>}</tr></thead>
          <tbody>
            {rows.map((p) => (
              <tr key={p.ref}>
                <td><b>{p.ref}</b><span className="sub">{p.orderId}</span></td>
                <td>{p.method}</td>
                <td>{p.email}</td>
                <td><b>{formatPence(p.amount)}</b></td>
                <td><Pill value={p.status} /></td>
                <td>{fmtDate(new Date(p.createdAt * 1000).toISOString())}</td>
                {writable && <td>{p.status === 'paid' && p.orderId && <button type="button" className="btn btn--outline btn--xs btn--danger" onClick={() => confirm(`Refund ${formatPence(p.amount)} for ${p.orderId}?`) && adminApi.orders.refund(p.orderId).then(() => { toast('Refund issued.'); reload(); }).catch((e) => toast(errMsg(e), 'err'))}>Refund</button>}</td>}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </>
  );
}

export function Promotions() {
  const toast = useToast();
  const writable = useWritable('promotions');
  const { data, error, loading, reload } = useLoad(adminApi.promotions.list, []);
  const { data: products } = useLoad(() => adminApi.products.list(), []);
  const [f, setF] = useState({ productId: '', percentOff: 10, startsAt: '', endsAt: '' });
  const submit = async (e: FormEvent) => {
    e.preventDefault();
    try {
      await adminApi.promotions.create({ productId: Number(f.productId), percentOff: f.percentOff, startsAt: f.startsAt ? new Date(f.startsAt).toISOString() : undefined, endsAt: f.endsAt ? new Date(f.endsAt).toISOString() : new Date(Date.now() + 7 * 86400000).toISOString() });
      toast('Promotion scheduled.');
      reload();
    } catch (err) {
      toast(errMsg(err), 'err');
    }
  };
  return (
    <>
      {writable && (
        <form className="adm-form" onSubmit={submit}>
          <label className="field">Product<select className="select" value={f.productId} onChange={(e) => setF({ ...f, productId: e.target.value })} required><option value="">Choose…</option>{(products ?? []).map((p) => <option key={p.id} value={p.id}>{p.name}</option>)}</select></label>
          <label className="field">Discount %<input className="input" type="number" min={1} max={90} value={f.percentOff} onChange={(e) => setF({ ...f, percentOff: Number(e.target.value) })} /></label>
          <label className="field">Starts (blank = now)<input className="input" type="datetime-local" value={f.startsAt} onChange={(e) => setF({ ...f, startsAt: e.target.value })} /></label>
          <label className="field">Ends (blank = 7 days)<input className="input" type="datetime-local" value={f.endsAt} onChange={(e) => setF({ ...f, endsAt: e.target.value })} /></label>
          <div className="form-actions span-all"><button type="submit" className="btn btn--primary btn--sm">Schedule promotion</button><span className="form-hint">Applied and reverted automatically by the 5-minute cron; “was” prices show as strike-through in the store.</span></div>
        </form>
      )}
      <LoadState loading={loading && !data} error={error} />
      <div className="adm-table">
        <table>
          <thead><tr><th>Target</th><th>Discount</th><th>Window</th><th>State</th>{writable && <th>Actions</th>}</tr></thead>
          <tbody>
            {(data ?? []).map((p) => (
              <tr key={p.id}>
                <td>{p.productName ?? `Collection #${p.collectionId}`}</td>
                <td>{p.percentOff}% off</td>
                <td>{fmtDateTime(p.startsAt)} → {fmtDateTime(p.endsAt)}</td>
                <td><Pill value={p.revertedAt ? 'closed' : p.appliedAt ? 'active' : 'scheduled'} /></td>
                {writable && <td><button type="button" className="btn btn--outline btn--xs btn--danger" onClick={() => adminApi.promotions.remove(p.id).then(() => { toast('Promotion removed and prices restored.'); reload(); })}>End & remove</button></td>}
              </tr>
            ))}
            {!loading && !data?.length && <tr><td colSpan={5} className="form-hint">No scheduled promotions.</td></tr>}
          </tbody>
        </table>
      </div>
    </>
  );
}

export function Reviews() {
  const toast = useToast();
  const writable = useWritable('reviews');
  const [pending, setPending] = useState(false);
  const [q, setQ] = useState('');
  const { data, error, loading, reload } = useLoad(() => adminApi.reviews.list(pending), [pending]);
  const rows = (data ?? []).filter((r) => !q || `${r.author} ${r.productName} ${r.title ?? ''} ${r.body ?? ''}`.toLowerCase().includes(q.toLowerCase()));
  const act = (fn: () => Promise<unknown>, msg: string) => fn().then(() => { toast(msg); reload(); }).catch((e) => toast(errMsg(e), 'err'));
  return (
    <>
      <div className="adm-toolbar" style={{ gridTemplateColumns: 'minmax(0,1fr) auto auto' }}>
        <label className="field" htmlFor="rq">Search<input id="rq" className="input" value={q} onChange={(e) => setQ(e.target.value)} placeholder="Author, product, text…" /></label>
        <label className="check" style={{ alignSelf: 'end', height: 42 }}><input type="checkbox" checked={pending} onChange={(e) => setPending(e.target.checked)} /> Pending only</label>
        <button type="button" className="btn btn--outline btn--sm" style={{ alignSelf: 'end' }} onClick={() => exportCsv([['Product', 'Author', 'Rating', 'Title', 'Body', 'Verified', 'Approved', 'Date'], ...rows.map((r) => [r.productName, r.author, r.rating, r.title ?? '', r.body ?? '', r.verified, r.approved, r.createdAt])], 'reviews.csv')}>CSV</button>
      </div>
      <LoadState loading={loading && !data} error={error} />
      {rows.map((r) => (
        <div className="rev" key={r.id}>
          <div className="rev__head">
            <span className="stars">{'★'.repeat(r.rating)}</span><b>{r.title ?? '(no title)'}</b>
            <span className="form-hint">{r.author} · {r.productName} · {fmtDate(r.createdAt)}</span>
            {r.verified && <Pill value="paid" label="Verified buyer" />}
            <Pill value={r.approved ? 'approved' : 'pending'} />
          </div>
          <p>{r.body}</p>
          {writable && (
            <div className="table-actions">
              {r.approved ? <button type="button" className="btn btn--outline btn--xs" onClick={() => act(() => adminApi.reviews.update(r.id, { approved: false }), 'Review hidden.')}>Hide</button> : <button type="button" className="btn btn--primary btn--xs" onClick={() => act(() => adminApi.reviews.update(r.id, { approved: true }), 'Review approved.')}>Approve</button>}
              <button type="button" className="btn btn--outline btn--xs btn--danger" onClick={() => confirm('Delete this review?') && act(() => adminApi.reviews.remove(r.id), 'Review deleted.')}>Delete</button>
            </div>
          )}
        </div>
      ))}
      {!loading && rows.length === 0 && <p className="form-hint">No reviews.</p>}
    </>
  );
}

export function Newsletter() {
  const toast = useToast();
  const writable = useWritable('newsletter');
  const { data, error, loading } = useLoad(adminApi.newsletter.list, []);
  const [q, setQ] = useState('');
  const [sel, setSel] = useState<Set<string>>(new Set());
  const [compose, setCompose] = useState(false);
  const [subject, setSubject] = useState('');
  const [body, setBody] = useState('');
  const rows = (data ?? []).filter((s) => !q || `${s.email} ${s.name ?? ''}`.toLowerCase().includes(q.toLowerCase()));
  const toggle = (e: string) => setSel((s) => { const n = new Set(s); if (n.has(e)) n.delete(e); else n.add(e); return n; });
  const send = async (e: FormEvent) => {
    e.preventDefault();
    try {
      const r = await adminApi.newsletter.send([...sel], subject, body);
      toast(`${r.sent}/${r.total} emails sent.`);
      setCompose(false); setSel(new Set());
    } catch (err) {
      toast(errMsg(err), 'err');
    }
  };
  return (
    <>
      <div className="adm-toolbar" style={{ gridTemplateColumns: 'auto minmax(0,1fr) auto auto' }}>
        <label className="check" style={{ alignSelf: 'end', height: 42 }}><input type="checkbox" checked={rows.length > 0 && rows.every((r) => sel.has(r.email))} onChange={(e) => setSel(e.target.checked ? new Set(rows.map((r) => r.email)) : new Set())} /> All</label>
        <label className="field" htmlFor="nq">Search<input id="nq" className="input" value={q} onChange={(e) => setQ(e.target.value)} placeholder="Email or name…" /></label>
        {writable && <button type="button" className="btn btn--primary btn--sm" style={{ alignSelf: 'end' }} disabled={sel.size === 0} onClick={() => setCompose(true)}>Email {sel.size || ''} selected</button>}
        <button type="button" className="btn btn--outline btn--sm" style={{ alignSelf: 'end' }} onClick={() => exportCsv([['Email', 'Name', 'Tier', 'Orders', 'Source', 'Since'], ...rows.map((s) => [s.email, s.name ?? '', s.tier, s.orderCount, s.source ?? '', s.createdAt])], 'newsletter.csv')}>CSV</button>
      </div>
      <LoadState loading={loading && !data} error={error} />
      <div className="adm-table">
        <table>
          <thead><tr><th></th><th>Subscriber</th><th>Tier</th><th>Orders</th><th>Source</th><th>Status</th><th>Since</th></tr></thead>
          <tbody>
            {rows.map((s) => (
              <tr key={s.email}>
                <td><input type="checkbox" checked={sel.has(s.email)} onChange={() => toggle(s.email)} aria-label={`Select ${s.email}`} /></td>
                <td><b>{s.email}</b><span className="sub">{s.name ?? '—'}</span></td>
                <td>{s.tier}</td><td>{s.orderCount}</td><td>{s.source ?? '—'}</td>
                <td><Pill value={s.unsubscribedAt ? 'cancelled' : 'active'} label={s.unsubscribedAt ? 'unsubscribed' : 'active'} /></td>
                <td>{fmtDate(s.createdAt)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <Modal open={compose} title={`Email ${sel.size} subscriber(s)`} onClose={() => setCompose(false)}>
        <form onSubmit={send} style={{ display: 'contents' }}>
          <label className="field span-all">Subject<input className="input" required value={subject} onChange={(e) => setSubject(e.target.value)} /></label>
          <label className="field span-all">Message<textarea className="textarea" rows={7} required value={body} onChange={(e) => setBody(e.target.value)} /></label>
          <div className="form-actions span-all"><button type="submit" className="btn btn--primary btn--sm">Send</button><button type="button" className="btn btn--outline btn--sm" onClick={() => setCompose(false)}>Cancel</button></div>
        </form>
      </Modal>
    </>
  );
}

export function Messages() {
  const toast = useToast();
  const writable = useWritable('messages');
  const [status, setStatus] = useState('');
  const { data, error, loading, reload } = useLoad(() => adminApi.messages.list(status || undefined), [status]);
  const [replyTo, setReplyTo] = useState<number | null>(null);
  const [reply, setReply] = useState('');
  const send = async (e: FormEvent) => {
    e.preventDefault();
    if (replyTo == null) return;
    try {
      await adminApi.messages.reply(replyTo, reply);
      toast('Reply sent.'); setReplyTo(null); setReply(''); reload();
    } catch (err) {
      toast(errMsg(err), 'err');
    }
  };
  return (
    <>
      <div className="adm-toolbar"><label className="field" htmlFor="ms">Status<select id="ms" className="select" value={status} onChange={(e) => setStatus(e.target.value)}><option value="">All</option><option value="open">Open</option><option value="answered">Answered</option><option value="closed">Closed</option></select></label></div>
      <LoadState loading={loading && !data} error={error} />
      {(data ?? []).map((m) => (
        <div className="rev" key={m.id}>
          <div className="rev__head"><b>{m.subject ?? '(no subject)'}</b><span className="form-hint">{m.name} · {m.email} · {fmtDateTime(m.createdAt)}{m.orderId ? ` · ${m.orderId}` : ''}</span><Pill value={m.status} /></div>
          <p>{m.body}</p>
          {m.reply && <p className="status" style={{ alignSelf: 'flex-start' }}>↩ {m.reply}</p>}
          {writable && (
            <div className="table-actions">
              <button type="button" className="btn btn--primary btn--xs" onClick={() => { setReplyTo(m.id); setReply(m.reply ?? ''); }}>Reply</button>
              {m.status !== 'closed' && <button type="button" className="btn btn--outline btn--xs" onClick={() => adminApi.messages.setStatus(m.id, 'closed').then(reload)}>Close</button>}
            </div>
          )}
        </div>
      ))}
      {!loading && !data?.length && <p className="form-hint">No messages.</p>}
      <Modal open={replyTo != null} title="Reply by email" onClose={() => setReplyTo(null)}>
        <form onSubmit={send} style={{ display: 'contents' }}>
          <label className="field span-all">Reply<textarea className="textarea" rows={6} required value={reply} onChange={(e) => setReply(e.target.value)} /></label>
          <div className="form-actions span-all"><button type="submit" className="btn btn--primary btn--sm">Send reply</button><button type="button" className="btn btn--outline btn--sm" onClick={() => setReplyTo(null)}>Cancel</button></div>
        </form>
      </Modal>
    </>
  );
}

export function StockAlerts() {
  const { data, error, loading } = useLoad(adminApi.stockAlerts, []);
  return (
    <>
      <p className="form-hint" style={{ marginBottom: 12 }}>Customers who asked to be told when a size is back. Emails go out automatically when stock returns.</p>
      <LoadState loading={loading && !data} error={error} />
      <div className="adm-table">
        <table>
          <thead><tr><th>Product</th><th>SKU</th><th>Email</th><th>Status</th><th>Requested</th></tr></thead>
          <tbody>
            {(data ?? []).map((a) => <tr key={a.id}><td>{a.productName}</td><td>{a.sku}</td><td>{a.email}</td><td><Pill value={a.sentAt ? 'paid' : 'pending'} label={a.sentAt ? `sent ${fmtDate(a.sentAt)}` : 'waiting'} /></td><td>{fmtDate(a.createdAt)}</td></tr>)}
            {!loading && !data?.length && <tr><td colSpan={5} className="form-hint">No alerts requested.</td></tr>}
          </tbody>
        </table>
      </div>
    </>
  );
}

export function Content() {
  const toast = useToast();
  const writable = useWritable('content');
  const { data, error, loading, reload } = useLoad(adminApi.content.list, []);
  const [edit, setEdit] = useState<ContentBlock | null>(null);
  const [bodyText, setBodyText] = useState('');
  const open = (c: ContentBlock) => { setEdit(c); setBodyText(JSON.stringify(c.body, null, 2)); };
  const save = async (e: FormEvent) => {
    e.preventDefault();
    if (!edit) return;
    try {
      const body = JSON.parse(bodyText);
      await adminApi.content.save(edit.key, { ...edit, body });
      toast('Content saved.'); setEdit(null); reload();
    } catch (err) {
      toast(err instanceof SyntaxError ? 'Body must be valid JSON.' : errMsg(err), 'err');
    }
  };
  return (
    <>
      {writable && <button type="button" className="btn btn--primary btn--sm" style={{ marginBottom: 12 }} onClick={() => open({ key: '', type: 'block', title: '', body: {}, status: 'draft', publishAt: null })}>+ New block</button>}
      <LoadState loading={loading && !data} error={error} />
      <div className="adm-table">
        <table>
          <thead><tr><th>Key</th><th>Type</th><th>Title</th><th>Status</th><th>Updated</th>{writable && <th>Actions</th>}</tr></thead>
          <tbody>
            {(data ?? []).map((c) => (
              <tr key={c.key}>
                <td><code>{c.key}</code></td><td>{c.type}</td><td>{c.title}</td><td><Pill value={c.status} /></td><td>{fmtDateTime(c.updatedAt)}</td>
                {writable && <td><div className="table-actions"><button type="button" className="btn btn--outline btn--xs" onClick={() => open(c)}>Edit</button><button type="button" className="btn btn--outline btn--xs btn--danger" onClick={() => confirm(`Delete ${c.key}?`) && adminApi.content.remove(c.key).then(reload)}>Delete</button></div></td>}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <Modal open={!!edit} title={edit?.key ? `Edit ${edit.key}` : 'New content block'} onClose={() => setEdit(null)}>
        {edit && (
          <form onSubmit={save} style={{ display: 'contents' }}>
            <label className="field">Key<input className="input" value={edit.key} disabled={!!data?.some((c) => c.key === edit.key)} onChange={(e) => setEdit({ ...edit, key: e.target.value })} required placeholder="hero, faq, legal.terms…" /></label>
            <label className="field">Type<select className="select" value={edit.type} onChange={(e) => setEdit({ ...edit, type: e.target.value as ContentBlock['type'] })}>{['block', 'page', 'banner', 'faq', 'email'].map((t) => <option key={t}>{t}</option>)}</select></label>
            <label className="field">Title<input className="input" value={edit.title ?? ''} onChange={(e) => setEdit({ ...edit, title: e.target.value })} /></label>
            <label className="field">Status<select className="select" value={edit.status} onChange={(e) => setEdit({ ...edit, status: e.target.value as ContentBlock['status'] })}>{['draft', 'scheduled', 'published'].map((t) => <option key={t}>{t}</option>)}</select></label>
            <label className="field span-all">Body (JSON)<textarea className="textarea" rows={12} style={{ fontFamily: 'ui-monospace, monospace', fontSize: 13 }} value={bodyText} onChange={(e) => setBodyText(e.target.value)} /></label>
            <div className="form-actions span-all"><button type="submit" className="btn btn--primary btn--sm">Save</button><button type="button" className="btn btn--outline btn--sm" onClick={() => setEdit(null)}>Cancel</button></div>
          </form>
        )}
      </Modal>
    </>
  );
}

export function Settings() {
  const toast = useToast();
  const writable = useWritable('settings');
  const { data, error, loading, setData } = useLoad(adminApi.settings.get, []);
  const [busy, setBusy] = useState(false);
  if (!data) return <LoadState loading={loading} error={error} />;
  const set = <K extends keyof SettingsT>(k: K, v: SettingsT[K]) => setData({ ...data, [k]: v });
  const submit = async (e: FormEvent) => {
    e.preventDefault();
    setBusy(true);
    try {
      setData(await adminApi.settings.update(data));
      toast('Settings saved.');
    } catch (err) {
      toast(errMsg(err), 'err');
    } finally {
      setBusy(false);
    }
  };
  const T = ({ k, label, type = 'text' }: { k: keyof SettingsT; label: string; type?: string }) => (
    <label className="field">{label}<input className="input" type={type} value={String(data[k] ?? '')} disabled={!writable} onChange={(e) => set(k, (type === 'number' ? Number(e.target.value) : e.target.value) as never)} /></label>
  );
  return (
    <form className="adm-form" onSubmit={submit}>
      <T k="storeName" label="Store name" /><T k="supportEmail" label="Support email" />
      <label className="field">Currency<select className="select" value={data.currency} disabled={!writable} onChange={(e) => set('currency', e.target.value)}><option>GBP</option><option>EUR</option><option>USD</option></select></label>
      <T k="freeDeliveryThreshold" label="Free delivery over (pence)" type="number" />
      <T k="dispatchCutoff" label="Dispatch cut-off (HH:MM)" /><T k="returnsDays" label="Returns window (days)" type="number" />
      <label className="check" style={{ paddingTop: 22 }}><input type="checkbox" checked={data.maintenanceMode} disabled={!writable} onChange={(e) => set('maintenanceMode', e.target.checked)} /> Maintenance mode</label>
      <div />
      <T k="companyName" label="Company name" /><T k="companyNumber" label="Company number" /><T k="vatNumber" label="VAT number" />
      <label className="field span-all">Registered address<input className="input" value={data.registeredAddress ?? ''} disabled={!writable} onChange={(e) => set('registeredAddress', e.target.value)} /></label>
      {writable && <div className="form-actions span-all"><button type="submit" className="btn btn--primary btn--sm" disabled={busy}>Save settings</button></div>}
    </form>
  );
}

export function Audit() {
  const { data, error, loading } = useLoad(adminApi.audit, []);
  return (
    <>
      <LoadState loading={loading && !data} error={error} />
      <div className="adm-table">
        <table>
          <thead><tr><th>When</th><th>Who</th><th>Action</th><th>Entity</th><th>Details</th></tr></thead>
          <tbody>
            {(data ?? []).map((a) => <tr key={a.id}><td>{fmtDateTime(a.created_at)}</td><td>{a.actor ?? '—'}</td><td>{a.action}</td><td>{a.entity} <span className="sub">{a.entity_id}</span></td><td><code style={{ fontSize: 12 }}>{a.diff ? JSON.stringify(a.diff).slice(0, 120) : ''}</code></td></tr>)}
            {!loading && !data?.length && <tr><td colSpan={5} className="form-hint">No activity yet.</td></tr>}
          </tbody>
        </table>
      </div>
    </>
  );
}
