import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { getConsent, setConsent, type Consent } from '@/lib/consent';
import './CookieBanner.css';

/** UK GDPR / PECR: equal-weight accept and reject, no pre-ticked boxes. */
export function CookieBanner() {
  const [open, setOpen] = useState(false);
  useEffect(() => setOpen(getConsent() === null), []);
  if (!open) return null;

  const choose = (c: Consent) => {
    setConsent(c);
    setOpen(false);
  };

  return (
    <div className="cookie" role="dialog" aria-live="polite" aria-label="Cookie preferences">
      <p>
        We use essential cookies to run the shop and, with your permission, analytics cookies to improve it. Read our{' '}
        <Link to="/legal/cookies">cookie policy</Link>.
      </p>
      <div className="cookie__actions">
        <button type="button" className="btn btn--outline btn--sm" onClick={() => choose('essential')}>
          Reject non-essential
        </button>
        <button type="button" className="btn btn--primary btn--sm" onClick={() => choose('all')}>
          Accept all
        </button>
      </div>
    </div>
  );
}
