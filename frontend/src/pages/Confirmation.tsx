import { Link, useLocation, useParams, useSearchParams } from 'react-router-dom';
import { orderToken } from '@/lib/orderAccess';
import { useOrder } from '@/api/hooks';
import { useAuthStore } from '@/store/auth';
import { Seo } from '@/components/ui/Seo';
import { CheckIcon } from '@/components/ui/Icons';
import { formatPence } from '@/lib/money';
import { ProductImage } from '@/components/ui/ProductImage';
import type { Order } from '@/types';
import './Confirmation.css';

export default function Confirmation() {
  const { id = '' } = useParams();
  const location = useLocation();
  const [params] = useSearchParams();
  const state = location.state as { order?: Order; email?: string } | null;
  const fromState = state?.order;
  // Token from the email link or this tab's checkout; email from the track form (kept out of the URL).
  const q = useOrder(id, { token: params.get('token') ?? orderToken(id), email: state?.email });
  const user = useAuthStore((s) => s.user);
  const order = fromState?.id === id ? fromState : q.data;

  if (!order) {
    return (
      <main className="container confirm" aria-busy={q.isPending}>
        {q.isError ? (
          <>
            <h1 className="page-title">We could not find that order</h1>
            <p className="page-intro">
              Check the link in your confirmation email, or <Link to="/help#track">track your order</Link>.
            </p>
          </>
        ) : (
          <div className="skeleton" style={{ height: 300, width: '100%' }} />
        )}
      </main>
    );
  }

  const eta = new Date(order.estimatedDelivery).toLocaleDateString('en-GB', { weekday: 'long', day: 'numeric', month: 'long' });
  const a = order.shippingAddress;

  return (
    <main className="container confirm">
      <Seo title={`Order ${order.id} confirmed`} />
      <span className="confirm__tick">
        <CheckIcon />
      </span>
      <h1 className="page-title">Thank you — your order is confirmed</h1>
      <p className="confirm__lead">
        Order <b>#{order.id}</b>. We've sent a confirmation to <b>{order.email}</b>. Estimated delivery: <b>{eta}</b>.
      </p>

      <div className="card confirm__box">
        {order.lines.map((l, i) => (
          <div key={`${l.sku}-${i}`} className="confirm__line">
            <span className="confirm__thumb">
              <ProductImage src={l.image} width={64} height={64} className="pimg--mini" />
            </span>
            <span>
              <b>{l.name}</b> · {l.variantLabel} × {l.qty}
            </span>
            <span style={{ fontWeight: 600 }}>{formatPence(l.unitPrice * l.qty)}</span>
          </div>
        ))}
        <div className="confirm__row">
          <span>Subtotal</span>
          <span>{formatPence(order.subtotal)}</span>
        </div>
        <div className="confirm__row">
          <span>Delivery</span>
          <span>{order.delivery === 0 ? 'Free' : formatPence(order.delivery)}</span>
        </div>
        {order.discount > 0 && (
          <div className="confirm__row">
            <span>Discount</span>
            <span>−{formatPence(order.discount)}</span>
          </div>
        )}
        <div className="confirm__total">
          <span>Total paid</span>
          <span>{formatPence(order.total)}</span>
        </div>
        <span className="muted" style={{ fontSize: 14 }}>
          Delivering to: {a.firstName} {a.lastName}, {a.line1}, {a.city}, {a.postcode}
        </span>
      </div>

      {!user && (
        <div className="panel confirm__account">
          <span>
            <b>Save your details for next time.</b>
            <br />
            <span className="muted">Create an account with one click to track this order.</span>
          </span>
          <Link to={`/account?register=1&email=${encodeURIComponent(order.email)}`} className="btn btn--primary">
            Create account
          </Link>
        </div>
      )}

      <Link to="/" className="btn btn--secondary btn--lg">
        Continue shopping
      </Link>
    </main>
  );
}
