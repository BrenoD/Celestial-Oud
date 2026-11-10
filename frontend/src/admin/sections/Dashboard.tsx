import { Link } from 'react-router-dom';
import { formatPence } from '@/lib/money';
import { adminApi } from '../api';
import { BarChart, LoadState, Metrics, Pill, fmtDateTime, useLoad } from '../components/ui';

export default function Dashboard() {
  const { data, error, loading } = useLoad(adminApi.dashboard, []);
  if (!data) return <LoadState loading={loading} error={error} />;
  return (
    <>
      <Metrics
        items={[
          [formatPence(data.totalRevenue), 'Revenue (paid)'],
          [String(data.totalOrders), 'Orders'],
          [String(data.totalCustomers), 'Customers'],
          [String(data.pendingOrders), 'To dispatch'],
          [String(data.pendingReviews), 'Reviews to moderate'],
          [String(data.openMessages), 'Open messages'],
        ]}
      />
      <BarChart series={data.revenueByDay} title="Revenue — last 14 days" />
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(320px, 1fr))', gap: 16 }}>
        <div className="adm-table">
          <table style={{ minWidth: 0 }}>
            <thead>
              <tr>
                <th>Recent orders</th>
                <th>Status</th>
                <th>Total</th>
              </tr>
            </thead>
            <tbody>
              {data.recentOrders.map((o) => (
                <tr key={o.id}>
                  <td>
                    <Link to={`/admin/orders?q=${o.id}`}>{o.id}</Link>
                    <span className="sub">{o.customerName} · {fmtDateTime(o.createdAt)}</span>
                  </td>
                  <td>
                    <Pill value={o.status} />
                  </td>
                  <td>{formatPence(o.total)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
          <div className="adm-table">
            <table style={{ minWidth: 0 }}>
              <thead>
                <tr>
                  <th>Low stock (≤ 5)</th>
                  <th>Units</th>
                </tr>
              </thead>
              <tbody>
                {data.lowStock.length === 0 && (
                  <tr>
                    <td colSpan={2} className="form-hint">
                      All sizes above the threshold.
                    </td>
                  </tr>
                )}
                {data.lowStock.map((s) => (
                  <tr key={s.sku}>
                    <td>
                      {s.name} <span className="sub">{s.label} · {s.sku}</span>
                    </td>
                    <td>
                      <Pill value={s.stock === 0 ? 'cancelled' : 'processing'} label={String(s.stock)} />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <div className="adm-table">
            <table style={{ minWidth: 0 }}>
              <thead>
                <tr>
                  <th>Top products (30 days)</th>
                  <th>Units</th>
                  <th>Revenue</th>
                </tr>
              </thead>
              <tbody>
                {data.topProducts.map((p) => (
                  <tr key={p.name}>
                    <td>{p.name}</td>
                    <td>{p.units}</td>
                    <td>{formatPence(p.revenue)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </>
  );
}
