// Package config reads runtime configuration from the environment.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Env             string // development | production
	Port            string
	DatabaseURL     string
	JWTSecret       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	CORSOrigins     []string
	PublicURL       string // storefront URL used in emails
	APIURL          string // this API's public URL (webhooks)

	StripeSecretKey      string
	StripePublishableKey string
	StripeWebhookSecret  string

	ResendAPIKey string
	EmailFrom    string
	AdminEmails  []string

	CronSecret        string
	IdealPostcodesKey string // optional UK address lookup provider

	// PaymentsMode: "stripe" when STRIPE_SECRET_KEY is set; otherwise "mock"
	// (orders are marked paid immediately — test environments only) or
	// "disabled". Production without Stripe is "disabled" unless
	// PAYMENTS_MODE=mock is set explicitly, so a live shop can never hand out
	// free orders by accident.
	PaymentsMode string

	// TrustProxy: "cloudflare" | "xff-last" | "" — see httpx.TrustProxy.
	TrustProxy string
	// CookieSameSite for the refresh-token cookie: "Strict" when the storefront
	// and API share a site (celestialoud.com / api.celestialoud.com), "None"
	// only for cross-site preview hosts.
	CookieSameSite string
	CookieSecure   bool
}

func Load() (Config, error) {
	c := Config{
		// Unset APP_ENV means production: safe defaults unless a developer opts out.
		Env:             get("APP_ENV", "production"),
		Port:            get("PORT", "8080"),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		JWTSecret:       os.Getenv("JWT_SECRET"),
		AccessTokenTTL:  dur("ACCESS_TOKEN_TTL", 15*time.Minute),
		RefreshTokenTTL: dur("REFRESH_TOKEN_TTL", 30*24*time.Hour),
		CORSOrigins:     list("CORS_ORIGINS", "http://localhost:5173,http://localhost:4173"),
		PublicURL:       get("PUBLIC_URL", "http://localhost:5173"),
		APIURL:          get("API_URL", "http://localhost:8080"),

		StripeSecretKey:      os.Getenv("STRIPE_SECRET_KEY"),
		StripePublishableKey: os.Getenv("STRIPE_PUBLISHABLE_KEY"),
		StripeWebhookSecret:  os.Getenv("STRIPE_WEBHOOK_SECRET"),

		ResendAPIKey: os.Getenv("RESEND_API_KEY"),
		EmailFrom:    get("EMAIL_FROM", "Celestial Oud <noreply@celestialoud.com>"),
		AdminEmails:  list("ADMIN_EMAILS", ""),

		CronSecret:        os.Getenv("CRON_SECRET"),
		IdealPostcodesKey: os.Getenv("IDEAL_POSTCODES_KEY"),

		TrustProxy:     strings.ToLower(os.Getenv("TRUST_PROXY")),
		CookieSameSite: get("COOKIE_SAMESITE", "Strict"),
	}
	c.CookieSecure = c.Env == "production" || strings.HasPrefix(c.APIURL, "https://")
	switch c.TrustProxy {
	case "", "cloudflare", "xff-last":
	default:
		return c, fmt.Errorf("TRUST_PROXY must be cloudflare, xff-last or empty")
	}
	switch {
	case c.StripeSecretKey != "":
		c.PaymentsMode = "stripe"
	case strings.EqualFold(os.Getenv("PAYMENTS_MODE"), "mock") || c.Env != "production":
		c.PaymentsMode = "mock"
	default:
		c.PaymentsMode = "disabled"
	}
	if c.DatabaseURL == "" {
		return c, fmt.Errorf("DATABASE_URL is required")
	}
	if c.JWTSecret == "" {
		if c.Env == "production" {
			return c, fmt.Errorf("JWT_SECRET is required in production")
		}
		c.JWTSecret = "dev-secret-change-me"
	}
	if len(c.JWTSecret) < 32 && c.Env == "production" {
		return c, fmt.Errorf("JWT_SECRET must be at least 32 characters")
	}
	if c.Env == "production" && c.PaymentsMode == "mock" {
		// Allowed only for explicit test hosts; logged loudly at boot.
		fmt.Fprintln(os.Stderr, "WARNING: PAYMENTS_MODE=mock in production — orders are marked paid without charging.")
	}
	return c, nil
}

func (c Config) IsProd() bool        { return c.Env == "production" }
func (c Config) StripeEnabled() bool { return c.StripeSecretKey != "" }

func get(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func dur(k string, def time.Duration) time.Duration {
	if v := os.Getenv(k); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
		if n, err := strconv.Atoi(v); err == nil {
			return time.Duration(n) * time.Second
		}
	}
	return def
}

func list(k, def string) []string {
	raw := get(k, def)
	var out []string
	for _, s := range strings.Split(raw, ",") {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}
