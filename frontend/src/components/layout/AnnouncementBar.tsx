import { config } from '@/config';
import { formatPenceShort } from '@/lib/money';
import './AnnouncementBar.css';

export function AnnouncementBar() {
  return (
    <div className="announce" role="region" aria-label="Store offers">
      <span>Free UK delivery over {formatPenceShort(config.freeDeliveryThreshold)}</span>
      <span>Order by {config.dispatchCutoff} — dispatched today</span>
      <span>Free sample with every order</span>
    </div>
  );
}
