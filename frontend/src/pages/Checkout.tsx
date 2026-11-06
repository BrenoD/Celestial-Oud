import { useEffect, useMemo, useState, type FormEvent } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useCreateOrder, useDeliveryMethods, usePostcodeLookup } from '@/api/hooks';
import { selectSubtotal, useBasketStore } from '@/store/basket';
import { useAuthStore } from '@/store/auth';
import { Seo } from '@/components/ui/Seo';
import { formatPence } from '@/lib/money';
import { assetUrl, config } from '@/config';
import { freeSamples } from '@/data/catalogue';
import type { Address } from '@/types';
import './Checkout.css';
import './Basket.css';

interface Form {
  email: string;
  marketing: boolean;
  firstName: string;
  lastName: string;
  postcode: string;
  line1: string;
  line2: string;
  city: string;
  phone: string;
  cardNumber: string;
  expiry: string;
  cvc: string;
  cardName: string;
  billingSame: boolean;
  agree: boolean;
}

const EMPTY: Form = {
  email: '',
  marketing: false,
  firstName: '',
  lastName: '',
  postcode: '',
  line1: '',
  line2: '',
  city: '',
  phone: '',
  cardNumber: '',
  expiry: '',
  cvc: '',
  cardName: '',
  billingSame: true,
  agree: false,
};

type Errors = Partial<Record<keyof Form, string>>;

interface FieldProps {
  name: keyof Form;
  label: string;
  value: string;
  error?: string;
  onChange: (v: string) => void;
  type?: string;
  placeholder?: string;
  autoComplete?: string;
  inputMode?: 'numeric' | 'tel' | 'email';
}

/** Defined at module level so React keeps the same input mounted across renders. */
function Field({ name, label, value, error, onChange, type = 'text', placeholder, autoComplete, inputMode }: FieldProps) {
  const id = `chk-${name}`;
  return (
    <label className="field" htmlFor={id}>
      {label}
      <input
        id={id}
        className="input"
        type={type}
        value={value}
        placeholder={placeholder}
        autoComplete={autoComplete}
        inputMode={inputMode}
        aria-invalid={error ? true : undefined}
        aria-describedby={error ? `${id}-err` : undefined}
        onChange={(e) => onChange(e.target.value)}
      />
      {error && (
        <span id={`${id}-err`} className="field__error">
          {error}
        </span>
      )}
    </label>
  );
}

function validate(f: Form): Errors {
  const e: Errors = {};
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(f.email)) e.email = 'Enter a valid email address';
  if (!f.firstName.trim()) e.firstName = 'Enter your first name';
  if (!f.lastName.trim()) e.lastName = 'Enter your last name';
  if (!/^[A-Za-z]{1,2}\d[A-Za-z\d]? ?\d[A-Za-z]{2}$/.test(f.postcode.trim())) e.postcode = 'Enter a valid UK postcode';
  if (!f.line1.trim()) e.line1 = 'Enter your address';
  if (!f.city.trim()) e.city = 'Enter your town or city';
  if (f.cardNumber.replace(/\s/g, '').length < 13) e.cardNumber = 'Enter your card number';
  if (!/^\d{2}\s?\/\s?\d{2}$/.test(f.expiry)) e.expiry = 'Use MM / YY';
  if (!/^\d{3,4}$/.test(f.cvc)) e.cvc = 'Enter the 3 or 4 digit code';
  if (!f.cardName.trim()) e.cardName = 'Enter the name on the card';
  if (!f.agree) e.agree = 'You need to accept the terms to continue';
  return e;
}

export default function Checkout() {
  const navigate = useNavigate();
  const basket = useBasketStore();
  const subtotal = useBasketStore(selectSubtotal);
  const user = useAuthStore((s) => s.user);
  const methods = useDeliveryMethods();
  const lookup = usePostcodeLookup();
  const createOrder = useCreateOrder();

  const [f, setF] = useState<Form>({ ...EMPTY, email: user?.email ?? '', firstName: user?.firstName ?? '', lastName: user?.lastName ?? '' });
  const [errors, setErrors] = useState<Errors>({});
  const [methodId, setMethodId] = useState<string>('standard');
  const [addresses, setAddresses] = useState<Array<Pick<Address, 'line1' | 'line2' | 'city' | 'postcode'>>>([]);

  // One key per checkout attempt: a retry after a network blip creates the
  // same order on the server instead of a duplicate.
  const idempotencyKey = useMemo(() => crypto.randomUUID(), []);

  useEffect(() => {
    if (basket.lines.length === 0 && !createOrder.isSuccess) navigate('/basket', { replace: true });
  }, [basket.lines.length, createOrder.isSuccess, navigate]);

  const set = <K extends keyof Form>(k: K, v: Form[K]) => {
    setF((prev) => ({ ...prev, [k]: v }));
    if (errors[k]) setErrors((prev) => ({ ...prev, [k]: undefined }));
  };

  const method = methods.data?.find((m) => m.id === methodId) ?? methods.data?.[0];
  const freeStandard = subtotal >= config.freeDeliveryThreshold && method?.id === 'standard';
  const deliveryCost = method ? (freeStandard ? 0 : method.price) : 0;
  const total = Math.max(0, subtotal - basket.discountAmount) + deliveryCost;

  const findAddress = () => {
    lookup.mutate(f.postcode, { onSuccess: (list) => setAddresses(list) });
  };
  const pickAddress = (a: (typeof addresses)[number]) => {
    setF((p) => ({ ...p, line1: a.line1, line2: a.line2 ?? '', city: a.city, postcode: a.postcode }));
    setErrors((p) => ({ ...p, line1: undefined, line2: undefined, city: undefined, postcode: undefined }));
    setAddresses([]);
  };

  const submit = (e: FormEvent) => {
    e.preventDefault();
    const errs = validate(f);
    setErrors(errs);
    if (Object.keys(errs).length) {
      document.querySelector<HTMLElement>('[aria-invalid="true"]')?.focus();
      return;
    }
    if (!method) return;
    createOrder.mutate(
      {
        idempotencyKey,
        req: {
          email: f.email,
          marketingOptIn: f.marketing,
          lines: basket.lines.map((l) => ({ sku: l.sku, qty: l.qty })),
          freeSampleSku: basket.freeSampleSku ?? undefined,
          giftWrap: basket.giftWrap,
          giftMessage: basket.giftWrap ? basket.giftMessage : undefined,
          discountCode: basket.discountCode ?? undefined,
          deliveryMethodId: method.id,
          shippingAddress: {
            firstName: f.firstName,
            lastName: f.lastName,
            line1: f.line1,
            line2: f.line2 || undefined,
            city: f.city,
            postcode: f.postcode.toUpperCase(),
            country: 'GB',
            phone: f.phone || undefined,
          },
          // In production the PSP (Stripe/Adyen) tokenises the card in the
          // browser; raw card data never reaches our API.
          paymentToken: `tok_${idempotencyKey}`,
        },
      },
      {
        onSuccess: (order) => {
          basket.clear();
          navigate(`/checkout/confirmed/${order.id}`, { replace: true, state: { order } });
        },
      },
    );
  };

  // Plain function call (not a component) so the real <Field> keeps its identity across renders.
  const field = (p: Omit<FieldProps, 'value' | 'error' | 'onChange'>) => (
    <Field key={p.name} {...p} value={String(f[p.name])} error={errors[p.name]} onChange={(v) => set(p.name, v as never)} />
  );

  return (
    <main className="container container--narrow chk">
      <Seo title="Checkout" />
      <div className="chk__grid">
        <form className="chk__form" onSubmit={submit} noValidate>
          <div className="express">
            <span className="muted" style={{ fontSize: 13, textAlign: 'center' }}>
              Express checkout
            </span>
            <div className="express__row">
              <button type="button" className="express__btn express__btn--apple" disabled title="Available once payments are connected">
                Apple Pay
              </button>
              <button type="button" className="express__btn express__btn--google" disabled title="Available once payments are connected">
                Google Pay
              </button>
              <button type="button" className="express__btn express__btn--paypal" disabled title="Available once payments are connected">
                PayPal
              </button>
            </div>
            <span className="muted" style={{ fontSize: 13, textAlign: 'center' }}>
              or pay with card
            </span>
          </div>

          <section className="chk__section">
            <h2>1. Contact</h2>
            {field({ name: 'email', label: 'Email address', type: 'email', placeholder: 'you@example.com', autoComplete: 'email', inputMode: 'email' })}
            <label className="check" style={{ fontSize: 14 }}>
              <input type="checkbox" checked={f.marketing} onChange={(e) => set('marketing', e.target.checked)} /> Email me offers and new launches (optional)
            </label>
            {!user && (
              <span className="muted" style={{ fontSize: 14 }}>
                Already have an account? <Link to="/account?next=/checkout">Sign in</Link>
              </span>
            )}
          </section>

          <section className="chk__section">
            <h2>2. Delivery address</h2>
            <div className="chk__two">
              {field({ name: 'firstName', label: 'First name', autoComplete: 'given-name' })}
              {field({ name: 'lastName', label: 'Last name', autoComplete: 'family-name' })}
            </div>
            <div className="chk__postcode">
              {field({ name: 'postcode', label: 'Postcode', placeholder: 'e.g. SW1A 1AA', autoComplete: 'postal-code' })}
              <button type="button" className="btn btn--outline" onClick={findAddress} disabled={lookup.isPending || !f.postcode.trim()}>
                {lookup.isPending ? 'Searching…' : 'Find address'}
              </button>
            </div>
            {lookup.isError && <span className="field__error">{(lookup.error as Error).message}</span>}
            {addresses.length > 0 && (
              <ul className="addr-list card" aria-label="Matching addresses">
                {addresses.map((a, i) => (
                  <li key={i}>
                    <button type="button" onClick={() => pickAddress(a)}>
                      {a.line1}, {a.city}, {a.postcode}
                    </button>
                  </li>
                ))}
              </ul>
            )}
            {field({ name: 'line1', label: 'Address line 1', autoComplete: 'address-line1' })}
            {field({ name: 'line2', label: 'Address line 2 (optional)', autoComplete: 'address-line2' })}
            <div className="chk__two">
              {field({ name: 'city', label: 'Town / City', autoComplete: 'address-level2' })}
              {field({ name: 'phone', label: 'Phone (for delivery updates)', type: 'tel', autoComplete: 'tel', inputMode: 'tel' })}
            </div>
          </section>

          <section className="chk__section">
            <h2>3. Delivery method</h2>
            <div className="methods" role="radiogroup" aria-label="Delivery method">
              {methods.isPending && <div className="skeleton" style={{ height: 60 }} />}
              {methods.data?.map((m) => {
                const free = subtotal >= config.freeDeliveryThreshold && m.id === 'standard';
                return (
                  <button key={m.id} type="button" role="radio" aria-checked={methodId === m.id} className="method" onClick={() => setMethodId(m.id)}>
                    <span>
                      <b>{m.name}</b>
                      <span className="muted" style={{ fontSize: 13 }}>
                        {m.eta}
                      </span>
                    </span>
                    <span style={{ fontWeight: 600 }}>{free ? 'Free' : formatPence(m.price)}</span>
                  </button>
                );
              })}
            </div>
          </section>

          <section className="chk__section">
            <h2>4. Payment</h2>
            {field({ name: 'cardNumber', label: 'Card number', placeholder: '1234 1234 1234 1234', autoComplete: 'cc-number', inputMode: 'numeric' })}
            <div className="chk__two">
              {field({ name: 'expiry', label: 'Expiry date', placeholder: 'MM / YY', autoComplete: 'cc-exp', inputMode: 'numeric' })}
              {field({ name: 'cvc', label: 'Security code', placeholder: 'CVC', autoComplete: 'cc-csc', inputMode: 'numeric' })}
            </div>
            {field({ name: 'cardName', label: 'Name on card', autoComplete: 'cc-name' })}
            <label className="check" style={{ fontSize: 14 }}>
              <input type="checkbox" checked={f.billingSame} onChange={(e) => set('billingSame', e.target.checked)} /> Billing address same as delivery
            </label>
          </section>

          <label className="check" style={{ fontSize: 14, alignItems: 'flex-start', lineHeight: 1.5 }}>
            <input type="checkbox" checked={f.agree} onChange={(e) => set('agree', e.target.checked)} aria-invalid={errors.agree ? true : undefined} style={{ marginTop: 2 }} />
            <span>
              I agree to the <Link to="/legal/terms">Terms & Conditions</Link> and <Link to="/help#returns">Returns policy</Link>. Opened fragrances cannot be returned for hygiene reasons.
            </span>
          </label>
          {errors.agree && <span className="field__error">{errors.agree}</span>}

          {createOrder.isError && (
            <p className="status status--error" role="alert">
              We could not take your payment: {(createOrder.error as Error).message}. Nothing has been charged — please try again.
            </p>
          )}

          <button type="submit" className="btn btn--primary btn--lg btn--block" style={{ height: 54, fontSize: 16 }} disabled={createOrder.isPending || !method}>
            {createOrder.isPending ? 'Processing…' : `Pay ${formatPence(total)}`}
          </button>
          <Link to="/basket" className="text-link" style={{ textAlign: 'center' }}>
            ← Return to basket
          </Link>
        </form>

        <aside aria-label="Order summary" className="summary summary--ivory card">
          <span className="summary__title" style={{ fontSize: 24 }}>
            Order summary
          </span>
          {basket.lines.map((l) => (
            <div key={l.sku} className="summary__item">
              <img src={assetUrl(l.image)} alt="" width={64} height={64} />
              <span>
                <b>{l.name}</b>
                <br />
                {l.variantLabel} × {l.qty}
              </span>
              <span style={{ fontWeight: 600 }}>{formatPence(l.unitPrice * l.qty)}</span>
            </div>
          ))}
          <div className="summary__row" style={{ borderTop: '1px solid var(--line-2)', paddingTop: 14 }}>
            <span>Subtotal</span>
            <span>{formatPence(subtotal)}</span>
          </div>
          <div className="summary__row">
            <span>Delivery{method ? ` (${method.name})` : ''}</span>
            <span>{method ? (deliveryCost === 0 ? 'Free' : formatPence(deliveryCost)) : '—'}</span>
          </div>
          {basket.discountCode && (
            <div className="summary__row" style={{ color: 'var(--success)' }}>
              <span>Discount ({basket.discountCode})</span>
              <span>−{formatPence(basket.discountAmount)}</span>
            </div>
          )}
          <div className="summary__row">
            <span>Free sample</span>
            <span>{freeSamples.find((x) => x.sku === basket.freeSampleSku)?.name ?? '—'}</span>
          </div>
          {basket.giftWrap && (
            <div className="summary__row">
              <span>Gift wrap</span>
              <span>Free</span>
            </div>
          )}
          <div className="summary__total" style={{ borderTopColor: 'var(--line-2)' }}>
            <span>Total</span>
            <span>{formatPence(total)}</span>
          </div>
          <span className="muted" style={{ fontSize: 12 }}>
            Including VAT
          </span>
        </aside>
      </div>
    </main>
  );
}
