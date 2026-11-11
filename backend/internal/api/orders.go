package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/BrenoD/Celestial-Oud/backend/internal/httpx"
	"github.com/BrenoD/Celestial-Oud/backend/internal/money"
	"github.com/BrenoD/Celestial-Oud/backend/internal/store"
)

type orderResponse struct {
	*store.Order
	Payment *Intent `json:"payment,omitempty"`
}

// createOrder: price server-side → insert pending order (reserving stock) →
// create Stripe PaymentIntent → return clientSecret. Safe to retry with the
// same Idempotency-Key: the stored response is replayed.
func (a *API) createOrder(w http.ResponseWriter, r *http.Request) {
	if a.cfg.PaymentsMode == "disabled" {
		httpx.Fail(w, r, httpx.E(503, "payments_unavailable", "Checkout is temporarily unavailable. Please try again later."))
		return
	}
	idem := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idem == "" {
		httpx.Fail(w, r, httpx.BadRequest("Idempotency-Key header is required."))
		return
	}
	if len(idem) > 100 {
		idem = idem[:100]
	}
	if status, body, ok, err := a.st.IdempotentResponse(r.Context(), idem, "orders"); err == nil && ok {
		w.Header().Set("Idempotent-Replayed", "true")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write(body)
		return
	}

	var in store.CreateOrderInput
	if err := httpx.DecodeLoose(r, &in); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	in.Email = strings.TrimSpace(in.Email)
	if !strings.Contains(in.Email, "@") {
		httpx.Fail(w, r, httpx.Unprocessable("invalid_email", "Enter a valid email address."))
		return
	}
	addr := in.ShippingAddress
	if addr.FirstName == "" || addr.LastName == "" || addr.Line1 == "" || addr.City == "" || !ukPostcode.MatchString(strings.TrimSpace(addr.Postcode)) {
		httpx.Fail(w, r, httpx.Unprocessable("invalid_address", "Please complete the delivery address."))
		return
	}
	in.ShippingAddress.Postcode = strings.ToUpper(strings.TrimSpace(addr.Postcode))
	in.ShippingAddress.Country = "GB"

	settings, _ := a.st.Settings(r.Context())
	po, err := a.st.PriceOrder(r.Context(), in, settings.FreeDeliveryThreshold)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	eta := businessDays(time.Now(), etaDays(po.Method.ID))
	order, err := a.st.CreateOrder(r.Context(), in, po, userID(r), idem, eta)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	if in.MarketingOptIn {
		_, _ = a.st.Subscribe(r.Context(), in.Email, addr.FirstName, "checkout")
	}

	res := orderResponse{Order: order}
	if a.pay.Enabled() {
		intent, err := a.pay.CreateIntent(r.Context(), order.ID, in.Email, order.Total, idem)
		if err != nil {
			httpx.Logger(r).Error("stripe intent", "err", err)
			_, _ = a.st.UpdateOrderStatus(r.Context(), order.ID, "cancelled")
			httpx.Fail(w, r, httpx.E(502, "payment_unavailable", "We could not start the payment. Nothing has been charged — please try again."))
			return
		}
		_ = a.st.SetOrderIntent(r.Context(), order.ID, intent.ID)
		res.Payment = intent
	} else {
		// PaymentsMode "mock" (test environments): treat as paid so the flow is
		// testable end-to-end. Never reached in production unless PAYMENTS_MODE=mock.
		if paid, _, err := a.st.MarkPaid(r.Context(), order.ID); err == nil {
			res.Order = paid
			a.afterPaid(r, paid)
		}
	}
	a.st.SaveIdempotent(r.Context(), idem, "orders", 201, res)
	httpx.JSON(w, 201, res)
}

func etaDays(method string) int {
	switch method {
	case "next-day", "express":
		return 1
	default:
		return 3
	}
}

func businessDays(from time.Time, n int) time.Time {
	d := from
	if d.Hour() >= 14 {
		d = d.AddDate(0, 0, 1)
	}
	for n > 0 {
		d = d.AddDate(0, 0, 1)
		if d.Weekday() != time.Saturday && d.Weekday() != time.Sunday {
			n--
		}
	}
	return d
}

// getOrder: owners see their orders; guests must pass ?email= matching the order.
func (a *API) getOrder(w http.ResponseWriter, r *http.Request) {
	o, err := a.st.OrderByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	u := httpx.UserFrom(r)
	email := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("email")))
	switch {
	case u != nil && httpx.HasRole(u.Role, "viewer"):
	case u != nil && o.UserID != nil && *o.UserID == u.ID:
	case email != "" && email == strings.ToLower(o.Email):
	case time.Since(o.CreatedAt) < 2*time.Hour && o.UserID == nil:
		// Guest just placed it (confirmation page); the id is unguessable enough for this window.
	default:
		httpx.Fail(w, r, httpx.ErrNotFound)
		return
	}
	httpx.JSON(w, 200, o)
}

// confirmOrder lets the client finalise after Stripe.js reports success; the
// webhook remains the source of truth and is idempotent with this path.
func (a *API) confirmOrder(w http.ResponseWriter, r *http.Request) {
	o, err := a.st.OrderByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	if o.Status != "pending_payment" {
		httpx.JSON(w, 200, o)
		return
	}
	if o.StripePaymentIntent == nil {
		httpx.Fail(w, r, httpx.Unprocessable("no_payment", "This order has no payment attached."))
		return
	}
	status, err := a.pay.IntentStatus(r.Context(), *o.StripePaymentIntent)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	if status != "succeeded" {
		httpx.Fail(w, r, httpx.Unprocessable("payment_pending", "The payment has not been confirmed yet."))
		return
	}
	paid, first, err := a.st.MarkPaid(r.Context(), o.ID)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	if first {
		a.afterPaid(r, paid)
	}
	httpx.JSON(w, 200, paid)
}

// afterPaid sends emails and issues Discovery Set credit. Called exactly once per order.
func (a *API) afterPaid(r *http.Request, o *store.Order) {
	var items strings.Builder
	for _, l := range o.Lines {
		fmt.Fprintf(&items, "%s · %s × %d — %s<br>", l.Name, l.VariantLabel, l.Qty, money.GBP(l.UnitPrice*l.Qty))
		if l.SKU == "discovery-set" {
			if code, err := a.st.IssueCredit(r.Context(), o.UserID, o.Email, o.ID, l.UnitPrice*l.Qty); err == nil {
				fmt.Fprintf(&items, "Your Discovery Set credit code: <b>%s</b> (valid 12 months on any 50ml/100ml bottle)<br>", code)
			}
		}
	}
	data := map[string]any{"order_id": o.ID, "total": money.GBP(o.Total), "items_html": items.String(), "customer": o.CustomerName}
	a.mail.Send(r.Context(), "order_confirmation", o.Email, data)
	a.mail.Admin(r.Context(), "new_order", data)
	for _, l := range o.Lines {
		if l.SizeID != nil {
			if stock, err := a.st.AdjustStock(r.Context(), l.SKU, 0); err == nil && stock <= 5 {
				a.mail.Admin(r.Context(), "low_stock", map[string]any{"sku": l.SKU, "stock": stock})
			}
		}
	}
}

// ── Stripe webhook ───────────────────────────────────────────────────────────

func (a *API) stripeWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		httpx.Fail(w, r, httpx.BadRequest("Unreadable body."))
		return
	}
	ev, err := a.pay.ParseWebhook(r, body)
	if err != nil {
		httpx.Fail(w, r, httpx.E(400, "bad_signature", "Webhook signature verification failed."))
		return
	}
	var pi struct {
		ID       string            `json:"id"`
		Metadata map[string]string `json:"metadata"`
		LastErr  *struct {
			Message string `json:"message"`
		} `json:"last_payment_error"`
	}
	_ = json.Unmarshal(ev.Data.Raw, &pi)
	orderID := pi.Metadata["order_id"]
	var oid *string
	if orderID != "" {
		oid = &orderID
	}
	fresh, err := a.st.RecordPaymentEvent(r.Context(), ev.ID, string(ev.Type), oid, ev.Data.Raw)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	if !fresh {
		httpx.JSON(w, 200, map[string]bool{"received": true, "duplicate": true})
		return
	}
	switch ev.Type {
	case "payment_intent.succeeded":
		if orderID != "" {
			if o, first, err := a.st.MarkPaid(r.Context(), orderID); err == nil && first {
				a.afterPaid(r, o)
			}
		}
	case "payment_intent.payment_failed":
		reason := "unknown"
		if pi.LastErr != nil {
			reason = pi.LastErr.Message
		}
		a.mail.Admin(r.Context(), "payment_failed", map[string]any{"intent": pi.ID, "reason": reason})
	case "charge.refunded":
		if o, err := a.st.OrderByIntent(r.Context(), pi.Metadata["payment_intent"]); err == nil && o.Status != "refunded" {
			if _, err := a.st.UpdateOrderStatus(r.Context(), o.ID, "refunded"); err == nil {
				a.mail.Send(r.Context(), "order_refunded", o.Email, map[string]any{"order_id": o.ID, "total": money.GBP(o.Total)})
			}
		}
	}
	httpx.JSON(w, 200, map[string]bool{"received": true})
}

// ── Cron tick ────────────────────────────────────────────────────────────────

func (a *API) cronTick(w http.ResponseWriter, r *http.Request) {
	if a.cfg.CronSecret == "" || r.Header.Get("X-Cron-Secret") != a.cfg.CronSecret {
		httpx.Fail(w, r, httpx.ErrForbidden)
		return
	}
	ctx := r.Context()
	applied, reverted, perr := a.st.ApplyDuePromotions(ctx)
	expired, _ := a.st.ExpireUnpaidOrders(ctx)
	alerts, _ := a.st.DueStockAlerts(ctx)
	for _, al := range alerts {
		name, slug, _ := strings.Cut(al.ProductName, "|")
		a.mail.Send(ctx, "back_in_stock", al.Email, map[string]any{"product": name, "slug": slug})
	}
	a.st.Purge(ctx)
	out := map[string]any{"promotionsApplied": applied, "promotionsReverted": reverted, "ordersExpired": expired, "stockAlertsSent": len(alerts)}
	if perr != nil {
		out["error"] = perr.Error()
	}
	httpx.JSON(w, 200, out)
}
