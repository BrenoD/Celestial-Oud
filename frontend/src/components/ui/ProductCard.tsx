import { Link } from 'react-router-dom';
import type { ProductSummary } from '@/types';
import { formatPenceShort } from '@/lib/money';
import { Stars } from './Stars';
import { ProductImage } from './ProductImage';
import './ProductCard.css';

interface Props {
  product: ProductSummary;
  /** "rating" shows the white "★ 4.8 rating" badge used in Top rated. */
  badgeStyle?: 'tag' | 'rating';
  compact?: boolean;
}

export function ProductCard({ product: p, badgeStyle = 'tag', compact = false }: Props) {
  const href = `/products/${p.slug}`;
  return (
    <article className={`pcard card${compact ? ' pcard--compact' : ''}`}>
      <Link to={href} className="pcard__media" aria-label={p.name}>
        <ProductImage src={p.images[0]} width={400} height={276} />
        {badgeStyle === 'rating' ? (
          <span className="badge badge--light pcard__badge">
            <span style={{ color: 'var(--star)' }}>★</span> {p.rating.toFixed(1)} rating
          </span>
        ) : (
          p.badge && <span className="badge pcard__badge">{p.badge}</span>
        )}
      </Link>
      <div className="pcard__body">
        <Link to={href} className="pcard__name">
          {p.name}
        </Link>
        <span className="pcard__notes">{p.notesLine}</span>
        <Stars rating={p.rating} count={p.reviewCount} />
        <span className="pcard__price">From {formatPenceShort(p.fromPrice)}</span>
        {!compact && (
          <Link to={href} className="btn btn--primary pcard__cta">
            Choose size
          </Link>
        )}
      </div>
    </article>
  );
}

export function ProductCardSkeleton() {
  return (
    <div className="pcard card" aria-hidden="true">
      <div className="skeleton" style={{ height: 260, borderRadius: 0 }} />
      <div className="pcard__body">
        <div className="skeleton" style={{ height: 22, width: '70%' }} />
        <div className="skeleton" style={{ height: 14, width: '50%' }} />
        <div className="skeleton" style={{ height: 14, width: '40%' }} />
        <div className="skeleton" style={{ height: 46, marginTop: 'auto' }} />
      </div>
    </div>
  );
}
