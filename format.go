package main

import (
	"fmt"
	"math"
	"strings"
)

var currencySymbols = map[string]string{
	"usd": "$", "eur": "€", "gbp": "£", "jpy": "¥", "aud": "A$", "cad": "C$", "chf": "CHF ", "aed": "AED ", "sar": "SAR ", "kwd": "KWD ",
}

// currencies offered in the menu, in order.
var currencies = []string{"usd", "eur", "gbp", "jpy", "aud", "cad", "chf", "aed", "sar", "kwd"}

// formatPrice gives "$91,234" (no cents at Bitcoin prices) or "$1,234.56" below 1,000.
func formatPrice(price float64, currency string) string {
	sym, ok := currencySymbols[currency]
	if !ok {
		sym = strings.ToUpper(currency) + " "
	}
	decimals := 0
	if price < 1000 {
		decimals = 2
	}
	return sym + groupThousands(price, decimals)
}

func groupThousands(v float64, decimals int) string {
	neg := v < 0
	s := fmt.Sprintf("%.*f", decimals, math.Abs(v))
	intPart, frac := s, ""
	if i := strings.IndexByte(s, '.'); i >= 0 {
		intPart, frac = s[:i], s[i:]
	}
	var b strings.Builder
	for i, r := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	out := b.String() + frac
	if neg {
		out = "-" + out
	}
	return out
}

// formatChange gives "▲ 1.23%" / "▼ 0.45%".
func formatChange(pct float64) string {
	arrow := "▲"
	if pct < 0 {
		arrow = "▼"
	}
	return fmt.Sprintf("%s %.2f%%", arrow, math.Abs(pct))
}

// menuBarTitle is the text shown in the macOS menu bar.
func menuBarTitle(q Quote, showChange bool) string {
	t := "₿ " + formatPrice(q.Price, q.Currency)
	if showChange && q.HasChange {
		t += " " + formatChange(q.Change24h)
	}
	return t
}
