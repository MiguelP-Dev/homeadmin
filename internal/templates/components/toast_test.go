package components_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/homeadmin/internal/templates/components"
)

func renderToast(t *testing.T) string {
	t.Helper()
	buf := &bytes.Buffer{}
	err := components.ToastContainer().Render(context.Background(), buf)
	if err != nil {
		t.Fatalf("failed to render ToastContainer: %v", err)
	}
	return buf.String()
}

func TestToastContainer_ContainsContainerID(t *testing.T) {
	output := renderToast(t)
	if !strings.Contains(output, `id="toast-container"`) {
		t.Errorf("expected toast-container id in output, got: %s", output)
	}
}

func TestToastContainer_ContainsScriptTag(t *testing.T) {
	output := renderToast(t)
	if !strings.Contains(output, "<script>") {
		t.Errorf("expected <script> tag in output, got: %s", output)
	}
}

func TestToastContainer_ScriptContainsHtmxListener(t *testing.T) {
	output := renderToast(t)
	if !strings.Contains(output, "htmx:afterRequest") {
		t.Errorf("expected htmx:afterRequest listener in script, got: %s", output)
	}
}

func TestToastContainer_ScriptContainsShowToast(t *testing.T) {
	output := renderToast(t)
	if !strings.Contains(output, "showToast") {
		t.Errorf("expected showToast function in script, got: %s", output)
	}
}

// --- Triangulation: verify styling variants and behavior ---

func TestToastContainer_HasSuccessStyle(t *testing.T) {
	output := renderToast(t)
	if !strings.Contains(output, "bg-green-500") {
		t.Error("expected bg-green-500 for success toasts")
	}
}

func TestToastContainer_HasErrorStyle(t *testing.T) {
	output := renderToast(t)
	if !strings.Contains(output, "bg-red-500") {
		t.Error("expected bg-red-500 for error toasts")
	}
}

func TestToastContainer_HasWarningStyle(t *testing.T) {
	output := renderToast(t)
	if !strings.Contains(output, "bg-yellow-500") {
		t.Error("expected bg-yellow-500 for warning toasts")
	}
}

func TestToastContainer_HasCSSTransition(t *testing.T) {
	output := renderToast(t)
	if !strings.Contains(output, "transition-opacity") {
		t.Error("expected transition-opacity CSS class for fade animation")
	}
	if !strings.Contains(output, "duration-300") {
		t.Error("expected duration-300 CSS class for transition timing")
	}
}

func TestToastContainer_ContainerHasFixedPositioning(t *testing.T) {
	output := renderToast(t)
	if !strings.Contains(output, "fixed") {
		t.Error("expected 'fixed' positioning class on container")
	}
	if !strings.Contains(output, "bottom-4") {
		t.Error("expected 'bottom-4' class on container")
	}
	if !strings.Contains(output, "right-4") {
		t.Error("expected 'right-4' class on container")
	}
}

func TestToastContainer_ScriptParsesJSONHeader(t *testing.T) {
	output := renderToast(t)
	if !strings.Contains(output, "JSON.parse") {
		t.Error("expected JSON.parse to parse HX-Trigger header")
	}
}

func TestToastContainer_ScriptExposesWindowShowToast(t *testing.T) {
	output := renderToast(t)
	if !strings.Contains(output, "window.showToast") {
		t.Error("expected window.showToast to be exposed globally")
	}
}
