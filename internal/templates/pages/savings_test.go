package pages_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/homeadmin/internal/database"
	"github.com/homeadmin/internal/templates/pages"
)

func mustRenderSavings(savings []database.Savings, total float64, lang string) string {
	buf := &bytes.Buffer{}
	err := pages.Savings(savings, total, lang, "test-csrf").Render(context.Background(), buf)
	if err != nil {
		panic(err)
	}
	return buf.String()
}

func TestSavings_ShowsTitleAndTotal(t *testing.T) {
	output := mustRenderSavings([]database.Savings{}, 1250.50, "en")
	if !strings.Contains(output, "Savings") {
		t.Error("expected 'Savings' title in savings page output")
	}
	if !strings.Contains(output, "Total Saved") {
		t.Error("expected 'Total Saved' label in savings page output")
	}
}

func TestSavings_TranslatedStrings(t *testing.T) {
	tests := []struct {
		lang string
		want []string
	}{
		{"en", []string{
			"Savings", "Add Savings", "Total Saved",
			"Are you sure you want to delete this savings entry?",
			`aria-label="Delete"`,
		}},
		{"es", []string{
			"Ahorros", "Agregar Ahorro", "Total Ahorrado",
			"¿Estás seguro de que quieres eliminar este ahorro?",
			`aria-label="Eliminar"`,
		}},
	}
	for _, tt := range tests {
		t.Run(tt.lang, func(t *testing.T) {
			output := mustRenderSavings([]database.Savings{
				{ID: 7, Description: "Vacation fund", Amount: 500, Target: 2000},
			}, 500, tt.lang)
			for _, want := range tt.want {
				if !strings.Contains(output, want) {
					t.Errorf("lang %q: expected %q in savings output", tt.lang, want)
				}
			}
		})
	}
}

func TestSavings_DeleteConfirmUsesTranslatedAttribute(t *testing.T) {
	tests := []struct {
		lang string
		want string
	}{
		{"en", "Are you sure you want to delete this savings entry?"},
		{"es", "¿Estás seguro de que quieres eliminar este ahorro?"},
	}
	for _, tt := range tests {
		t.Run(tt.lang, func(t *testing.T) {
			output := mustRenderSavings([]database.Savings{
				{ID: 7, Description: "Vacation fund", Amount: 500, Target: 2000},
			}, 500, tt.lang)
			if !strings.Contains(output, `onclick="return confirm(`) {
				t.Errorf("lang %q: expected a real onclick confirm attribute in savings output", tt.lang)
			}
			if !strings.Contains(output, tt.want) {
				t.Errorf("lang %q: expected %q in onclick confirm attribute", tt.lang, tt.want)
			}
			if strings.Contains(output, `attributes="map[`) {
				t.Errorf("lang %q: expected no stringified attributes map in output", tt.lang)
			}
			if strings.Contains(output, "Are you sure?") {
				t.Error("expected legacy hardcoded 'Are you sure?' confirm to be replaced")
			}
		})
	}
}

func TestSavings_EmptyState(t *testing.T) {
	tests := []struct {
		lang string
		want string
	}{
		{"en", "No savings yet"},
		{"es", "Sin ahorros aún"},
	}
	for _, tt := range tests {
		t.Run(tt.lang, func(t *testing.T) {
			output := mustRenderSavings([]database.Savings{}, 0, tt.lang)
			if !strings.Contains(output, tt.want) {
				t.Errorf("lang %q: expected %q empty state in savings output", tt.lang, tt.want)
			}
		})
	}
}
