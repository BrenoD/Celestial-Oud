import { Suspense, useEffect, useState } from 'react';
import { Link, NavLink, Outlet, useNavigate, useParams } from 'react-router-dom';
import { useAuthStore } from '@/store/auth';
import { logout } from '@/api/client';
import { usePolling } from '@/lib/usePolling';
import { useMockApi } from '@/config';
import { Logo } from '@/components/layout/Logo';
import { Seo } from '@/components/ui/Seo';
import { NAV_GROUPS, SECTION_TITLES, canView, isStaff } from './rbac';
import { ToastProvider, useLoad } from './components/ui';
import { adminApi } from './api';
import { resetMockAdmin } from './api.mock';
import './admin.css';

function Nav({ section, role }: { section: string; role?: string }) {
  const activeIdx = NAV_GROUPS.findIndex((g) => g.items.some((i) => i.key === section));
  const [open, setOpen] = useState<number>(Math.max(0, activeIdx));
  useEffect(() => setOpen(Math.max(0, activeIdx)), [activeIdx]);
  const { data, reload } = useLoad(() => adminApi.dashboard().catch(() => null), []);
  // Keep the badges (new chats above all) fresh while the admin is open.
  usePolling(reload, 30_000);
  const badges: Record<string, number> = {
    orders: data?.pendingOrders ?? 0,
    reviews: data?.pendingReviews ?? 0,
    messages: data?.openMessages ?? 0,
    chats: data?.unreadChats ?? 0,
  };
  return (
    <nav className="adm__nav" aria-label="Admin">
      {NAV_GROUPS.map((g, gi) => {
        const items = g.items.filter((i) => canView(role, i.key));
        if (!items.length) return null;
        const expanded = open === gi;
        return (
          <div key={g.label}>
            <button type="button" className="nav-group__btn" aria-expanded={expanded} onClick={() => setOpen(expanded ? -1 : gi)}>
              {g.label}
              <span className="nav-group__arrow" aria-hidden="true" />
            </button>
            {expanded && (
              <div className="nav-group__items">
                {items.map((i) => (
                  <NavLink key={i.key} to={`/admin/${i.key}`} className={({ isActive }) => (isActive ? 'is-active' : '')}>
                    {i.label}
                    {badges[i.key] > 0 && <span className="nav-badge">{badges[i.key]}</span>}
                  </NavLink>
                ))}
              </div>
            )}
          </div>
        );
      })}
    </nav>
  );
}

function Gate() {
  const user = useAuthStore((s) => s.user);
  return (
    <main className="container container--narrow page" style={{ maxWidth: 560, textAlign: 'center', paddingTop: 80 }}>
      <Seo title="Admin" />
      <span className="eyebrow">Admin</span>
      <h1 className="page-title" style={{ marginTop: 8 }}>
        Staff sign-in required
      </h1>
      <p className="page-intro">
        {user ? `${user.email} is a customer account.` : 'Sign in with a staff account to open the admin.'}
        {useMockApi && (
          <>
            <br />
            Preview mode: use an email containing <b>admin</b>, <b>manager</b>, <b>support</b> or <b>viewer</b> (e.g. admin@celestialoud.com) with any password.
          </>
        )}
      </p>
      <div style={{ display: 'flex', gap: 12, justifyContent: 'center', marginTop: 24 }}>
        <Link to="/account?next=/admin" className="btn btn--primary">
          Sign in
        </Link>
        <Link to="/" className="btn btn--outline">
          Back to store
        </Link>
      </div>
    </main>
  );
}

export default function AdminLayout() {
  const { section = 'dashboard' } = useParams();
  const user = useAuthStore((s) => s.user);
  const navigate = useNavigate();
  if (!isStaff(user?.role)) return <Gate />;

  const title = SECTION_TITLES[section] ?? 'Admin';
  return (
    <ToastProvider>
      <div className="adm">
        <Seo title={`${title} · Admin`} />
        {useMockApi && (
          <div className="adm-preview">
            Preview mode — changes are saved in this browser only.{' '}
            <button type="button" className="btn btn--ghost" style={{ color: 'inherit' }} onClick={() => { resetMockAdmin(); location.reload(); }}>
              Reset demo data
            </button>
          </div>
        )}
        <header className="adm__top">
          <Logo compact />
          <span className="adm__badge">Admin</span>
          <span className="adm__user">
            <span className="adm__user-email" title={user?.email}>
              {user?.email} · <b>{user?.role}</b>
            </span>
            <span className="table-actions">
            <Link to="/" className="btn btn--outline btn--xs">
              View store
            </Link>
            <button type="button" className="btn btn--outline btn--xs" onClick={() => { void logout().then(() => navigate('/')); }}>
              Sign out
            </button>
            </span>
          </span>
        </header>
        <div className="adm__body">
          <Nav section={section} role={user?.role} />
          <div className="adm__panel">
            <div className="adm__head">
              <span className="eyebrow">Celestial Oud · Back office</span>
              <h1>{title}</h1>
            </div>
            {canView(user?.role, section) ? (
              <Suspense fallback={<div className="skeleton" style={{ height: 240 }} />}>
                <Outlet />
              </Suspense>
            ) : (
              <p className="status status--danger">Your role does not have access to this section.</p>
            )}
          </div>
        </div>
      </div>
    </ToastProvider>
  );
}
