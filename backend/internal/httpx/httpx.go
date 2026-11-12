// Package httpx holds JSON helpers, the API error type and middleware.
package httpx

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Error is returned to clients as {"message","code","details"}.
type Error struct {
	Status  int    `json:"-"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
	Details any    `json:"details,omitempty"`
}

func (e *Error) Error() string { return e.Message }

func E(status int, code, msg string) *Error { return &Error{Status: status, Code: code, Message: msg} }

var (
	ErrNotFound     = E(404, "not_found", "Not found")
	ErrUnauthorized = E(401, "unauthorized", "Authentication required. Please sign in.")
	ErrForbidden    = E(403, "forbidden", "You do not have permission to perform this action.")
)

func BadRequest(msg string) *Error { return E(400, "bad_request", msg) }
func Unprocessable(code, msg string) *Error {
	return E(422, code, msg)
}

func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(v)
}

func NoContent(w http.ResponseWriter) { w.WriteHeader(http.StatusNoContent) }

// Fail maps any error to a JSON response. Unknown errors become 500 and are logged.
func Fail(w http.ResponseWriter, r *http.Request, err error) {
	var apiErr *Error
	switch {
	case errors.As(err, &apiErr):
	case errors.Is(err, pgx.ErrNoRows):
		apiErr = ErrNotFound
	default:
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "23505":
				apiErr = E(409, "conflict", "A record with this value already exists.")
			case "23503":
				apiErr = E(409, "in_use", "This record is referenced by other data and cannot be changed.")
			case "23514", "22P02", "22001", "22003":
				// Never echo database internals to clients.
				Logger(r).Warn("rejected value", "code", pgErr.Code, "constraint", pgErr.ConstraintName)
				apiErr = BadRequest("One of the values is not valid.")
			}
		}
		if apiErr == nil {
			Logger(r).Error("unhandled error", "err", err, "path", r.URL.Path)
			apiErr = E(500, "internal", "Something went wrong on our side. Please try again.")
		}
	}
	JSON(w, apiErr.Status, apiErr)
}

// Decode reads a JSON body of at most 1 MiB into v.
func Decode(r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(nil, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		if errors.Is(err, io.EOF) {
			return BadRequest("Request body is required.")
		}
		return decodeErr(err)
	}
	return nil
}

func decodeErr(err error) error {
	var tooBig *http.MaxBytesError
	if errors.As(err, &tooBig) {
		return E(413, "too_large", "Request body is too large.")
	}
	return BadRequest("The request body is not valid JSON.")
}

// DecodeLoose is Decode without DisallowUnknownFields (for partial updates).
func DecodeLoose(r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(nil, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		if errors.Is(err, io.EOF) {
			return BadRequest("Request body is required.")
		}
		return decodeErr(err)
	}
	return nil
}

// DecodeBytes decodes an already-read body (used where the raw bytes are also hashed).
func DecodeBytes(b []byte, v any) error {
	if len(b) == 0 {
		return BadRequest("Request body is required.")
	}
	if err := json.Unmarshal(b, v); err != nil {
		return decodeErr(err)
	}
	return nil
}

// ── Context values ───────────────────────────────────────────────────────────

type ctxKey int

const (
	ctxRequestID ctxKey = iota
	ctxLogger
	ctxUser
)

type UserClaims struct {
	ID    string
	Email string
	Role  string
}

func WithUser(ctx context.Context, u *UserClaims) context.Context {
	return context.WithValue(ctx, ctxUser, u)
}

func UserFrom(r *http.Request) *UserClaims {
	u, _ := r.Context().Value(ctxUser).(*UserClaims)
	return u
}

func RequestID(r *http.Request) string {
	id, _ := r.Context().Value(ctxRequestID).(string)
	return id
}

func Logger(r *http.Request) *slog.Logger {
	if l, ok := r.Context().Value(ctxLogger).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}

// ── Middleware ───────────────────────────────────────────────────────────────

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (s *statusWriter) WriteHeader(code int) { s.status = code; s.ResponseWriter.WriteHeader(code) }

func RequestLogger(base *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get("X-Request-Id")
			if !validRequestID(id) {
				id = newID()
			}
			log := base.With("request_id", id)
			ctx := context.WithValue(r.Context(), ctxRequestID, id)
			ctx = context.WithValue(ctx, ctxLogger, log)
			w.Header().Set("X-Request-Id", id)
			sw := &statusWriter{ResponseWriter: w, status: 200}
			start := time.Now()
			next.ServeHTTP(sw, r.WithContext(ctx))
			log.Info("http", "method", r.Method, "path", r.URL.Path, "status", sw.status, "ms", time.Since(start).Milliseconds())
		})
	}
}

func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				Logger(r).Error("panic", "err", fmt.Sprint(rec))
				JSON(w, 500, E(500, "internal", "Something went wrong on our side."))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func validRequestID(id string) bool {
	if id == "" || len(id) > 64 {
		return false
	}
	for _, c := range id {
		if !(c == '-' || c == '_' || c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z') {
			return false
		}
	}
	return true
}

func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Cache-Control", "no-store")
		h.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=(), usb=()")
		h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		next.ServeHTTP(w, r)
	})
}

// RateLimiter is a fixed-window per-IP limiter kept in memory. Each container
// instance limits independently, which is fine as a brute-force guard; the
// Cloudflare WAF in front handles volumetric abuse.
type RateLimiter struct {
	mu     sync.Mutex
	hits   map[string]*bucket
	limit  int
	window time.Duration
}

type bucket struct {
	count int
	reset time.Time
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{hits: map[string]*bucket{}, limit: limit, window: window}
	go func() {
		for range time.Tick(window) {
			rl.mu.Lock()
			now := time.Now()
			for k, b := range rl.hits {
				if now.After(b.reset) {
					delete(rl.hits, k)
				}
			}
			rl.mu.Unlock()
		}
	}()
	return rl
}

// Allow counts one hit for key and reports whether it is within the limit.
func (rl *RateLimiter) Allow(key string) (bool, time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	b := rl.hits[key]
	now := time.Now()
	if b == nil || now.After(b.reset) {
		b = &bucket{reset: now.Add(rl.window)}
		rl.hits[key] = b
	}
	b.count++
	return b.count <= rl.limit, time.Until(b.reset)
}

// Middleware limits per client IP (in-memory, per instance): a coarse flood
// guard in front of the shared, database-backed per-route limits.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ok, retry := rl.Allow(ClientIP(r)); !ok {
			TooMany(w, retry)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func TooMany(w http.ResponseWriter, retry time.Duration) {
	w.Header().Set("Retry-After", fmt.Sprint(int(retry.Seconds())+1))
	JSON(w, 429, E(429, "rate_limited", "Too many attempts. Please wait a moment and try again."))
}

// TrustProxy says which forwarding header carries the real client address:
//
//	"cloudflare" — CF-Connecting-IP (set by Cloudflare's edge; clients cannot forge it there)
//	"xff-last"   — the right-most X-Forwarded-For entry (appended by a single trusted proxy, e.g. Render)
//	""           — none: use the TCP peer address.
//
// Trusting a header the client controls would let anyone bypass rate limits.
var TrustProxy = ""

func ClientIP(r *http.Request) string {
	switch TrustProxy {
	case "cloudflare":
		if v := strings.TrimSpace(r.Header.Get("CF-Connecting-IP")); net.ParseIP(v) != nil {
			return v
		}
	case "xff-last":
		if v := r.Header.Get("X-Forwarded-For"); v != "" {
			parts := strings.Split(v, ",")
			if ip := strings.TrimSpace(parts[len(parts)-1]); net.ParseIP(ip) != nil {
				return ip
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Role ranks mirror printforge: viewer < support < manager < admin.
var roleRank = map[string]int{"customer": -1, "viewer": 0, "support": 1, "manager": 2, "admin": 3}

func HasRole(role, min string) bool {
	r, ok := roleRank[role]
	if !ok {
		return false
	}
	return r >= roleRank[min]
}
