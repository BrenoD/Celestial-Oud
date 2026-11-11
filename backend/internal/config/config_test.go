package config

import "testing"

func TestPaymentsMode(t *testing.T) {
	cases := []struct {
		env, stripe, mode, want string
	}{
		{"development", "", "", "mock"},
		{"production", "", "", "disabled"},
		{"production", "", "mock", "mock"},
		{"production", "sk_test_x", "", "stripe"},
		{"production", "sk_test_x", "mock", "stripe"},
	}
	for _, c := range cases {
		t.Setenv("DATABASE_URL", "postgres://x")
		t.Setenv("JWT_SECRET", "0123456789abcdef0123456789abcdef")
		t.Setenv("APP_ENV", c.env)
		t.Setenv("STRIPE_SECRET_KEY", c.stripe)
		t.Setenv("PAYMENTS_MODE", c.mode)
		cfg, err := Load()
		if err != nil {
			t.Fatal(err)
		}
		if cfg.PaymentsMode != c.want {
			t.Errorf("env=%s stripe=%q mode=%q: got %s, want %s", c.env, c.stripe, c.mode, cfg.PaymentsMode, c.want)
		}
	}
}
