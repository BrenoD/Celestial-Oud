package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/BrenoD/Celestial-Oud/backend/internal/auth"
	"github.com/BrenoD/Celestial-Oud/backend/internal/httpx"
	"github.com/BrenoD/Celestial-Oud/backend/internal/money"
	"github.com/BrenoD/Celestial-Oud/backend/internal/store"
)

// Role gates mirror printforge's SECTION_WRITE_ROLE table.
func (a *API) adminRoutes(r chi.Router) {
	manager := auth.RequireRole("manager")
	support := auth.RequireRole("support")
	admin := auth.RequireRole("admin")

	// Dashboard & analytics (viewer+)
	r.Get("/dashboard", a.adminDashboard)
	r.Get("/analytics", a.adminAnalytics)
	r.Get("/audit", a.adminAudit)

	// Catalogue (read viewer+, write manager+)
	r.Get("/products", a.adminListProducts)
	r.Get("/products/{id}", a.adminGetProduct)
	r.With(manager).Post("/products", a.adminSaveProduct)
	r.With(manager).Put("/products/{id}", a.adminSaveProduct)
	r.With(manager).Delete("/products/{id}", a.adminDeleteProduct)
	r.With(manager).Post("/products/stock", a.adminAdjustStock)
	r.Get("/collections", a.adminListCollections)
	r.With(manager).Post("/collections", a.adminSaveCollection)
	r.With(manager).Put("/collections/{id}", a.adminSaveCollection)
	r.With(manager).Delete("/collections/{id}", a.adminDeleteCollection)
	r.Get("/stock-alerts", a.adminStockAlerts)

	// Orders (support+ for operations)
	r.Get("/orders", a.adminListOrders)
	r.Get("/orders/{id}", a.adminGetOrder)
	r.With(support).Patch("/orders/{id}/status", a.adminOrderStatus)
	r.With(support).Post("/orders/{id}/tracking", a.adminOrderTracking)
	r.With(manager).Post("/orders/{id}/refund", a.adminRefund)

	// Customers & payments (manager+)
	r.With(manager).Get("/customers", a.adminCustomers)
	r.With(manager).Get("/payments", a.adminPayments)

	// Marketing
	r.Get("/discounts", a.adminListDiscounts)
	r.With(manager).Post("/discounts", a.adminSaveDiscount)
	r.With(manager).Put("/discounts/{code}", a.adminSaveDiscount)
	r.With(manager).Delete("/discounts/{code}", a.adminDeleteDiscount)
	r.Get("/promotions", a.adminListPromotions)
	r.With(manager).Post("/promotions", a.adminCreatePromotion)
	r.With(manager).Delete("/promotions/{id}", a.adminDeletePromotion)
	r.Get("/reviews", a.adminReviews)
	r.With(manager).Patch("/reviews/{id}", a.adminUpdateReview)
	r.With(manager).Delete("/reviews/{id}", a.adminDeleteReview)
	r.Get("/newsletter", a.adminNewsletter)
	r.With(manager).Post("/newsletter/send", a.adminNewsletterSend)

	// Messages (support+)
	r.With(support).Get("/messages", a.adminMessages)
	r.With(support).Post("/messages/{id}/reply", a.adminReply)
	r.With(support).Patch("/messages/{id}", a.adminMessageStatus)

	// Delivery methods (manager+)
	r.Get("/delivery-methods", a.adminDeliveryMethods)
	r.With(manager).Post("/delivery-methods", a.adminSaveDelivery)
	r.With(manager).Put("/delivery-methods/{id}", a.adminSaveDelivery)
	r.With(manager).Delete("/delivery-methods/{id}", a.adminDeleteDelivery)

	// Content & blog (manager+)
	r.Get("/content", a.adminContent)
	r.With(manager).Put("/content/{key}", a.adminSaveContent)
	r.With(manager).Delete("/content/{key}", a.adminDeleteContent)
	r.Get("/blog", a.adminBlog)
	r.With(manager).Post("/blog", a.adminSaveBlog)
	r.With(manager).Put("/blog/{id}", a.adminSaveBlog)
	r.With(manager).Delete("/blog/{id}", a.adminDeleteBlog)

	// Generic sections: taxes, emails, notifications, integrations (manager+)
	r.Get("/sections/{section}", a.adminSectionList)
	r.With(manager).Post("/sections/{section}", a.adminSectionCreate)
	r.With(manager).Patch("/sections/{section}/{id}", a.adminSectionUpdate)
	r.With(manager).Delete("/sections/{section}/{id}", a.adminSectionDelete)

	// Settings & staff (admin)
	r.Get("/settings", a.adminSettings)
	r.With(admin).Patch("/settings", a.adminUpdateSettings)
	r.With(manager).Get("/users", a.adminStaff)
	r.With(admin).Post("/users", a.adminInviteStaff)
	r.With(admin).Patch("/users/{id}", a.adminSetRole)
	r.With(admin).Post("/users/{id}/reset-password", a.adminResetPassword)
}

func (a *API) audit(r *http.Request, action, entity, id string, diff any) {
	a.st.Audit(r.Context(), userID(r), action, entity, id, diff)
}

// ── Dashboard ────────────────────────────────────────────────────────────────

func (a *API) adminDashboard(w http.ResponseWriter, r *http.Request) {
	d, err := a.st.Dashboard(r.Context())
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.JSON(w, 200, d)
}

func (a *API) adminAnalytics(w http.ResponseWriter, r *http.Request) {
	days := qInt(r, "period", 30)
	if days != 7 && days != 30 && days != 90 && days != 365 {
		days = 30
	}
	b, err := a.st.Breakdown(r.Context(), days)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.JSON(w, 200, b)
}

func (a *API) adminAudit(w http.ResponseWriter, r *http.Request) {
	rows, err := a.st.ListAudit(r.Context(), 200)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.JSON(w, 200, rows)
}

// ── Products & collections ───────────────────────────────────────────────────

func (a *API) adminListProducts(w http.ResponseWriter, r *http.Request) {
	page, err := a.st.ListProducts(r.Context(), store.ProductQuery{Q: r.URL.Query().Get("q"), Sort: "name", Page: 1, PageSize: 100, AllStatuses: true,
		Collection: r.URL.Query().Get("collection")})
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.JSON(w, 200, page)
}

func (a *API) adminGetProduct(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "id")
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	p, err := a.st.ProductByID(r.Context(), id)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.JSON(w, 200, p)
}

func (a *API) adminSaveProduct(w http.ResponseWriter, r *http.Request) {
	var in store.ProductInput
	if err := httpx.DecodeLoose(r, &in); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	var id int64
	if chi.URLParam(r, "id") != "" {
		n, err := pathInt(r, "id")
		if err != nil {
			httpx.Fail(w, r, err)
			return
		}
		id = n
	}
	p, err := a.st.SaveProduct(r.Context(), id, in)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	a.audit(r, "save", "product", p.Slug, in)
	httpx.JSON(w, 200, p)
}

func (a *API) adminDeleteProduct(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "id")
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	if err := a.st.DeleteProduct(r.Context(), id); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	a.audit(r, "delete", "product", chi.URLParam(r, "id"), nil)
	httpx.NoContent(w)
}

func (a *API) adminAdjustStock(w http.ResponseWriter, r *http.Request) {
	var in struct {
		SKU   string `json:"sku"`
		Delta int    `json:"delta"`
	}
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	stock, err := a.st.AdjustStock(r.Context(), in.SKU, in.Delta)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	a.audit(r, "stock", "size", in.SKU, in)
	httpx.JSON(w, 200, map[string]any{"sku": in.SKU, "stock": stock})
}

func (a *API) adminListCollections(w http.ResponseWriter, r *http.Request) {
	cs, err := a.st.ListCollections(r.Context(), true)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.JSON(w, 200, cs)
}

func (a *API) adminSaveCollection(w http.ResponseWriter, r *http.Request) {
	var in store.Collection
	if err := httpx.DecodeLoose(r, &in); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	if chi.URLParam(r, "id") != "" {
		n, err := pathInt(r, "id")
		if err != nil {
			httpx.Fail(w, r, err)
			return
		}
		in.ID = n
	}
	in.Slug = strings.Trim(strings.ToLower(strings.ReplaceAll(strings.TrimSpace(in.Slug), " ", "-")), "-")
	if in.Slug == "" {
		in.Slug = strings.Trim(strings.ToLower(strings.ReplaceAll(strings.TrimSpace(in.Name), " ", "-")), "-")
	}
	if in.Name == "" {
		httpx.Fail(w, r, httpx.BadRequest("A name is required."))
		return
	}
	c, err := a.st.UpsertCollection(r.Context(), in)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	a.audit(r, "save", "collection", c.Slug, in)
	httpx.JSON(w, 200, c)
}

func (a *API) adminDeleteCollection(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "id")
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	if err := a.st.DeleteCollection(r.Context(), id); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.NoContent(w)
}

func (a *API) adminStockAlerts(w http.ResponseWriter, r *http.Request) {
	al, err := a.st.ListStockAlerts(r.Context())
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.JSON(w, 200, al)
}

// ── Orders ───────────────────────────────────────────────────────────────────

func (a *API) adminListOrders(w http.ResponseWriter, r *http.Request) {
	orders, total, err := a.st.ListOrders(r.Context(), store.OrderFilter{Status: r.URL.Query().Get("status"), Q: r.URL.Query().Get("q"),
		Limit: qInt(r, "limit", 50), Offset: qInt(r, "offset", 0)})
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.JSON(w, 200, map[string]any{"items": orders, "total": total})
}

func (a *API) adminGetOrder(w http.ResponseWriter, r *http.Request) {
	o, err := a.st.OrderByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.JSON(w, 200, o)
}

func (a *API) adminOrderStatus(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Status string `json:"status"`
	}
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	o, err := a.st.UpdateOrderStatus(r.Context(), chi.URLParam(r, "id"), in.Status)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	a.audit(r, "status", "order", o.ID, in)
	switch in.Status {
	case "dispatched":
		a.mail.Send(r.Context(), "order_dispatched", o.Email, map[string]any{"order_id": o.ID, "carrier": trackingField(o, 0), "tracking": trackingField(o, 1)})
	case "delivered":
		a.mail.Send(r.Context(), "order_delivered", o.Email, map[string]any{"order_id": o.ID})
	case "refunded":
		a.mail.Send(r.Context(), "order_refunded", o.Email, map[string]any{"order_id": o.ID, "total": money.GBP(o.Total)})
	}
	httpx.JSON(w, 200, o)
}

func trackingField(o *store.Order, i int) string {
	if o.Tracking == nil {
		return "—"
	}
	if i == 0 {
		return o.Tracking.Carrier
	}
	return o.Tracking.Number
}

func (a *API) adminOrderTracking(w http.ResponseWriter, r *http.Request) {
	var in struct{ Carrier, Number, URL string }
	if err := httpx.DecodeLoose(r, &in); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	o, err := a.st.SetTracking(r.Context(), chi.URLParam(r, "id"), in.Carrier, in.Number, in.URL)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	a.audit(r, "tracking", "order", o.ID, in)
	a.mail.Send(r.Context(), "order_dispatched", o.Email, map[string]any{"order_id": o.ID, "carrier": in.Carrier, "tracking": in.Number})
	httpx.JSON(w, 200, o)
}

func (a *API) adminRefund(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Amount int `json:"amount"` // 0 = full
	}
	_ = httpx.DecodeLoose(r, &in)
	o, err := a.st.OrderByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	if o.StripePaymentIntent != nil {
		if err := a.pay.Refund(r.Context(), *o.StripePaymentIntent, in.Amount); err != nil {
			httpx.Fail(w, r, httpx.E(502, "refund_failed", "Stripe refused the refund: "+err.Error()))
			return
		}
	}
	if in.Amount == 0 || in.Amount >= o.Total {
		o, err = a.st.UpdateOrderStatus(r.Context(), o.ID, "refunded")
		if err != nil {
			httpx.Fail(w, r, err)
			return
		}
	}
	a.audit(r, "refund", "order", o.ID, in)
	a.mail.Send(r.Context(), "order_refunded", o.Email, map[string]any{"order_id": o.ID, "total": money.GBP(max(in.Amount, 0))})
	httpx.JSON(w, 200, o)
}

// ── Customers & payments ─────────────────────────────────────────────────────

func (a *API) adminCustomers(w http.ResponseWriter, r *http.Request) {
	cs, err := a.st.ListCustomers(r.Context(), strings.TrimSpace(r.URL.Query().Get("q")))
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.JSON(w, 200, cs)
}

func (a *API) adminPayments(w http.ResponseWriter, r *http.Request) {
	rows, err := a.pay.List(r.Context(), min(qInt(r, "limit", 100), 100))
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.JSON(w, 200, map[string]any{"payments": rows, "stripeEnabled": a.pay.Enabled()})
}

// ── Discounts & promotions ───────────────────────────────────────────────────

func (a *API) adminListDiscounts(w http.ResponseWriter, r *http.Request) {
	ds, err := a.st.ListDiscounts(r.Context())
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.JSON(w, 200, ds)
}

func (a *API) adminSaveDiscount(w http.ResponseWriter, r *http.Request) {
	var in store.DiscountCode
	if err := httpx.DecodeLoose(r, &in); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	if c := chi.URLParam(r, "code"); c != "" {
		in.Code = c
	}
	d, err := a.st.UpsertDiscount(r.Context(), in)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	a.audit(r, "save", "discount", d.Code, in)
	httpx.JSON(w, 200, d)
}

func (a *API) adminDeleteDiscount(w http.ResponseWriter, r *http.Request) {
	if err := a.st.DeleteDiscount(r.Context(), chi.URLParam(r, "code")); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.NoContent(w)
}

func (a *API) adminListPromotions(w http.ResponseWriter, r *http.Request) {
	ps, err := a.st.ListPromotions(r.Context())
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.JSON(w, 200, ps)
}

func (a *API) adminCreatePromotion(w http.ResponseWriter, r *http.Request) {
	var in store.ScheduledPromotion
	if err := httpx.DecodeLoose(r, &in); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	if in.StartsAt.IsZero() {
		in.StartsAt = time.Now()
	}
	p, err := a.st.CreatePromotion(r.Context(), in)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	// Apply immediately when the window is already open.
	_, _, _ = a.st.ApplyDuePromotions(r.Context())
	a.audit(r, "create", "promotion", "", in)
	httpx.JSON(w, 201, p)
}

func (a *API) adminDeletePromotion(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "id")
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	if err := a.st.DeletePromotion(r.Context(), id); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.NoContent(w)
}

// ── Reviews ──────────────────────────────────────────────────────────────────

func (a *API) adminReviews(w http.ResponseWriter, r *http.Request) {
	rs, err := a.st.AllReviews(r.Context(), r.URL.Query().Get("pending") == "1")
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.JSON(w, 200, rs)
}

func (a *API) adminUpdateReview(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "id")
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	var in map[string]any
	if err := httpx.DecodeLoose(r, &in); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	if err := a.st.UpdateReview(r.Context(), id, in); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	a.audit(r, "update", "review", chi.URLParam(r, "id"), in)
	httpx.NoContent(w)
}

func (a *API) adminDeleteReview(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "id")
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	if err := a.st.DeleteReview(r.Context(), id); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.NoContent(w)
}

// ── Newsletter ───────────────────────────────────────────────────────────────

func (a *API) adminNewsletter(w http.ResponseWriter, r *http.Request) {
	subs, err := a.st.ListSubscribers(r.Context())
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.JSON(w, 200, subs)
}

func (a *API) adminNewsletterSend(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Emails  []string `json:"emails"`
		Subject string   `json:"subject"`
		Body    string   `json:"body"`
	}
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	if len(in.Emails) == 0 || strings.TrimSpace(in.Subject) == "" || strings.TrimSpace(in.Body) == "" {
		httpx.Fail(w, r, httpx.BadRequest("Recipients, subject and message are required."))
		return
	}
	htmlBody := strings.ReplaceAll(in.Body, "\n", "<br>")
	sent := 0
	for _, e := range in.Emails {
		if err := a.mail.Raw(r.Context(), e, in.Subject, htmlBody); err == nil {
			sent++
		}
	}
	a.audit(r, "newsletter", "campaign", in.Subject, map[string]int{"recipients": len(in.Emails), "sent": sent})
	httpx.JSON(w, 200, map[string]int{"sent": sent, "total": len(in.Emails)})
}

// ── Messages ─────────────────────────────────────────────────────────────────

func (a *API) adminMessages(w http.ResponseWriter, r *http.Request) {
	ms, err := a.st.ListMessages(r.Context(), r.URL.Query().Get("status"))
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.JSON(w, 200, ms)
}

func (a *API) adminReply(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "id")
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	var in struct {
		Reply string `json:"reply"`
	}
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	m, err := a.st.ReplyMessage(r.Context(), id, in.Reply)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	subject := "your message"
	if m.Subject != nil {
		subject = *m.Subject
	}
	a.mail.Send(r.Context(), "message_reply", m.Email, map[string]any{"subject": subject, "reply": in.Reply})
	httpx.JSON(w, 200, m)
}

func (a *API) adminMessageStatus(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "id")
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	var in struct {
		Status string `json:"status"`
	}
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	if err := a.st.SetMessageStatus(r.Context(), id, in.Status); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.NoContent(w)
}

// ── Delivery, content, blog ──────────────────────────────────────────────────

func (a *API) adminDeliveryMethods(w http.ResponseWriter, r *http.Request) {
	ms, err := a.st.ListDeliveryMethods(r.Context(), true)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.JSON(w, 200, ms)
}

func (a *API) adminSaveDelivery(w http.ResponseWriter, r *http.Request) {
	var in store.DeliveryMethod
	if err := httpx.DecodeLoose(r, &in); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	if id := chi.URLParam(r, "id"); id != "" {
		in.ID = id
	}
	if in.Name == "" {
		httpx.Fail(w, r, httpx.BadRequest("A name is required."))
		return
	}
	if err := a.st.UpsertDeliveryMethod(r.Context(), in); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	a.audit(r, "save", "delivery_method", in.ID, in)
	httpx.JSON(w, 200, in)
}

func (a *API) adminDeleteDelivery(w http.ResponseWriter, r *http.Request) {
	if err := a.st.DeleteDeliveryMethod(r.Context(), chi.URLParam(r, "id")); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.NoContent(w)
}

func (a *API) adminContent(w http.ResponseWriter, r *http.Request) {
	cs, err := a.st.ListContent(r.Context())
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.JSON(w, 200, cs)
}

func (a *API) adminSaveContent(w http.ResponseWriter, r *http.Request) {
	var in store.ContentBlock
	if err := httpx.DecodeLoose(r, &in); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	in.Key = chi.URLParam(r, "key")
	if err := a.st.UpsertContent(r.Context(), in); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	a.audit(r, "save", "content", in.Key, nil)
	httpx.NoContent(w)
}

func (a *API) adminDeleteContent(w http.ResponseWriter, r *http.Request) {
	if err := a.st.DeleteContent(r.Context(), chi.URLParam(r, "key")); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.NoContent(w)
}

func (a *API) adminBlog(w http.ResponseWriter, r *http.Request) {
	ps, err := a.st.ListBlog(r.Context(), true)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.JSON(w, 200, ps)
}

func (a *API) adminSaveBlog(w http.ResponseWriter, r *http.Request) {
	var in store.BlogPost
	if err := httpx.DecodeLoose(r, &in); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	if chi.URLParam(r, "id") != "" {
		n, err := pathInt(r, "id")
		if err != nil {
			httpx.Fail(w, r, err)
			return
		}
		in.ID = n
	}
	p, err := a.st.UpsertBlog(r.Context(), in)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.JSON(w, 200, p)
}

func (a *API) adminDeleteBlog(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "id")
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	if err := a.st.DeleteBlog(r.Context(), id); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.NoContent(w)
}

// ── Generic sections ─────────────────────────────────────────────────────────

func (a *API) adminSectionList(w http.ResponseWriter, r *http.Request) {
	rows, err := a.st.ListSection(r.Context(), chi.URLParam(r, "section"))
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.JSON(w, 200, rows)
}

func (a *API) adminSectionCreate(w http.ResponseWriter, r *http.Request) {
	var in map[string]any
	if err := httpx.DecodeLoose(r, &in); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	row, err := a.st.CreateSectionItem(r.Context(), chi.URLParam(r, "section"), in)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	a.audit(r, "create", chi.URLParam(r, "section"), "", in)
	httpx.JSON(w, 201, row)
}

func (a *API) adminSectionUpdate(w http.ResponseWriter, r *http.Request) {
	var in map[string]any
	if err := httpx.DecodeLoose(r, &in); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	row, err := a.st.UpdateSectionItem(r.Context(), chi.URLParam(r, "section"), chi.URLParam(r, "id"), in)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	a.audit(r, "update", chi.URLParam(r, "section"), chi.URLParam(r, "id"), in)
	httpx.JSON(w, 200, row)
}

func (a *API) adminSectionDelete(w http.ResponseWriter, r *http.Request) {
	if err := a.st.DeleteSectionItem(r.Context(), chi.URLParam(r, "section"), chi.URLParam(r, "id")); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.NoContent(w)
}

// ── Settings & staff ─────────────────────────────────────────────────────────

func (a *API) adminSettings(w http.ResponseWriter, r *http.Request) {
	s, err := a.st.Settings(r.Context())
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.JSON(w, 200, s)
}

func (a *API) adminUpdateSettings(w http.ResponseWriter, r *http.Request) {
	cur, err := a.st.Settings(r.Context())
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	if err := httpx.DecodeLoose(r, &cur); err != nil { // merge over current values
		httpx.Fail(w, r, err)
		return
	}
	s, err := a.st.UpdateSettings(r.Context(), cur)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	a.audit(r, "update", "settings", "1", s)
	httpx.JSON(w, 200, s)
}

func (a *API) adminStaff(w http.ResponseWriter, r *http.Request) {
	us, err := a.st.ListStaff(r.Context())
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.JSON(w, 200, us)
}

func (a *API) adminInviteStaff(w http.ResponseWriter, r *http.Request) {
	var in struct{ Email, FirstName, LastName, Role, Password string }
	if err := httpx.DecodeLoose(r, &in); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	if !strings.Contains(in.Email, "@") {
		httpx.Fail(w, r, httpx.BadRequest("A valid email is required."))
		return
	}
	hash := ""
	if in.Password != "" {
		h, err := auth.HashPassword(in.Password)
		if err != nil {
			httpx.Fail(w, r, err)
			return
		}
		hash = h
	}
	u, err := a.st.InviteStaff(r.Context(), in.Email, in.FirstName, in.LastName, in.Role, hash)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	if hash == "" { // no password given: send a reset link so they set their own
		plain, h := auth.NewOpaqueToken()
		if err := a.st.SaveReset(r.Context(), u.ID, h); err == nil {
			a.mail.Send(r.Context(), "password_reset", u.Email, map[string]any{"url": a.cfg.PublicURL + "/account/reset?token=" + plain})
		}
	}
	a.audit(r, "invite", "user", u.ID, map[string]string{"role": in.Role})
	httpx.JSON(w, 201, u)
}

func (a *API) adminSetRole(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Role string `json:"role"`
	}
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	if chi.URLParam(r, "id") == httpx.UserFrom(r).ID && in.Role != "admin" {
		httpx.Fail(w, r, httpx.BadRequest("You cannot remove your own admin role."))
		return
	}
	u, err := a.st.SetRole(r.Context(), chi.URLParam(r, "id"), in.Role)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	a.audit(r, "role", "user", u.ID, in)
	httpx.JSON(w, 200, u)
}

func (a *API) adminResetPassword(w http.ResponseWriter, r *http.Request) {
	u, err := a.st.UserByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	plain, h := auth.NewOpaqueToken()
	if err := a.st.SaveReset(r.Context(), u.ID, h); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	a.mail.Send(r.Context(), "password_reset", u.Email, map[string]any{"url": a.cfg.PublicURL + "/account/reset?token=" + plain})
	a.audit(r, "reset_password", "user", u.ID, nil)
	httpx.NoContent(w)
}
