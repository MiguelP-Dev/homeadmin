package i18n

import (
	"fmt"
	"strings"
	"time"

	"github.com/homeadmin/internal/database"
)

// esMonths holds the Spanish month names in lowercase, following the RAE
// convention for dates ("2 de enero de 2006"). Go's time.Format month names
// are English-only, so the table lives here (design D7).
var esMonths = [...]string{
	"enero", "febrero", "marzo", "abril", "mayo", "junio",
	"julio", "agosto", "septiembre", "octubre", "noviembre", "diciembre",
}

// groupDigits inserts sep every three digits of the integer part.
func groupDigits(s, sep string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteString(sep)
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// FormatCurrency renders v as USD with locale grouping: en "$1,234.56",
// es "$1.234,56". Currency stays USD per REQ-I18N-5.
func FormatCurrency(lang string, v float64) string {
	amount := fmt.Sprintf("%.2f", v)
	neg := strings.HasPrefix(amount, "-")
	if neg {
		amount = amount[1:]
	}
	intPart, frac, ok := strings.Cut(amount, ".")
	if !ok {
		frac = "00"
	}
	sign := ""
	if neg {
		sign = "-"
	}
	if lang == "es" {
		return sign + "$" + groupDigits(intPart, ".") + "," + frac
	}
	return sign + "$" + groupDigits(intPart, ",") + "." + frac
}

// FormatDate renders t with locale conventions: en "Jan 2, 2006",
// es "2 de enero de 2006" (design D7).
func FormatDate(lang string, t time.Time) string {
	if lang == "es" {
		return fmt.Sprintf("%d de %s de %d", t.Day(), esMonths[t.Month()-1], t.Year())
	}
	return t.Format("Jan 2, 2006")
}

// MonthName returns the localized month name (en "January", es "enero") for
// dashboard labels.
func MonthName(lang string, t time.Time) string {
	if lang == "es" {
		return esMonths[t.Month()-1]
	}
	return t.Month().String()
}

// CategoryLabel translates a stored category for display. Unknown stored
// values pass through unchanged — display-only translation, stored values
// are never modified (REQ-I18N-5).
func CategoryLabel(lang, category string) string {
	key := "category." + strings.ToLower(strings.ReplaceAll(category, " ", "_"))
	if got := T(lang, key); got != key {
		return got
	}
	return category
}

// VisibilityLabel translates a stored visibility value for display. Unknown
// values pass through unchanged (REQ-I18N-5).
func VisibilityLabel(lang string, v database.VisibilityType) string {
	key := "visibility." + string(v)
	if got := T(lang, key); got != key {
		return got
	}
	return string(v)
}

// TypeLabel translates a transaction type (income/expense) for display.
func TypeLabel(lang, txType string) string {
	key := "expense." + txType
	if got := T(lang, key); got != key {
		return got
	}
	return txType
}
