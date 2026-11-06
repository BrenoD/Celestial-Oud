import { Link, useParams } from 'react-router-dom';
import { Seo } from '@/components/ui/Seo';
import { config } from '@/config';
import NotFound from './NotFound';

const PAGES: Record<string, { title: string; body: string[] }> = {
  terms: {
    title: 'Terms & Conditions',
    body: [
      '[Company details: Celestial Oud Ltd, company number, registered address, VAT number.]',
      `Consumer Contracts Regulations 2013: you may cancel your order within ${config.returnsDays} days of delivery. Sealed cosmetic products that have been unsealed after delivery are excluded from the right to cancel for health and hygiene reasons; this is stated before purchase on every product page and at checkout.`,
      '[Pricing, payment, delivery, faulty goods, liability, governing law (England and Wales).]',
    ],
  },
  privacy: {
    title: 'Privacy Policy',
    body: [
      '[Data controller, what we collect (contact, delivery, order and payment-token data), why, lawful basis, retention.]',
      'Under UK GDPR you can request a copy of your data, correct it, or ask us to delete it from the Privacy & data section of your account or by emailing ' + config.brand.supportEmail + '.',
      '[Processors: payment provider, courier, email platform, analytics. International transfers. ICO complaint route.]',
    ],
  },
  cookies: {
    title: 'Cookie Policy',
    body: [
      'Essential cookies keep your basket and sign-in working; they cannot be switched off. Analytics cookies are only set if you choose “Accept all” in the cookie banner, and you can change your mind at any time by clearing site data.',
      '[Table of cookies: name, purpose, duration, provider.]',
    ],
  },
  accessibility: {
    title: 'Accessibility Statement',
    body: [
      'We aim to meet WCAG 2.2 AA. Pages work with keyboard only, text can be resized to 200%, and all interactive controls have visible focus and accessible names.',
      '[Known issues and contact route for accessibility problems.]',
    ],
  },
  'product-safety': {
    title: 'Product Safety & Ingredients',
    body: [
      'All Celestial Oud fragrances are cosmetics under the UK Cosmetics Regulation. Each product has a Responsible Person in the UK, a Cosmetic Product Safety Report and an SCPN notification. Full INCI ingredient lists and declared allergens are shown on every product page.',
      'Perfume is flammable: keep away from heat and flame, and out of reach of children.',
    ],
  },
};

export default function Legal() {
  const { page = '' } = useParams();
  const data = PAGES[page];
  if (!data) return <NotFound />;
  return (
    <main className="container container--narrow page" style={{ maxWidth: 820 }}>
      <Seo title={data.title} />
      <nav aria-label="Breadcrumb" className="breadcrumb">
        <Link to="/">Home</Link> › <span>{data.title}</span>
      </nav>
      <h1 className="page-title">{data.title}</h1>
      <p className="page-intro">Last updated: [date]</p>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 16, marginTop: 24, fontSize: 15, lineHeight: 1.7, color: 'var(--muted)' }}>
        {data.body.map((p, i) => (
          <p key={i}>{p}</p>
        ))}
      </div>
      <nav aria-label="Legal pages" style={{ display: 'flex', flexWrap: 'wrap', gap: 16, marginTop: 40, fontSize: 14 }}>
        {Object.entries(PAGES).map(([slug, p]) => (
          <Link key={slug} to={`/legal/${slug}`} aria-current={slug === page ? 'page' : undefined}>
            {p.title}
          </Link>
        ))}
      </nav>
    </main>
  );
}
