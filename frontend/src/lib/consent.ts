const KEY = 'co.consent';
export type Consent = 'all' | 'essential';

export function getConsent(): Consent | null {
  try {
    const v = localStorage.getItem(KEY);
    return v === 'all' || v === 'essential' ? v : null;
  } catch {
    return null;
  }
}

export function setConsent(c: Consent) {
  try {
    localStorage.setItem(KEY, c);
  } catch {
    /* storage unavailable (private mode) — banner will show again */
  }
  window.dispatchEvent(new CustomEvent('co:consent', { detail: c }));
}
