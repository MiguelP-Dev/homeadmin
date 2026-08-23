package pages_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/homeadmin/internal/templates/pages"
)

func mustRenderSavingsForm(csrfToken, formAction, submitLabel, errorMsg string, lang string) string {
	buf := &bytes.Buffer{}
	err := pages.SavingsForm(csrfToken, formAction, submitLabel, errorMsg, lang).Render(context.Background(), buf)
	if err != nil {
		panic(err)
	}
	return buf.String()
}

func TestSavingsForm_TranslatedStrings(t *testing.T) {
	tests := []struct {
		lang        string
		submitLabel string
		want        []string
	}{
		{"en", "Create Savings", []string{
			"Create Savings", "Description:", "Amount:", "Target:", "Back to Savings",
		}},
		{"es", "Crear Ahorro", []string{
			"Crear Ahorro", "Descripción:", "Monto:", "Meta:", "Volver a ahorros",
		}},
	}
	for _, tt := range tests {
		t.Run(tt.lang, func(t *testing.T) {
			output := mustRenderSavingsForm("tok", "/savings", tt.submitLabel, "", tt.lang)
			for _, want := range tt.want {
				if !strings.Contains(output, want) {
					t.Errorf("lang %q: expected %q in savings form output", tt.lang, want)
				}
			}
		})
	}
}
