import { Suspense } from 'react';
import { Outlet, useLocation } from 'react-router-dom';
import { AnnouncementBar } from './AnnouncementBar';
import { Header } from './Header';
import { Footer } from './Footer';
import { CookieBanner } from './CookieBanner';
import { AddedToast } from './AddedToast';
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
      <CookieBanner />
    </div>
  );
}

/** Distraction-free checkout shell. */
export function CheckoutLayout() {
  const { pathname } = useLocation();
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
