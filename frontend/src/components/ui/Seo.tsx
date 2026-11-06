import { useEffect } from 'react';

/** Minimal head management — enough for a SPA behind a CDN. */
export function Seo({ title, description }: { title: string; description?: string }) {
  useEffect(() => {
    document.title = title ? `${title} — Celestial Oud` : 'Celestial Oud';
    if (description) {
      let m = document.querySelector<HTMLMetaElement>('meta[name="description"]');
      if (!m) {
        m = document.createElement('meta');
        m.name = 'description';
        document.head.appendChild(m);
      }
      m.content = description;
    }
  }, [title, description]);
  return null;
}
