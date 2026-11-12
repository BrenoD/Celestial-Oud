import { useMemo, useRef, useState } from 'react';
import { Link, useParams, useSearchParams } from 'react-router-dom';
import { useProducts } from '@/api/hooks';
import { ProductCard, ProductCardSkeleton } from '@/components/ui/ProductCard';
import { useOverlay } from '@/lib/useOverlay';
import { Seo } from '@/components/ui/Seo';
import type { ProductQuery, ScentFamily, Wear } from '@/types';
import { collections } from '@/data/catalogue';
import './Collection.css';

const FAMILIES: Array<[ScentFamily, string]> = [
  ['woody-oud', 'Woody & Oud'],
  ['amber-oriental', 'Amber & Oriental'],
  ['floral', 'Floral'],
  ['fresh-musk', 'Fresh & Musk'],
  ['gourmand', 'Gourmand'],
  ['spicy', 'Spicy'],
];
const NOTES = ['oud', 'amber', 'rose', 'vanilla', 'saffron', 'musk'];
const SIZES = [2, 30, 50, 100];
const PRICES: Array<[string, string]> = [
  ['-3000', 'Under £30'],
  ['3000-5000', '£30 – £50'],
  ['5000-8000', '£50 – £80'],
  ['8000-', 'Over £80'],
];
const WEAR: Array<[Wear, string]> = [
  ['day', 'Day'],
  ['evening', 'Evening'],
  ['summer', 'Summer'],
  ['winter', 'Winter'],
];
const SORTS: Array<[NonNullable<ProductQuery['sort']>, string]> = [
  ['bestselling', 'Bestselling'],
  ['newest', 'Newest'],
  ['price-asc', 'Price: low to high'],
  ['price-desc', 'Price: high to low'],
  ['top-rated', 'Top rated'],
];
const PAGE_SIZE = 9;

/** Filters live in the URL so results are shareable and back/forward works. */
function useFilters() {
  const [sp, setSp] = useSearchParams();
  const get = (k: string) => sp.get(k)?.split(',').filter(Boolean) ?? [];
  const toggle = (k: string, v: string) => {
    const cur = get(k);
    const next = cur.includes(v) ? cur.filter((x) => x !== v) : [...cur, v];
    const n = new URLSearchParams(sp);
    if (next.length) n.set(k, next.join(','));
    else n.delete(k);
    n.delete('page');
    setSp(n, { replace: true });
  };
  const set = (k: string, v: string | null) => {
    const n = new URLSearchParams(sp);
    if (v) n.set(k, v);
    else n.delete(k);
    if (k !== 'page') n.delete('page');
    setSp(n, { replace: k !== 'page' });
  };
  const clear = () => {
    const n = new URLSearchParams();
    const q = sp.get('q');
    if (q) n.set('q', q);
    setSp(n, { replace: true });
  };
  return { sp, get, toggle, set, clear };
}

type Filters = ReturnType<typeof useFilters>;

function Fieldset({ f, legend, name, options }: { f: Filters; legend: string; name: string; options: Array<[string, string]> }) {
  return (
    <fieldset className="filters__group">
      <legend>{legend}</legend>
      {options.map(([v, label]) => (
        <label key={v} className="check">
          <input type="checkbox" checked={f.get(name).includes(v)} onChange={() => f.toggle(name, v)} /> {label}
        </label>
      ))}
    </fieldset>
  );
}

/** Module-level so checkboxes keep focus between renders (keyboard users). */
function FilterGroups({ f }: { f: Filters }) {
  return (
    <>
      <Fieldset f={f} legend="Scent family" name="family" options={FAMILIES} />
      <Fieldset f={f} legend="Key notes" name="notes" options={NOTES.map((n) => [n, n[0].toUpperCase() + n.slice(1)])} />
      <Fieldset f={f} legend="Size" name="size" options={SIZES.map((s) => [String(s), s === 2 ? '2ml sample' : `${s}ml`])} />
      <Fieldset f={f} legend="Price" name="price" options={PRICES} />
      <Fieldset f={f} legend="Wear" name="wear" options={WEAR} />
    </>
  );
}

/** Mobile filters: a bottom sheet over a dimmed page with a live result count. */
function FilterSheet({ f, total, active, onClose }: { f: Filters; total?: number; active: number; onClose: () => void }) {
  const panel = useRef<HTMLDivElement>(null);
  useOverlay(true, onClose, panel, 900);
  return (
    <>
      <button type="button" className="backdrop" aria-label="Close filters" tabIndex={-1} onClick={onClose} />
      <div ref={panel} id="filter-sheet" className="sheet sheet--bottom" role="dialog" aria-modal="true" aria-label="Filters">
        <div className="sheet__head">
          <p className="sheet__title">Filters{active ? ` (${active})` : ''}</p>
          <button type="button" className="sheet__close" aria-label="Close filters" onClick={onClose} data-autofocus>
            ✕
          </button>
        </div>
        <div className="sheet__body filters filters--sheet stagger">
          <FilterGroups f={f} />
        </div>
        <div className="sheet__foot">
          <button type="button" className="btn btn--outline" onClick={f.clear} disabled={!active}>
            Clear all
          </button>
          <button type="button" className="btn btn--primary" onClick={onClose}>
            {total === undefined ? 'Show results' : `Show ${total} ${total === 1 ? 'result' : 'results'}`}
          </button>
        </div>
      </div>
    </>
  );
}

export default function Collection({ search = false }: { search?: boolean }) {
  const { slug = 'all' } = useParams();
  const f = useFilters();
  const [filtersOpen, setFiltersOpen] = useState(false);

  const q = f.sp.get('q') ?? '';
  const page = Number(f.sp.get('page') ?? 1);
  const query = useMemo<ProductQuery>(
    () => ({
      collection: search ? 'all' : slug,
      q: search ? q : undefined,
      family: f.get('family') as ScentFamily[],
      notes: f.get('notes'),
      size: f.get('size').map(Number),
      price: f.get('price'),
      wear: f.get('wear') as Wear[],
      sort: (f.sp.get('sort') as ProductQuery['sort']) ?? 'bestselling',
      page: 1,
      pageSize: PAGE_SIZE * page,
    }),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [slug, search, f.sp.toString()],
  );

  const { data, isPending, isError, isFetching } = useProducts(query);
  const collection = collections.find((c) => c.slug === slug);
  const title = search ? (q ? `Results for “${q}”` : 'Search') : (collection?.name ?? 'Collection');
  const intro = search ? (data ? `${data.total} products` : '') : `${collection?.description ?? 'Long-lasting oud, amber and musk perfumes.'}${data ? ` ${data.total} products.` : ''}`;

  const active: Array<{ k: string; v: string; label: string }> = [
    ...f.get('family').map((v) => ({ k: 'family', v, label: FAMILIES.find((x) => x[0] === v)?.[1] ?? v })),
    ...f.get('notes').map((v) => ({ k: 'notes', v, label: v[0].toUpperCase() + v.slice(1) })),
    ...f.get('size').map((v) => ({ k: 'size', v, label: v === '2' ? '2ml sample' : `${v}ml` })),
    ...f.get('price').map((v) => ({ k: 'price', v, label: PRICES.find((x) => x[0] === v)?.[1] ?? v })),
    ...f.get('wear').map((v) => ({ k: 'wear', v, label: WEAR.find((x) => x[0] === v)?.[1] ?? v })),
  ];

  return (
    <main className="container page">
      <Seo title={title} description={intro} />
      <nav aria-label="Breadcrumb" className="breadcrumb">
        <Link to="/">Home</Link> › <span>{title}</span>
      </nav>
      <div className="plp-head">
        <div>
          <h1 className="page-title">{title}</h1>
          <p className="page-intro">{intro}</p>
        </div>
        <label className="plp-sort">
          Sort by
          <select className="select" value={f.sp.get('sort') ?? 'bestselling'} onChange={(e) => f.set('sort', e.target.value)}>
            {SORTS.map(([v, l]) => (
              <option key={v} value={v}>
                {l}
              </option>
            ))}
          </select>
        </label>
      </div>

      <div className="plp">
        <button type="button" className="btn btn--outline plp__filter-toggle" aria-expanded={filtersOpen} aria-controls="filter-sheet" onClick={() => setFiltersOpen(true)}>
          {`Filters${active.length ? ` (${active.length})` : ''}`}
        </button>
        {filtersOpen && <FilterSheet f={f} total={data?.total} active={active.length} onClose={() => setFiltersOpen(false)} />}
        <aside aria-label="Filters" className="filters">
          <FilterGroups f={f} />
        </aside>

        <div className="plp__results" aria-busy={isFetching}>
          {active.length > 0 && (
            <div className="plp__active">
              <span className="muted">Active filters:</span>
              {active.map((a) => (
                <button key={`${a.k}:${a.v}`} type="button" className="chip" onClick={() => f.toggle(a.k, a.v)} aria-label={`Remove filter ${a.label}`}>
                  {a.label} ✕
                </button>
              ))}
              <button type="button" className="btn btn--ghost" onClick={f.clear}>
                Clear all
              </button>
            </div>
          )}

          {isError ? (
            <p className="status status--error">We could not load products right now. Please try again shortly.</p>
          ) : (
            <div className="product-grid product-grid--dense">
              {isPending
                ? Array.from({ length: PAGE_SIZE }, (_, i) => <ProductCardSkeleton key={i} />)
                : data.items.map((p) => <ProductCard key={p.id} product={p} compact />)}
            </div>
          )}

          {data && data.items.length === 0 && (
            <div className="plp__empty card">
              <p>No products match these filters.</p>
              <button type="button" className="btn btn--outline" onClick={f.clear}>
                Clear filters
              </button>
            </div>
          )}

          {data && data.items.length > 0 && (
            <div className="plp__more">
              <span className="muted">
                Showing {data.items.length} of {data.total} products
              </span>
              {data.items.length < data.total && (
                <button type="button" className="btn btn--outline" onClick={() => f.set('page', String(page + 1))} disabled={isFetching}>
                  {isFetching ? 'Loading…' : 'Load more'}
                </button>
              )}
            </div>
          )}
        </div>
      </div>
    </main>
  );
}
