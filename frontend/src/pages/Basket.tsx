import { useState, type FormEvent } from 'react';
import { Link } from 'react-router-dom';
import { selectCount, selectSubtotal, useBasketStore } from '@/store/basket';
import { useValidateDiscount } from '@/api/hooks';
import { Qty } from '@/components/ui/Qty';
import { Seo } from '@/components/ui/Seo';
import { formatPence, formatPenceShort } from '@/lib/money';
import { assetUrl, config } from '@/config';
import { freeSamples } from '@/data/catalogue';
import './Basket.css';

export function FreeDeliveryBar({ subtotal }: { subtotal: number }) {
  const threshold = config.freeDeliveryThreshold;
  const remaining = Math.max(0, threshold - subtotal);
  const pct = Math.min(100, Math.round((subtotal / threshold) * 100));
  return (
    <div className="panel freebar">
      <span>
        {remaining > 0 ? (
          <>
            You're <b>{formatPence(remaining)}</b> away from <b>free UK delivery</b>
          </>
        ) : (
          <>
            <b>You've unlocked free UK delivery.</b>
          </>
        )}
      </span>
      <div className="freebar__track" role="progressbar" aria-valuenow={pct} aria-valuemin={0} aria-valuemax={100} aria-label="Progress to free delivery">
        <div className="freebar__fill" style={{ width: `${pct}%` }} />
      </div>
    </div>
  );
}

export default function Basket() {
  const s = useBasketStore();
  const count = useBasketStore(selectCount);
  const subtotal = useBasketStore(selectSubtotal);
  const [code, setCode] = useState(s.discountCode ?? '');
  const discount = useValidateDiscount();

  const applyCode = (e: FormEvent) => {
    e.preventDefault();
    if (!code.trim()) return;
    discount.mutate(
      { code: code.trim(), subtotal },
      { onSuccess: (r) => s.applyDiscount(r.code, r.amount), onError: () => s.clearDiscount() },
    );
  };

  const total = Math.max(0, subtotal - s.discountAmount);

  return (
    <main className="container page">
      <Seo title="Your basket" />
      <nav aria-label="Breadcrumb" className="breadcrumb">
        <Link to="/">Home</Link> › <span>Basket</span>
      </nav>
      <h1 className="page-title">Your basket</h1>

      <div className="basket">
        <div className="basket__lines">
          <FreeDeliveryBar subtotal={subtotal} />

          {s.lines.length === 0 ? (
            <div className="card basket__empty">
              <span className="basket__empty-title">Your basket is empty</span>
              <Link to="/collections/bestsellers" className="btn btn--primary">
                Shop bestsellers
              </Link>
            </div>
          ) : (
            s.lines.map((l) => (
              <div key={l.sku} className="card line">
                <img src={assetUrl(l.image)} alt="" width={110} height={110} loading="lazy" />
                <div className="line__info">
                  <Link to={`/products/${l.slug}`} className="line__name">
                    {l.name}
                  </Link>
                  <span className="muted" style={{ fontSize: 14 }}>
                    {l.variantLabel}
                  </span>
                  <button type="button" className="btn btn--ghost" style={{ alignSelf: 'flex-start', fontSize: 13 }} onClick={() => s.remove(l.sku)}>
                    Remove
                  </button>
                </div>
                <Qty value={l.qty} onChange={(q) => s.setQty(l.sku, q)} size="sm" />
                <span className="line__price">{formatPence(l.unitPrice * l.qty)}</span>
              </div>
            ))
          )}

          {s.lines.length > 0 && (
            <fieldset className="card samples">
              <legend>Choose your free sample</legend>
              <div className="samples__row">
                {freeSamples.map((f) => (
                  <button key={f.sku} type="button" className="option option--sm" aria-pressed={s.freeSampleSku === f.sku} onClick={() => s.setFreeSample(f.sku)}>
                    {f.name}
                  </button>
                ))}
              </div>
            </fieldset>
          )}

          <Link to="/collections/all" className="text-link">
            ← Continue shopping
          </Link>
        </div>

        <aside aria-label="Order summary" className="card summary">
          <span className="summary__title">Order summary</span>
          <div className="summary__row">
            <span>Subtotal ({count} items)</span>
            <span>{formatPence(subtotal)}</span>
          </div>
          <div className="summary__row">
            <span>Delivery</span>
            <span>{subtotal >= config.freeDeliveryThreshold ? 'Free' : 'Calculated at checkout'}</span>
          </div>
          <div className="summary__row">
            <span>Free sample</span>
            <span>{freeSamples.find((f) => f.sku === s.freeSampleSku)?.name ?? 'Choose one'}</span>
          </div>
          {s.discountCode && (
            <div className="summary__row" style={{ color: 'var(--success)' }}>
              <span>Discount ({s.discountCode})</span>
              <span>−{formatPence(s.discountAmount)}</span>
            </div>
          )}

          <form className="field" onSubmit={applyCode}>
            Discount code or gift card
            <span style={{ display: 'flex', gap: 8 }}>
              <input className="input" value={code} onChange={(e) => setCode(e.target.value)} aria-label="Discount code" autoComplete="off" />
              <button type="submit" className="btn btn--outline btn--sm" disabled={discount.isPending}>
                Apply
              </button>
            </span>
            {discount.isError && <span className="field__error">{(discount.error as Error).message}</span>}
          </form>

          <label className="check">
            <input type="checkbox" checked={s.giftWrap} onChange={(e) => s.setGiftWrap(e.target.checked)} /> Add free gift wrap & message
          </label>
          {s.giftWrap && (
            <label className="field">
              Gift message
              <textarea className="textarea" rows={3} maxLength={200} value={s.giftMessage} onChange={(e) => s.setGiftWrap(true, e.target.value)} />
            </label>
          )}

          <div className="summary__total">
            <span>Total</span>
            <span>{formatPence(total)}</span>
          </div>
          <span className="muted" style={{ fontSize: 12, marginTop: -8 }}>
            Including VAT
          </span>
          {s.lines.length > 0 ? (
            <Link to="/checkout" className="btn btn--primary btn--lg btn--block">
              Checkout securely
            </Link>
          ) : (
            <span className="btn btn--primary btn--lg btn--block" aria-disabled="true">
              Checkout securely
            </span>
          )}
          <span className="muted" style={{ fontSize: 12, textAlign: 'center' }}>
            Visa · Mastercard · Apple Pay · PayPal · Klarna
          </span>
          <span className="muted" style={{ fontSize: 12, textAlign: 'center' }}>
            Free UK delivery over {formatPenceShort(config.freeDeliveryThreshold)}
          </span>
        </aside>
      </div>
    </main>
  );
}
