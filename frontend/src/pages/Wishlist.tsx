import { Link } from 'react-router-dom';
import { useQueries } from '@tanstack/react-query';
import { api } from '@/api';
import { keys } from '@/api/hooks';
import { useWishlistStore } from '@/store/wishlist';
import { ProductCard, ProductCardSkeleton } from '@/components/ui/ProductCard';
import { Seo } from '@/components/ui/Seo';

export default function Wishlist() {
  const { slugs, toggle } = useWishlistStore();
  const results = useQueries({
    queries: slugs.map((slug) => ({ queryKey: keys.product(slug), queryFn: () => api.getProduct(slug), staleTime: 300_000 })),
  });

  return (
    <main className="container page">
      <Seo title="Wishlist" />
      <nav aria-label="Breadcrumb" className="breadcrumb">
        <Link to="/">Home</Link> › <span>Wishlist</span>
      </nav>
      <h1 className="page-title">Your wishlist</h1>
      <p className="page-intro">Saved fragrances stay here on this device. Sign in to keep them across devices.</p>

      {slugs.length === 0 ? (
        <div className="card" style={{ padding: 40, textAlign: 'center', marginTop: 24, display: 'flex', flexDirection: 'column', gap: 14, alignItems: 'center' }}>
          <span style={{ fontFamily: 'var(--font-display)', fontWeight: 600, fontSize: 26 }}>Nothing saved yet</span>
          <Link to="/collections/all" className="btn btn--primary">
            Browse fragrances
          </Link>
        </div>
      ) : (
        <div className="product-grid product-grid--dense" style={{ marginTop: 24 }}>
          {results.map((r, i) =>
            r.data ? (
              <div key={slugs[i]} style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
                <ProductCard product={{ ...r.data, fromPrice: Math.min(...r.data.sizes.map((s) => s.price)) }} compact />
                <button type="button" className="btn btn--ghost" onClick={() => toggle(slugs[i])}>
                  Remove
                </button>
              </div>
            ) : (
              <ProductCardSkeleton key={slugs[i]} />
            ),
          )}
        </div>
      )}
    </main>
  );
}
