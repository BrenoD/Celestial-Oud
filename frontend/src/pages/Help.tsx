import { useEffect, useState, type FormEvent } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useContact, useFaqs } from '@/api/hooks';
import { Accordion } from '@/components/ui/Accordion';
import { Seo } from '@/components/ui/Seo';
import { config } from '@/config';
import { openChat } from '@/components/chat/chatStore';
import './Help.css';

export default function Help() {
  const faqs = useFaqs();
  const contact = useContact();
  const navigate = useNavigate();
  const [track, setTrack] = useState({ order: '', email: '' });
  const [msg, setMsg] = useState({ name: '', email: '', orderId: '', message: '' });

  const submitTrack = (e: FormEvent) => {
    e.preventDefault();
    const id = track.order.trim().replace(/^#/, '').toUpperCase();
    // The email goes in navigation state, not the URL (no personal data in logs or history).
    if (id) navigate(`/checkout/confirmed/${encodeURIComponent(id)}`, { state: { email: track.email.trim() } });
  };
  // Links in chat emails land on /help#chat: open the chat straight away.
  useEffect(() => {
    if (location.hash === '#chat') openChat();
  }, []);
  const submitContact = (e: FormEvent) => {
    e.preventDefault();
    contact.mutate({ ...msg, orderId: msg.orderId || undefined });
  };

  return (
    <main className="container container--narrow page help">
      <Seo title="Help & contact" description="Quick answers about orders, delivery and returns." />
      <div>
        <nav aria-label="Breadcrumb" className="breadcrumb">
          <Link to="/">Home</Link> › <span>Help & contact</span>
        </nav>
        <h1 className="page-title">Help & contact</h1>
        <p className="page-intro">Quick answers about orders, delivery and returns.</p>
      </div>

      <div className="help__top">
        <form id="track" className="panel help__track" onSubmit={submitTrack}>
          <h2>Track my order</h2>
          <div className="help__two">
            <label className="field">
              Order number
              <input className="input" placeholder="#CO-XXXX" value={track.order} onChange={(e) => setTrack({ ...track, order: e.target.value })} required />
            </label>
            <label className="field">
              Email address
              <input className="input" type="email" value={track.email} onChange={(e) => setTrack({ ...track, email: e.target.value })} required />
            </label>
          </div>
          <button type="submit" className="btn btn--primary" style={{ alignSelf: 'flex-start' }}>
            Track order
          </button>
        </form>
        <a href="#returns" className="card help__tile">
          <span className="help__tile-title">Returns</span>
          <span className="muted" style={{ fontSize: 14 }}>
            Unopened items within {config.returnsDays} days.
          </span>
        </a>
        <button id="chat" type="button" className="card help__tile help__tile--chat" onClick={openChat}>
          <span className="help__tile-title">Live chat</span>
          <span className="muted" style={{ fontSize: 14 }}>
            Talk to the team now. We usually reply within minutes.
          </span>
        </button>
        <a href="#contact" className="card help__tile">
          <span className="help__tile-title">Contact us</span>
          <span className="muted" style={{ fontSize: 14 }}>
            Mon–Fri, 9am–5pm. Reply within 1 working day.
          </span>
        </a>
      </div>

      <section id="delivery" className="help__policy">
        <h2>Delivery</h2>
        <p>
          Orders placed before {config.dispatchCutoff} on working days are dispatched the same day. Standard UK delivery takes 2–3 working days; next-day and
          tracked express options are offered at checkout. Perfume is classed as a flammable liquid, so we use couriers approved to carry it; this is why
          some remote postcodes may take a day longer.
        </p>
      </section>

      <section id="returns" className="help__policy">
        <h2>Returns</h2>
        <p>
          You have {config.returnsDays} days from delivery to cancel and return unopened, sealed items for a full refund. For hygiene reasons, opened
          fragrances cannot be returned unless they are faulty or not as described. This does not affect your statutory rights.
        </p>
      </section>

      <section id="faq" className="help__faq">
        <h2>Frequently asked questions</h2>
        {faqs.isPending && <div className="skeleton" style={{ height: 240 }} />}
        {faqs.data && <Accordion size="lg" defaultOpen="0" items={faqs.data.map((f, i) => ({ id: String(i), title: f.q, body: <p>{f.a}</p> }))} />}
      </section>

      <section id="contact" className="help__contact">
        <div className="help__contact-info">
          <h2>Still need help?</h2>
          <span className="muted">Email: {config.brand.supportEmail}</span>
          <span className="muted">WhatsApp: [number]</span>
          <span className="muted">Mon–Fri, 9am–5pm</span>
          <span className="muted" style={{ fontSize: 13, paddingTop: 12 }}>
            Policies: <a href="#delivery">Delivery</a> · <a href="#returns">Returns</a> · <Link to="/legal/terms">Terms</Link> ·{' '}
            <Link to="/legal/privacy">Privacy</Link> · <Link to="/legal/cookies">Cookies</Link>
          </span>
        </div>
        {contact.isSuccess ? (
          <p className="status" role="status">
            Thanks — we've received your message and will reply within one working day.
          </p>
        ) : (
          <form className="help__form" onSubmit={submitContact}>
            <div className="help__two">
              <label className="field">
                Name
                <input className="input" required value={msg.name} onChange={(e) => setMsg({ ...msg, name: e.target.value })} />
              </label>
              <label className="field">
                Email
                <input className="input" type="email" required value={msg.email} onChange={(e) => setMsg({ ...msg, email: e.target.value })} />
              </label>
            </div>
            <label className="field">
              Order number (optional)
              <input className="input" value={msg.orderId} onChange={(e) => setMsg({ ...msg, orderId: e.target.value })} />
            </label>
            <label className="field">
              Message
              <textarea className="textarea" rows={5} required value={msg.message} onChange={(e) => setMsg({ ...msg, message: e.target.value })} />
            </label>
            {contact.isError && <p className="field__error">Could not send your message. Please email us directly.</p>}
            <button type="submit" className="btn btn--primary" style={{ alignSelf: 'flex-start', height: 48, padding: '0 28px' }} disabled={contact.isPending}>
              {contact.isPending ? 'Sending…' : 'Send message'}
            </button>
          </form>
        )}
      </section>
    </main>
  );
}
