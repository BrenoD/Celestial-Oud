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
    const body = document.body;
    // iOS Safari ignores overflow:hidden on <html> for touch scrolling, so the
    // page is pinned in place (position: fixed at the current offset) instead.
    const scrollY = window.scrollY;
    const saved = { overflow: root.style.overflow, position: body.style.position, top: body.style.top, left: body.style.left, right: body.style.right, width: body.style.width };
    root.style.overflow = 'hidden';
    Object.assign(body.style, { position: 'fixed', top: `-${scrollY}px`, left: '0', right: '0', width: '100%' });
    root.classList.add('has-overlay');

    // On touch screens never focus a text field automatically: that pops the
    // keyboard up before the visitor asked for it. Focus the dialog instead.
    const el = panel.current;
    const touch = window.matchMedia('(pointer: coarse)').matches;
    let first = el?.querySelector<HTMLElement>('[data-autofocus]') ?? el?.querySelector<HTMLElement>(FOCUSABLE) ?? null;
    if (touch && first && /^(INPUT|TEXTAREA|SELECT)$/.test(first.tagName)) first = el;
    if (first === el && el) el.tabIndex = -1;
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
      root.style.overflow = saved.overflow;
      Object.assign(body.style, { position: saved.position, top: saved.top, left: saved.left, right: saved.right, width: saved.width });
      // 'instant': the page may use smooth scrolling, which would animate the jump back.
      window.scrollTo({ top: scrollY, behavior: 'instant' as ScrollBehavior });
      root.classList.remove('has-overlay');
      if (trigger && document.contains(trigger)) trigger.focus({ preventScroll: true });
    };
  }, [open, panel, closeAbove]);
}
