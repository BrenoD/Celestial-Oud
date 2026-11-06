import { useState, type FormEvent } from 'react';
import { Link, useNavigate, useParams, useSearchParams } from 'react-router-dom';
import { useOrders, useRegister, useSignIn } from '@/api/hooks';
import { useAuthStore } from '@/store/auth';
import { useWishlistStore } from '@/store/wishlist';
import { Seo } from '@/components/ui/Seo';
import { formatPence } from '@/lib/money';
import type { OrderStatus } from '@/types';
import './Account.css';

const TABS = [
  { id: 'orders', label: 'Orders' },
  { id: 'wishlist', label: 'Wishlist' },
  { id: 'scent-profile', label: 'Scent profile' },
  { id: 'addresses', label: 'Addresses' },
  { id: 'preferences', label: 'Communication preferences' },
  { id: 'privacy', label: 'Privacy & data' },
] as const;
type TabId = (typeof TABS)[number]['id'];

const STATUS_LABEL: Record<OrderStatus, string> = {
  confirmed: 'Confirmed',
  dispatched: 'Dispatched',
  delivered: 'Delivered',
  cancelled: 'Cancelled',
};

function SignIn() {
  const [sp] = useSearchParams();
  const navigate = useNavigate();
  const signInStore = useAuthStore((s) => s.signIn);
  const signIn = useSignIn();
  const register = useRegister();
  const [mode, setMode] = useState<'signin' | 'register'>(sp.get('register') ? 'register' : 'signin');
  const [email, setEmail] = useState(sp.get('email') ?? '');
  const [password, setPassword] = useState('');
  const [firstName, setFirstName] = useState('');
  const [lastName, setLastName] = useState('');
  const next = sp.get('next') ?? '/account';

  const submit = (e: FormEvent) => {
    e.preventDefault();
    const onSuccess = (r: { token: string; user: Parameters<typeof signInStore>[1] }) => {
      signInStore(r.token, r.user);
      navigate(next, { replace: true });
    };
    if (mode === 'signin') signIn.mutate({ email, password }, { onSuccess });
    else register.mutate({ email, password, firstName, lastName }, { onSuccess });
  };
  const pending = signIn.isPending || register.isPending;
  const error = (signIn.error ?? register.error) as Error | null;

  return (
    <div className="acct-auth">
      <form className="card acct-auth__form" onSubmit={submit}>
        <h1 style={{ fontSize: 34 }}>{mode === 'signin' ? 'Sign in' : 'Create account'}</h1>
        {mode === 'register' && (
          <div className="acct-auth__two">
            <label className="field">
              First name
              <input className="input" required autoComplete="given-name" value={firstName} onChange={(e) => setFirstName(e.target.value)} />
            </label>
            <label className="field">
              Last name
              <input className="input" required autoComplete="family-name" value={lastName} onChange={(e) => setLastName(e.target.value)} />
            </label>
          </div>
        )}
        <label className="field">
          Email address
          <input className="input" type="email" required autoComplete="email" value={email} onChange={(e) => setEmail(e.target.value)} />
        </label>
        <label className="field">
          Password
          <input className="input" type="password" required minLength={8} autoComplete={mode === 'signin' ? 'current-password' : 'new-password'} value={password} onChange={(e) => setPassword(e.target.value)} />
        </label>
        {mode === 'signin' && (
          <Link to="/account/reset" className="text-link" style={{ alignSelf: 'flex-start' }}>
            Forgotten your password?
          </Link>
        )}
        {error && (
          <p className="status status--error" role="alert">
            {error.message}
          </p>
        )}
        <button type="submit" className="btn btn--primary btn--lg" disabled={pending}>
          {pending ? 'Please wait…' : mode === 'signin' ? 'Sign in' : 'Create account'}
        </button>
        <span className="muted" style={{ fontSize: 13, textAlign: 'center' }}>
          or continue with
        </span>
        <div className="acct-auth__two">
          <button type="button" className="btn btn--outline" disabled title="Available once social login is configured">
            Google
          </button>
          <button type="button" className="btn" style={{ background: '#000', color: '#fff' }} disabled title="Available once social login is configured">
            Apple
          </button>
        </div>
      </form>

      <div className="panel acct-auth__side">
        <h2 style={{ fontSize: 34 }}>{mode === 'signin' ? 'New here?' : 'Already have an account?'}</h2>
        {mode === 'signin' ? (
          <>
            <span className="muted">Create an account to:</span>
            <ul className="acct-auth__list">
              <li>Track your orders and reorder in one click</li>
              <li>Save your wishlist and scent profile</li>
              <li>Get 10% off your first order</li>
            </ul>
          </>
        ) : (
          <span className="muted">Sign in to see your orders, wishlist and saved addresses.</span>
        )}
        <button type="button" className="btn btn--primary btn--lg" style={{ marginTop: 'auto' }} onClick={() => setMode(mode === 'signin' ? 'register' : 'signin')}>
          {mode === 'signin' ? 'Create account' : 'Sign in'}
        </button>
      </div>
    </div>
  );
}

function Dashboard() {
  const { tab = 'orders' } = useParams<{ tab: TabId }>();
  const navigate = useNavigate();
  const { user, signOut } = useAuthStore();
  const orders = useOrders(tab === 'orders');
  const wishlist = useWishlistStore((s) => s.slugs);
  const current = TABS.find((t) => t.id === tab) ?? TABS[0];

  return (
    <div className="acct">
      <nav aria-label="Account menu" className="card acct__menu">
        {TABS.map((t) => (
          <Link key={t.id} to={`/account/${t.id}`} className={`acct__tab${t.id === current.id ? ' is-active' : ''}`} aria-current={t.id === current.id ? 'page' : undefined}>
            {t.label}
          </Link>
        ))}
        <button
          type="button"
          className="acct__tab acct__tab--out"
          onClick={() => {
            signOut();
            navigate('/');
          }}
        >
          Sign out
        </button>
      </nav>

      <div className="acct__content">
        <h1 style={{ fontSize: 38 }}>{current.label}</h1>
        {user && current.id === 'orders' && (
          <p className="muted">
            Signed in as {user.firstName} ({user.email}).
          </p>
        )}

        {current.id === 'orders' && (
          <div className="acct__orders">
            {orders.isPending && <div className="skeleton" style={{ height: 80 }} />}
            {orders.isError && <p className="status status--error">Could not load your orders.</p>}
            {orders.data?.length === 0 && <p className="muted">You have not placed any orders yet.</p>}
            {orders.data?.map((o) => (
              <div key={o.id} className="card order">
                <span className="order__meta">
                  <b>#{o.id}</b>
                  <span className="muted" style={{ fontSize: 14 }}>
                    {new Date(o.createdAt).toLocaleDateString('en-GB', { day: 'numeric', month: 'short', year: 'numeric' })} · {formatPence(o.total)}
                  </span>
                </span>
                <span className={`pill${o.status === 'dispatched' ? ' pill--accent' : ''}`}>{STATUS_LABEL[o.status]}</span>
                <span className="order__actions">
                  {o.status === 'dispatched' && (
                    <Link to={`/help#track`} className="btn btn--primary btn--sm">
                      Track
                    </Link>
                  )}
                  {o.status === 'delivered' ? (
                    <Link to="/collections/all" className="btn btn--outline btn--sm">
                      Write a review
                    </Link>
                  ) : (
                    <Link to="/collections/bestsellers" className="btn btn--outline btn--sm">
                      Buy again
                    </Link>
                  )}
                </span>
              </div>
            ))}
          </div>
        )}

        {current.id === 'wishlist' && (
          <div className="acct__note">
            {wishlist.length ? (
              <>
                You have {wishlist.length} saved {wishlist.length === 1 ? 'fragrance' : 'fragrances'}. <Link to="/wishlist">View wishlist</Link>.
              </>
            ) : (
              <>
                No saved fragrances yet. Tap the heart on any product to save it. <Link to="/collections/all">Browse all</Link>.
              </>
            )}
          </div>
        )}
        {current.id === 'scent-profile' && <div className="acct__note">[Quiz result, favourite notes and notes to avoid — powers recommendations.]</div>}
        {current.id === 'addresses' && <div className="acct__note">[Saved delivery addresses and default address.]</div>}
        {current.id === 'preferences' && (
          <div className="acct__note">
            <label className="check">
              <input type="checkbox" /> Email — offers and new launches
            </label>
            <label className="check">
              <input type="checkbox" /> SMS — delivery updates
            </label>
            <label className="check">
              <input type="checkbox" /> WhatsApp — delivery updates
            </label>
          </div>
        )}
        {current.id === 'privacy' && (
          <div className="acct__note">
            <p>Under UK GDPR you can download a copy of the data we hold about you, or delete your account.</p>
            <div style={{ display: 'flex', gap: 10, flexWrap: 'wrap', marginTop: 12 }}>
              <button type="button" className="btn btn--outline btn--sm">
                Download my data
              </button>
              <button type="button" className="btn btn--outline btn--sm" style={{ borderColor: 'var(--danger)', color: 'var(--danger)' }}>
                Delete my account
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

export default function Account() {
  const token = useAuthStore((s) => s.token);
  return (
    <main className="container container--narrow page">
      <Seo title="My account" />
      <nav aria-label="Breadcrumb" className="breadcrumb">
        <Link to="/">Home</Link> › <span>My account</span>
      </nav>
      {token ? <Dashboard /> : <SignIn />}
    </main>
  );
}
