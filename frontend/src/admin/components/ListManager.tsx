/**
 * Config-driven CRUD list — React port of printforge's AdminListManager:
 * inline create form + searchable table + edit modal. Used by collections,
 * discount codes, delivery methods, taxes, emails, blog, users, …
 */
import { useMemo, useState, type FormEvent, type ReactNode } from 'react';
import { Modal, errMsg, useLoad, useToast } from './ui';

export type FieldType = 'text' | 'number' | 'select' | 'toggle' | 'textarea' | 'date' | 'json';

export interface Field<T> {
  id: keyof T & string;
  label: string;
  type?: FieldType;
  options?: Array<{ value: string; label: string } | string>;
  placeholder?: string;
  required?: boolean;
  span?: 2;
  /** Hide in the inline create form (still editable in the modal). */
  modalOnly?: boolean;
  /** Read-only once created (e.g. primary key). */
  immutable?: boolean;
}

export interface Column<T> {
  key: keyof T & string;
  label: string;
  render?: (item: T) => ReactNode;
}

export interface ListConfig<T extends object> {
  noun: string;
  idKey: keyof T & string;
  fields: Field<T>[];
  columns: Column<T>[];
  api: {
    list: () => Promise<T[]>;
    create?: (payload: Partial<T>) => Promise<unknown>;
    update?: (id: string | number, payload: Partial<T>) => Promise<unknown>;
    remove?: (id: string | number) => Promise<unknown>;
  };
  readOnly?: boolean;
  searchKeys?: Array<keyof T & string>;
  defaults?: Partial<T>;
  /** Extra read-only columns / actions per row. */
  rowActions?: (item: T, reload: () => void) => ReactNode;
}

function FieldInput<T extends object>({ f, value, onChange, disabled, id }: { f: Field<T>; value: unknown; onChange: (v: unknown) => void; disabled?: boolean; id: string }) {
  const t = f.type ?? 'text';
  if (t === 'toggle') {
    return (
      <label className="check" htmlFor={id} style={{ paddingTop: 22 }}>
        <input id={id} type="checkbox" checked={!!value} onChange={(e) => onChange(e.target.checked)} disabled={disabled} /> {f.label}
      </label>
    );
  }
  return (
    <label className={`field${f.span === 2 ? ' span-2' : ''}`} htmlFor={id}>
      {f.label}
      {t === 'select' ? (
        <select id={id} className="select" value={String(value ?? '')} onChange={(e) => onChange(e.target.value)} disabled={disabled}>
          {(f.options ?? []).map((o) => {
            const v = typeof o === 'string' ? o : o.value;
            const l = typeof o === 'string' ? o : o.label;
            return (
              <option key={v} value={v}>
                {l}
              </option>
            );
          })}
        </select>
      ) : t === 'textarea' || t === 'json' ? (
        <textarea id={id} className="textarea" rows={t === 'json' ? 8 : 4} value={typeof value === 'string' ? value : value == null ? '' : JSON.stringify(value, null, 2)} onChange={(e) => onChange(e.target.value)} disabled={disabled} placeholder={f.placeholder} style={t === 'json' ? { fontFamily: 'ui-monospace, monospace', fontSize: 13 } : undefined} />
      ) : (
        <input id={id} className="input" type={t === 'number' ? 'number' : t === 'date' ? 'datetime-local' : 'text'} value={value == null ? '' : t === 'date' && typeof value === 'string' ? value.slice(0, 16) : String(value)} onChange={(e) => onChange(t === 'number' ? (e.target.value === '' ? null : Number(e.target.value)) : e.target.value)} disabled={disabled} placeholder={f.placeholder} required={f.required} />
      )}
    </label>
  );
}

export function ListManager<T extends object>({ cfg }: { cfg: ListConfig<T> }) {
  const toast = useToast();
  const { data, error, loading, reload } = useLoad(cfg.api.list, []);
  const [q, setQ] = useState('');
  const [draft, setDraft] = useState<Partial<T>>(cfg.defaults ?? {});
  const [editing, setEditing] = useState<Partial<T> | null>(null);
  const [busy, setBusy] = useState(false);

  const items = useMemo(() => {
    const list = data ?? [];
    if (!q) return list;
    const n = q.toLowerCase();
    const keys = cfg.searchKeys ?? cfg.columns.map((c) => c.key);
    return list.filter((i) => keys.some((k) => String((i as Record<string, unknown>)[k] ?? '').toLowerCase().includes(n)));
  }, [data, q, cfg]);

  const normalise = (payload: Partial<T>): Partial<T> => {
    const out: Record<string, unknown> = { ...payload };
    for (const f of cfg.fields) {
      const v = out[f.id];
      if (f.type === 'json' && typeof v === 'string') {
        try {
          out[f.id] = v.trim() ? JSON.parse(v) : {};
        } catch {
          throw new Error(`${f.label} must be valid JSON.`);
        }
      }
      if (f.type === 'date' && typeof v === 'string') out[f.id] = v ? new Date(v).toISOString() : null;
      if (f.type === 'number' && v === '') out[f.id] = null;
    }
    return out as Partial<T>;
  };

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    const missing = cfg.fields.find((f) => f.required && !f.modalOnly && !(draft as Record<string, unknown>)[f.id]);
    if (missing) return toast(`Fill in ${missing.label}.`, 'err');
    setBusy(true);
    try {
      await cfg.api.create?.(normalise(draft));
      toast(`${cfg.noun} created.`);
      setDraft(cfg.defaults ?? {});
      reload();
    } catch (err) {
      toast(errMsg(err), 'err');
    } finally {
      setBusy(false);
    }
  };

  const saveEdit = async (e: FormEvent) => {
    e.preventDefault();
    if (!editing) return;
    setBusy(true);
    try {
      await cfg.api.update?.(String((editing as Record<string, unknown>)[cfg.idKey]), normalise(editing));
      toast(`${cfg.noun} updated.`);
      setEditing(null);
      reload();
    } catch (err) {
      toast(errMsg(err), 'err');
    } finally {
      setBusy(false);
    }
  };

  const remove = async (item: T) => {
    if (!confirm(`Remove this ${cfg.noun.toLowerCase()}?`)) return;
    try {
      await cfg.api.remove?.(String((item as Record<string, unknown>)[cfg.idKey]));
      toast(`${cfg.noun} removed.`);
      reload();
    } catch (err) {
      toast(errMsg(err), 'err');
    }
  };

  const canWrite = !cfg.readOnly;

  return (
    <>
      {canWrite && cfg.api.create ? (
        <form className="adm-form" onSubmit={submit} aria-label={`Create ${cfg.noun}`}>
          {cfg.fields.filter((f) => !f.modalOnly).map((f) => (
            <FieldInput key={f.id} f={f} id={`new-${f.id}`} value={(draft as Record<string, unknown>)[f.id]} onChange={(v) => setDraft((d) => ({ ...d, [f.id]: v }))} />
          ))}
          <div className="form-actions">
            <button type="submit" className="btn btn--primary btn--sm" disabled={busy}>
              Save {cfg.noun.toLowerCase()}
            </button>
            <button type="button" className="btn btn--outline btn--sm" onClick={() => setDraft(cfg.defaults ?? {})}>
              Clear
            </button>
          </div>
        </form>
      ) : (
        !canWrite && <p className="form-hint" style={{ marginBottom: 12 }}>Read-only: your role can view this section but not change it.</p>
      )}

      <div className="adm-toolbar">
        <label className="field" htmlFor="lm-search">
          Search
          <input id="lm-search" className="input" value={q} onChange={(e) => setQ(e.target.value)} placeholder={`Filter ${cfg.noun.toLowerCase()}s…`} />
        </label>
        <span className="form-hint">{loading ? 'Loading…' : `${items.length} record${items.length === 1 ? '' : 's'}`}</span>
      </div>

      {error && <p className="status status--danger">{error}</p>}
      <div className="adm-table">
        <table>
          <thead>
            <tr>
              {cfg.columns.map((c) => (
                <th key={c.key}>{c.label}</th>
              ))}
              {(canWrite || cfg.rowActions) && <th>Actions</th>}
            </tr>
          </thead>
          <tbody>
            {items.map((item) => (
              <tr key={String((item as Record<string, unknown>)[cfg.idKey])}>
                {cfg.columns.map((c) => (
                  <td key={c.key}>{c.render ? c.render(item) : String((item as Record<string, unknown>)[c.key] ?? '')}</td>
                ))}
                {(canWrite || cfg.rowActions) && (
                  <td>
                    <div className="table-actions">
                      {cfg.rowActions?.(item, reload)}
                      {canWrite && cfg.api.update && (
                        <button type="button" className="btn btn--outline btn--xs" onClick={() => setEditing(item)}>
                          Edit
                        </button>
                      )}
                      {canWrite && cfg.api.remove && (
                        <button type="button" className="btn btn--outline btn--xs btn--danger" onClick={() => remove(item)}>
                          Remove
                        </button>
                      )}
                    </div>
                  </td>
                )}
              </tr>
            ))}
            {!loading && items.length === 0 && (
              <tr>
                <td colSpan={cfg.columns.length + 1} className="form-hint">
                  No records.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      <Modal open={!!editing} title={`Edit ${cfg.noun.toLowerCase()}`} onClose={() => setEditing(null)}>
        {editing && (
          <form onSubmit={saveEdit} style={{ display: 'contents' }}>
            {cfg.fields.map((f) => (
              <FieldInput key={f.id} f={f} id={`edit-${f.id}`} value={(editing as Record<string, unknown>)[f.id]} disabled={f.immutable} onChange={(v) => setEditing((d) => ({ ...(d ?? {}), [f.id]: v }))} />
            ))}
            <div className="form-actions span-all">
              <button type="submit" className="btn btn--primary btn--sm" disabled={busy}>
                Save changes
              </button>
              <button type="button" className="btn btn--outline btn--sm" onClick={() => setEditing(null)}>
                Cancel
              </button>
            </div>
          </form>
        )}
      </Modal>
    </>
  );
}

/* ── Toggle list (notifications, integrations) ─────────────────────────────── */
export function ToggleList({ api, flagKey, onLabel, offLabel, readOnly }: { api: { list: () => Promise<Array<Record<string, unknown>>>; toggle: (id: string | number, v: boolean) => Promise<unknown> }; flagKey: string; onLabel: string; offLabel: string; readOnly?: boolean }) {
  const toast = useToast();
  const { data, error, loading, setData } = useLoad(api.list, []);
  const flip = async (row: Record<string, unknown>, v: boolean) => {
    const id = (row.id ?? row.key) as string | number;
    setData((d) => (d ?? []).map((r) => (r === row ? { ...r, [flagKey]: v } : r)));
    try {
      await api.toggle(id, v);
      toast(`${row.name}: ${v ? onLabel : offLabel}.`);
    } catch (e) {
      setData((d) => (d ?? []).map((r) => ((r.id ?? r.key) === id ? { ...r, [flagKey]: !v } : r)));
      toast(errMsg(e), 'err');
    }
  };
  if (error) return <p className="status status--danger">{error}</p>;
  if (loading) return <div className="skeleton" style={{ height: 160 }} />;
  return (
    <div className="toggle-list">
      {(data ?? []).map((row) => (
        <div className="toggle-row" key={String(row.id ?? row.key)}>
          <div>
            <strong>{String(row.name)}</strong>
            <span className="form-hint">{String(row.description ?? row.subject ?? '')}</span>
          </div>
          <label className="switch">
            <input type="checkbox" checked={!!row[flagKey]} disabled={readOnly} onChange={(e) => flip(row, e.target.checked)} />
            <span className={`status${row[flagKey] ? ' status--ok' : ' status--muted'}`}>{row[flagKey] ? onLabel : offLabel}</span>
          </label>
        </div>
      ))}
    </div>
  );
}
