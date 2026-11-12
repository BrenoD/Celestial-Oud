// Package auth: password hashing, JWT access tokens and opaque refresh tokens.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/BrenoD/Celestial-Oud/backend/internal/httpx"
)

type Service struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
	// RoleOf, when set, is consulted for staff tokens so a demoted or deleted
	// staff member loses access immediately instead of when the JWT expires.
	RoleOf func(ctx context.Context, userID string) string
}

func New(secret string, accessTTL, refreshTTL time.Duration) *Service {
	return &Service{secret: []byte(secret), accessTTL: accessTTL, refreshTTL: refreshTTL}
}

// ValidatePassword: 8–72 bytes (bcrypt's limit), not a common password and
// not the email's local part. Follows NCSC guidance (length + blocklist, no
// composition rules).
func ValidatePassword(pw, email string) error {
	if len(pw) < 8 {
		return httpx.Unprocessable("weak_password", "Password must be at least 8 characters long.")
	}
	if len(pw) > 72 {
		return httpx.Unprocessable("weak_password", "Password must be at most 72 characters long.")
	}
	low := strings.ToLower(pw)
	if commonPasswords[low] {
		return httpx.Unprocessable("weak_password", "This password is too common. Try three random words instead.")
	}
	if local, _, ok := strings.Cut(strings.ToLower(email), "@"); ok && len(local) >= 4 && strings.Contains(low, local) {
		return httpx.Unprocessable("weak_password", "Your password must not contain your email address.")
	}
	return nil
}

func HashPassword(pw string) (string, error) {
	if len(pw) < 8 || len(pw) > 72 {
		return "", ValidatePassword(pw, "")
	}
	b, err := bcrypt.GenerateFromPassword([]byte(pw), 12)
	return string(b), err
}

// dummyHash is compared against when the account does not exist, so a login
// for an unknown email costs the same time as a wrong password.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("celestial-oud-timing-guard"), 12)

func CheckPassword(hash, pw string) bool {
	if len(pw) > 72 {
		pw = pw[:72]
	}
	if hash == "" {
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(pw))
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}

type Claims struct {
	Email string `json:"email"`
	Role  string `json:"role"`
	jwt.RegisteredClaims
}

func (s *Service) IssueAccess(userID, email, role string) (string, time.Time, error) {
	exp := time.Now().Add(s.accessTTL)
	claims := Claims{
		Email: email,
		Role:  role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "celestial-oud",
		},
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
	return tok, exp, err
}

func (s *Service) ParseAccess(token string) (*httpx.UserClaims, error) {
	var c Claims
	_, err := jwt.ParseWithClaims(token, &c, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.secret, nil
	}, jwt.WithIssuer("celestial-oud"), jwt.WithLeeway(30*time.Second), jwt.WithExpirationRequired(),
		jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return nil, err
	}
	return &httpx.UserClaims{ID: c.Subject, Email: c.Email, Role: c.Role}, nil
}

// NewOpaqueToken returns (plaintext, sha256-hex) for refresh/reset tokens.
// Only the hash is stored.
func NewOpaqueToken() (string, string) {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	plain := base64.RawURLEncoding.EncodeToString(b)
	return plain, HashToken(plain)
}

func HashToken(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}

func (s *Service) RefreshTTL() time.Duration { return s.refreshTTL }

// BearerToken extracts the token from an Authorization header.
func BearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimSpace(h[7:])
	}
	return ""
}

// Middleware attaches the user to the context when a valid token is present.
// It never rejects; use Require / RequireRole for that.
func (s *Service) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if tok := BearerToken(r); tok != "" {
			if u, err := s.ParseAccess(tok); err == nil {
				if u.Role != "customer" && s.RoleOf != nil {
					u.Role = s.RoleOf(r.Context(), u.ID)
				}
				if u.Role != "" {
					r = r.WithContext(httpx.WithUser(r.Context(), u))
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

func Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if httpx.UserFrom(r) == nil {
			httpx.Fail(w, r, httpx.ErrUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func RequireRole(min string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u := httpx.UserFrom(r)
			if u == nil {
				httpx.Fail(w, r, httpx.ErrUnauthorized)
				return
			}
			if !httpx.HasRole(u.Role, min) {
				httpx.Fail(w, r, httpx.ErrForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
