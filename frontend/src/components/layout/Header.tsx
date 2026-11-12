import { useEffect, useRef, useState, type FormEvent } from 'react';
import { useOverlay } from '@/lib/useOverlay';
import { Link, NavLink, useLocation, useNavigate } from 'react-router-dom';
import { useBasketStore, selectCount } from '@/store/basket';
import { useWishlistStore } from '@/store/wishlist';
import { BagIcon, CloseIcon, HeartIcon, MenuIcon, SearchIcon, UserIcon } from '@/components/ui/Icons';
import { Logo } from './Logo';
import { NAV } from './nav';
import './Header.css';

/** Mobile menu: a full-height drawer over a dimmed page (rendered only while open). */
function MobileMenu({ onClose, count, wishCount }: { onClose: () => void; count: number; wishCount: number }) {
  const panel = useRef<HTMLDivElement>(null);
  useOverlay(true, onClose, panel, 900);
  return (
    <>
      <button type="button" className="backdrop" aria-label="Close menu" tabIndex={-1} onClick={onClose} />
      <div ref={panel} id="mobile-menu" className="sheet sheet--left mnav" role="dialog" aria-modal="true" aria-label="Menu">
        <div className="sheet__head">
          <p className="sheet__title">Menu</p>
          <button type="button" className="sheet__close" aria-label="Close menu" onClick={onClose} data-autofocus>
            <CloseIcon />
          </button>
        </div>
        <nav aria-label="Main" className="sheet__body">
          <ul className="mnav__list stagger">
            {NAV.map((n, i) => (
              <li key={n.to} style={{ ['--i' as string]: i }}>
                <NavLink to={n.to} className={({ isActive }) => `mnav__link${n.accent ? ' mnav__link--accent' : ''}${isActive ? ' is-active' : ''}`}>
                  {n.label}
                  <span aria-hidden="true" className="mnav__arrow">→</span>
                </NavLink>
              </li>
            ))}
          </ul>
          <ul className="mnav__secondary stagger">
            <li style={{ ['--i' as string]: NAV.length }}>
              <Link to="/account">My account</Link>
            </li>
            <li style={{ ['--i' as string]: NAV.length + 1 }}>
              <Link to="/wishlist">Wishlist{wishCount ? ` (${wishCount})` : ''}</Link>
            </li>
            <li style={{ ['--i' as string]: NAV.length + 2 }}>
              <Link to="/basket">Basket{count ? ` (${count})` : ''}</Link>
            </li>
            <li style={{ ['--i' as string]: NAV.length + 3 }}>
              <Link to="/help">Help & contact</Link>
            </li>
          </ul>
        </nav>
      </div>
    </>
  );
}

export function Header() {
  const count = useBasketStore(selectCount);
  const wishCount = useWishlistStore((s) => s.slugs.length);
  const navigate = useNavigate();
  const location = useLocation();
  const [q, setQ] = useState('');
  const [menuOpen, setMenuOpen] = useState(false);

  useEffect(() => setMenuOpen(false), [location.pathname]);

  const submit = (e: FormEvent) => {
    e.preventDefault();
    const term = q.trim();
    navigate(term ? `/search?q=${encodeURIComponent(term)}` : '/collections/all');
  };

  return (
    <header className="hdr">
      <div className="container hdr__top">
        <button type="button" className={`hdr__burger${menuOpen ? ' is-on' : ''}`} aria-label={menuOpen ? 'Close menu' : 'Open menu'} aria-expanded={menuOpen} aria-controls="mobile-menu" onClick={() => setMenuOpen((v) => !v)}>
          {menuOpen ? <CloseIcon /> : <MenuIcon />}
        </button>
        <Logo />
        <form className="hdr__search" role="search" onSubmit={submit}>
          <label htmlFor="site-search" className="visually-hidden">
            Search
          </label>
          <input
            id="site-search"
            type="search"
            value={q}
            onChange={(e) => setQ(e.target.value)}
            placeholder="Search perfumes, notes (oud, amber, rose…)"
            autoComplete="off"
          />
          <button type="submit" aria-label="Search">
            <SearchIcon />
          </button>
        </form>
        <nav aria-label="Account" className="hdr__actions">
          <Link to="/account" aria-label="Account" className="hdr__icon">
            <UserIcon />
          </Link>
          <Link to="/wishlist" aria-label={`Wishlist, ${wishCount} items`} className="hdr__icon">
            <HeartIcon filled={wishCount > 0} />
          </Link>
          <Link to="/basket" className="hdr__basket" aria-label={`Basket, ${count} items`}>
            <BagIcon />
            <span className="hdr__basket-label">Basket</span>
            <span className="hdr__count" aria-hidden="true">
              {count}
            </span>
          </Link>
        </nav>
      </div>
      {menuOpen && <MobileMenu onClose={() => setMenuOpen(false)} count={count} wishCount={wishCount} />}
      <nav aria-label="Main" className="hdr__nav">
        <div className="container hdr__nav-inner">
          {NAV.map((n) => (
            <NavLink key={n.to} to={n.to} className={({ isActive }) => `hdr__link${n.accent ? ' hdr__link--accent' : ''}${isActive ? ' is-active' : ''}`}>
              {n.label}
            </NavLink>
          ))}
        </div>
      </nav>
    </header>
  );
}
