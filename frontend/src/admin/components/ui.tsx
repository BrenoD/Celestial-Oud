import { createContext, useCallback, useContext, useEffect, useRef, useState, type ReactNode } from 'react';
import { formatPence } from '@/lib/money';

/* ── Toast ─────────────────────────────────────────────────────────────────── */
type ToastFn = (msg: string, kind?: 'ok' | 'err') => void;
const ToastCtx = createContext<ToastFn>(() => {});
export const useToast = () => useContext(ToastCtx);

export function ToastProvider({ children }: { children: ReactNode }) {
  const [t, setT] = useState<{ msg: string; kind: 'ok' | 'err' } | null>(null);
  const toast = useCallback<ToastFn>((msg, kind = 'ok') => {
    setT({ msg, kind });
    setTimeout(() => setT(null), 3200);
  }, []);
  return (
    <ToastCtx.Provider value={toast}>
      {children}
      {t && (
        <div className={`adm-toast${t.kind === 'err' ? ' adm-toast--err' : ''}`} role="status">
          {t.msg}
        </div>
      )}
    </ToastCtx.Provider>
  );
}

export const errMsg = (e: unknown) => (e instanceof Error ? e.message : 'Something went wrong');

/* ── Modal (native <dialog>) ───────────────────────────────────────────────── */
export function Modal({ open, title, onClose, children, footer }: { open: boolean; title: string; onClose: () => void; children: ReactNode; footer?: ReactNode }) {
  const ref = useRef<HTMLDialogElement>(null);
  useEffect(() => {
    const d = ref.current;
    if (!d) return;
    if (open && !d.open) d.showModal();
    if (!open && d.open) d.close();
  }, [open]);
  return (
    <dialog ref={ref} className="modal" onClose={onClose} onClick={(e) => e.target === ref.current && onClose()}>
      <div className="modal__head">
        <span>{title}</span>
        <button type="button" className="modal__close" aria-label="Close" onClick={onClose}>
          ✕
        </button>
      </div>
      <div className="modal__body">{children}</div>
      {footer && <div className="form-actions" style={{ padding: '0 18px 18px' }}>{footer}</div>}
    </dialog>
  );
}

/* ── Metrics & chart ───────────────────────────────────────────────────────── */
export function Metrics({ items }: { items: Array<[string, string]> }) {
  return (
    <div className="metrics">
      {items.map(([v, l]) => (
        <div className="metric" key={l}>
          <strong>{v}</strong>
          <span>{l}</span>
        </div>
      ))}
    </div>
  );
}

export function BarChart({ series, title }: { series: Array<{ day: string; revenue: number; orders: number }>; title: string }) {
  const max = Math.max(1, ...series.map((s) => s.revenue));
  return (
    <div className="chart">
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 10, fontSize: 13 }}>
        <b>{title}</b>
        <span className="form-hint">{formatPence(series.reduce((n, s) => n + s.revenue, 0))} total</span>
      </div>
      <div className="chart__bars" role="img" aria-label={title}>
        {series.map((s) => (
          <div key={s.day} className="chart__bar" style={{ height: `${Math.max(2, (s.revenue / max) * 100)}%` }} title={`${s.day}: ${formatPence(s.revenue)} · ${s.orders} orders`} />
        ))}
      </div>
      <div className="chart__labels">
        {series.map((s) => (
          <span key={s.day}>{s.day.slice(5)}</span>
        ))}
      </div>
    </div>
  );
}

/* ── Status pill ───────────────────────────────────────────────────────────── */
const KIND: Record<string, string> = {
  paid: 'ok', delivered: 'ok', answered: 'ok', active: 'ok', published: 'ok', approved: 'ok', true: 'ok',
  processing: 'warn', dispatched: 'warn', pending_payment: 'warn', open: 'warn', pending: 'warn', scheduled: 'warn', draft: 'muted',
  cancelled: 'danger', refunded: 'danger', canceled: 'danger', closed: 'muted', archived: 'muted', false: 'muted', inactive: 'muted',
};
export function Pill({ value, label }: { value: string | boolean; label?: string }) {
  const k = KIND[String(value)] ?? '';
  return <span className={`status${k ? ` status--${k}` : ''}`}>{label ?? String(value).replace(/_/g, ' ')}</span>;
}

/* ── Simple CSV export ─────────────────────────────────────────────────────── */
export function exportCsv(rows: Array<Array<string | number | boolean | null | undefined>>, filename: string) {
  const csv = rows.map((r) => r.map((v) => `"${String(v ?? '').replace(/"/g, '""')}"`).join(',')).join('\n');
  const url = URL.createObjectURL(new Blob([csv], { type: 'text/csv' }));
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}

export const fmtDate = (iso?: string | null) => (iso ? new Date(iso).toLocaleDateString('en-GB', { day: '2-digit', month: 'short', year: 'numeric' }) : '—');
export const fmtDateTime = (iso?: string | null) => (iso ? new Date(iso).toLocaleString('en-GB', { day: '2-digit', month: 'short', hour: '2-digit', minute: '2-digit' }) : '—');

/* ── Async data hook ───────────────────────────────────────────────────────── */
export function useLoad<T>(fn: () => Promise<T>, deps: unknown[] = []) {
  const [data, setData] = useState<T | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const reload = useCallback(() => {
    setLoading(true);
    fn()
      .then((d) => {
        setData(d);
        setError(null);
      })
      .catch((e) => setError(errMsg(e)))
      .finally(() => setLoading(false));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, deps);
  useEffect(() => {
    reload();
  }, [reload]);
  return { data, error, loading, reload, setData };
}

export function LoadState({ loading, error }: { loading: boolean; error: string | null }) {
  if (error) return <p className="status status--danger">{error}</p>;
  if (loading) return <div className="skeleton" style={{ height: 120 }} />;
  return null;
}
