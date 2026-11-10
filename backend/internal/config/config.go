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

	CronSecret       string
	IdealPostcodesKey string // optional UK address lookup provider
}

func Load() (Config, error) {
	c := Config{
		Env:             get("APP_ENV", "development"),
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
		EmailFrom:    get("EMAIL_FROM", "Celestial Oud <noreply@celestialoud.co.uk>"),
		AdminEmails:  list("ADMIN_EMAILS", ""),

		CronSecret:        os.Getenv("CRON_SECRET"),
		IdealPostcodesKey: os.Getenv("IDEAL_POSTCODES_KEY"),
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
	if len(c.JWTSecret) < 16 && c.Env == "production" {
		return c, fmt.Errorf("JWT_SECRET must be at least 16 characters")
	}
	return c, nil
}

func (c Config) IsProd() bool       { return c.Env == "production" }
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
