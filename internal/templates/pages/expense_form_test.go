package pages_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/homeadmin/internal/database"
	"github.com/homeadmin/internal/templates/pages"
)

func mustRenderExpenseForm(csrfToken, formAction, submitLabel, errorMsg string, values pages.ExpenseFormValues) string {
	buf := &bytes.Buffer{}
	err := pages.ExpenseForm(csrfToken, formAction, submitLabel, errorMsg, values, "en").Render(context.Background(), buf)
	if err != nil {
		panic(err)
	}
	return buf.String()
}

func TestExpenseForm_CreateVariantHasCsrfAndAction(t *testing.T) {
	output := mustRenderExpenseForm("tok123", "/expenses", "Create Expense", "", pages.ExpenseFormValues{})
	if !strings.Contains(output, `name="csrf" value="tok123"`) {
		t.Error("expected csrf hidden input with the token")
	}
	if !strings.Contains(output, `action="/expenses"`) {
		t.Error("expected create form action /expenses")
	}
	if !strings.Contains(output, "Create Expense") {
		t.Error("expected create heading and submit label")
	}
}

func TestExpenseForm_EditVariantPrefillsValues(t *testing.T) {
	output := mustRenderExpenseForm("tok456", "/expenses/7/update", "Update Expense", "", pages.ExpenseFormValues{
		Amount:      "85.50",
		Description: "Groceries",
		Category:    "Groceries",
		Date:        "2026-07-15",
		Visibility:  database.VisibleOnly,
		IsFixed:     true,
	})
	if !strings.Contains(output, `action="/expenses/7/update"`) {
		t.Error("expected edit form action /expenses/7/update")
	}
	if !strings.Contains(output, `value="Groceries"`) {
		t.Error("expected pre-filled description")
	}
	if !strings.Contains(output, `value="85.50"`) {
		t.Error("expected pre-filled amount")
	}
	if !strings.Contains(output, `value="2026-07-15"`) {
		t.Error("expected pre-filled date")
	}
	if !strings.Contains(output, `value="visible_only" selected`) {
		t.Error("expected visible_only option selected")
	}
	if !strings.Contains(output, `name="isFixed" value="true" checked`) {
		t.Error("expected isFixed checkbox checked")
	}
}

func TestExpenseForm_ShowsError(t *testing.T) {
	output := mustRenderExpenseForm("tok", "/expenses", "Create Expense", "description is required", pages.ExpenseFormValues{})
	if !strings.Contains(output, "description is required") {
		t.Error("expected the error message rendered above the form")
	}
}
