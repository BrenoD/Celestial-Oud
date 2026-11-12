package httpx

import (
	"net/mail"
	"regexp"
	"strings"
	"unicode/utf8"
)

// NormEmail trims and lower-cases an address and checks it is a plain,
// syntactically valid mailbox (no display name), at most 254 bytes.
func NormEmail(s string) (string, bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" || len(s) > 254 {
		return s, false
	}
	a, err := mail.ParseAddress(s)
	if err != nil || a.Address != s || a.Name != "" {
		return s, false
	}
	at := strings.LastIndexByte(s, '@')
	if at < 1 || !strings.Contains(s[at+1:], ".") || strings.HasSuffix(s, ".") {
		return s, false
	}
	return s, true
}

// InvalidEmail is the shared 422 for bad addresses.
var InvalidEmail = Unprocessable("invalid_email", "Enter a valid email address.")

// Field length guard. Lengths are counted in characters, not bytes.
type Limit struct {
	Name  string
	Value string
	Max   int
}

// CheckLengths returns a 422 naming the first field over its limit.
func CheckLengths(fields ...Limit) error {
	for _, f := range fields {
		if utf8.RuneCountInString(f.Value) > f.Max {
			return Unprocessable("too_long", f.Name+" is too long (maximum "+itoa(f.Max)+" characters).")
		}
	}
	return nil
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

// UKPostcode accepts the standard UK formats (incl. GIR 0AA).
var UKPostcode = regexp.MustCompile(`(?i)^([A-Z]{1,2}[0-9][A-Z0-9]?\s?[0-9][A-Z]{2}|GIR\s?0AA)$`)
