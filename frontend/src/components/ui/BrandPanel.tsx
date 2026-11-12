import { assetUrl } from '@/config';
import './BrandPanel.css';

/**
 * Neutral stand-in for lifestyle photography (hero, collection tiles, promo
 * banners, about page). Ivory panel with the brand emblem — replaced by real
 * imagery once the client supplies it.
 */
export function BrandPanel({ label, size = 'md', className }: { label?: string; size?: 'sm' | 'md' | 'lg'; className?: string }) {
  return (
    <div className={`bpanel bpanel--${size}${className ? ` ${className}` : ''}`} role="img" aria-label={label ?? 'Celestial Oud'}>
      <img src={assetUrl('/images/logo-240.webp')} alt="" width={120} height={130} loading="lazy" />
      {label && <span className="bpanel__label">{label}</span>}
    </div>
  );
}
