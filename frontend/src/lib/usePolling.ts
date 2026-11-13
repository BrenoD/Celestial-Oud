import { useEffect, useRef } from 'react';

/**
 * Calls `fn` every `ms` while `enabled` and the tab is visible. Hidden tabs
 * stop polling (no wasted requests); coming back runs `fn` at once.
 */
export function usePolling(fn: () => void, ms: number, enabled = true) {
  const ref = useRef(fn);
  ref.current = fn;
  useEffect(() => {
    if (!enabled) return;
    let timer: number | undefined;
    const tick = () => {
      if (document.visibilityState === 'visible') ref.current();
    };
    const start = () => {
      window.clearInterval(timer);
      timer = window.setInterval(tick, ms);
    };
    const onVisible = () => {
      if (document.visibilityState === 'visible') {
        ref.current();
        start();
      } else window.clearInterval(timer);
    };
    start();
    document.addEventListener('visibilitychange', onVisible);
    return () => {
      window.clearInterval(timer);
      document.removeEventListener('visibilitychange', onVisible);
    };
  }, [ms, enabled]);
}
