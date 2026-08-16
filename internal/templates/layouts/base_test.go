package layouts_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/homeadmin/internal/templates/layouts"
)

func renderBase(t *testing.T, title, csrfToken, email string, isAdmin bool, lang string, activePath string) string {
	t.Helper()
	buf := &bytes.Buffer{}
	err := layouts.Base(title, csrfToken, email, isAdmin, lang, activePath).Render(context.Background(), buf)
	if err != nil {
		t.Fatalf("failed to render Base layout: %v", err)
	}
	return buf.String()
}

func TestBase_IncludesToastContainer(t *testing.T) {
	output := renderBase(t, "Test Page", "csrf-token-123", "", false, "en", "/")
	if !strings.Contains(output, `id="toast-container"`) {
		t.Error("expected toast-container in base layout output")
	}
}

func TestBase_IncludesCSRFToken(t *testing.T) {
	output := renderBase(t, "Test Page", "csrf-token-123", "", false, "en", "/")
	if !strings.Contains(output, "csrf-token-123") {
		t.Error("expected csrf token in base layout output")
	}
}

func TestBase_IncludesTitle(t *testing.T) {
	output := renderBase(t, "My Dashboard", "tok", "", false, "en", "/")
	if !strings.Contains(output, "<title>My Dashboard</title>") {
		t.Errorf("expected title in output, got: %s", output)
	}
}

func TestBase_IncludesHtmxScript(t *testing.T) {
	output := renderBase(t, "Test", "tok", "", false, "en", "/")
	if !strings.Contains(output, "/static/js/htmx.min.js") {
		t.Error("expected vendored HTMX script in base layout")
	}
}

// New tests for nav integration

func TestBase_IncludesNav(t *testing.T) {
	output := renderBase(t, "Test", "tok", "", false, "en", "/")
	if !strings.Contains(output, "<nav") {
		t.Error("expected <nav> element in base layout output")
	}
}

func TestBase_IncludesNavWithUsername(t *testing.T) {
	output := renderBase(t, "Test", "tok", "testuser", false, "en", "/")
	if !strings.Contains(output, "testuser") {
		t.Error("expected username in base layout output when username is provided")
	}
}

// --- NEW WU-4 TESTS: lang attribute, theme script in head ---

// Base: html lang attribute reflects the lang param
func TestBase_HtmlLangAttribute_DefaultEn(t *testing.T) {
	output := renderBase(t, "Test", "tok", "", false, "en", "/")
	if !strings.Contains(output, `lang="en"`) {
		t.Error("expected lang=\"en\" on <html> element for en")
	}
}

func TestBase_HtmlLangAttribute_Spanish(t *testing.T) {
	output := renderBase(t, "Test", "tok", "", false, "es", "/")
	if !strings.Contains(output, `lang="es"`) {
		t.Error("expected lang=\"es\" on <html> element for es")
	}
}

// Base: theme script is in <head>, NOT at bottom of <body>
func TestBase_ThemeScriptInHead(t *testing.T) {
	output := renderBase(t, "Test", "tok", "", false, "en", "/")

	headEnd := strings.Index(output, "</head>")
	bodyEnd := strings.Index(output, "</body>")

	if headEnd == -1 || bodyEnd == -1 {
		t.Fatal("expected both </head> and </body> tags in output")
	}

	// The theme script (localStorage.getItem('theme')) should appear BEFORE </head>
	themeScriptMarker := "localStorage.getItem('theme')"
	headSection := output[:headEnd]
	bodySection := output[headEnd:bodyEnd]

	if !strings.Contains(headSection, themeScriptMarker) {
		t.Error("expected theme script (localStorage.getItem('theme')) inside <head>")
	}

	// There should be NO duplicate theme script in <body>
	bodyOccurrences := strings.Count(bodySection, themeScriptMarker)
	if bodyOccurrences > 0 {
		t.Errorf("expected NO theme script in <body>, found %d occurrence(s)", bodyOccurrences)
	}
}

// Base: passes lang and activePath to Nav component
func TestBase_PassesLangToNav(t *testing.T) {
	output := renderBase(t, "Test", "tok", "user@test.com", false, "es", "/")
	// Spanish nav labels should appear since lang="es" is passed through
	if !strings.Contains(output, "Gastos") {
		t.Error("expected Spanish nav label 'Gastos' when lang='es' is passed to Base")
	}
}

// Base: activePath is passed through (check that nav receives it)
func TestBase_PassesActivePathToNav(t *testing.T) {
	output := renderBase(t, "Test", "tok", "user@test.com", false, "en", "/dashboard")
	if !strings.Contains(output, `aria-current="page"`) {
		t.Error("expected aria-current=\"page\" when activePath=/dashboard is passed")
	}
}
