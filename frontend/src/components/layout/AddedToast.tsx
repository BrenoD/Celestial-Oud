import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { useBasketStore } from '@/store/basket';
import './AddedToast.css';

export function AddedToast() {
  const lastAdded = useBasketStore((s) => s.lastAdded);
  const [visible, setVisible] = useState(false);

  useEffect(() => {
    if (!lastAdded) return;
    setVisible(true);
    const t = setTimeout(() => setVisible(false), 3500);
    return () => clearTimeout(t);
  }, [lastAdded]);

  if (!visible || !lastAdded) return null;
  return (
    <div className="toast" role="status">
      <span className="toast__tick" aria-hidden="true">✓</span>
      <span className="toast__text">
        <b>{lastAdded.name}</b> added to your basket.
      </span>
      <Link to="/basket" className="btn btn--secondary btn--sm">
        View basket
      </Link>
      <span className="toast__timer" aria-hidden="true" />
    </div>
  );
}
