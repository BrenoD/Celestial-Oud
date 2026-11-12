import { useEffect } from 'react';

function meta(name: string): HTMLMetaElement {
  let m = document.querySelector<HTMLMetaElement>(`meta[name="${name}"]`);
  if (!m) {
    m = document.createElement('meta');
    m.name = name;
    document.head.appendChild(m);
  }
  return m;
}

/** Minimal head management — enough for a SPA behind a CDN. */
export function Seo({ title, description, noindex }: { title: string; description?: string; noindex?: boolean }) {
  useEffect(() => {
    document.title = title ? `${title} — Celestial Oud` : 'Celestial Oud';
    if (description) meta('description').content = description;
    // Token landing pages (email links) must not be indexed.
    if (noindex) meta('robots').content = 'noindex';
    else document.querySelector('meta[name="robots"]')?.remove();
  }, [title, description, noindex]);
  return null;
}
