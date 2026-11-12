import { useState, type FormEvent } from 'react';
import { Link } from 'react-router-dom';
import { useNewsletter, useProducts } from '@/api/hooks';
import { ProductCard, ProductCardSkeleton } from '@/components/ui/ProductCard';
import { ClockIcon, GiftIcon, LockIcon, TruckIcon } from '@/components/ui/Icons';
import { Seo } from '@/components/ui/Seo';
import { BrandPanel } from '@/components/ui/BrandPanel';
import './Home.css';

const COLLECTION_TILES = [
  { slug: 'oud', name: 'Oud' },
  { slug: 'amber', name: 'Amber & Resins' },
  { slug: 'musk', name: 'Musk & Florals' },
  { slug: 'gifts', name: 'Gift Sets' },
];

function Grid({ query, badgeStyle }: { query: Parameters<typeof useProducts>[0]; badgeStyle?: 'tag' | 'rating' }) {
  const { data, isPending, isError } = useProducts(query);
  if (isError) return <p className="status status--error">We could not load products right now. Please try again shortly.</p>;
  return (
    <div className="product-grid">
      {isPending
        ? Array.from({ length: 4 }, (_, i) => <ProductCardSkeleton key={i} />)
        : data.items.map((p) => <ProductCard key={p.id} product={p} badgeStyle={badgeStyle} />)}
    </div>
  );
}

function Newsletter() {
  const [email, setEmail] = useState('');
  const m = useNewsletter();
  const submit = (e: FormEvent) => {
    e.preventDefault();
    if (email) m.mutate(email);
  };
  return (
    <form className="home-nl" onSubmit={submit}>
      <label htmlFor="nl" className="home-nl__title">
        Get 10% off your first order
      </label>
      {m.isSuccess ? (
        <p className="status" role="status">
          Thanks — check your inbox for your code.
        </p>
      ) : (
        <div className="home-nl__row">
          <input id="nl" type="email" required className="input" placeholder="Your email address" value={email} onChange={(e) => setEmail(e.target.value)} />
          <button type="submit" className="btn btn--primary" disabled={m.isPending}>
            {m.isPending ? 'Signing up…' : 'Sign up'}
          </button>
        </div>
      )}
      {m.isError && <p className="field__error">Something went wrong. Please try again.</p>}
    </form>
  );
}

export default function Home() {
  return (
    <>
      <Seo title="Rich oud perfumes, made to last all day" />

      {/* Hero */}
      <section className="container hero">
        <div className="hero__copy">
          <span className="eyebrow">Eau de Parfum · Long-lasting</span>
          <h1 className="hero__title">
            Rich oud perfumes.
            <br />
            Made to last all day.
          </h1>
          <p className="hero__lead">Luxury oud, amber and musk fragrances — without the designer price tag.</p>
          <div className="hero__ctas">
            <Link to="/collections/bestsellers" className="btn btn--primary btn--lg">
              Shop bestsellers
            </Link>
            <Link to="/products/discovery-set" className="btn btn--secondary btn--lg">
              Try 5 samples
            </Link>
          </div>
        </div>
        <div className="hero__media">
          <BrandPanel size="lg" label="Imagery coming soon" />
        </div>
      </section>

      {/* Trust strip */}
      <section aria-label="Why shop with us" className="trust">
        <div className="container trust__grid">
          <div>
            <TruckIcon />
            <span>
              <b>Fast UK delivery</b> — next day available
            </span>
          </div>
          <div>
            <ClockIcon />
            <span>
              <b>Long-lasting</b> — high concentration
            </span>
          </div>
          <div>
            <GiftIcon />
            <span>
              <b>Free gift wrap</b> on request
            </span>
          </div>
          <div>
            <LockIcon />
            <span>
              <b>Secure checkout</b> — Apple Pay, PayPal
            </span>
          </div>
        </div>
      </section>

      <main className="container">
        {/* Collections: compact navigation row */}
        <section className="home-section" style={{ paddingTop: 40 }}>
          <div className="section-head section-head--compact">
            <h2>Shop by collection</h2>
            <Link to="/collections/all" className="text-link">
              View all
            </Link>
          </div>
          <div className="tiles">
            {COLLECTION_TILES.map((c) => (
              <Link key={c.slug} to={`/collections/${c.slug}`} className="tile card">
                <span className="tile__media">
                  <BrandPanel size="sm" />
                </span>
                <span className="tile__name">{c.name}</span>
                <span className="tile__cta" aria-hidden="true">
                  →
                </span>
              </Link>
            ))}
          </div>
        </section>

        {/* Bestsellers */}
        <section className="home-section">
          <div className="section-head">
            <h2>Bestsellers</h2>
            <Link to="/collections/bestsellers" className="text-link">
              View all bestsellers
            </Link>
          </div>
          <Grid query={{ collection: 'bestsellers', pageSize: 4 }} />
        </section>

        {/* Top rated: the main showcase */}
        <section className="home-section">
          <div className="section-head">
            <div>
              <h2>Top rated</h2>
              <p className="section-head__sub">Our highest-rated fragrances, chosen by customers.</p>
            </div>
            <Link to="/collections/top-rated" className="text-link">
              See all top rated
            </Link>
          </div>
          <Grid query={{ collection: 'top-rated', sort: 'top-rated', pageSize: 8 }} badgeStyle="rating" />
        </section>

        {/* Promo banners */}
        <section className="home-section promos">
          <Link to="/products/discovery-set" className="promo panel">
            <div className="promo__copy">
              <span className="eyebrow">Not sure yet?</span>
              <span className="promo__title">
                Discovery Set
                <br />5 × 2ml samples
              </span>
              <span className="muted">Get the cost back as credit on your first full bottle.</span>
              <span className="btn btn--secondary promo__btn">Shop the set</span>
            </div>
          </Link>
          <Link to="/collections/gifts" className="promo panel">
            <div className="promo__copy">
              <span className="eyebrow">Christmas gifting</span>
              <span className="promo__title">
                Gift sets,
                <br />
                ready to give
              </span>
              <span className="muted">Boxed, wrapped and dispatched today.</span>
              <span className="btn btn--primary promo__btn">Shop gifts</span>
            </div>
          </Link>
        </section>

        {/* Quiz strip */}
        <section className="quiz panel" style={{ marginBottom: 56 }}>
          <div>
            <span className="quiz__title">Find your scent in 60 seconds</span>
            <span>Answer 5 quick questions and get your perfect match.</span>
          </div>
          <Link to="/quiz" className="btn btn--primary">
            Take the quiz
          </Link>
        </section>
      </main>

      {/* Story + newsletter (third plane) */}
      <section className="story">
        <div className="container story__grid">
          <div className="story__copy">
            <span className="story__title">Our story</span>
            <span className="muted">[One or two lines about Celestial Oud — who makes it and why.]</span>
            <Link to="/about" className="text-link">
              Read our story
            </Link>
          </div>
          <Newsletter />
        </div>
      </section>
    </>
  );
}
