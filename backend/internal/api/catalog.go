package api

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/BrenoD/Celestial-Oud/backend/internal/httpx"
	"github.com/BrenoD/Celestial-Oud/backend/internal/store"
)

func (a *API) listCollections(w http.ResponseWriter, r *http.Request) {
	cs, err := a.st.ListCollections(r.Context(), false)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=300")
	httpx.JSON(w, 200, cs)
}

func (a *API) getCollection(w http.ResponseWriter, r *http.Request) {
	c, err := a.st.CollectionBySlug(r.Context(), chi.URLParam(r, "slug"))
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=300")
	httpx.JSON(w, 200, c)
}

func (a *API) listProducts(w http.ResponseWriter, r *http.Request) {
	q := store.ProductQuery{
		Collection: r.URL.Query().Get("collection"),
		Q:          strings.TrimSpace(r.URL.Query().Get("q")),
		Family:     qList(r, "family"),
		Notes:      qList(r, "notes"),
		Price:      qList(r, "price"),
		Wear:       qList(r, "wear"),
		Sort:       r.URL.Query().Get("sort"),
		Page:       qInt(r, "page", 1),
		PageSize:   qInt(r, "pageSize", 24),
	}
	for _, s := range qList(r, "size") {
		q.Size = append(q.Size, qAtoi(s))
	}
	page, err := a.st.ListProducts(r.Context(), q)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=60")
	httpx.JSON(w, 200, page)
}

func qAtoi(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			break
		}
		n = n*10 + int(c-'0')
	}
	return n
}

func (a *API) getProduct(w http.ResponseWriter, r *http.Request) {
	p, err := a.st.ProductBySlug(r.Context(), chi.URLParam(r, "slug"), false)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	if uid := userID(r); uid != nil {
		a.st.RecordView(r.Context(), *uid, p.ID)
	}
	w.Header().Set("Cache-Control", "public, max-age=60")
	httpx.JSON(w, 200, p)
}

func (a *API) related(w http.ResponseWriter, r *http.Request) {
	items, err := a.st.RelatedProducts(r.Context(), chi.URLParam(r, "slug"), 4)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=300")
	httpx.JSON(w, 200, items)
}

func (a *API) listReviews(w http.ResponseWriter, r *http.Request) {
	rs, err := a.st.ProductReviews(r.Context(), chi.URLParam(r, "slug"))
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.JSON(w, 200, rs)
}

func (a *API) submitReview(w http.ResponseWriter, r *http.Request) {
	var in store.ReviewInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	u := httpx.UserFrom(r)
	if in.Author == "" {
		if me, err := a.st.UserByID(r.Context(), u.ID); err == nil {
			in.Author = strings.TrimSpace(me.FirstName + " " + firstLetter(me.LastName))
		}
	}
	rv, err := a.st.SubmitReview(r.Context(), chi.URLParam(r, "slug"), &u.ID, in)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	a.mail.Admin(r.Context(), "new_review", map[string]any{"author": rv.Author, "product": rv.ProductName, "rating": rv.Rating})
	httpx.JSON(w, 201, rv)
}

func firstLetter(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + "."
}

func (a *API) notifyStock(w http.ResponseWriter, r *http.Request) {
	var in struct {
		SKU   string `json:"sku"`
		Email string `json:"email"`
	}
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	if err := a.st.NotifyStock(r.Context(), in.SKU, in.Email); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.NoContent(w)
}

func (a *API) listNotes(w http.ResponseWriter, r *http.Request) {
	ns, err := a.st.ListNotes(r.Context())
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.JSON(w, 200, ns)
}

func (a *API) deliveryMethods(w http.ResponseWriter, r *http.Request) {
	ms, err := a.st.ListDeliveryMethods(r.Context(), false)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.JSON(w, 200, ms)
}

func (a *API) validateDiscount(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Code     string `json:"code"`
		Subtotal int    `json:"subtotal"`
	}
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	d, amount, err := a.st.ValidateDiscount(r.Context(), in.Code, in.Subtotal)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.JSON(w, 200, map[string]any{"code": d.Code, "amount": amount, "type": d.Type})
}

var ukPostcode = regexp.MustCompile(`(?i)^[A-Z]{1,2}\d[A-Z\d]? ?\d[A-Z]{2}$`)

// lookupPostcode proxies to Ideal Postcodes when a key is configured; without
// one it validates the format and returns an empty list so the UI falls back
// to manual entry.
func (a *API) lookupPostcode(w http.ResponseWriter, r *http.Request) {
	pc := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("postcode")))
	if !ukPostcode.MatchString(pc) {
		httpx.Fail(w, r, httpx.Unprocessable("invalid_postcode", "Enter a valid UK postcode."))
		return
	}
	type addr struct {
		Line1    string `json:"line1"`
		Line2    string `json:"line2,omitempty"`
		City     string `json:"city"`
		Postcode string `json:"postcode"`
	}
	if a.cfg.IdealPostcodesKey == "" {
		httpx.JSON(w, 200, []addr{})
		return
	}
	res, err := http.Get("https://api.ideal-postcodes.co.uk/v1/postcodes/" + strings.ReplaceAll(pc, " ", "") + "?api_key=" + a.cfg.IdealPostcodesKey)
	if err != nil {
		httpx.JSON(w, 200, []addr{})
		return
	}
	defer res.Body.Close()
	var payload struct {
		Result []struct {
			Line1, Line2, PostTown, Postcode string `json:",omitempty"`
			L1                               string `json:"line_1"`
			L2                               string `json:"line_2"`
			Town                             string `json:"post_town"`
			PC                               string `json:"postcode"`
		} `json:"result"`
	}
	_ = json.NewDecoder(res.Body).Decode(&payload)
	out := make([]addr, 0, len(payload.Result))
	for _, x := range payload.Result {
		out = append(out, addr{Line1: x.L1, Line2: x.L2, City: x.Town, Postcode: x.PC})
	}
	httpx.JSON(w, 200, out)
}

// ── Content ──────────────────────────────────────────────────────────────────

func (a *API) faqs(w http.ResponseWriter, r *http.Request) {
	f, err := a.st.FAQs(r.Context())
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=600")
	httpx.JSON(w, 200, f)
}

func (a *API) contentBlock(w http.ResponseWriter, r *http.Request) {
	c, err := a.st.ContentBlock(r.Context(), chi.URLParam(r, "key"))
	if err != nil || c.Status != "published" {
		httpx.Fail(w, r, httpx.ErrNotFound)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=300")
	httpx.JSON(w, 200, c)
}

func (a *API) listBlog(w http.ResponseWriter, r *http.Request) {
	posts, err := a.st.ListBlog(r.Context(), false)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.JSON(w, 200, posts)
}

func (a *API) getBlog(w http.ResponseWriter, r *http.Request) {
	p, err := a.st.BlogBySlug(r.Context(), chi.URLParam(r, "slug"))
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.JSON(w, 200, p)
}

func (a *API) publicSettings(w http.ResponseWriter, r *http.Request) {
	s, err := a.st.Settings(r.Context())
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=300")
	httpx.JSON(w, 200, map[string]any{
		"storeName": s.StoreName, "supportEmail": s.SupportEmail, "currency": s.Currency,
		"freeDeliveryThreshold": s.FreeDeliveryThreshold, "dispatchCutoff": s.DispatchCutoff, "returnsDays": s.ReturnsDays,
		"maintenanceMode": s.MaintenanceMode, "stripePublishableKey": a.cfg.StripePublishableKey,
	})
}

func (a *API) subscribe(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email  string `json:"email"`
		Name   string `json:"name"`
		Source string `json:"source"`
	}
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	isNew, err := a.st.Subscribe(r.Context(), in.Email, in.Name, in.Source)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	if isNew {
		a.mail.Send(r.Context(), "newsletter_welcome", in.Email, map[string]any{"name": in.Name})
	}
	httpx.NoContent(w)
}

func (a *API) unsubscribe(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email string `json:"email"`
	}
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	if err := a.st.Unsubscribe(r.Context(), in.Email); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.NoContent(w)
}

func (a *API) contact(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name    string  `json:"name"`
		Email   string  `json:"email"`
		OrderID *string `json:"orderId"`
		Subject *string `json:"subject"`
		Message string  `json:"message"`
	}
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	if in.OrderID != nil && strings.TrimSpace(*in.OrderID) == "" {
		in.OrderID = nil
	}
	m, err := a.st.CreateMessage(r.Context(), store.Message{UserID: userID(r), OrderID: in.OrderID, Name: in.Name, Email: in.Email, Subject: in.Subject, Body: in.Message})
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	a.mail.Admin(r.Context(), "new_message", map[string]any{"name": m.Name, "email": m.Email, "body": m.Body})
	httpx.NoContent(w)
}
