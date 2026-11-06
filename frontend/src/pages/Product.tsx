import { useEffect, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { useProduct, useRelated, useReviews } from '@/api/hooks';
import { useBasketStore } from '@/store/basket';
import { useWishlistStore } from '@/store/wishlist';
import { Accordion } from '@/components/ui/Accordion';
import { ProductCard } from '@/components/ui/ProductCard';
import { Qty } from '@/components/ui/Qty';
import { Stars } from '@/components/ui/Stars';
import { Seo } from '@/components/ui/Seo';
import { GiftIcon, HeartIcon, ReturnIcon, TruckIcon } from '@/components/ui/Icons';
import { formatPence, formatPenceShort } from '@/lib/money';
import { config } from '@/config';
import { ProductImage } from '@/components/ui/ProductImage';
import NotFound from './NotFound';
import './Product.css';

export default function Product() {
  const { slug = '' } = useParams();
  const { data: p, isPending, isError, error } = useProduct(slug);
  const related = useRelated(slug);
  const reviews = useReviews(slug);
  const add = useBasketStore((s) => s.add);
  const wish = useWishlistStore();

  const [sizeIdx, setSizeIdx] = useState(0);
  const [qty, setQty] = useState(1);
  const [photo, setPhoto] = useState(0);
  const [added, setAdded] = useState(false);

  // Default to the middle size (50ml) when there are three, like the mockup.
  useEffect(() => {
    if (p) {
      setSizeIdx(p.sizes.length >= 3 ? 1 : 0);
      setQty(1);
      setPhoto(0);
      setAdded(false);
    }
  }, [p]);

  if (isError && (error as { status?: number }).status === 404) return <NotFound />;
  if (isPending || isError) {
    return (
      <main className="container page" aria-busy="true">
        <div className="pdp">
          <div className="skeleton" style={{ flex: '1 1 520px', height: 560 }} />
          <div style={{ flex: '1 1 420px', display: 'flex', flexDirection: 'column', gap: 16 }}>
            <div className="skeleton" style={{ height: 48, width: '60%' }} />
            <div className="skeleton" style={{ height: 20, width: '80%' }} />
            <div className="skeleton" style={{ height: 52 }} />
          </div>
        </div>
      </main>
    );
  }

  const size = p.sizes[sizeIdx] ?? p.sizes[0];
  const inWishlist = wish.slugs.includes(p.slug);
  const sample = p.samplePrice !== undefined && !p.tags.includes('set');

  const onAdd = () => {
    add(p, size, qty);
    setAdded(true);
  };

  const bars: Array<[string, string, number]> = [
    ['Intensity', p.profile.intensity >= 70 ? 'Strong' : p.profile.intensity >= 40 ? 'Medium' : 'Soft', p.profile.intensity],
    ['Longevity', p.profile.longevity >= 75 ? '8+ hours' : p.profile.longevity >= 50 ? '5–8 hours' : '3–5 hours', p.profile.longevity],
    ['Sweetness', p.profile.sweetness >= 70 ? 'High' : p.profile.sweetness >= 40 ? 'Medium' : 'Low', p.profile.sweetness],
    ['Freshness', p.profile.freshness >= 70 ? 'High' : p.profile.freshness >= 40 ? 'Medium' : 'Low', p.profile.freshness],
  ];

  return (
    <main className="container page">
      <Seo title={p.name} description={p.tagline} />
      <nav aria-label="Breadcrumb" className="breadcrumb">
        <Link to="/">Home</Link> › <Link to={`/collections/${p.collection.slug}`}>{p.collection.name}</Link> › <span>{p.name}</span>
      </nav>

      <section className="pdp">
        <div className="pdp__gallery">
          <div className="pdp__main">
            <ProductImage src={p.images[photo] ?? p.images[0]} alt={`${p.name} bottle`} width={492} height={560} loading="eager" fetchPriority="high" />
          </div>
          {p.images.length > 1 && (
            <div className="pdp__thumbs">
              {p.images.map((img, i) => (
                <button key={img + i} type="button" aria-label={`Photo ${i + 1}`} aria-pressed={photo === i} onClick={() => setPhoto(i)}>
                  <ProductImage src={img} width={120} height={96} className="pimg--mini" />
                </button>
              ))}
            </div>
          )}
        </div>

        <div className="pdp__buy">
          <span className="eyebrow">Eau de Parfum · {p.collection.name} collection</span>
          <h1 className="pdp__title">{p.name}</h1>
          <p className="pdp__tagline">{p.tagline}</p>
          <a href="#reviews" className="pdp__rating">
            <Stars rating={p.rating} showCount={false} size={14} /> {p.rating.toFixed(1)} · {p.reviewCount} reviews
          </a>
          <div className="pdp__price">
            {formatPence(size.price)}
            {size.compareAtPrice && <s className="pdp__was">{formatPence(size.compareAtPrice)}</s>}
          </div>

          <div className="pdp__sizes">
            <span style={{ fontSize: 14, fontWeight: 600 }}>Size: {size.label}</span>
            <div className="pdp__size-row" role="group" aria-label="Size">
              {p.sizes.map((s, i) => (
                <button key={s.sku} type="button" className="option" aria-pressed={i === sizeIdx} onClick={() => setSizeIdx(i)} disabled={!s.inStock}>
                  {s.label}
                  {!s.inStock && <span className="visually-hidden"> (out of stock)</span>}
                </button>
              ))}
            </div>
            {!size.inStock && <span className="status">This size is currently out of stock.</span>}
          </div>

          <div className="pdp__actions">
            <Qty value={qty} onChange={setQty} />
            <button type="button" className="btn btn--primary btn--lg pdp__add" onClick={onAdd} disabled={!size.inStock}>
              Add to basket
            </button>
            <button type="button" className="pdp__wish" aria-pressed={inWishlist} aria-label={inWishlist ? 'Remove from wishlist' : 'Add to wishlist'} onClick={() => wish.toggle(p.slug)}>
              <HeartIcon filled={inWishlist} />
            </button>
          </div>
          {added && (
            <span role="status" className="status">
              Added to your basket. <Link to="/basket">View basket</Link>
            </span>
          )}

          {sample && (
            <button
              type="button"
              className="pdp__sample"
              onClick={() => add(p, { sku: `${p.slug}-2ml`, label: '2ml sample', ml: 2, price: p.samplePrice!, inStock: true })}
            >
              <span>
                <b>Not sure? Try a 2ml sample first</b>
                <span className="muted">{formatPenceShort(p.samplePrice!)} — credited back on your full bottle</span>
              </span>
              <span className="pdp__sample-cta">Add sample</span>
            </button>
          )}

          <ul className="pdp__trust">
            <li>
              <TruckIcon size={20} />
              <span>
                <b>In stock.</b> Order by {config.dispatchCutoff} for dispatch today.
              </span>
            </li>
            <li>
              <GiftIcon size={20} />
              <span>Free gift wrap and message at checkout.</span>
            </li>
            <li>
              <ReturnIcon />
              <span>Returns on unopened items within {config.returnsDays} days.</span>
            </li>
          </ul>

          <Accordion
            defaultOpen="description"
            items={[
              { id: 'description', title: 'Description', body: <p>{p.description}</p> },
              { id: 'wear', title: 'How to wear it', body: <p>{p.howToWear}</p> },
              { id: 'ingredients', title: 'Ingredients & allergens', body: <p>{p.ingredients}</p> },
              {
                id: 'delivery',
                title: 'Delivery & returns',
                body: (
                  <p>
                    Free UK delivery over {formatPenceShort(config.freeDeliveryThreshold)}; next-day available. Unopened items can be returned within{' '}
                    {config.returnsDays} days. <Link to="/help#returns">Full policy</Link>.
                  </p>
                ),
              },
            ]}
          />
        </div>
      </section>

      {/* Scent profile */}
      <section className="scent panel">
        <div className="scent__col">
          <h2>Scent notes</h2>
          {(['top', 'heart', 'base'] as const).map((k) => (
            <div key={k} className="scent__tier">
              <span className="eyebrow">{k}</span>
              <span className="scent__notes">{p.pyramid[k].join(' · ')}</span>
            </div>
          ))}
        </div>
        <div className="scent__col">
          <h2>Profile</h2>
          {bars.map(([label, word, pct]) => (
            <div key={label} className="bar">
              <div className="bar__head">
                <span>{label}</span>
                <span style={{ color: 'var(--bronze-dark)' }}>{word}</span>
              </div>
              <div className="bar__track" role="meter" aria-valuenow={pct} aria-valuemin={0} aria-valuemax={100} aria-label={label}>
                <div className="bar__fill" style={{ width: `${pct}%` }} />
              </div>
            </div>
          ))}
          <span className="muted" style={{ fontSize: 14 }}>
            Best for: {p.bestFor}
          </span>
        </div>
      </section>

      {/* Reviews */}
      <section id="reviews" className="reviews">
        <div className="reviews__summary">
          <h2>Reviews</h2>
          <span className="reviews__score">{p.rating.toFixed(1)}</span>
          <Stars rating={p.rating} showCount={false} size={20} />
          <span className="muted" style={{ fontSize: 14 }}>
            Based on {p.reviewCount} verified reviews
          </span>
          <Link to="/account" className="btn btn--outline" style={{ alignSelf: 'flex-start' }}>
            Write a review
          </Link>
        </div>
        <div className="reviews__list">
          {reviews.data?.length ? (
            reviews.data.map((r) => (
              <article key={r.id} className="review card">
                <div className="review__head">
                  <Stars rating={r.rating} showCount={false} />
                  <b>{r.title}</b>
                  {r.verified && <span className="pill pill--accent">Verified buyer</span>}
                </div>
                <p>{r.body}</p>
                <span className="muted" style={{ fontSize: 13 }}>
                  {r.author} · {new Date(r.createdAt).toLocaleDateString('en-GB', { month: 'short', year: 'numeric' })}
                  {r.longevity && <> · Longevity: {r.longevity}</>}
                </span>
              </article>
            ))
          ) : (
            <div className="review card muted" style={{ fontSize: 14 }}>
              No reviews yet. Be the first to review {p.name}.
            </div>
          )}
        </div>
      </section>

      {/* Related */}
      {related.data && related.data.length > 0 && (
        <section style={{ marginTop: 56 }}>
          <h2 style={{ fontSize: 32, marginBottom: 20 }}>You may also like</h2>
          <div className="product-grid product-grid--dense">
            {related.data.map((r) => (
              <ProductCard key={r.id} product={r} compact />
            ))}
          </div>
        </section>
      )}
    </main>
  );
}
