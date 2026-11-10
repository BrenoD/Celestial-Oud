// Package money formats integer pence.
package money

import "fmt"

func GBP(pence int) string {
	sign := ""
	if pence < 0 {
		sign = "-"
		pence = -pence
	}
	return fmt.Sprintf("%s£%d.%02d", sign, pence/100, pence%100)
}

// Percent applies pct% to pence, rounding half up.
func Percent(pence, pct int) int {
	return (pence*pct + 50) / 100
}
