package components_test

import (
	"testing"

	"github.com/homeadmin/internal/templates/components"
)

// Table-driven coverage of nav section matching: exact match for /dashboard,
// prefix-with-boundary match for section roots so nested routes
// (/expenses/new) stay highlighted without false positives (/expensesfoo).
func TestIsActive(t *testing.T) {
	tests := []struct {
		name       string
		activePath string
		section    string
		want       bool
	}{
		{name: "dashboard exact match", activePath: "/dashboard", section: "/dashboard", want: true},
		{name: "dashboard is exact only", activePath: "/dashboard/other", section: "/dashboard", want: false},
		{name: "dashboard not active on other page", activePath: "/expenses", section: "/dashboard", want: false},
		{name: "expenses root match", activePath: "/expenses", section: "/expenses", want: true},
		{name: "expenses nested route keeps section active", activePath: "/expenses/new", section: "/expenses", want: true},
		{name: "savings edit route", activePath: "/savings/new", section: "/savings", want: true},
		{name: "household root", activePath: "/household", section: "/household", want: true},
		{name: "admin root", activePath: "/admin", section: "/admin", want: true},
		{name: "prefix must respect boundary", activePath: "/expensesfoo", section: "/expenses", want: false},
		{name: "root path activates nothing", activePath: "/", section: "/expenses", want: false},
		{name: "different section does not match", activePath: "/savings", section: "/expenses", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := components.IsActive(tt.activePath, tt.section); got != tt.want {
				t.Errorf("IsActive(%q, %q) = %v, want %v", tt.activePath, tt.section, got, tt.want)
			}
		})
	}
}
