package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/BrenoD/Celestial-Oud/backend/internal/httpx"
)

// ── Generic config-driven sections (printforge adminSections parity) ─────────
//
// Each section maps to a table and a whitelist of writable columns. Column
// names come only from this map, so building SQL from them is safe; values are
// always parameterised.

type SectionField struct {
	Name string // JSON key == column name
	Type string // text | int | numeric | bool | json
}

type Section struct {
	Table  string
	IDCol  string
	Fields []SectionField
	Order  string
}

var Sections = map[string]Section{
	"taxes":         {Table: "tax_rules", IDCol: "id", Order: "id", Fields: []SectionField{{"region", "text"}, {"rate", "numeric"}, {"status", "text"}}},
	"emails":        {Table: "email_templates", IDCol: "key", Order: "key", Fields: []SectionField{{"key", "text"}, {"name", "text"}, {"subject", "text"}, {"enabled", "bool"}}},
	"notifications": {Table: "admin_notifications", IDCol: "id", Order: "id", Fields: []SectionField{{"key", "text"}, {"name", "text"}, {"description", "text"}, {"enabled", "bool"}}},
	"integrations":  {Table: "integrations", IDCol: "id", Order: "id", Fields: []SectionField{{"key", "text"}, {"name", "text"}, {"description", "text"}, {"connected", "bool"}, {"config", "json"}}},
}

func (s *Store) section(key string) (Section, error) {
	sec, ok := Sections[key]
	if !ok {
		return sec, httpx.E(404, "unknown_section", "Unknown admin section.")
	}
	return sec, nil
}

func (sec Section) cols() string {
	names := []string{sec.IDCol}
	for _, f := range sec.Fields {
		if f.Name != sec.IDCol {
			names = append(names, f.Name)
		}
	}
	return strings.Join(names, ", ")
}

// rowsToMaps generically converts pgx rows to []map[string]any (JSON-ready).
func rowsToMaps(rows pgx.Rows) ([]map[string]any, error) {
	defer rows.Close()
	descs := rows.FieldDescriptions()
	out := []map[string]any{}
	for rows.Next() {
		vals, err := rows.Values()
		if err != nil {
			return nil, err
		}
		m := make(map[string]any, len(vals))
		for i, d := range descs {
			v := vals[i]
			if b, ok := v.([]byte); ok {
				var js any
				if json.Unmarshal(b, &js) == nil {
					v = js
				} else {
					v = string(b)
				}
			}
			m[string(d.Name)] = v
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) ListSection(ctx context.Context, key string) ([]map[string]any, error) {
	sec, err := s.section(key)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.Query(ctx, fmt.Sprintf(`select %s from %s order by %s`, sec.cols(), sec.Table, sec.Order))
	if err != nil {
		return nil, err
	}
	return rowsToMaps(rows)
}

func (sec Section) filter(payload map[string]any) (cols []string, vals []any, err error) {
	for _, f := range sec.Fields {
		v, ok := payload[f.Name]
		if !ok {
			continue
		}
		switch f.Type {
		case "json":
			b, _ := json.Marshal(v)
			v = b
		case "text":
			if str, ok := v.(string); ok && len(str) > 2000 {
				return nil, nil, httpx.BadRequest(fmt.Sprintf("Field %q is too long.", f.Name))
			}
		}
		cols = append(cols, f.Name)
		vals = append(vals, v)
	}
	if len(cols) == 0 {
		return nil, nil, httpx.BadRequest("No valid fields were provided.")
	}
	return cols, vals, nil
}

func (s *Store) CreateSectionItem(ctx context.Context, key string, payload map[string]any) (map[string]any, error) {
	sec, err := s.section(key)
	if err != nil {
		return nil, err
	}
	cols, vals, err := sec.filter(payload)
	if err != nil {
		return nil, err
	}
	ph := make([]string, len(cols))
	for i := range cols {
		ph[i] = fmt.Sprintf("$%d", i+1)
	}
	rows, err := s.db.Query(ctx, fmt.Sprintf(`insert into %s (%s) values (%s) returning %s`, sec.Table, strings.Join(cols, ","), strings.Join(ph, ","), sec.cols()), vals...)
	if err != nil {
		return nil, err
	}
	out, err := rowsToMaps(rows)
	if err != nil || len(out) == 0 {
		return nil, err
	}
	return out[0], nil
}

func (s *Store) UpdateSectionItem(ctx context.Context, key, id string, payload map[string]any) (map[string]any, error) {
	sec, err := s.section(key)
	if err != nil {
		return nil, err
	}
	delete(payload, sec.IDCol)
	cols, vals, err := sec.filter(payload)
	if err != nil {
		return nil, err
	}
	set := make([]string, len(cols))
	for i, c := range cols {
		set[i] = fmt.Sprintf("%s = $%d", c, i+2)
	}
	rows, err := s.db.Query(ctx, fmt.Sprintf(`update %s set %s where %s::text = $1 returning %s`, sec.Table, strings.Join(set, ", "), sec.IDCol, sec.cols()), append([]any{id}, vals...)...)
	if err != nil {
		return nil, err
	}
	out, err := rowsToMaps(rows)
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, httpx.ErrNotFound
	}
	return out[0], nil
}

func (s *Store) DeleteSectionItem(ctx context.Context, key, id string) error {
	sec, err := s.section(key)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(ctx, fmt.Sprintf(`delete from %s where %s::text = $1`, sec.Table, sec.IDCol), id)
	return err
}

// ── Settings ─────────────────────────────────────────────────────────────────

func (s *Store) Settings(ctx context.Context) (Settings, error) {
	var st Settings
	err := s.db.QueryRow(ctx, `select store_name, support_email, currency, free_delivery_threshold, dispatch_cutoff, returns_days, maintenance_mode,
		company_name, company_number, registered_address, vat_number from store_settings where id = 1`).
		Scan(&st.StoreName, &st.SupportEmail, &st.Currency, &st.FreeDeliveryThreshold, &st.DispatchCutoff, &st.ReturnsDays, &st.MaintenanceMode,
			&st.CompanyName, &st.CompanyNumber, &st.RegisteredAddress, &st.VATNumber)
	return st, err
}

func (s *Store) UpdateSettings(ctx context.Context, st Settings) (Settings, error) {
	_, err := s.db.Exec(ctx, `update store_settings set store_name=$1, support_email=$2, currency=$3, free_delivery_threshold=$4, dispatch_cutoff=$5,
		returns_days=$6, maintenance_mode=$7, company_name=$8, company_number=$9, registered_address=$10, vat_number=$11 where id = 1`,
		st.StoreName, st.SupportEmail, st.Currency, st.FreeDeliveryThreshold, st.DispatchCutoff, st.ReturnsDays, st.MaintenanceMode,
		st.CompanyName, st.CompanyNumber, st.RegisteredAddress, st.VATNumber)
	if err != nil {
		return st, err
	}
	return s.Settings(ctx)
}

// ── Customers & staff ────────────────────────────────────────────────────────

func (s *Store) ListCustomers(ctx context.Context, q string) ([]Customer, error) {
	rows, err := s.db.Query(ctx, `select `+userCols+`,
		(select count(*) from orders o where o.user_id = users.id and o.status not in ('cancelled','pending_payment')),
		coalesce((select sum(o.total) from orders o where o.user_id = users.id and o.status not in ('cancelled','pending_payment','refunded')), 0)
		from users where role = 'customer' and ($1 = '' or email ilike '%'||$1||'%' or first_name ilike '%'||$1||'%' or last_name ilike '%'||$1||'%')
		order by created_at desc limit 500`, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Customer{}
	for rows.Next() {
		var c Customer
		if err := rows.Scan(&c.ID, &c.Email, &c.PasswordHash, &c.FirstName, &c.LastName, &c.Phone, &c.Role, &c.Tier, &c.Points,
			&c.MarketingEmail, &c.MarketingSMS, &c.MarketingWhatsApp, &c.EmailVerifiedAt, &c.CreatedAt, &c.OrderCount, &c.TotalSpent); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) ListStaff(ctx context.Context) ([]User, error) {
	rows, err := s.db.Query(ctx, `select `+userCols+` from users where role <> 'customer' order by created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []User{}
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *u)
	}
	return out, rows.Err()
}

func (s *Store) SetRole(ctx context.Context, id, role string) (*User, error) {
	if !httpx.HasRole(role, "viewer") && role != "customer" {
		return nil, httpx.BadRequest("Unknown role.")
	}
	return scanUser(s.db.QueryRow(ctx, `update users set role = $2 where id = $1 returning `+userCols, id, role))
}

// InviteStaff creates (or promotes) a user with a staff role.
func (s *Store) InviteStaff(ctx context.Context, email, first, last, role, hash string) (*User, error) {
	if !httpx.HasRole(role, "viewer") {
		return nil, httpx.BadRequest("Unknown role.")
	}
	u, err := scanUser(s.db.QueryRow(ctx, `insert into users (email, first_name, last_name, role, password_hash) values ($1,$2,$3,$4,nullif($5,''))
		returning `+userCols, strings.TrimSpace(email), first, last, role, hash))
	if err != nil && strings.Contains(err.Error(), "users_email_norm_idx") {
		return nil, httpx.E(409, "email_taken", "This email already has an account.")
	}
	return u, err
}

func (s *Store) AdminCount(ctx context.Context) int {
	var n int
	_ = s.db.QueryRow(ctx, `select count(*) from users where role = 'admin'`).Scan(&n)
	return n
}

func (s *Store) Audit(ctx context.Context, actor *string, action, entity, entityID string, diff any) {
	_, _ = s.db.Exec(ctx, `insert into audit_log (actor_id, action, entity, entity_id, diff) values ($1,$2,$3,$4,$5)`, actor, action, entity, entityID, jsonb(diff))
}

func (s *Store) ListAudit(ctx context.Context, limit int) ([]map[string]any, error) {
	rows, err := s.db.Query(ctx, `select a.id, a.action, a.entity, a.entity_id, a.diff, a.created_at, u.email as actor
		from audit_log a left join users u on u.id = a.actor_id order by a.id desc limit $1`, limit)
	if err != nil {
		return nil, err
	}
	return rowsToMaps(rows)
}

// ── Analytics ────────────────────────────────────────────────────────────────

type Dashboard struct {
	TotalRevenue   int              `json:"totalRevenue"`
	TotalOrders    int              `json:"totalOrders"`
	TotalCustomers int              `json:"totalCustomers"`
	PendingOrders  int              `json:"pendingOrders"`
	PendingReviews int              `json:"pendingReviews"`
	OpenMessages   int              `json:"openMessages"`
	LowStock       []map[string]any `json:"lowStock"`
	RecentOrders   []Order          `json:"recentOrders"`
	RevenueByDay   []DayPoint       `json:"revenueByDay"`
	TopProducts    []map[string]any `json:"topProducts"`
}

type DayPoint struct {
	Day     string `json:"day"`
	Revenue int    `json:"revenue"`
	Orders  int    `json:"orders"`
}

func (s *Store) Dashboard(ctx context.Context) (Dashboard, error) {
	var d Dashboard
	const paid = `status not in ('pending_payment','cancelled','refunded')`
	err := s.db.QueryRow(ctx, `select
		coalesce((select sum(total) from orders where `+paid+`),0),
		(select count(*) from orders where `+paid+`),
		(select count(*) from users where role = 'customer'),
		(select count(*) from orders where status in ('paid','processing')),
		(select count(*) from product_reviews where not approved),
		(select count(*) from messages where status = 'open')`).
		Scan(&d.TotalRevenue, &d.TotalOrders, &d.TotalCustomers, &d.PendingOrders, &d.PendingReviews, &d.OpenMessages)
	if err != nil {
		return d, err
	}
	rows, err := s.db.Query(ctx, `select p.name, s.sku, s.label, s.stock from product_sizes s join products p on p.id = s.product_id
		where p.status = 'active' and not s.is_sample and s.stock <= 5 order by s.stock, p.name limit 20`)
	if err != nil {
		return d, err
	}
	d.LowStock, err = rowsToMaps(rows)
	if err != nil {
		return d, err
	}
	d.RecentOrders, err = s.queryOrders(ctx, `order by o.created_at desc limit 8`)
	if err != nil {
		return d, err
	}
	d.RevenueByDay, err = s.revenueByDay(ctx, 14)
	if err != nil {
		return d, err
	}
	rows, err = s.db.Query(ctx, `select i.name, sum(i.qty) as units, sum(i.qty * i.unit_price) as revenue from order_items i
		join orders o on o.id = i.order_id where o.`+paid+` and o.created_at > now() - interval '30 days'
		group by i.name order by units desc limit 5`)
	if err != nil {
		return d, err
	}
	d.TopProducts, err = rowsToMaps(rows)
	return d, err
}

func (s *Store) revenueByDay(ctx context.Context, days int) ([]DayPoint, error) {
	rows, err := s.db.Query(ctx, `select to_char(d, 'YYYY-MM-DD'), coalesce(sum(o.total),0), count(o.id)
		from generate_series(current_date - ($1::int - 1), current_date, '1 day') d
		left join orders o on o.created_at::date = d and o.status not in ('pending_payment','cancelled','refunded')
		group by d order by d`, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []DayPoint{}
	for rows.Next() {
		var p DayPoint
		if err := rows.Scan(&p.Day, &p.Revenue, &p.Orders); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

type Breakdown struct {
	PeriodDays   int        `json:"period"`
	Orders       int        `json:"orders"`
	Revenue      int        `json:"revenue"`
	NewCustomers int        `json:"newCustomers"`
	AvgTicket    int        `json:"ticket"`
	LTV          int        `json:"ltv"`
	RepeatRate   float64    `json:"repeatRate"`
	Series       []DayPoint `json:"series"`
}

func (s *Store) Breakdown(ctx context.Context, days int) (Breakdown, error) {
	b := Breakdown{PeriodDays: days}
	since := time.Now().AddDate(0, 0, -days)
	err := s.db.QueryRow(ctx, `with o as (select * from orders where created_at >= $1 and status not in ('pending_payment','cancelled','refunded'))
		select (select count(*) from o), coalesce((select sum(total) from o),0),
		(select count(*) from users where role = 'customer' and created_at >= $1),
		coalesce((select avg(total)::int from o),0),
		coalesce((select avg(t)::int from (select sum(total) t from o where user_id is not null group by user_id) x),0),
		coalesce((select count(*) filter (where c > 1)::float / nullif(count(*),0) from (select user_id, count(*) c from o where user_id is not null group by user_id) y),0)`, since).
		Scan(&b.Orders, &b.Revenue, &b.NewCustomers, &b.AvgTicket, &b.LTV, &b.RepeatRate)
	if err != nil {
		return b, err
	}
	b.Series, err = s.revenueByDay(ctx, days)
	return b, err
}

// ── Scheduled promotions ─────────────────────────────────────────────────────

func (s *Store) ListPromotions(ctx context.Context) ([]ScheduledPromotion, error) {
	rows, err := s.db.Query(ctx, `select sp.id, sp.product_id, p.name, sp.collection_id, sp.percent_off, sp.starts_at, sp.ends_at, sp.applied_at, sp.reverted_at
		from scheduled_promotions sp left join products p on p.id = sp.product_id order by sp.starts_at desc`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ScheduledPromotion{}
	for rows.Next() {
		var p ScheduledPromotion
		if err := rows.Scan(&p.ID, &p.ProductID, &p.ProductName, &p.CollectionID, &p.PercentOff, &p.StartsAt, &p.EndsAt, &p.AppliedAt, &p.RevertedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) CreatePromotion(ctx context.Context, p ScheduledPromotion) (ScheduledPromotion, error) {
	if p.ProductID == nil && p.CollectionID == nil {
		return p, httpx.BadRequest("Choose a product or a collection.")
	}
	if p.PercentOff < 1 || p.PercentOff > 90 {
		return p, httpx.BadRequest("Discount must be between 1% and 90%.")
	}
	if !p.EndsAt.After(p.StartsAt) {
		return p, httpx.BadRequest("End date must be after the start date.")
	}
	err := s.db.QueryRow(ctx, `insert into scheduled_promotions (product_id, collection_id, percent_off, starts_at, ends_at) values ($1,$2,$3,$4,$5) returning id`,
		p.ProductID, p.CollectionID, p.PercentOff, p.StartsAt, p.EndsAt).Scan(&p.ID)
	return p, err
}

func (s *Store) DeletePromotion(ctx context.Context, id int64) error {
	// Revert first if it is live.
	_, _ = s.revertPromotion(ctx, id)
	_, err := s.db.Exec(ctx, `delete from scheduled_promotions where id = $1`, id)
	return err
}

// ApplyDuePromotions is called by the cron tick.
func (s *Store) ApplyDuePromotions(ctx context.Context) (applied, reverted int, err error) {
	rows, err := s.db.Query(ctx, `select id from scheduled_promotions where applied_at is null and starts_at <= now() and ends_at > now()`)
	if err != nil {
		return 0, 0, err
	}
	var toApply []int64
	for rows.Next() {
		var id int64
		_ = rows.Scan(&id)
		toApply = append(toApply, id)
	}
	rows.Close()
	for _, id := range toApply {
		if e := s.applyPromotion(ctx, id); e == nil {
			applied++
		} else {
			err = e
		}
	}
	rows, err2 := s.db.Query(ctx, `select id from scheduled_promotions where applied_at is not null and reverted_at is null and ends_at <= now()`)
	if err2 != nil {
		return applied, 0, err2
	}
	var toRevert []int64
	for rows.Next() {
		var id int64
		_ = rows.Scan(&id)
		toRevert = append(toRevert, id)
	}
	rows.Close()
	for _, id := range toRevert {
		if ok, e := s.revertPromotion(ctx, id); ok && e == nil {
			reverted++
		}
	}
	return applied, reverted, err
}

func (s *Store) applyPromotion(ctx context.Context, id int64) error {
	return s.Tx(ctx, func(tx pgx.Tx) error {
		// Sizes without an existing compare_at_price get it set to the current price, then price is reduced.
		_, err := tx.Exec(ctx, `update product_sizes s set compare_at_price = coalesce(s.compare_at_price, s.price),
			price = round(coalesce(s.compare_at_price, s.price) * (100 - sp.percent_off) / 100.0)
			from scheduled_promotions sp where sp.id = $1 and not s.is_sample and
			(s.product_id = sp.product_id or s.product_id in (select product_id from product_collections where collection_id = sp.collection_id))`, id)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `update scheduled_promotions set applied_at = now() where id = $1`, id)
		return err
	})
}

func (s *Store) revertPromotion(ctx context.Context, id int64) (bool, error) {
	var applied bool
	if err := s.db.QueryRow(ctx, `select applied_at is not null and reverted_at is null from scheduled_promotions where id = $1`, id).Scan(&applied); err != nil || !applied {
		return false, nil
	}
	return true, s.Tx(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `update product_sizes s set price = s.compare_at_price, compare_at_price = null
			from scheduled_promotions sp where sp.id = $1 and s.compare_at_price is not null and
			(s.product_id = sp.product_id or s.product_id in (select product_id from product_collections where collection_id = sp.collection_id))`, id)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `update scheduled_promotions set reverted_at = now() where id = $1`, id)
		return err
	})
}

// ── Housekeeping (cron) ──────────────────────────────────────────────────────

func (s *Store) Purge(ctx context.Context) {
	_, _ = s.db.Exec(ctx, `delete from idempotency_keys where created_at < now() - interval '24 hours'`)
	_, _ = s.db.Exec(ctx, `delete from refresh_tokens where expires_at < now() or revoked_at < now() - interval '7 days'`)
	_, _ = s.db.Exec(ctx, `delete from password_resets where expires_at < now()`)
	_, _ = s.db.Exec(ctx, `update store_credits set remaining = 0 where expires_at < now() and remaining > 0`)
}
