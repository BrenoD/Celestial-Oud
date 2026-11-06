import { Link } from 'react-router-dom';
import { Seo } from '@/components/ui/Seo';

export default function NotFound() {
  return (
    <main className="container page" style={{ textAlign: 'center', paddingTop: 80, paddingBottom: 96 }}>
      <Seo title="Page not found" />
      <span className="eyebrow">404</span>
      <h1 className="page-title" style={{ marginTop: 12 }}>
        We can't find that page
      </h1>
      <p className="page-intro">It may have moved, or the link may be out of date.</p>
      <div style={{ display: 'flex', gap: 12, justifyContent: 'center', flexWrap: 'wrap', marginTop: 28 }}>
        <Link to="/" className="btn btn--primary">
          Back to home
        </Link>
        <Link to="/collections/bestsellers" className="btn btn--secondary">
          Shop bestsellers
        </Link>
      </div>
    </main>
  );
}
