package pages_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/homeadmin/internal/templates/pages"
)

func renderRegister(csrfToken string, errorMsg string) string {
	buf := &bytes.Buffer{}
	err := pages.Register(csrfToken, errorMsg).Render(context.Background(), buf)
	if err != nil {
		panic(err)
	}
	return buf.String()
}

func TestRegister_ContainsForm(t *testing.T) {
	output := renderRegister("tok", "")
	if !strings.Contains(output, "<form") {
		t.Error("expected <form> in register output")
	}
}

func TestRegister_HasEmailField(t *testing.T) {
	output := renderRegister("tok", "")
	if !strings.Contains(output, `name="email"`) {
		t.Error("expected email field in register form")
	}
}

func TestRegister_HasPasswordField(t *testing.T) {
	output := renderRegister("tok", "")
	if !strings.Contains(output, `name="password"`) {
		t.Error("expected password field in register form")
	}
}

func TestRegister_HasNameField(t *testing.T) {
	output := renderRegister("tok", "")
	if !strings.Contains(output, `name="name"`) {
		t.Error("expected name field in register form")
	}
}

func TestRegister_HasCSRFField(t *testing.T) {
	output := renderRegister("csrf-token-xyz", "")
	if !strings.Contains(output, `value="csrf-token-xyz"`) {
		t.Error("expected CSRF token hidden input with correct value")
	}
}

func TestRegister_SubmitsToRegister(t *testing.T) {
	output := renderRegister("tok", "")
	if !strings.Contains(output, `action="/register"`) {
		t.Error("expected form to submit to /register")
	}
}

func TestRegister_HasLoginLink(t *testing.T) {
	output := renderRegister("tok", "")
	if !strings.Contains(output, "/login") {
		t.Error("expected link to /login in register page")
	}
}

func TestRegister_PasswordHasMinLength(t *testing.T) {
	output := renderRegister("tok", "")
	if !strings.Contains(output, `minlength="8"`) {
		t.Error("expected password field to have minlength=\"8\"")
	}
}

func TestRegister_ShowsErrorWhenProvided(t *testing.T) {
	output := renderRegister("tok", "Email already registered")
	if !strings.Contains(output, "Email already registered") {
		t.Error("expected error message to appear in output")
	}
}

func TestRegister_HidesErrorWhenEmpty(t *testing.T) {
	output := renderRegister("tok", "")
	if strings.Contains(output, "text-red-500") {
		t.Error("expected no error message when errorMsg is empty")
	}
}

func TestRegister_SubmitButton(t *testing.T) {
	output := renderRegister("tok", "")
	if !strings.Contains(output, `type="submit"`) {
		t.Error("expected submit button in register form")
	}
}

// Triangulation: different CSRF tokens and error messages

func TestRegister_DifferentCSRFTokens(t *testing.T) {
	output1 := renderRegister("token-a", "")
	output2 := renderRegister("token-b", "")
	if !strings.Contains(output1, `value="token-a"`) {
		t.Error("first token not found")
	}
	if !strings.Contains(output2, `value="token-b"`) {
		t.Error("second token not found")
	}
}

func TestRegister_DifferentErrorMessages(t *testing.T) {
	output1 := renderRegister("tok", "Email already registered")
	output2 := renderRegister("tok", "Password too weak")
	if !strings.Contains(output1, "Email already registered") {
		t.Error("first error message not found")
	}
	if !strings.Contains(output2, "Password too weak") {
		t.Error("second error message not found")
	}
}
