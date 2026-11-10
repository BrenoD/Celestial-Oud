import { useState, type FormEvent } from 'react';
import { useAuthStore } from '@/store/auth';
import { formatPence } from '@/lib/money';
import { adminApi } from '../api';
import { canWrite } from '../rbac';
import type { AdminProduct, ProductInput } from '../types';
import { LoadState, Modal, Pill, errMsg, useLoad, useToast } from '../components/ui';

const FAMILIES = ['woody-oud', 'amber-oriental', 'floral', 'fresh-musk', 'gourmand', 'spicy'];
const WEAR = ['day', 'evening', 'summer', 'winter'];
const TAGS = ['bestseller', 'new', 'top-rated', 'gift', 'set', 'offer', 'oud'];
const list = (s: string) => s.split(/[,\n]/).map((x) => x.trim()).filter(Boolean);

const empty = (): ProductInput => ({
  name: '', tagline: '', notesLine: '', badge: '', tags: [], family: 'woody-oud', concentration: 'Eau de Parfum', wear: [], images: [],
  pyramid: { top: [], heart: [], base: [] }, profile: { intensity: 50, longevity: 50, sweetness: 50, freshness: 50 }, bestFor: '', description: '',
  howToWear: 'Spray 2–3 times on pulse points — neck, wrists, behind the ears. Do not rub.', ingredients: '', allergens: [], status: 'draft', sort: 0,
  keyNotes: [], collectionSlugs: [],
  sizes: [{ label: '30ml', ml: 30, price: 0, stock: 0 }, { label: '50ml', ml: 50, price: 0, stock: 0 }, { label: '100ml', ml: 100, price: 0, stock: 0 }, { label: '2ml sample', ml: 2, price: 400, stock: 0, isSample: true }],
});

const fromProduct = (p: AdminProduct): ProductInput => ({
  slug: p.slug, name: p.name, tagline: p.tagline, notesLine: p.notesLine, badge: p.badge ?? '', tags: p.tags, family: p.family, concentration: p.concentration ?? 'Eau de Parfum',
  wear: p.wear, images: p.images, pyramid: p.pyramid, profile: p.profile, bestFor: p.bestFor, description: p.description, howToWear: p.howToWear, ingredients: p.ingredients,
  allergens: p.allergens ?? [], responsiblePerson: p.responsiblePerson ?? '', cpsrRef: p.cpsrRef ?? '', scpnRef: p.scpnRef ?? '', status: p.status, sort: p.sort,
  keyNotes: p.keyNotes, collectionSlugs: p.collectionSlugs ?? [p.collection.slug],
  sizes: [...p.sizes.map((s) => ({ ...s, stock: s.stock ?? 0 })), ...(p.samplePrice !== undefined ? [{ sku: `${p.slug}-2ml`, label: '2ml sample', ml: 2, price: p.samplePrice, stock: 999, isSample: true }] : [])],
});

function Editor({ initial, id, onDone }: { initial: ProductInput; id: number | null; onDone: () => void }) {
  const toast = useToast();
  const [f, setF] = useState<ProductInput>(initial);
  const [busy, setBusy] = useState(false);
  const { data: collections } = useLoad(adminApi.collections.list, []);
  const set = <K extends keyof ProductInput>(k: K, v: ProductInput[K]) => setF((p) => ({ ...p, [k]: v }));
  const toggle = (k: 'tags' | 'wear' | 'collectionSlugs', v: string) => set(k, f[k].includes(v) ? f[k].filter((x) => x !== v) : [...f[k], v]);
  const setSize = (i: number, patch: Partial<ProductInput['sizes'][number]>) => set('sizes', f.sizes.map((s, j) => (j === i ? { ...s, ...patch } : s)));

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    if (!f.name.trim()) return toast('A name is required.', 'err');
    setBusy(true);
    try {
      await adminApi.products.save(id, { ...f, badge: f.badge || null, responsiblePerson: f.responsiblePerson || null, cpsrRef: f.cpsrRef || null, scpnRef: f.scpnRef || null });
      toast(id ? 'Product updated.' : 'Product created.');
      onDone();
    } catch (err) {
      toast(errMsg(err), 'err');
    } finally {
      setBusy(false);
    }
  };

  const Text = ({ k, label, span, ph }: { k: keyof ProductInput; label: string; span?: boolean; ph?: string }) => (
    <label className={`field${span ? ' span-all' : ''}`}>
      {label}
      <input className="input" value={String(f[k] ?? '')} placeholder={ph} onChange={(e) => set(k, e.target.value as never)} />
    </label>
  );

  return (
    <form onSubmit={submit} style={{ display: 'contents' }}>
      <Text k="name" label="Name" />
      <Text k="slug" label="Slug (URL)" ph="auto from name" />
      <Text k="tagline" label="Tagline" span />
      <Text k="notesLine" label="Notes line (card)" ph="Oud · Saffron · Rose" />
      <Text k="badge" label="Badge" ph="Bestseller / New / Gift" />
      <label className="field">
        Scent family
        <select className="select" value={f.family} onChange={(e) => set('family', e.target.value)}>
          {FAMILIES.map((x) => <option key={x}>{x}</option>)}
        </select>
      </label>
      <label className="field">
        Status
        <select className="select" value={f.status} onChange={(e) => set('status', e.target.value as ProductInput['status'])}>
          <option value="draft">draft</option>
          <option value="active">active</option>
          <option value="archived">archived</option>
        </select>
      </label>
      <div className="field span-all">
        Tags
        <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
          {TAGS.map((t) => (
            <button key={t} type="button" className="option option--sm" aria-pressed={f.tags.includes(t)} onClick={() => toggle('tags', t)}>{t}</button>
          ))}
        </div>
      </div>
      <div className="field">
        Wear
        <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
          {WEAR.map((t) => (
            <button key={t} type="button" className="option option--sm" aria-pressed={f.wear.includes(t)} onClick={() => toggle('wear', t)}>{t}</button>
          ))}
        </div>
      </div>
      <div className="field">
        Collections
        <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
          {(collections ?? []).map((c) => (
            <button key={c.slug} type="button" className="option option--sm" aria-pressed={f.collectionSlugs.includes(c.slug)} onClick={() => toggle('collectionSlugs', c.slug)}>{c.name}</button>
          ))}
        </div>
      </div>

      <div className="field span-all">
        Sizes & stock
        <div className="sizes-grid">
          {['Label', 'ml', 'Price (pence)', 'Was (pence)', 'Stock', 'Sample', ''].map((h, i) => <span key={i} className="sizes-grid__head">{h}</span>)}
          {f.sizes.map((s, i) => (
            <div key={i} style={{ display: 'contents' }}>
              <input className="input" value={s.label} onChange={(e) => setSize(i, { label: e.target.value })} aria-label="Size label" />
              <input className="input" type="number" value={s.ml ?? 0} onChange={(e) => setSize(i, { ml: Number(e.target.value) })} aria-label="Millilitres" />
              <input className="input" type="number" value={s.price} onChange={(e) => setSize(i, { price: Number(e.target.value) })} aria-label="Price in pence" />
              <input className="input" type="number" value={s.compareAtPrice ?? ''} onChange={(e) => setSize(i, { compareAtPrice: e.target.value === '' ? undefined : Number(e.target.value) })} aria-label="Compare-at price" />
              <input className="input" type="number" value={s.stock} onChange={(e) => setSize(i, { stock: Number(e.target.value) })} aria-label="Stock" />
              <input type="checkbox" checked={!!s.isSample} onChange={(e) => setSize(i, { isSample: e.target.checked })} aria-label="Is 2ml sample" />
              <button type="button" className="btn btn--outline btn--xs btn--danger" onClick={() => set('sizes', f.sizes.filter((_, j) => j !== i))}>✕</button>
            </div>
          ))}
        </div>
        <button type="button" className="btn btn--outline btn--xs" style={{ alignSelf: 'flex-start', marginTop: 6 }} onClick={() => set('sizes', [...f.sizes, { label: '', ml: 0, price: 0, stock: 0 }])}>+ Add size</button>
        <span className="form-hint">Prices in pence (5500 = £55.00). Leave “Was” empty unless on offer.</span>
      </div>

      <label className="field">Key notes (filters)<input className="input" value={f.keyNotes.join(', ')} onChange={(e) => set('keyNotes', list(e.target.value))} placeholder="oud, saffron, rose" /></label>
      <label className="field">Best for<input className="input" value={f.bestFor} onChange={(e) => set('bestFor', e.target.value)} /></label>
      <label className="field">Top notes<input className="input" value={f.pyramid.top.join(', ')} onChange={(e) => set('pyramid', { ...f.pyramid, top: list(e.target.value) })} /></label>
      <label className="field">Heart notes<input className="input" value={f.pyramid.heart.join(', ')} onChange={(e) => set('pyramid', { ...f.pyramid, heart: list(e.target.value) })} /></label>
      <label className="field span-all">Base notes<input className="input" value={f.pyramid.base.join(', ')} onChange={(e) => set('pyramid', { ...f.pyramid, base: list(e.target.value) })} /></label>
      {(['intensity', 'longevity', 'sweetness', 'freshness'] as const).map((k) => (
        <label key={k} className="field">
          {k[0].toUpperCase() + k.slice(1)}: {f.profile[k]}
          <input type="range" min={0} max={100} value={f.profile[k]} onChange={(e) => set('profile', { ...f.profile, [k]: Number(e.target.value) })} />
        </label>
      ))}
      <label className="field span-all">Description<textarea className="textarea" rows={4} value={f.description} onChange={(e) => set('description', e.target.value)} /></label>
      <label className="field span-all">How to wear<textarea className="textarea" rows={2} value={f.howToWear} onChange={(e) => set('howToWear', e.target.value)} /></label>
      <label className="field span-all">Ingredients (INCI) — required by UK cosmetics law<textarea className="textarea" rows={3} value={f.ingredients} onChange={(e) => set('ingredients', e.target.value)} /></label>
      <label className="field">Allergens<input className="input" value={f.allergens.join(', ')} onChange={(e) => set('allergens', list(e.target.value))} placeholder="Linalool, Limonene" /></label>
      <Text k="responsiblePerson" label="Responsible Person" />
      <Text k="cpsrRef" label="CPSR reference" />
      <Text k="scpnRef" label="SCPN notification" />
      <label className="field span-all">Image URLs (one per line)<textarea className="textarea" rows={2} value={f.images.join('\n')} onChange={(e) => set('images', list(e.target.value))} /></label>
      <div className="form-actions span-all">
        <button type="submit" className="btn btn--primary btn--sm" disabled={busy}>{busy ? 'Saving…' : 'Save product'}</button>
        <button type="button" className="btn btn--outline btn--sm" onClick={onDone}>Cancel</button>
      </div>
    </form>
  );
}

export default function Products() {
  const role = useAuthStore((s) => s.user?.role);
  const writable = canWrite(role, 'products');
  const toast = useToast();
  const [q, setQ] = useState('');
  const { data, error, loading, reload } = useLoad(() => adminApi.products.list(q), [q]);
  const [editor, setEditor] = useState<{ id: number | null; input: ProductInput } | null>(null);

  const edit = async (id: number) => {
    try {
      setEditor({ id, input: fromProduct(await adminApi.products.get(id)) });
    } catch (e) {
      toast(errMsg(e), 'err');
    }
  };
  const remove = async (id: number, name: string) => {
    if (!confirm(`Remove ${name}? Products that were ever ordered are archived instead.`)) return;
    try {
      await adminApi.products.remove(id);
      toast('Removed.');
      reload();
    } catch (e) {
      toast(errMsg(e), 'err');
    }
  };

  return (
    <>
      <div className="adm-toolbar">
        <label className="field" htmlFor="pq">Search<input id="pq" className="input" value={q} onChange={(e) => setQ(e.target.value)} placeholder="Name or notes…" /></label>
        {writable && <button type="button" className="btn btn--primary btn--sm" onClick={() => setEditor({ id: null, input: empty() })}>+ New product</button>}
      </div>
      <LoadState loading={loading && !data} error={error} />
      <div className="adm-table">
        <table>
          <thead><tr><th>Product</th><th>Status</th><th>From</th><th>Stock</th><th>Rating</th><th>Actions</th></tr></thead>
          <tbody>
            {(data ?? []).map((p) => (
              <tr key={p.id}>
                <td><b>{p.name}</b><span className="sub">{p.notesLine} · {p.tags.join(', ')}</span></td>
                <td><Pill value={p.status} /></td>
                <td>{Number.isFinite(p.fromPrice) ? formatPence(p.fromPrice) : '—'}</td>
                <td><Pill value={p.totalStock === 0 ? 'cancelled' : p.totalStock <= 10 ? 'processing' : 'paid'} label={String(p.totalStock)} /></td>
                <td>{p.rating ? `★ ${p.rating.toFixed(1)} (${p.reviewCount})` : '—'}</td>
                <td>
                  <div className="table-actions">
                    <a className="btn btn--outline btn--xs" href={`/products/${p.slug}`} target="_blank" rel="noreferrer">View</a>
                    {writable && <button type="button" className="btn btn--outline btn--xs" onClick={() => edit(p.id)}>Edit</button>}
                    {writable && <button type="button" className="btn btn--outline btn--xs btn--danger" onClick={() => remove(p.id, p.name)}>Remove</button>}
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <Modal open={!!editor} title={editor?.id ? 'Edit product' : 'New product'} onClose={() => setEditor(null)}>
        {editor && <Editor key={editor.id ?? 'new'} id={editor.id} initial={editor.input} onDone={() => { setEditor(null); reload(); }} />}
      </Modal>
    </>
  );
}
