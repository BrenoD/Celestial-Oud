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
}

// PriceOrder resolves SKUs, checks stock and computes totals.
func (s *Store) PriceOrder(ctx context.Context, in CreateOrderInput, freeThreshold int) (*PricedOrder, error) {
	if len(in.Lines) == 0 {
		return nil, httpx.Unprocessable("empty_basket", "Your basket is empty.")
	}
	skus := make([]string, 0, len(in.Lines))
	qty := map[string]int{}
	for _, l := range in.Lines {
		if l.Qty < 1 || l.Qty > 10 {
			return nil, httpx.Unprocessable("bad_qty", "Quantities must be between 1 and 10.")
		}
		if _, dup := qty[l.SKU]; !dup {
			skus = append(skus, l.SKU)
		}
		qty[l.SKU] += l.Qty
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

func newOrderID() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(900000))
	return fmt.Sprintf("CO-%06d", n.Int64()+100000)
}

// CreateOrder inserts the order + lines and reserves stock. Status is
// pending_payment; MarkPaid is called by the Stripe webhook (or immediately in
// dev when Stripe is not configured).
func (s *Store) CreateOrder(ctx context.Context, in CreateOrderInput, po *PricedOrder, userID *string, idem string, eta time.Time) (*Order, error) {
	var id string
	err := s.Tx(ctx, func(tx pgx.Tx) error {
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
			shipping_address, free_sample_sku, gift_wrap, gift_message, marketing_opt_in, notes, idempotency_key, estimated_delivery)
			values ($1,$2,$3,'pending_payment',$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)`,
			id, userID, strings.ToLower(strings.TrimSpace(in.Email)), po.Subtotal, po.Discount, po.Delivery, po.Total, po.DiscountCode, po.Method.ID,
			jsonb(in.ShippingAddress), in.FreeSampleSKU, in.GiftWrap, in.GiftMessage, in.MarketingOptIn, in.Notes, strPtr(idem), eta)
		if err != nil {
			return err
		}
		for _, l := range po.Lines {
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
		if po.DiscountCode != nil {
			if _, err := tx.Exec(ctx, `update discount_codes set uses = uses + 1 where code = $1`, *po.DiscountCode); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.OrderByID(ctx, id)
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

func (s *Store) ExpireUnpaidOrders(ctx context.Context) (int64, error) {
	var n int64
	err := s.Tx(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `select id from orders where status = 'pending_payment' and created_at < now() - interval '24 hours'`)
		if err != nil {
			return err
		}
		var ids []string
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return err
			}
			ids = append(ids, id)
		}
		rows.Close()
		for _, id := range ids {
			if _, err := tx.Exec(ctx, `update product_sizes s set stock = s.stock + i.qty from order_items i where i.order_id = $1 and i.sku = s.sku`, id); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `update orders set status = 'cancelled', payment_status = 'expired' where id = $1`, id); err != nil {
				return err
			}
			n++
		}
		return nil
	})
	return n, err
}

// ── Idempotency ──────────────────────────────────────────────────────────────

func (s *Store) IdempotentResponse(ctx context.Context, key, scope string) (int, json.RawMessage, bool, error) {
	var status int
	var body json.RawMessage
	err := s.db.QueryRow(ctx, `select status_code, response from idempotency_keys where key = $1 and scope = $2`, key, scope).Scan(&status, &body)
	if err == pgx.ErrNoRows {
		return 0, nil, false, nil
	}
	return status, body, err == nil, err
}

func (s *Store) SaveIdempotent(ctx context.Context, key, scope string, status int, body any) {
	_, _ = s.db.Exec(ctx, `insert into idempotency_keys (key, scope, status_code, response) values ($1,$2,$3,$4) on conflict do nothing`, key, scope, status, jsonb(body))
}

// ── Payment events ───────────────────────────────────────────────────────────

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
