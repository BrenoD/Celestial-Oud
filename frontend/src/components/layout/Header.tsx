import { useEffect, useState, type FormEvent } from 'react';
import { Link, NavLink, useLocation, useNavigate } from 'react-router-dom';
import { useBasketStore, selectCount } from '@/store/basket';
import { useWishlistStore } from '@/store/wishlist';
import { BagIcon, CloseIcon, HeartIcon, MenuIcon, SearchIcon, UserIcon } from '@/components/ui/Icons';
import { Logo } from './Logo';
import { NAV } from './nav';
import './Header.css';

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
        <button type="button" className="hdr__burger" aria-label={menuOpen ? 'Close menu' : 'Open menu'} aria-expanded={menuOpen} onClick={() => setMenuOpen((v) => !v)}>
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
      <nav aria-label="Main" className={`hdr__nav${menuOpen ? ' is-open' : ''}`}>
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
