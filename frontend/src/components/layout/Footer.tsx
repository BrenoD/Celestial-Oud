import { Link } from 'react-router-dom';
import { openCookieSettings } from '@/lib/consent';
import './Footer.css';

export function Footer() {
  return (
    <footer className="ftr">
      <div className="container ftr__grid">
        <div className="ftr__brand">
          <span className="ftr__wordmark">CELESTIAL OUD</span>
          <span className="muted">Oud, amber and musk fragrances.</span>
        </div>
        <nav aria-label="Shop" className="ftr__col">
          <span className="ftr__head">Shop</span>
          <Link to="/collections/all">All fragrances</Link>
          <Link to="/collections/bestsellers">Bestsellers</Link>
          <Link to="/collections/discovery-sets">Discovery sets</Link>
          <Link to="/collections/gifts">Gift sets</Link>
        </nav>
        <nav aria-label="Help" className="ftr__col">
          <span className="ftr__head">Help</span>
          <Link to="/help#track">Track my order</Link>
          <Link to="/help#delivery">Delivery</Link>
          <Link to="/help#returns">Returns</Link>
          <Link to="/help#faq">FAQs</Link>
          <Link to="/help#contact">Contact us</Link>
        </nav>
        <nav aria-label="About" className="ftr__col">
          <span className="ftr__head">About</span>
          <Link to="/about">Our story</Link>
          <Link to="/about#oud">What is oud?</Link>
          <Link to="/about#guide">Fragrance guide</Link>
          <Link to="/help#contact">Wholesale</Link>
        </nav>
      </div>
      <div className="container ftr__legal">
        <span>© Celestial Oud [Ltd] · Company no. [XXXXXXXX] · [Registered address]</span>
        <span className="ftr__links">
          <Link to="/legal/terms">Terms</Link>
          <Link to="/legal/privacy">Privacy</Link>
          <Link to="/legal/cookies">Cookies</Link>
          <button type="button" className="ftr__linkbtn" onClick={openCookieSettings}>
            Cookie settings
          </button>
          <Link to="/legal/product-safety">Product safety</Link>
          <Link to="/legal/accessibility">Accessibility</Link>
        </span>
        <span>Secure card payments · Visa · Mastercard · Amex</span>
      </div>
    </footer>
  );
}
