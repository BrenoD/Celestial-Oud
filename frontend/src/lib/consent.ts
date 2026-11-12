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

export const COOKIE_SETTINGS_EVENT = 'co:cookie-settings';

/** Reopens the cookie banner so the visitor can change their choice. */
export function openCookieSettings() {
  window.dispatchEvent(new Event(COOKIE_SETTINGS_EVENT));
}
