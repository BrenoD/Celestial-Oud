package api

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/BrenoD/Celestial-Oud/backend/internal/auth"
	"github.com/BrenoD/Celestial-Oud/backend/internal/httpx"
	"github.com/BrenoD/Celestial-Oud/backend/internal/mail"
	"github.com/BrenoD/Celestial-Oud/backend/internal/money"
	"github.com/BrenoD/Celestial-Oud/backend/internal/store"
)

type orderResponse struct {
	*store.Order
	Payment *Intent `json:"payment,omitempty"`
}

// validateAddress normalises and caps a UK delivery address.
func validateAddress(addr *store.Address) error {
	addr.FirstName, addr.LastName = strings.TrimSpace(addr.FirstName), strings.TrimSpace(addr.LastName)
	addr.Line1, addr.City = strings.TrimSpace(addr.Line1), strings.TrimSpace(addr.City)
	addr.Postcode = strings.ToUpper(strings.TrimSpace(addr.Postcode))
	if addr.FirstName == "" || addr.LastName == "" || addr.Line1 == "" || addr.City == "" || !httpx.UKPostcode.MatchString(addr.Postcode) {
		return httpx.Unprocessable("invalid_address", "Please complete the delivery address with a UK postcode.")
	}
	addr.Country = "GB"
	return httpx.CheckLengths(
		httpx.Limit{Name: "First name", Value: addr.FirstName, Max: 60}, httpx.Limit{Name: "Last name", Value: addr.LastName, Max: 60},
		httpx.Limit{Name: "Address line 1", Value: addr.Line1, Max: 100}, httpx.Limit{Name: "Address line 2", Value: deref(addr.Line2), Max: 100},
		httpx.Limit{Name: "Town or city", Value: addr.City, Max: 60}, httpx.Limit{Name: "Phone", Value: deref(addr.Phone), Max: 30},
		httpx.Limit{Name: "Label", Value: deref(addr.Label), Max: 30})
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// createOrder: price server-side → insert pending order (reserving stock) →
// create Stripe PaymentIntent → return clientSecret. Safe to retry with the
// same Idempotency-Key and body: the stored response is replayed.
func (a *API) createOrder(w http.ResponseWriter, r *http.Request) {
	if a.cfg.PaymentsMode == "disabled" {
		httpx.Fail(w, r, httpx.E(503, "payments_unavailable", "Checkout is temporarily unavailable. Please try again later."))
		return
	}
	idem := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if len(idem) < 16 || len(idem) > 100 {
		httpx.Fail(w, r, httpx.BadRequest("A valid Idempotency-Key header is required."))
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 64<<10))
	if err != nil {
		httpx.Fail(w, r, httpx.E(413, "too_large", "Request body is too large."))
		return
	}
	caller := "guest"
	if u := httpx.UserFrom(r); u != nil {
		caller = u.ID
	}
	fp := sha256.Sum256(append([]byte(caller+"\n"), body...))
	fingerprint := hex.EncodeToString(fp[:])
	if status, stored, ok, err := a.st.IdempotentResponse(r.Context(), idem, "orders", fingerprint); err != nil {
		httpx.Fail(w, r, err)
		return
	} else if ok {
		w.Header().Set("Idempotent-Replayed", "true")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write(stored)
		return
	}

	var in store.CreateOrderInput
	if err := httpx.DecodeBytes(body, &in); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	email, ok := httpx.NormEmail(in.Email)
	if !ok {
		httpx.Fail(w, r, httpx.InvalidEmail)
		return
	}
	in.Email = email
	if !a.allow(w, r, "order-email", email) {
		return
	}
	if err := validateAddress(&in.ShippingAddress); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	if err := httpx.CheckLengths(httpx.Limit{Name: "Gift message", Value: deref(in.GiftMessage), Max: 200},
		httpx.Limit{Name: "Order notes", Value: deref(in.Notes), Max: 500}, httpx.Limit{Name: "Discount code", Value: deref(in.DiscountCode), Max: 40}); err != nil {
		httpx.Fail(w, r, err)
		return
	}

	settings, _ := a.st.Settings(r.Context())
	po, err := a.st.PriceOrder(r.Context(), in, settings.FreeDeliveryThreshold)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	eta := businessDays(time.Now(), etaDays(po.Method.ID))
	accessToken, accessHash := auth.NewOpaqueToken()
	order, err := a.st.CreateOrder(r.Context(), in, po, userID(r), idem, accessHash, eta)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	if in.MarketingOptIn {
		a.requestSubscription(r, email, in.ShippingAddress.FirstName, "checkout")
	}

	res := orderResponse{Order: order}
	if a.pay.Enabled() {
		intent, err := a.pay.CreateIntent(r.Context(), order.ID, email, order.Total, idem)
		if err != nil {
			httpx.Logger(r).Error("stripe intent", "err", err)
			_, _ = a.st.ExpireOrder(r.Context(), order.ID)
			httpx.Fail(w, r, httpx.E(502, "payment_unavailable", "We could not start the payment. Nothing has been charged — please try again."))
			return
		}
		_ = a.st.SetOrderIntent(r.Context(), order.ID, intent.ID)
		res.Payment = intent
	} else {
		// PaymentsMode "mock" (test environments only): treat as paid so the flow is
		// testable end-to-end. Never reached in production unless PAYMENTS_MODE=mock.
		if paid, _, err := a.st.MarkPaid(r.Context(), order.ID); err == nil {
			res.Order = paid
			a.afterPaid(r, paid, accessToken)
		}
	}
	res.Order.AccessToken = accessToken
	a.st.SaveIdempotent(r.Context(), idem, "orders", fingerprint, 201, res)
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

// orderAccess loads an order the caller may see: staff, the owning account,
// the holder of the order's access token, or someone who knows its email.
// Everyone else gets 404 (not 403), so order numbers cannot be probed.
func (a *API) orderAccess(r *http.Request, id, token, email string) (*store.Order, error) {
	if len(id) > 20 {
		return nil, httpx.ErrNotFound
	}
	o, err := a.st.OrderByID(r.Context(), id)
	if err != nil {
		return nil, httpx.ErrNotFound
	}
	u := httpx.UserFrom(r)
	switch {
	case u != nil && httpx.HasRole(u.Role, "viewer"):
	case u != nil && o.UserID != nil && *o.UserID == u.ID:
	case token != "" && subtle.ConstantTimeCompare([]byte(auth.HashToken(token)), []byte(a.st.OrderAccessHash(r.Context(), id))) == 1:
	case email != "" && subtle.ConstantTimeCompare([]byte(strings.ToLower(strings.TrimSpace(email))), []byte(strings.ToLower(o.Email))) == 1:
	default:
		return nil, httpx.ErrNotFound
	}
	return o, nil
}

func (a *API) getOrder(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	o, err := a.orderAccess(r, chi.URLParam(r, "id"), q.Get("token"), q.Get("email"))
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.JSON(w, 200, o)
}

// confirmOrder lets the client finalise after Stripe.js reports success; the
// webhook remains the source of truth and is idempotent with this path.
func (a *API) confirmOrder(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Token string `json:"token"`
	}
	_ = httpx.DecodeLoose(r, &in)
	o, err := a.orderAccess(r, chi.URLParam(r, "id"), in.Token, "")
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
		a.afterPaid(r, paid, in.Token)
	}
	httpx.JSON(w, 200, paid)
}

// afterPaid sends emails and issues Discovery Set credit. Called exactly once per order.
func (a *API) afterPaid(r *http.Request, o *store.Order, accessToken string) {
	var items strings.Builder
	for _, l := range o.Lines {
		fmt.Fprintf(&items, "%s · %s × %d — %s<br>", html.EscapeString(l.Name), html.EscapeString(l.VariantLabel), l.Qty, money.GBP(l.UnitPrice*l.Qty))
		if l.SKU == "discovery-set" {
			if code, err := a.st.IssueCredit(r.Context(), o.UserID, o.Email, o.ID, l.UnitPrice*l.Qty); err == nil {
				fmt.Fprintf(&items, "Your Discovery Set credit code: <b>%s</b> (valid 12 months on any 50ml/100ml bottle)<br>", html.EscapeString(code))
			}
		}
	}
	orderURL := a.cfg.PublicURL + "/checkout/confirmed/" + url.PathEscape(o.ID)
	if accessToken != "" {
		orderURL += "?token=" + url.QueryEscape(accessToken)
	}
	data := map[string]any{"order_id": o.ID, "total": money.GBP(o.Total), "items_html": mail.SafeHTML(items.String()), "customer": o.CustomerName, "order_url": orderURL}
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
	if a.st.PaymentEventSeen(r.Context(), ev.ID) {
		httpx.JSON(w, 200, map[string]bool{"received": true, "duplicate": true})
		return
	}
	var obj struct {
		ID             string            `json:"id"`
		Amount         int               `json:"amount"`
		AmountReceived int               `json:"amount_received"`
		Currency       string            `json:"currency"`
		PaymentIntent  string            `json:"payment_intent"` // on charge objects
		Metadata       map[string]string `json:"metadata"`
		LastErr        *struct {
			Message string `json:"message"`
		} `json:"last_payment_error"`
	}
	_ = json.Unmarshal(ev.Data.Raw, &obj)
	orderID := obj.Metadata["order_id"]
	var oid *string
	if orderID != "" {
		oid = &orderID
	}
	log := httpx.Logger(r)
	switch ev.Type {
	case "payment_intent.succeeded":
		o, err := a.st.OrderByID(r.Context(), orderID)
		if err != nil {
			log.Warn("webhook for unknown order", "order", orderID, "intent", obj.ID)
			break // acknowledge: retrying cannot help
		}
		// Only a payment for exactly this order, amount and currency marks it paid.
		if o.StripePaymentIntent == nil || *o.StripePaymentIntent != obj.ID || obj.AmountReceived != o.Total || !strings.EqualFold(obj.Currency, o.Currency) {
			log.Error("payment does not match order", "order", o.ID, "intent", obj.ID, "amount", obj.AmountReceived, "total", o.Total, "currency", obj.Currency)
			a.mail.Admin(r.Context(), "payment_failed", map[string]any{"intent": obj.ID, "reason": "Payment does not match order " + o.ID + " — check it manually."})
			break
		}
		paid, first, err := a.st.MarkPaid(r.Context(), o.ID)
		if err != nil {
			httpx.Fail(w, r, err) // 500: Stripe retries, the event is not recorded yet
			return
		}
		if first {
			a.afterPaid(r, paid, "")
		}
	case "payment_intent.payment_failed":
		reason := "unknown"
		if obj.LastErr != nil {
			reason = obj.LastErr.Message
		}
		a.mail.Admin(r.Context(), "payment_failed", map[string]any{"intent": obj.ID, "reason": reason})
	case "charge.refunded":
		if o, err := a.st.OrderByIntent(r.Context(), obj.PaymentIntent); err == nil && o.Status != "refunded" {
			if _, err := a.st.UpdateOrderStatus(r.Context(), o.ID, "refunded"); err == nil {
				a.mail.Send(r.Context(), "order_refunded", o.Email, map[string]any{"order_id": o.ID, "total": money.GBP(o.Total)})
			}
		}
	}
	// Recorded only after processing succeeded, so a failure is retried by Stripe.
	if _, err := a.st.RecordPaymentEvent(r.Context(), ev.ID, string(ev.Type), oid, ev.Data.Raw); err != nil {
		log.Error("record payment event", "err", err)
	}
	httpx.JSON(w, 200, map[string]bool{"received": true})
}

// ── Cron tick ────────────────────────────────────────────────────────────────

func (a *API) cronTick(w http.ResponseWriter, r *http.Request) {
	if a.cfg.CronSecret == "" || subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Cron-Secret")), []byte(a.cfg.CronSecret)) != 1 {
		httpx.Fail(w, r, httpx.ErrForbidden)
		return
	}
	ctx := r.Context()
	applied, reverted, perr := a.st.ApplyDuePromotions(ctx)
	expired := a.expireUnpaid(r)
	alerts, _ := a.st.DueStockAlerts(ctx)
	for _, al := range alerts {
		name, slug, _ := strings.Cut(al.ProductName, "|")
		a.mail.Send(ctx, "back_in_stock", al.Email, map[string]any{"product": name, "slug": slug})
	}
	a.st.Purge(ctx)
	out := map[string]any{"promotionsApplied": applied, "promotionsReverted": reverted, "ordersExpired": expired, "stockAlertsSent": len(alerts)}
	if perr != nil {
		httpx.Logger(r).Error("promotions", "err", perr)
		out["promotionsError"] = true
	}
	httpx.JSON(w, 200, out)
}

// unpaidHold is how long an unpaid order keeps its stock reserved.
const unpaidHold = time.Hour

// expireUnpaid cancels stale PaymentIntents first (so a late payment cannot
// land on a cancelled order), then releases the reserved stock.
func (a *API) expireUnpaid(r *http.Request) int {
	stale, err := a.st.StaleUnpaid(r.Context(), unpaidHold)
	if err != nil {
		httpx.Logger(r).Error("stale unpaid", "err", err)
		return 0
	}
	n := 0
	for _, o := range stale {
		if o.Intent != nil {
			status, err := a.pay.CancelIntent(r.Context(), *o.Intent)
			if err != nil {
				httpx.Logger(r).Warn("cancel intent", "order", o.ID, "err", err)
				continue
			}
			if status == "succeeded" {
				if paid, first, err := a.st.MarkPaid(r.Context(), o.ID); err == nil && first {
					a.afterPaid(r, paid, "")
				}
				continue
			}
		}
		if ok, err := a.st.ExpireOrder(r.Context(), o.ID); err == nil && ok {
			n++
		}
	}
	return n
}
