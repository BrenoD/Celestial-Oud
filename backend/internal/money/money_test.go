package money

import "testing"

func TestGBP(t *testing.T) {
	cases := map[int]string{4500: "£45.00", 499: "£4.99", 5: "£0.05", -1250: "-£12.50"}
	for in, want := range cases {
		if got := GBP(in); got != want {
			t.Errorf("GBP(%d) = %s, want %s", in, got, want)
		}
	}
}

func TestPercent(t *testing.T) {
	if got := Percent(17400, 10); got != 1740 {
		t.Errorf("got %d", got)
	}
	if got := Percent(999, 10); got != 100 { // 99.9 rounds to 100
		t.Errorf("got %d", got)
	}
}
