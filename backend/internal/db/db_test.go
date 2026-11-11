package db

import "testing"

func TestUsesTransactionPooler(t *testing.T) {
	cases := []struct {
		host string
		port uint16
		want bool
	}{
		{"aws-1-eu-west-3.pooler.supabase.com", 6543, true},
		{"aws-1-eu-west-3.pooler.supabase.com", 5432, true},
		{"db.bualtfdrfvsnnydwtmgz.supabase.co", 5432, false},
		{"localhost", 5432, false},
		{"localhost", 6543, true},
	}
	for _, c := range cases {
		if got := UsesTransactionPooler(c.host, c.port); got != c.want {
			t.Errorf("%s:%d = %v, want %v", c.host, c.port, got, c.want)
		}
	}
	t.Setenv("DB_TRANSACTION_POOLER", "false")
	if UsesTransactionPooler("aws-1-eu-west-3.pooler.supabase.com", 6543) {
		t.Error("override false ignored")
	}
}
