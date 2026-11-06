import { Link } from 'react-router-dom';
import { assetUrl } from '@/config';

export function Logo({ compact = false }: { compact?: boolean }) {
  return (
    <Link to="/" className="logo" aria-label="Celestial Oud — home">
      <img src={assetUrl('/images/logo_dark.png')} alt="" width={compact ? 40 : 48} height={compact ? 44 : 52} />
      <span className="logo__text">
        <span className="logo__wordmark">CELESTIAL OUD</span>
        {!compact && <span className="logo__sub">Fragrances</span>}
      </span>
    </Link>
  );
}
