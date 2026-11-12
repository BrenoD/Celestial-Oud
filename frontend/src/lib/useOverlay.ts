import { useEffect, useRef, type RefObject } from 'react';

const FOCUSABLE = 'a[href],button:not([disabled]),input:not([disabled]),select:not([disabled]),textarea:not([disabled]),[tabindex]:not([tabindex="-1"])';

/**
 * Behaviour shared by every panel that opens over the page (menu, filters,
 * chat): Esc closes, focus moves into the panel and is trapped there, the page
 * behind stops scrolling, focus returns to the trigger on close, and the panel
 * closes itself if the viewport grows past `closeAbove` (e.g. the desktop nav
 * takes over).
 */
export function useOverlay(open: boolean, close: () => void, panel: RefObject<HTMLElement | null>, closeAbove?: number) {
  const closeRef = useRef(close);
  closeRef.current = close;

  useEffect(() => {
    if (!open) return;
    const trigger = document.activeElement as HTMLElement | null;
    const root = document.documentElement;
    const prevOverflow = root.style.overflow;
    root.style.overflow = 'hidden';
    root.classList.add('has-overlay');

    const el = panel.current;
    const first = el?.querySelector<HTMLElement>('[data-autofocus]') ?? el?.querySelector<HTMLElement>(FOCUSABLE);
    first?.focus({ preventScroll: true });

    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.preventDefault();
        closeRef.current();
        return;
      }
      if (e.key !== 'Tab' || !el) return;
      const items = Array.from(el.querySelectorAll<HTMLElement>(FOCUSABLE)).filter((n) => n.offsetParent !== null);
      if (!items.length) return;
      const [a, z] = [items[0], items[items.length - 1]];
      if (e.shiftKey && document.activeElement === a) {
        e.preventDefault();
        z.focus();
      } else if (!e.shiftKey && document.activeElement === z) {
        e.preventDefault();
        a.focus();
      }
    };
    document.addEventListener('keydown', onKey);

    const mq = closeAbove ? window.matchMedia(`(min-width: ${closeAbove + 1}px)`) : null;
    const onWide = (e: MediaQueryListEvent) => e.matches && closeRef.current();
    mq?.addEventListener('change', onWide);

    return () => {
      document.removeEventListener('keydown', onKey);
      mq?.removeEventListener('change', onWide);
      root.style.overflow = prevOverflow;
      root.classList.remove('has-overlay');
      if (trigger && document.contains(trigger)) trigger.focus({ preventScroll: true });
    };
  }, [open, panel, closeAbove]);
}
