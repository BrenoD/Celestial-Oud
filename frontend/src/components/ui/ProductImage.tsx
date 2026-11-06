import { assetUrl } from '@/config';
import './ProductImage.css';

interface Props {
  src?: string;
  alt?: string;
  width?: number;
  height?: number;
  loading?: 'lazy' | 'eager';
  fetchPriority?: 'high' | 'low' | 'auto';
  className?: string;
}

/**
 * Product photo with a neutral fallback. Real product photography does not
 * exist yet, so every product renders the placeholder until the API returns
 * image URLs; the layout is identical either way.
 */
export function ProductImage({ src, alt = '', width, height, loading = 'lazy', fetchPriority, className }: Props) {
  if (src) {
    return <img className={className} src={assetUrl(src)} alt={alt} width={width} height={height} loading={loading} fetchPriority={fetchPriority} />;
  }
  return (
    <span className={`pimg${className ? ` ${className}` : ''}`} role="img" aria-label={alt || 'Product photo coming soon'}>
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.2" aria-hidden="true">
        <path d="M9 3h6v3H9zM8 6h8l1 3v10a2 2 0 0 1-2 2H9a2 2 0 0 1-2-2V9z" />
        <path d="M10 13h4" />
      </svg>
      <span className="pimg__text">Photo coming soon</span>
    </span>
  );
}
