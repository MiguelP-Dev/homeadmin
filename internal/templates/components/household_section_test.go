package components_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/homeadmin/internal/database"
	"github.com/homeadmin/internal/services"
	"github.com/homeadmin/internal/templates/components"
)

func mustRenderHouseholdSection(block services.HouseholdBlock, lang string) string {
	buf := &bytes.Buffer{}
	err := components.HouseholdTransactionsSection(block, lang).Render(context.Background(), buf)
	if err != nil {
		panic(err)
	}
	return buf.String()
}

func TestHouseholdSection_NoExpensesEmptyStateTranslated(t *testing.T) {
	block := services.HouseholdBlock{Household: database.Household{Name: "My Family"}}
	tests := []struct {
		lang string
		want string
	}{
		{"en", "No expenses yet"},
		{"es", "Aún no hay gastos"},
	}
	for _, tt := range tests {
		t.Run(tt.lang, func(t *testing.T) {
			output := mustRenderHouseholdSection(block, tt.lang)
			if !strings.Contains(output, tt.want) {
				t.Errorf("lang %q: expected %q in household section output", tt.lang, tt.want)
			}
		})
	}
}
