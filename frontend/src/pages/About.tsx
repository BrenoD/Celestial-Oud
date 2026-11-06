import { Link } from 'react-router-dom';
import { Seo } from '@/components/ui/Seo';
import { assetUrl } from '@/config';
import './About.css';

export default function About() {
  return (
    <main>
      <Seo title="Our story" />
      <section className="container about-hero">
        <nav aria-label="Breadcrumb" className="breadcrumb">
          <Link to="/">Home</Link> › <span>Our story</span>
        </nav>
        <div className="about-hero__grid">
          <div className="about-hero__copy">
            <span className="eyebrow" style={{ fontSize: 13, letterSpacing: '0.18em' }}>
              Our story
            </span>
            <h1 className="about-hero__title">[Headline about why Celestial Oud exists]</h1>
            <p className="muted" style={{ fontSize: 17, lineHeight: 1.6 }}>
              [Two or three sentences: who founded it, the inspiration behind the name, how the perfumes are made.]
            </p>
          </div>
          <div className="about-hero__media">
            <img src={assetUrl('/images/oud_brass.jpg')} alt="Oud wood" width={572} height={517} />
          </div>
        </div>
      </section>

      <section className="container about-cards">
        <article id="oud" className="card about-card">
          <img src={assetUrl('/images/wood.jpg')} alt="" loading="lazy" width={300} height={200} />
          <div>
            <span className="about-card__title">What is oud?</span>
            <span className="muted">Oud comes from agarwood — a dark, resinous wood prized for centuries across the Middle East and Asia.</span>
          </div>
        </article>
        <article id="guide" className="card about-card">
          <img src={assetUrl('/images/amber.jpg')} alt="" loading="lazy" width={300} height={200} />
          <div>
            <span className="about-card__title">Made to last</span>
            <span className="muted">[How the fragrances are blended and why they last.]</span>
          </div>
        </article>
        <article className="card about-card">
          <img src={assetUrl('/images/box.jpg')} alt="" loading="lazy" width={300} height={200} />
          <div>
            <span className="about-card__title">Luxury, fairly priced</span>
            <span className="muted">[Why the prices are what they are — no designer mark-up.]</span>
          </div>
        </article>
      </section>

      <section className="about-cta">
        <div className="container about-cta__inner">
          <span className="about-cta__title">Ready to find your scent?</span>
          <span style={{ display: 'flex', gap: 12, flexWrap: 'wrap' }}>
            <Link to="/collections/all" className="btn btn--primary btn--lg">
              Shop all
            </Link>
            <Link to="/products/discovery-set" className="btn btn--secondary btn--lg">
              Try samples
            </Link>
          </span>
        </div>
      </section>
    </main>
  );
}
