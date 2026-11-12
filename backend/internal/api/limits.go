package api

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/BrenoD/Celestial-Oud/backend/internal/httpx"
)

// Per-route limits, shared by every API instance through Postgres. They are
// deliberately strict (printforge's table, tightened); the in-memory global
// limiter and Cloudflare's edge sit in front of them.
type limit struct {
	n      int
	window time.Duration
}

var limits = map[string]limit{
	"login":        {10, 15 * time.Minute},
	"login-fail":   {5, 15 * time.Minute}, // per email: lockout after 5 wrong passwords
	"register":     {5, time.Hour},
	"forgot":       {5, time.Hour},
	"forgot-email": {3, time.Hour},
	"reset":        {10, time.Hour},
	"refresh":      {60, 15 * time.Minute},
	"logout":       {30, 15 * time.Minute},
	"password":     {5, 15 * time.Minute},
	"order":        {10, 10 * time.Minute},
	"order-email":  {5, 10 * time.Minute},
	"order-read":   {30, 10 * time.Minute},
	"discount":     {20, 10 * time.Minute},
	"contact":      {5, time.Hour},
	"newsletter":   {5, time.Hour},
	"news-email":   {3, 24 * time.Hour},
	"unsubscribe":  {10, time.Hour},
	"notify":       {10, time.Hour},
	"review":       {5, time.Hour},
	"postcode":     {20, time.Hour},
	"gdpr":         {5, time.Hour},
	"admin-write":  {300, 10 * time.Minute},
}

// subjectKey hashes a subject (email, user id) so no personal data sits in
// the rate_limits table.
func subjectKey(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:12])
}

// allow counts one hit for rule+subject. On a breach it writes the 429 and
// returns false. If the counter store is unavailable it fails closed.
func (a *API) allow(w http.ResponseWriter, r *http.Request, rule, subject string) bool {
	l := limits[rule]
	ok, retry, err := a.st.Hit(r.Context(), rule+":"+subjectKey(subject), l.n, l.window)
	if err != nil {
		httpx.Logger(r).Error("rate limit store", "err", err)
		httpx.Fail(w, r, httpx.E(503, "unavailable", "Please try again in a moment."))
		return false
	}
	if !ok {
		httpx.TooMany(w, retry)
		return false
	}
	return true
}

// perIP is middleware applying rule to the client address.
func (a *API) perIP(rule string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if a.allow(w, r, rule, "ip:"+httpx.ClientIP(r)) {
				next.ServeHTTP(w, r)
			}
		})
	}
}

// perUser applies rule to the signed-in user (falls back to IP).
func (a *API) perUser(rule string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			subject := "ip:" + httpx.ClientIP(r)
			if u := httpx.UserFrom(r); u != nil {
				subject = "user:" + u.ID
			}
			if a.allow(w, r, rule, subject) {
				next.ServeHTTP(w, r)
			}
		})
	}
}

// writesOnly limits non-GET requests.
func (a *API) writesOnly(rule string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		limited := a.perUser(rule)(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}
			limited.ServeHTTP(w, r)
		})
	}
}
