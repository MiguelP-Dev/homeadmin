package components_test

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/homeadmin/internal/database"
	"github.com/homeadmin/internal/templates/components"
)

func mustRenderExpenseCard(e database.Expense) string {
	buf := &bytes.Buffer{}
	err := components.ExpenseCard(e).Render(context.Background(), buf)
	if err != nil {
		panic(err)
	}
	return buf.String()
}

func TestExpenseCard_ShowsDescription(t *testing.T) {
	e := database.Expense{ID: 1, Description: "Groceries", Amount: 50.00, Category: "food", Date: time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC)}
	output := mustRenderExpenseCard(e)
	if !strings.Contains(output, "Groceries") {
		t.Error("expected description 'Groceries' in expense card output")
	}
}

func TestExpenseCard_ShowsFormattedAmount(t *testing.T) {
	e := database.Expense{ID: 1, Description: "Rent", Amount: 1500.50, Category: "rent", Date: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)}
	output := mustRenderExpenseCard(e)
	if !strings.Contains(output, "$1,500.50") && !strings.Contains(output, "1500.50") {
		t.Error("expected amount 1500.50 formatted in expense card")
	}
}

func TestExpenseCard_ShowsCategory(t *testing.T) {
	e := database.Expense{ID: 1, Description: "Internet", Amount: 80.00, Category: "utilities", Date: time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC)}
	output := mustRenderExpenseCard(e)
	if !strings.Contains(output, "utilities") {
		t.Error("expected category 'utilities' in expense card output")
	}
}

func TestExpenseCard_ShowsFormattedDate(t *testing.T) {
	e := database.Expense{ID: 1, Description: "Dinner", Amount: 45.00, Category: "food", Date: time.Date(2026, 7, 4, 0, 0, 0, 0, time.UTC)}
	output := mustRenderExpenseCard(e)
	if !strings.Contains(output, "Jul") {
		t.Error("expected formatted date (month abbreviation) in expense card")
	}
	if !strings.Contains(output, "4") {
		t.Error("expected day number in expense card date")
	}
}

func TestExpenseCard_ShowsVisibility(t *testing.T) {
	e := database.Expense{ID: 1, Description: "Gift", Amount: 30.00, Category: "other", Date: time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC), Visibility: database.HiddenPrivate}
	output := mustRenderExpenseCard(e)
	if !strings.Contains(output, "hidden") && !strings.Contains(output, "private") {
		t.Error("expected visibility indicator for hidden_private expense")
	}
}

func TestExpenseCard_ShowsVisibleOnly(t *testing.T) {
	e := database.Expense{ID: 1, Description: "Bill", Amount: 100.00, Category: "utilities", Date: time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC), Visibility: database.VisibleOnly}
	output := mustRenderExpenseCard(e)
	if !strings.Contains(output, "yellow") && !strings.Contains(output, "visible") && !strings.Contains(output, "read") {
		t.Error("expected visibility indicator for visible_only expense")
	}
}

func TestExpenseCard_ShowsVisibleEditable(t *testing.T) {
	e := database.Expense{ID: 1, Description: "Groceries", Amount: 50.00, Category: "food", Date: time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC), Visibility: database.VisibleEditable}
	output := mustRenderExpenseCard(e)
	if !strings.Contains(output, "Editable") && !strings.Contains(output, "editable") {
		t.Error("expected visibility indicator for visible_editable expense")
	}
}

// Triangulation: different expenses produce different output

func TestExpenseCard_Triangulation_DifferentDescriptions(t *testing.T) {
	e1 := database.Expense{ID: 1, Description: "Groceries", Amount: 50.00, Category: "food", Date: time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC)}
	e2 := database.Expense{ID: 2, Description: "Rent", Amount: 1500.00, Category: "rent", Date: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)}
	out1 := mustRenderExpenseCard(e1)
	out2 := mustRenderExpenseCard(e2)
	if !strings.Contains(out1, "Groceries") {
		t.Error("first expense should show 'Groceries'")
	}
	if !strings.Contains(out2, "Rent") {
		t.Error("second expense should show 'Rent'")
	}
	if strings.Contains(out1, "1500") {
		t.Error("first expense should not contain second expense's amount")
	}
}

func TestExpenseCard_Triangulation_DifferentCategories(t *testing.T) {
	e1 := database.Expense{ID: 1, Description: "Gas", Amount: 40.00, Category: "transport", Date: time.Date(2026, 7, 5, 0, 0, 0, 0, time.UTC)}
	e2 := database.Expense{ID: 2, Description: "Medicine", Amount: 25.00, Category: "health", Date: time.Date(2026, 7, 8, 0, 0, 0, 0, time.UTC)}
	out1 := mustRenderExpenseCard(e1)
	out2 := mustRenderExpenseCard(e2)
	if !strings.Contains(out1, "transport") {
		t.Error("first expense should show category 'transport'")
	}
	if !strings.Contains(out2, "health") {
		t.Error("second expense should show category 'health'")
	}
}
