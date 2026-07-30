package pages_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/homeadmin/internal/templates/pages"
)

func renderLogin(csrfToken string, errorMsg string) string {
	buf := &bytes.Buffer{}
	err := pages.Login(csrfToken, errorMsg).Render(context.Background(), buf)
	if err != nil {
		panic(err)
	}
	return buf.String()
}

func TestLogin_ContainsForm(t *testing.T) {
	output := renderLogin("tok", "")
	if !strings.Contains(output, "<form") {
		t.Error("expected <form> in login output")
	}
}

func TestLogin_HasEmailField(t *testing.T) {
	output := renderLogin("tok", "")
	if !strings.Contains(output, `name="email"`) {
		t.Error("expected email field in login form")
	}
}

func TestLogin_HasPasswordField(t *testing.T) {
	output := renderLogin("tok", "")
	if !strings.Contains(output, `name="password"`) {
		t.Error("expected password field in login form")
	}
}

func TestLogin_HasCSRFField(t *testing.T) {
	output := renderLogin("csrf-token-abc", "")
	if !strings.Contains(output, `value="csrf-token-abc"`) {
		t.Error("expected CSRF token hidden input with correct value")
	}
}

func TestLogin_SubmitsToLogin(t *testing.T) {
	output := renderLogin("tok", "")
	if !strings.Contains(output, `action="/login"`) {
		t.Error("expected form to submit to /login")
	}
}

func TestLogin_HasRegisterLink(t *testing.T) {
	output := renderLogin("tok", "")
	if !strings.Contains(output, "/register") {
		t.Error("expected link to /register in login page")
	}
}

func TestLogin_ShowsErrorWhenProvided(t *testing.T) {
	output := renderLogin("tok", "Invalid credentials")
	if !strings.Contains(output, "Invalid credentials") {
		t.Error("expected error message to appear in output")
	}
}

func TestLogin_HidesErrorWhenEmpty(t *testing.T) {
	output := renderLogin("tok", "")
	if strings.Contains(output, "text-red-500") {
		t.Error("expected no error message div when errorMsg is empty")
	}
}

func TestLogin_SubmitButton(t *testing.T) {
	output := renderLogin("tok", "")
	if !strings.Contains(output, `type="submit"`) {
		t.Error("expected submit button in login form")
	}
}

// Triangulation: different CSRF tokens pass through correctly

func TestLogin_DifferentCSRFTokens(t *testing.T) {
	output1 := renderLogin("token-one", "")
	output2 := renderLogin("token-two", "")
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
	output1 := renderLogin("tok", "Invalid credentials")
	output2 := renderLogin("tok", "Account locked")
	if !strings.Contains(output1, "Invalid credentials") {
		t.Error("first error message not found")
	}
	if !strings.Contains(output2, "Account locked") {
		t.Error("second error message not found")
	}
}
