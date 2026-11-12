import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { COOKIE_SETTINGS_EVENT, getConsent, setConsent, type Consent } from '@/lib/consent';
import './CookieBanner.css';

/**
 * UK GDPR / PECR: accept and reject have identical weight (same style, same
 * size), nothing is pre-ticked, and the choice can be changed at any time from
 * "Cookie settings" in the footer (openCookieSettings).
 */
export function CookieBanner() {
  // Decided on the first render (no flash, no late layout shift).
  const [open, setOpen] = useState(() => getConsent() === null);
  useEffect(() => {
    const reopen = () => setOpen(true);
    window.addEventListener(COOKIE_SETTINGS_EVENT, reopen);
    return () => window.removeEventListener(COOKIE_SETTINGS_EVENT, reopen);
  }, []);
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
        <button type="button" className="btn btn--primary btn--sm cookie__btn" onClick={() => choose('essential')}>
          Reject non-essential
        </button>
        <button type="button" className="btn btn--primary btn--sm cookie__btn" onClick={() => choose('all')}>
          Accept all
        </button>
      </div>
    </div>
  );
}
