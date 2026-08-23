package pages_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/homeadmin/internal/database"
	"github.com/homeadmin/internal/templates/pages"
)

func mustRenderExpenseForm(csrfToken, formAction, submitLabel, errorMsg string, values pages.ExpenseFormValues, lang string) string {
	buf := &bytes.Buffer{}
	err := pages.ExpenseForm(csrfToken, formAction, submitLabel, errorMsg, values, lang).Render(context.Background(), buf)
	if err != nil {
		panic(err)
	}
	return buf.String()
}

func TestExpenseForm_CreateVariantHasCsrfAndAction(t *testing.T) {
	output := mustRenderExpenseForm("tok123", "/expenses", "Create Expense", "", pages.ExpenseFormValues{}, "en")
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
	}, "en")
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
	output := mustRenderExpenseForm("tok", "/expenses", "Create Expense", "description is required", pages.ExpenseFormValues{}, "en")
	if !strings.Contains(output, "description is required") {
		t.Error("expected the error message rendered above the form")
	}
}

func TestExpenseForm_TranslatedStrings(t *testing.T) {
	tests := []struct {
		lang string
		want []string
	}{
		{"en", []string{
			"Type:", "Description:", "Amount:", "Category:", "Date:",
			"Visibility:", "Fixed expense", "Back to Expenses",
			"Groceries", // category option rendered through i18n.CategoryLabel
		}},
		{"es", []string{
			"Tipo:", "Descripción:", "Monto:", "Categoría:", "Fecha:",
			"Visibilidad:", "Gasto fijo", "Volver a gastos",
			"Comer fuera",
		}},
	}
	for _, tt := range tests {
		t.Run(tt.lang, func(t *testing.T) {
			output := mustRenderExpenseForm("tok", "/expenses", "Create Expense", "", pages.ExpenseFormValues{}, tt.lang)
			output = strings.ReplaceAll(output, "&amp;", "&")
			for _, want := range tt.want {
				if !strings.Contains(output, want) {
					t.Errorf("lang %q: expected %q in expense form output", tt.lang, want)
				}
			}
		})
	}
}

func TestExpenseForm_TranslatedVisibilityOptions(t *testing.T) {
	tests := []struct {
		lang string
		want []string
	}{
		{"en", []string{"Visible & editable", "Visible only", "Hidden & private"}},
		{"es", []string{"Visible y editable", "Solo visible", "Oculto y privado"}},
	}
	for _, tt := range tests {
		t.Run(tt.lang, func(t *testing.T) {
			output := mustRenderExpenseForm("tok", "/expenses", "Create Expense", "", pages.ExpenseFormValues{}, tt.lang)
			output = strings.ReplaceAll(output, "&amp;", "&")
			for _, want := range tt.want {
				if !strings.Contains(output, want) {
					t.Errorf("lang %q: expected visibility option %q in expense form output", tt.lang, want)
				}
			}
		})
	}
}
