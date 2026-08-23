package pages_test

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/homeadmin/internal/database"
	"github.com/homeadmin/internal/templates/pages"
)

func mustRenderExpenses(expenses []database.Expense, lang string) string {
	buf := &bytes.Buffer{}
	err := pages.Expenses(expenses, lang, "test-csrf").Render(context.Background(), buf)
	if err != nil {
		panic(err)
	}
	return buf.String()
}

func TestExpenses_ShowsTitle(t *testing.T) {
	output := mustRenderExpenses([]database.Expense{
		{ID: 1, Description: "Rent", Amount: 1500, Category: "rent", Date: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)},
	}, "en")
	if !strings.Contains(output, "Expenses") {
		t.Error("expected 'Expenses' title in expenses page output")
	}
}

func TestExpenses_ShowsExpenseDescription(t *testing.T) {
	output := mustRenderExpenses([]database.Expense{
		{ID: 1, Description: "Groceries", Amount: 85.50, Category: "food", Date: time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC)},
	}, "en")
	if !strings.Contains(output, "Groceries") {
		t.Error("expected expense description 'Groceries' in output")
	}
}

func TestExpenses_ShowsMultipleExpenses(t *testing.T) {
	output := mustRenderExpenses([]database.Expense{
		{ID: 1, Description: "Rent", Amount: 1500, Category: "rent", Date: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)},
		{ID: 2, Description: "Internet", Amount: 80, Category: "utilities", Date: time.Date(2026, 7, 5, 0, 0, 0, 0, time.UTC)},
		{ID: 3, Description: "Gas", Amount: 45, Category: "transport", Date: time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC)},
	}, "en")
	if !strings.Contains(output, "Rent") {
		t.Error("expected 'Rent' in list")
	}
	if !strings.Contains(output, "Internet") {
		t.Error("expected 'Internet' in list")
	}
	if !strings.Contains(output, "Gas") {
		t.Error("expected 'Gas' in list")
	}
}

func TestExpenses_ShowsEmptyState(t *testing.T) {
	output := mustRenderExpenses([]database.Expense{}, "en")
	if !strings.Contains(output, "No expenses yet") {
		t.Error("expected 'No expenses yet' empty state when no expenses")
	}
}

func TestExpenses_HasCreateLink(t *testing.T) {
	output := mustRenderExpenses([]database.Expense{
		{ID: 1, Description: "Test", Amount: 10, Category: "other", Date: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)},
	}, "en")
	if !strings.Contains(output, "Create Expense") {
		t.Error("expected 'Create Expense' link/button in expenses page")
	}
}

// Triangulation: multiple expenses show correctly

func TestExpenses_Triangulation_NoExpensesVsSome(t *testing.T) {
	emptyOutput := mustRenderExpenses([]database.Expense{}, "en")
	fullOutput := mustRenderExpenses([]database.Expense{
		{ID: 1, Description: "Rent", Amount: 1500, Category: "rent", Date: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)},
	}, "en")

	if !strings.Contains(emptyOutput, "No expenses yet") {
		t.Error("empty list should show 'No expenses yet'")
	}
	if strings.Contains(fullOutput, "No expenses yet") {
		t.Error("non-empty list should not show 'No expenses yet'")
	}
	if !strings.Contains(fullOutput, "Rent") {
		t.Error("non-empty list should show expenses")
	}
}

func TestExpenses_Triangulation_DifferentExpenses(t *testing.T) {
	list1 := mustRenderExpenses([]database.Expense{
		{ID: 1, Description: "Item A", Amount: 10, Category: "food", Date: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)},
	}, "en")
	list2 := mustRenderExpenses([]database.Expense{
		{ID: 2, Description: "Item B", Amount: 20, Category: "rent", Date: time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC)},
	}, "en")

	if !strings.Contains(list1, "Item A") {
		t.Error("first list should contain 'Item A'")
	}
	if !strings.Contains(list2, "Item B") {
		t.Error("second list should contain 'Item B'")
	}
	if strings.Contains(list1, "Item B") {
		t.Error("first list should not contain 'Item B'")
	}
}

func TestExpenses_TranslatedStrings(t *testing.T) {
	tests := []struct {
		lang string
		want []string
	}{
		{"en", []string{"Expenses", "Create Expense", "No expenses yet"}},
		{"es", []string{"Gastos", "Crear gasto", "Aún no hay gastos"}},
	}
	for _, tt := range tests {
		t.Run(tt.lang, func(t *testing.T) {
			output := mustRenderExpenses([]database.Expense{}, tt.lang)
			for _, want := range tt.want {
				if !strings.Contains(output, want) {
					t.Errorf("lang %q: expected %q in expenses output", tt.lang, want)
				}
			}
		})
	}
}
