// Package auth: password hashing, JWT access tokens and opaque refresh tokens.
package auth

import (
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
}

func New(secret string, accessTTL, refreshTTL time.Duration) *Service {
	return &Service{secret: []byte(secret), accessTTL: accessTTL, refreshTTL: refreshTTL}
}

func HashPassword(pw string) (string, error) {
	if len(pw) < 8 {
		return "", httpx.Unprocessable("weak_password", "Password must be at least 8 characters long.")
	}
	b, err := bcrypt.GenerateFromPassword([]byte(pw), 12)
	return string(b), err
}

func CheckPassword(hash, pw string) bool {
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
	}, jwt.WithIssuer("celestial-oud"), jwt.WithLeeway(30*time.Second))
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
				r = r.WithContext(httpx.WithUser(r.Context(), u))
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
