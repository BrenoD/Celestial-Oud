import { useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { useAuthStore } from '@/store/auth';
import { formatPence } from '@/lib/money';
import { adminApi } from '../api';
import { canWrite } from '../rbac';
import type { AdminOrder } from '../types';
import { LoadState, Modal, Pill, errMsg, exportCsv, fmtDateTime, useLoad, useToast } from '../components/ui';

const NEXT: Record<string, Array<[string, string]>> = {
  paid: [['processing', 'Start processing'], ['cancelled', 'Cancel']],
  processing: [['cancelled', 'Cancel']],
  dispatched: [['delivered', 'Mark delivered']],
};

function OrderCard({ o, writable, reload }: { o: AdminOrder; writable: boolean; reload: () => void }) {
  const toast = useToast();
  const [open, setOpen] = useState(false);
  const [ship, setShip] = useState(false);
  const [carrier, setCarrier] = useState(o.tracking?.carrier ?? 'Royal Mail');
  const [number, setNumber] = useState(o.tracking?.number ?? '');
  const a = o.shippingAddress;

  const act = async (fn: () => Promise<unknown>, msg: string) => {
    try {
      await fn();
      toast(msg);
      reload();
    } catch (e) {
      toast(errMsg(e), 'err');
    }
  };

  return (
    <div className="ocard">
      <div className="ocard__head" onClick={() => setOpen((v) => !v)} role="button" tabIndex={0} aria-expanded={open}>
        <span className="ocard__id">{o.id}</span>
        <Pill value={o.status} />
        <span className="ocard__meta">{o.customerName} · {fmtDateTime(o.createdAt)} · {o.lines.reduce((n, l) => n + l.qty, 0)} items</span>
        <b>{formatPence(o.total)}</b>
        {writable && (
          <span className="table-actions" onClick={(e) => e.stopPropagation()}>
            {['paid', 'processing'].includes(o.status) && <button type="button" className="btn btn--primary btn--xs" onClick={() => setShip(true)}>Dispatch</button>}
            {(NEXT[o.status] ?? []).map(([s, l]) => (
              <button key={s} type="button" className={`btn btn--outline btn--xs${s === 'cancelled' ? ' btn--danger' : ''}`} onClick={() => (s !== 'cancelled' || confirm('Cancel this order and restock the items?')) && act(() => adminApi.orders.setStatus(o.id, s), `Order ${l.toLowerCase()}.`)}>{l}</button>
            ))}
            {['paid', 'processing', 'dispatched', 'delivered'].includes(o.status) && <button type="button" className="btn btn--outline btn--xs btn--danger" onClick={() => confirm(`Refund ${formatPence(o.total)} via Stripe? This cannot be undone.`) && act(() => adminApi.orders.refund(o.id), 'Refund issued.')}>Refund</button>}
          </span>
        )}
        <span aria-hidden="true">{open ? '▾' : '▸'}</span>
      </div>
      {open && (
        <div className="ocard__body">
          <div><label>Email</label>{o.email}</div>
          <div><label>Delivery</label>{o.deliveryMethodName ?? o.deliveryMethodId} · ETA {o.estimatedDelivery}</div>
          <div><label>Address</label>{a.line1}{a.line2 ? `, ${a.line2}` : ''}, {a.city}, {a.postcode}</div>
          <div><label>Tracking</label>{o.tracking ? `${o.tracking.carrier} · ${o.tracking.number}` : '—'}</div>
          <div><label>Extras</label>{o.freeSample ? `Sample: ${o.freeSample}` : 'No sample'}{o.giftWrap ? ' · Gift wrap' : ''}{o.discountCode ? ` · ${o.discountCode} −${formatPence(o.discount)}` : ''}</div>
          <div><label>Payment</label>{o.paymentStatus ?? '—'} {o.stripePaymentIntent ? `· ${o.stripePaymentIntent}` : ''}</div>
          {o.giftMessage && <div className="span-all"><label>Gift message</label>“{o.giftMessage}”</div>}
          <div className="ocard__lines">
            <label>Items</label>
            {o.lines.map((l) => (
              <div key={l.sku} style={{ display: 'flex', justifyContent: 'space-between', gap: 12 }}>
                <span>{l.name} <span className="sub" style={{ display: 'inline' }}>· {l.variantLabel} · {l.sku}</span></span>
                <span>×{l.qty} · {formatPence(l.unitPrice * l.qty)}</span>
              </div>
            ))}
            <div style={{ display: 'flex', justifyContent: 'space-between', borderTop: '1px solid var(--line)', paddingTop: 6 }}>
              <span>Subtotal {formatPence(o.subtotal)} · Delivery {formatPence(o.delivery)}{o.discount ? ` · Discount −${formatPence(o.discount)}` : ''}</span>
              <b>{formatPence(o.total)}</b>
            </div>
          </div>
        </div>
      )}
      <Modal open={ship} title={`Dispatch ${o.id}`} onClose={() => setShip(false)}>
        <label className="field">Carrier
          <select className="select" value={carrier} onChange={(e) => setCarrier(e.target.value)}>
            {['Royal Mail', 'DPD', 'Evri', 'UPS', 'FedEx', 'Other'].map((c) => <option key={c}>{c}</option>)}
          </select>
        </label>
        <label className="field">Tracking number<input className="input" value={number} onChange={(e) => setNumber(e.target.value)} placeholder="RM123456789GB" required /></label>
        <div className="form-actions span-all">
          <button type="button" className="btn btn--primary btn--sm" disabled={!number.trim()} onClick={() => act(() => adminApi.orders.setTracking(o.id, { carrier, number: number.trim() }), 'Marked as dispatched — customer emailed.').then(() => setShip(false))}>Confirm dispatch</button>
          <button type="button" className="btn btn--outline btn--sm" onClick={() => setShip(false)}>Cancel</button>
        </div>
      </Modal>
    </div>
  );
}

export default function Orders({ mode = 'orders' }: { mode?: 'orders' | 'shipments' }) {
  const role = useAuthStore((s) => s.user?.role);
  const writable = canWrite(role, mode);
  const [sp, setSp] = useSearchParams();
  const q = sp.get('q') ?? '';
  const status = sp.get('status') ?? (mode === 'shipments' ? 'dispatched,delivered' : '');
  const { data, error, loading, reload } = useLoad(() => adminApi.orders.list({ q: q || undefined, status: status || undefined }), [q, status]);
  const items = (data?.items ?? []).filter((o) => mode !== 'orders' || status || !['dispatched', 'delivered', 'cancelled', 'refunded'].includes(o.status));

  return (
    <>
      <div className="adm-toolbar" style={{ gridTemplateColumns: 'minmax(0,1fr) auto auto' }}>
        <label className="field" htmlFor="oq">Search<input id="oq" className="input" value={q} onChange={(e) => setSp((p) => { p.set('q', e.target.value); return p; }, { replace: true })} placeholder="Order id, email, name…" /></label>
        <label className="field" htmlFor="os">Status
          <select id="os" className="select" value={status} onChange={(e) => setSp((p) => { if (e.target.value) p.set('status', e.target.value); else p.delete('status'); return p; }, { replace: true })}>
            <option value="">{mode === 'orders' ? 'Open (paid, processing)' : 'All'}</option>
            {['pending_payment', 'paid', 'processing', 'dispatched', 'delivered', 'cancelled', 'refunded'].map((s) => <option key={s} value={s}>{s.replace('_', ' ')}</option>)}
          </select>
        </label>
        <button type="button" className="btn btn--outline btn--sm" style={{ alignSelf: 'end' }} onClick={() => exportCsv([['Order', 'Date', 'Customer', 'Email', 'Status', 'Total', 'Tracking'], ...items.map((o) => [o.id, o.createdAt, o.customerName, o.email, o.status, o.total / 100, o.tracking?.number ?? ''])], `${mode}.csv`)}>CSV</button>
      </div>
      <LoadState loading={loading && !data} error={error} />
      {!loading && items.length === 0 && <p className="form-hint">No orders match.</p>}
      {items.map((o) => <OrderCard key={o.id} o={o} writable={writable} reload={reload} />)}
    </>
  );
}
