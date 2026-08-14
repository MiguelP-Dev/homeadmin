package i18n

import (
	"testing"
	"time"

	"github.com/homeadmin/internal/database"
)

// TestFormatCurrency verifies locale grouping: en "$1,234.56", es "$1.234,56"
// (design D7, REQ-I18N-5 — USD always).
func TestFormatCurrency(t *testing.T) {
	tests := []struct {
		name string
		lang string
		v    float64
		want string
	}{
		{"en thousands", "en", 1234.56, "$1,234.56"},
		{"es thousands", "es", 1234.56, "$1.234,56"},
		{"en small", "en", 9.5, "$9.50"},
		{"es small", "es", 9.5, "$9,50"},
		{"en millions", "en", 1234567.89, "$1,234,567.89"},
		{"es millions", "es", 1234567.89, "$1.234.567,89"},
		{"en zero", "en", 0, "$0.00"},
		{"es zero", "es", 0, "$0,00"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatCurrency(tt.lang, tt.v); got != tt.want {
				t.Errorf("FormatCurrency(%q, %v) = %q, want %q", tt.lang, tt.v, got, tt.want)
			}
		})
	}
}

// TestFormatDate verifies the design D7 date formats: en "Jan 2, 2006",
// es "2 de enero de 2006" via the esMonths table.
func TestFormatDate(t *testing.T) {
	d := time.Date(2006, time.January, 2, 15, 4, 5, 0, time.UTC)
	if got := FormatDate("en", d); got != "Jan 2, 2006" {
		t.Errorf("FormatDate(en) = %q, want %q", got, "Jan 2, 2006")
	}
	if got := FormatDate("es", d); got != "2 de enero de 2006" {
		t.Errorf("FormatDate(es) = %q, want %q", got, "2 de enero de 2006")
	}
}

// TestFormatDateOtherMonth triangulates the esMonths table with a different
// month and multi-digit day.
func TestFormatDateOtherMonth(t *testing.T) {
	d := time.Date(2026, time.December, 25, 0, 0, 0, 0, time.UTC)
	if got := FormatDate("es", d); got != "25 de diciembre de 2026" {
		t.Errorf("FormatDate(es) = %q, want %q", got, "25 de diciembre de 2026")
	}
	if got := FormatDate("en", d); got != "Dec 25, 2026" {
		t.Errorf("FormatDate(en) = %q, want %q", got, "Dec 25, 2026")
	}
}

// TestMonthName verifies the localized month name (dashboard label per
// design: es "enero", en "January").
func TestMonthName(t *testing.T) {
	jan := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	if got := MonthName("es", jan); got != "enero" {
		t.Errorf("MonthName(es, jan) = %q, want %q", got, "enero")
	}
	if got := MonthName("en", jan); got != "January" {
		t.Errorf("MonthName(en, jan) = %q, want %q", got, "January")
	}
	jun := time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)
	if got := MonthName("es", jun); got != "junio" {
		t.Errorf("MonthName(es, jun) = %q, want %q", got, "junio")
	}
}

// TestCategoryLabel verifies display-only category translation: stored values
// map to localized labels, unknown stored values pass through unchanged
// (REQ-I18N-5).
func TestCategoryLabel(t *testing.T) {
	tests := []struct {
		name     string
		lang     string
		category string
		want     string
	}{
		{"es rent", "es", "Rent", "Alquiler"},
		{"en rent", "en", "Rent", "Rent"},
		{"es multi-word", "es", "Dining Out", "Comer fuera"},
		{"es utilities", "es", "Utilities", "Servicios"},
		{"unknown passthrough", "es", "Legacy", "Legacy"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CategoryLabel(tt.lang, tt.category); got != tt.want {
				t.Errorf("CategoryLabel(%q, %q) = %q, want %q", tt.lang, tt.category, got, tt.want)
			}
		})
	}
}

// TestVisibilityLabel verifies display-only visibility translation with raw
// passthrough for unknown values.
func TestVisibilityLabel(t *testing.T) {
	tests := []struct {
		name       string
		lang       string
		visibility database.VisibilityType
		want       string
	}{
		{"es editable", "es", database.VisibleEditable, "Visible y editable"},
		{"en editable", "en", database.VisibleEditable, "Visible & editable"},
		{"es visible only", "es", database.VisibleOnly, "Solo visible"},
		{"es hidden", "es", database.HiddenPrivate, "Oculto y privado"},
		{"unknown passthrough", "es", database.VisibilityType("legacy_mode"), "legacy_mode"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := VisibilityLabel(tt.lang, tt.visibility); got != tt.want {
				t.Errorf("VisibilityLabel(%q, %q) = %q, want %q", tt.lang, tt.visibility, got, tt.want)
			}
		})
	}
}
