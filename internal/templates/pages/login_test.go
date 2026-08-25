package pages_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/homeadmin/internal/templates/pages"
)

func renderLogin(csrfToken string, errorMsg string, lang string) string {
	buf := &bytes.Buffer{}
	err := pages.Login(csrfToken, errorMsg, lang).Render(context.Background(), buf)
	if err != nil {
		panic(err)
	}
	return buf.String()
}

func TestLogin_ContainsForm(t *testing.T) {
	output := renderLogin("tok", "", "en")
	if !strings.Contains(output, "<form") {
		t.Error("expected <form> in login output")
	}
}

func TestLogin_HasEmailField(t *testing.T) {
	output := renderLogin("tok", "", "en")
	if !strings.Contains(output, `name="email"`) {
		t.Error("expected email field in login form")
	}
}

func TestLogin_HasPasswordField(t *testing.T) {
	output := renderLogin("tok", "", "en")
	if !strings.Contains(output, `name="password"`) {
		t.Error("expected password field in login form")
	}
}

func TestLogin_HasCSRFField(t *testing.T) {
	output := renderLogin("csrf-token-abc", "", "en")
	if !strings.Contains(output, `value="csrf-token-abc"`) {
		t.Error("expected CSRF token hidden input with correct value")
	}
}

func TestLogin_SubmitsToLogin(t *testing.T) {
	output := renderLogin("tok", "", "en")
	if !strings.Contains(output, `action="/login"`) {
		t.Error("expected form to submit to /login")
	}
}

func TestLogin_HasRegisterLink(t *testing.T) {
	output := renderLogin("tok", "", "en")
	if !strings.Contains(output, "/register") {
		t.Error("expected link to /register in login page")
	}
}

func TestLogin_ShowsErrorWhenProvided(t *testing.T) {
	output := renderLogin("tok", "Invalid credentials", "en")
	if !strings.Contains(output, "Invalid credentials") {
		t.Error("expected error message to appear in output")
	}
}

func TestLogin_HidesErrorWhenEmpty(t *testing.T) {
	output := renderLogin("tok", "", "en")
	if strings.Contains(output, "text-red-500") {
		t.Error("expected no error message div when errorMsg is empty")
	}
}

func TestLogin_SubmitButton(t *testing.T) {
	output := renderLogin("tok", "", "en")
	if !strings.Contains(output, `type="submit"`) {
		t.Error("expected submit button in login form")
	}
}

// Triangulation: different CSRF tokens pass through correctly

func TestLogin_DifferentCSRFTokens(t *testing.T) {
	output1 := renderLogin("token-one", "", "en")
	output2 := renderLogin("token-two", "", "en")
	if !strings.Contains(output1, `value="token-one"`) {
		t.Error("first token not found in first render")
	}
	if !strings.Contains(output2, `value="token-two"`) {
		t.Error("second token not found in second render")
	}
	if strings.Contains(output1, "token-two") {
		t.Error("first render should not contain second token")
	}
}

func TestLogin_DifferentErrorMessages(t *testing.T) {
	output1 := renderLogin("tok", "Invalid credentials", "en")
	output2 := renderLogin("tok", "Account locked", "en")
	if !strings.Contains(output1, "Invalid credentials") {
		t.Error("first error message not found")
	}
	if !strings.Contains(output2, "Account locked") {
		t.Error("second error message not found")
	}
}

// English copy renders by default (lang=en).

func TestLogin_RendersEnglishLabels(t *testing.T) {
	output := renderLogin("tok", "", "en")
	// Note: templ escapes the apostrophe in "Don't" (&#39;), so assertions
	// avoid quoting the apostrophe directly.
	for _, want := range []string{"Login", "Email:", "Password:", "Register"} {
		if !strings.Contains(output, want) {
			t.Errorf("expected English label %q in login output", want)
		}
	}
	if !strings.Contains(output, "have an account? Register") {
		t.Error("expected English register prompt in login output")
	}
}

// Spanish labels when lang=es (root-cause regression: the page previously
// hardcoded English regardless of language).

func TestLogin_RendersSpanishLabels(t *testing.T) {
	output := renderLogin("tok", "", "es")
	for _, want := range []string{"Iniciar sesión", "Correo electrónico:", "Contraseña:", "¿No tienes una cuenta? Regístrate"} {
		if !strings.Contains(output, want) {
			t.Errorf("expected Spanish label %q in login output", want)
		}
	}
	if strings.Contains(output, ">Login</h1>") || strings.Contains(output, ">Login</button>") {
		t.Error("login page still renders English heading/button with lang=es")
	}
}

// Unknown languages fall back to English.
func TestLogin_UnknownLang_FallsBackToEnglish(t *testing.T) {
	output := renderLogin("tok", "", "fr")
	if !strings.Contains(output, "Login") {
		t.Error("expected English fallback for unknown lang")
	}
}
