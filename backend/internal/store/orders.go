package store

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/BrenoD/Celestial-Oud/backend/internal/httpx"
)

func nowUTC() time.Time { return time.Now().UTC() }

// PricedOrder is the server-side total; the client's numbers are never trusted.
type PricedOrder struct {
	Lines        []OrderLine
	Subtotal     int
	Discount     int
	Delivery     int
	Total        int
	DiscountCode *string
	Method       DeliveryMethod
	Sample       *OrderLine // free 2ml sample, priced at 0
}

// Order size caps: stop one anonymous order reserving the whole stock.
const (
	maxLines        = 20
	maxQtyPerSKU    = 10
	maxUnitsInOrder = 20
)

// PriceOrder resolves SKUs, checks stock and computes totals.
func (s *Store) PriceOrder(ctx context.Context, in CreateOrderInput, freeThreshold int) (*PricedOrder, error) {
	if len(in.Lines) == 0 {
		return nil, httpx.Unprocessable("empty_basket", "Your basket is empty.")
	}
	if len(in.Lines) > maxLines {
		return nil, httpx.Unprocessable("too_many_lines", "Your basket has too many lines. Please contact us for large orders.")
	}
	skus := make([]string, 0, len(in.Lines))
	qty := map[string]int{}
	units := 0
	for _, l := range in.Lines {
		if l.Qty < 1 || l.Qty > maxQtyPerSKU || len(l.SKU) > 80 {
			return nil, httpx.Unprocessable("bad_qty", "Quantities must be between 1 and 10.")
		}
		if _, dup := qty[l.SKU]; !dup {
			skus = append(skus, l.SKU)
		}
		qty[l.SKU] += l.Qty
		units += l.Qty
		// Limits apply to the total per item, so repeating a line cannot get round them.
		if qty[l.SKU] > maxQtyPerSKU {
			return nil, httpx.Unprocessable("bad_qty", "You can buy up to 10 of each item per order.")
		}
	}
	if units > maxUnitsInOrder {
		return nil, httpx.Unprocessable("too_many_units", "You can buy up to 20 items per order. Please contact us for large orders.")
	}
	rows, err := s.db.Query(ctx, `select s.sku, s.id, s.price, s.stock, s.label, s.is_sample, p.id, p.slug, p.name, p.images, p.tags
		from product_sizes s join products p on p.id = s.product_id where s.sku = any($1) and p.status = 'active'`, skus)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	found := map[string]OrderLine{}
	stock := map[string]int{}
	for rows.Next() {
		var l OrderLine
		var sizeID, productID int64
		var stockN int
		var isSample bool
		var imagesJSON []byte
		var tags []string
		if err := rows.Scan(&l.SKU, &sizeID, &l.UnitPrice, &stockN, &l.VariantLabel, &isSample, &productID, &l.Slug, &l.Name, &imagesJSON, &tags); err != nil {
			return nil, err
		}
		var imgs []string
		_ = json.Unmarshal(imagesJSON, &imgs)
		if len(imgs) > 0 {
			l.Image = &imgs[0]
		}
		isSet := false
		for _, t := range tags {
			if t == "set" || t == "gift" {
				isSet = true
			}
		}
		if !isSet && !isSample {
			l.VariantLabel = "Eau de Parfum · " + l.VariantLabel
		}
		l.ProductID = &productID
		l.SizeID = &sizeID
		found[l.SKU] = l
		stock[l.SKU] = stockN
	}
	po := &PricedOrder{}
	for _, sku := range skus {
		l, ok := found[sku]
		if !ok {
			return nil, httpx.Unprocessable("unknown_sku", fmt.Sprintf("%s is no longer available.", sku))
		}
		if stock[sku] < qty[sku] {
			return nil, httpx.Unprocessable("out_of_stock", fmt.Sprintf("%s (%s) is out of stock.", l.Name, l.VariantLabel))
		}
		l.Qty = qty[sku]
		po.Lines = append(po.Lines, l)
		po.Subtotal += l.UnitPrice * l.Qty
	}
	if in.FreeSampleSKU != nil && strings.TrimSpace(*in.FreeSampleSKU) != "" {
		sample, err := s.sampleLine(ctx, strings.TrimSpace(*in.FreeSampleSKU), qty)
		if err != nil {
			return nil, err
		}
		po.Sample = sample
	}
	freeDelivery := false
	if in.DiscountCode != nil && strings.TrimSpace(*in.DiscountCode) != "" {
		d, amount, err := s.ValidateDiscount(ctx, *in.DiscountCode, po.Subtotal)
		if err != nil {
			return nil, err
		}
		po.Discount = amount
		po.DiscountCode = &d.Code
		freeDelivery = d.Type == "free_delivery"
	}
	m, err := s.DeliveryMethod(ctx, in.DeliveryMethodID)
	if err != nil {
		return nil, err
	}
	po.Method = m
	po.Delivery = m.Price
	if freeDelivery || (m.FreeOver != nil && po.Subtotal-po.Discount >= *m.FreeOver) {
		po.Delivery = 0
	}
	po.Total = po.Subtotal - po.Discount + po.Delivery
	if po.Total < 0 {
		po.Total = 0
	}
	return po, nil
}

// sampleLine resolves the free sample: it must be an active 2ml sample SKU
// with stock left after the paid lines.
func (s *Store) sampleLine(ctx context.Context, sku string, paid map[string]int) (*OrderLine, error) {
	var l OrderLine
	var sizeID, productID int64
	var stock int
	err := s.db.QueryRow(ctx, `select s.sku, s.id, s.stock, s.label, p.id, p.slug, p.name
		from product_sizes s join products p on p.id = s.product_id
		where s.sku = $1 and s.is_sample and p.status = 'active'`, sku).Scan(&l.SKU, &sizeID, &stock, &l.VariantLabel, &productID, &l.Slug, &l.Name)
	if err != nil {
		return nil, httpx.Unprocessable("bad_sample", "That free sample is not available. Please choose another.")
	}
	if stock < paid[sku]+1 {
		return nil, httpx.Unprocessable("sample_out_of_stock", "That free sample is out of stock. Please choose another.")
	}
	l.VariantLabel = "Free sample · " + l.VariantLabel
	l.SizeID, l.ProductID, l.Qty, l.UnitPrice = &sizeID, &productID, 1, 0
	return &l, nil
}

func newOrderID() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(900000))
	return fmt.Sprintf("CO-%06d", n.Int64()+100000)
}

// CreateOrder inserts the order + lines and reserves stock. Status is
// pending_payment; MarkPaid is called by the Stripe webhook (or immediately in
// dev when Stripe is not configured).
func (s *Store) CreateOrder(ctx context.Context, in CreateOrderInput, po *PricedOrder, userID *string, idem, accessHash string, eta time.Time) (*Order, error) {
	var id string
	email := strings.ToLower(strings.TrimSpace(in.Email))
	var sampleSKU *string
	lines := po.Lines
	if po.Sample != nil {
		sampleSKU = &po.Sample.SKU
		lines = append(append([]OrderLine{}, po.Lines...), *po.Sample)
	}
	err := s.Tx(ctx, func(tx pgx.Tx) error {
		if po.DiscountCode != nil {
			if err := redeemDiscount(ctx, tx, *po.DiscountCode, email, userID); err != nil {
				return err
			}
		}
		for attempt := 0; attempt < 5; attempt++ {
			id = newOrderID()
			var exists bool
			if err := tx.QueryRow(ctx, `select exists(select 1 from orders where id = $1)`, id).Scan(&exists); err != nil {
				return err
			}
			if !exists {
				break
			}
		}
		_, err := tx.Exec(ctx, `insert into orders (id, user_id, email, status, subtotal, discount, delivery, total, discount_code, delivery_method_id,
			shipping_address, free_sample_sku, gift_wrap, gift_message, marketing_opt_in, notes, idempotency_key, estimated_delivery, access_token_hash)
			values ($1,$2,$3,'pending_payment',$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)`,
			id, userID, email, po.Subtotal, po.Discount, po.Delivery, po.Total, po.DiscountCode, po.Method.ID,
			jsonb(in.ShippingAddress), sampleSKU, in.GiftWrap, in.GiftMessage, in.MarketingOptIn, in.Notes, strPtr(idem), eta, accessHash)
		if err != nil {
			return err
		}
		for _, l := range lines {
			if _, err := tx.Exec(ctx, `insert into order_items (order_id, product_id, size_id, sku, name, variant_label, image, unit_price, qty)
				values ($1,$2,$3,$4,$5,$6,$7,$8,$9)`, id, l.ProductID, l.SizeID, l.SKU, l.Name, l.VariantLabel, l.Image, l.UnitPrice, l.Qty); err != nil {
				return err
			}
			tag, err := tx.Exec(ctx, `update product_sizes set stock = stock - $2 where sku = $1 and stock >= $2`, l.SKU, l.Qty)
			if err != nil {
				return err
			}
			if tag.RowsAffected() == 0 {
				return httpx.Unprocessable("out_of_stock", fmt.Sprintf("%s is out of stock.", l.Name))
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.OrderByID(ctx, id)
}

// redeemDiscount counts a use atomically (no overselling under concurrency)
// and enforces the per-customer limit, serialised per code+customer.
func redeemDiscount(ctx context.Context, tx pgx.Tx, code, email string, userID *string) error {
	var perCustomer *int
	err := tx.QueryRow(ctx, `update discount_codes set uses = uses + 1
		where code = $1 and active and (max_uses is null or uses < max_uses)
		  and (starts_at is null or starts_at <= now()) and (ends_at is null or ends_at > now())
		returning per_customer_limit`, code).Scan(&perCustomer)
	if err != nil {
		return httpx.Unprocessable("discount_exhausted", "That code is no longer available.")
	}
	if perCustomer == nil {
		return nil
	}
	if _, err := tx.Exec(ctx, `select pg_advisory_xact_lock(hashtext($1))`, code+"|"+email); err != nil {
		return err
	}
	var used int
	if err := tx.QueryRow(ctx, `select count(*) from orders where discount_code = $1 and status <> 'cancelled'
		and (lower(email) = $2 or ($3::uuid is not null and user_id = $3))`, code, email, userID).Scan(&used); err != nil {
		return err
	}
	if used >= *perCustomer {
		return httpx.Unprocessable("discount_used", "You have already used this code.")
	}
	return nil
}

// releaseDiscount gives back a use when an unpaid order is cancelled or expires.
func releaseDiscount(ctx context.Context, tx pgx.Tx, code *string) error {
	if code == nil {
		return nil
	}
	_, err := tx.Exec(ctx, `update discount_codes set uses = greatest(uses - 1, 0) where code = $1`, *code)
	return err
}

// OrderAccessHash returns the stored hash of the guest access token.
func (s *Store) OrderAccessHash(ctx context.Context, id string) string {
	var h *string
	_ = s.db.QueryRow(ctx, `select access_token_hash from orders where id = $1`, strings.ToUpper(id)).Scan(&h)
	return deref(h)
}

const orderCols = `o.id, o.user_id, o.email, o.status, o.payment_status, o.subtotal, o.discount, o.delivery, o.total, o.currency, o.discount_code,
	o.delivery_method_id, coalesce(d.name,''), o.shipping_address, o.free_sample_sku, o.gift_wrap, o.gift_message, o.notes,
	o.tracking_carrier, o.tracking_number, o.tracking_url, o.stripe_payment_intent, o.estimated_delivery, o.created_at, o.paid_at, o.dispatched_at, o.delivered_at,
	coalesce((select json_agg(json_build_object('sku',i.sku,'productId',i.product_id,'sizeId',i.size_id,'slug',p.slug,'name',i.name,
	  'variantLabel',i.variant_label,'image',i.image,'unitPrice',i.unit_price,'qty',i.qty) order by i.id)
	  from order_items i left join products p on p.id = i.product_id where i.order_id = o.id), '[]'::json)`

const orderFrom = ` from orders o left join delivery_methods d on d.id = o.delivery_method_id`

func scanOrder(row interface{ Scan(...any) error }) (*Order, error) {
	var o Order
	var addrJSON, linesJSON []byte
	var carrier, number, url *string
	var eta *time.Time
	err := row.Scan(&o.ID, &o.UserID, &o.Email, &o.Status, &o.PaymentStatus, &o.Subtotal, &o.Discount, &o.Delivery, &o.Total, &o.Currency, &o.DiscountCode,
		&o.DeliveryMethodID, &o.DeliveryMethodName, &addrJSON, &o.FreeSample, &o.GiftWrap, &o.GiftMessage, &o.Notes,
		&carrier, &number, &url, &o.StripePaymentIntent, &eta, &o.CreatedAt, &o.PaidAt, &o.DispatchedAt, &o.DeliveredAt, &linesJSON)
	if err != nil {
		return nil, notFound(err)
	}
	_ = json.Unmarshal(addrJSON, &o.ShippingAddress)
	_ = json.Unmarshal(linesJSON, &o.Lines)
	if o.Lines == nil {
		o.Lines = []OrderLine{}
	}
	o.CustomerName = strings.TrimSpace(o.ShippingAddress.FirstName + " " + o.ShippingAddress.LastName)
	if carrier != nil || number != nil {
		o.Tracking = &Tracking{Carrier: deref(carrier), Number: deref(number), URL: url}
	}
	if eta != nil {
		o.EstimatedDelivery = eta.Format("2006-01-02")
	}
	return &o, nil
}

func (s *Store) OrderByID(ctx context.Context, id string) (*Order, error) {
	return scanOrder(s.db.QueryRow(ctx, `select `+orderCols+orderFrom+` where o.id = $1`, strings.ToUpper(id)))
}

func (s *Store) OrderByIntent(ctx context.Context, intent string) (*Order, error) {
	return scanOrder(s.db.QueryRow(ctx, `select `+orderCols+orderFrom+` where o.stripe_payment_intent = $1`, intent))
}

func (s *Store) SetOrderIntent(ctx context.Context, id, intent string) error {
	_, err := s.db.Exec(ctx, `update orders set stripe_payment_intent = $2 where id = $1`, id, intent)
	return err
}

func (s *Store) ListOrdersForUser(ctx context.Context, userID string) ([]Order, error) {
	return s.queryOrders(ctx, `where o.user_id = $1 order by o.created_at desc`, userID)
}

type OrderFilter struct {
	Status string
	Q      string
	Limit  int
	Offset int
}

func (s *Store) ListOrders(ctx context.Context, f OrderFilter) ([]Order, int, error) {
	var where []string
	var args []any
	if f.Status != "" {
		args = append(args, strings.Split(f.Status, ","))
		where = append(where, fmt.Sprintf(`o.status = any($%d)`, len(args)))
	}
	if f.Q != "" {
		args = append(args, "%"+f.Q+"%")
		where = append(where, fmt.Sprintf(`(o.id ilike $%d or o.email ilike $%d or o.shipping_address::text ilike $%d)`, len(args), len(args), len(args)))
	}
	w := ""
	if len(where) > 0 {
		w = "where " + strings.Join(where, " and ")
	}
	var total int
	if err := s.db.QueryRow(ctx, `select count(*) from orders o `+w, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	if f.Limit <= 0 || f.Limit > 200 {
		f.Limit = 50
	}
	args = append(args, f.Limit, f.Offset)
	orders, err := s.queryOrders(ctx, w+fmt.Sprintf(` order by o.created_at desc limit $%d offset $%d`, len(args)-1, len(args)), args...)
	return orders, total, err
}

func (s *Store) queryOrders(ctx context.Context, tail string, args ...any) ([]Order, error) {
	rows, err := s.db.Query(ctx, `select `+orderCols+orderFrom+` `+tail, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Order{}
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *o)
	}
	return out, rows.Err()
}

// MarkPaid is idempotent: a replayed webhook is a no-op.
func (s *Store) MarkPaid(ctx context.Context, id string) (*Order, bool, error) {
	tag, err := s.db.Exec(ctx, `update orders set status = 'paid', payment_status = 'paid', paid_at = now() where id = $1 and status = 'pending_payment'`, id)
	if err != nil {
		return nil, false, err
	}
	o, err := s.OrderByID(ctx, id)
	if err != nil {
		return nil, false, err
	}
	if tag.RowsAffected() > 0 && o.UserID != nil {
		// Loyalty: 1 point per £1.
		_, _ = s.db.Exec(ctx, `update users set points = points + $2, tier = case when points + $2 >= 100000 then 'Gold' when points + $2 >= 30000 then 'Silver' else 'Bronze' end where id = $1`,
			*o.UserID, o.Total/100)
	}
	return o, tag.RowsAffected() > 0, nil
}

var orderTransitions = map[string][]string{
	"pending_payment": {"paid", "cancelled"},
	"paid":            {"processing", "dispatched", "cancelled", "refunded"},
	"processing":      {"dispatched", "cancelled", "refunded"},
	"dispatched":      {"delivered", "refunded"},
	"delivered":       {"refunded"},
}

// UpdateOrderStatus enforces the state machine and restocks on cancel.
func (s *Store) UpdateOrderStatus(ctx context.Context, id, status string) (*Order, error) {
	cur, err := s.OrderByID(ctx, id)
	if err != nil {
		return nil, err
	}
	allowed := false
	for _, n := range orderTransitions[cur.Status] {
		if n == status {
			allowed = true
		}
	}
	if !allowed {
		return nil, httpx.Unprocessable("bad_transition", fmt.Sprintf("An order that is %s cannot become %s.", cur.Status, status))
	}
	err = s.Tx(ctx, func(tx pgx.Tx) error {
		set := `status = $2`
		switch status {
		case "dispatched":
			set += `, dispatched_at = now()`
		case "delivered":
			set += `, delivered_at = now()`
		case "refunded":
			set += `, payment_status = 'refunded'`
		case "cancelled":
			if cur.Status == "pending_payment" {
				set += `, payment_status = 'cancelled'`
			}
		}
		if _, err := tx.Exec(ctx, `update orders set `+set+` where id = $1`, id, status); err != nil {
			return err
		}
		if status == "cancelled" && cur.Status == "pending_payment" {
			if err := releaseDiscount(ctx, tx, cur.DiscountCode); err != nil {
				return err
			}
		}
		if status == "cancelled" || status == "refunded" {
			for _, l := range cur.Lines {
				if _, err := tx.Exec(ctx, `update product_sizes set stock = stock + $2 where sku = $1`, l.SKU, l.Qty); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.OrderByID(ctx, id)
}

func (s *Store) SetTracking(ctx context.Context, id, carrier, number, url string) (*Order, error) {
	if strings.TrimSpace(number) == "" || len(number) > 100 {
		return nil, httpx.BadRequest("Invalid tracking code.")
	}
	_, err := s.db.Exec(ctx, `update orders set tracking_carrier = $2, tracking_number = $3, tracking_url = $4,
		status = case when status in ('paid','processing') then 'dispatched' else status end,
		dispatched_at = coalesce(dispatched_at, now()) where id = $1`, id, carrier, number, strPtr(url))
	if err != nil {
		return nil, err
	}
	return s.OrderByID(ctx, id)
}

type UnpaidOrder struct {
	ID     string
	Intent *string
}

// StaleUnpaid lists pending orders older than age; the caller cancels their
// PaymentIntents first, then expires them (releasing reserved stock).
func (s *Store) StaleUnpaid(ctx context.Context, age time.Duration) ([]UnpaidOrder, error) {
	rows, err := s.db.Query(ctx, `select id, stripe_payment_intent from orders
		where status = 'pending_payment' and created_at < now() - make_interval(secs => $1) order by created_at limit 200`, age.Seconds())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []UnpaidOrder
	for rows.Next() {
		var o UnpaidOrder
		if err := rows.Scan(&o.ID, &o.Intent); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// ExpireOrder cancels a still-unpaid order: restocks its lines (sample
// included) and gives back its discount use. No-op if it was paid meanwhile.
func (s *Store) ExpireOrder(ctx context.Context, id string) (bool, error) {
	done := false
	err := s.Tx(ctx, func(tx pgx.Tx) error {
		var code *string
		err := tx.QueryRow(ctx, `update orders set status = 'cancelled', payment_status = 'expired'
			where id = $1 and status = 'pending_payment' returning discount_code`, id).Scan(&code)
		if err == pgx.ErrNoRows {
			return nil
		}
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `update product_sizes s set stock = s.stock + i.qty from order_items i where i.order_id = $1 and i.sku = s.sku`, id); err != nil {
			return err
		}
		done = true
		return releaseDiscount(ctx, tx, code)
	})
	return done, err
}

// ── Idempotency ──────────────────────────────────────────────────────────────

// IdempotentResponse returns a stored response for key. The fingerprint (hash
// of the request body and caller) must match: a key cannot be replayed by
// someone else or reused for a different basket.
func (s *Store) IdempotentResponse(ctx context.Context, key, scope, fingerprint string) (int, json.RawMessage, bool, error) {
	var status int
	var body json.RawMessage
	var fp *string
	err := s.db.QueryRow(ctx, `select status_code, response, fingerprint from idempotency_keys where key = $1 and scope = $2`, key, scope).Scan(&status, &body, &fp)
	if err == pgx.ErrNoRows {
		return 0, nil, false, nil
	}
	if err != nil {
		return 0, nil, false, err
	}
	if fp == nil || *fp != fingerprint {
		return 0, nil, false, httpx.Unprocessable("idempotency_mismatch", "This request does not match the original one. Please refresh and try again.")
	}
	return status, body, true, nil
}

func (s *Store) SaveIdempotent(ctx context.Context, key, scope, fingerprint string, status int, body any) {
	_, _ = s.db.Exec(ctx, `insert into idempotency_keys (key, scope, status_code, response, fingerprint) values ($1,$2,$3,$4,$5) on conflict do nothing`,
		key, scope, status, jsonb(body), fingerprint)
}

// ── Payment events ───────────────────────────────────────────────────────────

// PaymentEventSeen reports whether a Stripe event was already processed.
func (s *Store) PaymentEventSeen(ctx context.Context, stripeID string) bool {
	var seen bool
	_ = s.db.QueryRow(ctx, `select exists(select 1 from payment_events where stripe_id = $1)`, stripeID).Scan(&seen)
	return seen
}

// RecordPaymentEvent returns false when the Stripe event was already processed.
func (s *Store) RecordPaymentEvent(ctx context.Context, stripeID, typ string, orderID *string, payload []byte) (bool, error) {
	tag, err := s.db.Exec(ctx, `insert into payment_events (stripe_id, type, order_id, payload) values ($1,$2,$3,$4) on conflict (stripe_id) do nothing`,
		stripeID, typ, orderID, payload)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// ── Store credits (Discovery Set credit-back) ────────────────────────────────

func (s *Store) IssueCredit(ctx context.Context, userID *string, email, orderID string, amount int) (string, error) {
	n, _ := rand.Int(rand.Reader, big.NewInt(1<<30))
	code := fmt.Sprintf("CREDIT-%X", n.Int64())
	_, err := s.db.Exec(ctx, `insert into store_credits (user_id, email, code, amount, remaining, origin_order_id, expires_at)
		values ($1,$2,$3,$4,$4,$5, now() + interval '12 months')`, userID, email, code, amount, orderID)
	return code, err
}
