// Package api wires HTTP routes to the store.
package api

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/BrenoD/Celestial-Oud/backend/internal/auth"
	"github.com/BrenoD/Celestial-Oud/backend/internal/config"
	"github.com/BrenoD/Celestial-Oud/backend/internal/httpx"
	"github.com/BrenoD/Celestial-Oud/backend/internal/mail"
	"github.com/BrenoD/Celestial-Oud/backend/internal/store"
)

type API struct {
	cfg  config.Config
	st   *store.Store
	auth *auth.Service
	mail *mail.Mailer
	log  *slog.Logger
	pay  *Payments
}

func New(cfg config.Config, st *store.Store, a *auth.Service, m *mail.Mailer, log *slog.Logger) *API {
	return &API{cfg: cfg, st: st, auth: a, mail: m, log: log, pay: NewPayments(cfg)}
}

func (a *API) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(httpx.Recoverer, httpx.RequestLogger(a.log), httpx.SecurityHeaders, middleware.Timeout(30*time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   a.cfg.CORSOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "Idempotency-Key", "X-Request-Id"},
		ExposedHeaders:   []string{"X-Request-Id"},
		AllowCredentials: false,
		MaxAge:           600,
	}))
	r.Use(a.auth.Middleware)

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { httpx.JSON(w, 200, map[string]string{"status": "ok"}) })
	r.Get("/readyz", a.ready)

	authLimiter := httpx.NewRateLimiter(20, time.Minute)
	writeLimiter := httpx.NewRateLimiter(60, time.Minute)

	r.Route("/v1", func(r chi.Router) {
		// ── Public catalogue & content ──
		r.Get("/collections", a.listCollections)
		r.Get("/collections/{slug}", a.getCollection)
		r.Get("/products", a.listProducts)
		r.Get("/products/{slug}", a.getProduct)
		r.Get("/products/{slug}/related", a.related)
		r.Get("/products/{slug}/reviews", a.listReviews)
		r.With(auth.Require, writeLimiter.Middleware).Post("/products/{slug}/reviews", a.submitReview)
		r.With(writeLimiter.Middleware).Post("/products/{slug}/notify", a.notifyStock)
		r.Get("/notes", a.listNotes)
		r.Get("/delivery-methods", a.deliveryMethods)
		r.Get("/address/lookup", a.lookupPostcode)
		r.With(writeLimiter.Middleware).Post("/discounts/validate", a.validateDiscount)
		r.Get("/content/faqs", a.faqs)
		r.Get("/content/{key}", a.contentBlock)
		r.Get("/blog", a.listBlog)
		r.Get("/blog/{slug}", a.getBlog)
		r.Get("/settings", a.publicSettings)
		r.With(writeLimiter.Middleware).Post("/newsletter", a.subscribe)
		r.With(writeLimiter.Middleware).Post("/newsletter/unsubscribe", a.unsubscribe)
		r.With(writeLimiter.Middleware).Post("/contact", a.contact)

		// ── Orders ──
		r.With(writeLimiter.Middleware).Post("/orders", a.createOrder)
		r.Get("/orders/{id}", a.getOrder)
		r.Post("/orders/{id}/confirm", a.confirmOrder)

		// ── Auth ──
		r.Route("/auth", func(r chi.Router) {
			r.Use(authLimiter.Middleware)
			r.Post("/register", a.register)
			r.Post("/login", a.login)
			r.Post("/refresh", a.refresh)
			r.Post("/logout", a.logout)
			r.Post("/forgot", a.forgot)
			r.Post("/reset", a.reset)
		})

		// ── Account ──
		r.Route("/account", func(r chi.Router) {
			r.Use(auth.Require)
			r.Get("/", a.me)
			r.Patch("/", a.updateMe)
			r.Delete("/", a.deleteMe)
			r.Post("/password", a.changePassword)
			r.Get("/export", a.exportMe)
			r.Get("/orders", a.myOrders)
			r.Get("/addresses", a.myAddresses)
			r.Post("/addresses", a.saveAddress)
			r.Put("/addresses/{id}", a.saveAddress)
			r.Delete("/addresses/{id}", a.deleteAddress)
			r.Get("/wishlist", a.wishlist)
			r.Post("/wishlist/{slug}", a.toggleWishlist)
			r.Get("/scent-profile", a.scentProfile)
			r.Put("/scent-profile", a.saveScentProfile)
		})

		// ── Admin ──
		r.Route("/admin", func(r chi.Router) {
			r.Use(auth.RequireRole("viewer"))
			a.adminRoutes(r)
		})
	})

	// ── Webhooks & cron (no CORS needs, secret-protected) ──
	r.Post("/webhooks/stripe", a.stripeWebhook)
	r.Post("/internal/cron/tick", a.cronTick)

	r.NotFound(func(w http.ResponseWriter, r *http.Request) { httpx.Fail(w, r, httpx.ErrNotFound) })
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		httpx.Fail(w, r, httpx.E(405, "method_not_allowed", "Method not allowed"))
	})
	return r
}

func (a *API) ready(w http.ResponseWriter, r *http.Request) {
	if _, err := a.st.Settings(r.Context()); err != nil {
		httpx.JSON(w, 503, map[string]string{"status": "degraded", "error": err.Error()})
		return
	}
	httpx.JSON(w, 200, map[string]string{"status": "ready"})
}

// ── Small helpers ────────────────────────────────────────────────────────────

func qInt(r *http.Request, key string, def int) int {
	if v, err := strconv.Atoi(r.URL.Query().Get(key)); err == nil {
		return v
	}
	return def
}

func qList(r *http.Request, key string) []string {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return nil
	}
	var out []string
	for _, s := range strings.Split(raw, ",") {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func pathInt(r *http.Request, key string) (int64, error) {
	n, err := strconv.ParseInt(chi.URLParam(r, key), 10, 64)
	if err != nil {
		return 0, httpx.BadRequest("Invalid id.")
	}
	return n, nil
}

func userID(r *http.Request) *string {
	if u := httpx.UserFrom(r); u != nil {
		return &u.ID
	}
	return nil
}
