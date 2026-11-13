import { Suspense, useState } from 'react';
import { Navigate, Outlet, useLocation } from 'react-router-dom';
import { useBasketStore } from '@/store/basket';
import { CookieBanner } from './CookieBanner';
import { AnnouncementBar } from './AnnouncementBar';
import { Header } from './Header';
import { Footer } from './Footer';
import { AddedToast } from './AddedToast';
import { ChatLauncher } from '@/components/chat/ChatLauncher';
import { Logo } from './Logo';
import { LockIcon } from '@/components/ui/Icons';
import { ScrollToTop } from './ScrollToTop';

function PageFallback() {
  return (
    <main className="container page" aria-busy="true">
      <div className="skeleton" style={{ height: 44, width: 320, marginBottom: 24 }} />
      <div className="skeleton" style={{ height: 420 }} />
    </main>
  );
}

/** Storefront shell: announcement bar → header → nav → page → footer. */
export function StoreLayout() {
  return (
    <div className="shell">
      <ScrollToTop />
      <a href="#main" className="skip-link">
        Skip to content
      </a>
      <AnnouncementBar />
      <Header />
      <div id="main">
        <Suspense fallback={<PageFallback />}>
          <Outlet />
        </Suspense>
      </div>
      <Footer />
      <AddedToast />
      <ChatLauncher />
    </div>
  );
}

/** Distraction-free checkout shell. */
export function CheckoutLayout() {
  const { pathname } = useLocation();
  // Decided once, before anything paints: arriving at checkout with an empty
  // basket never shows the checkout shell (no flash, no layout shift). Later
  // changes (the basket clears after a successful order) are left to the page.
  const [arrivedEmpty] = useState(() => useBasketStore.getState().lines.length === 0);
  if (arrivedEmpty && pathname.replace(/\/$/, '').endsWith('/checkout')) return <Navigate to="/basket" replace />;
  return (
    <div className="shell">
      <ScrollToTop />
      <header className="chk-hdr">
        <div className="container container--narrow chk-hdr__inner">
          <Logo compact />
          <span className="chk-hdr__secure">
            <LockIcon size={18} /> {pathname.includes('confirmed') ? 'Order complete' : 'Secure checkout'}
          </span>
        </div>
      </header>
      <div id="main">
        <Suspense fallback={<PageFallback />}>
          <Outlet />
        </Suspense>
      </div>
    </div>
  );
}

/**
 * Root of every route. The cookie banner lives here, above the store,
 * checkout and admin shells, so it stays mounted (no repaint or layout shift)
 * when navigation moves between them.
 */
export function RootLayout() {
  return (
    <>
      <Outlet />
      <CookieBanner />
    </>
  );
}
