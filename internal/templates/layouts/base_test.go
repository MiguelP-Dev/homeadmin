package layouts_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/homeadmin/internal/templates/layouts"
)

func TestBase_IncludesToastContainer(t *testing.T) {
	buf := &bytes.Buffer{}
	err := layouts.Base("Test Page", "csrf-token-123", "").Render(context.Background(), buf)
	if err != nil {
		t.Fatalf("failed to render Base layout: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, `id="toast-container"`) {
		t.Error("expected toast-container in base layout output")
	}
}

func TestBase_IncludesCSRFToken(t *testing.T) {
	buf := &bytes.Buffer{}
	err := layouts.Base("Test Page", "csrf-token-123", "").Render(context.Background(), buf)
	if err != nil {
		t.Fatalf("failed to render Base layout: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "csrf-token-123") {
		t.Error("expected csrf token in base layout output")
	}
}

func TestBase_IncludesTitle(t *testing.T) {
	buf := &bytes.Buffer{}
	err := layouts.Base("My Dashboard", "tok", "").Render(context.Background(), buf)
	if err != nil {
		t.Fatalf("failed to render Base layout: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "<title>My Dashboard</title>") {
		t.Errorf("expected title in output, got: %s", output)
	}
}

func TestBase_IncludesHtmxScript(t *testing.T) {
	buf := &bytes.Buffer{}
	err := layouts.Base("Test", "tok", "").Render(context.Background(), buf)
	if err != nil {
		t.Fatalf("failed to render Base layout: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "htmx.org") {
		t.Error("expected HTMX script in base layout")
	}
}

// New tests for nav integration

func TestBase_IncludesNav(t *testing.T) {
	buf := &bytes.Buffer{}
	err := layouts.Base("Test", "tok", "").Render(context.Background(), buf)
	if err != nil {
		t.Fatalf("failed to render Base layout: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "<nav") {
		t.Error("expected <nav> element in base layout output")
	}
}

func TestBase_IncludesNavWithUsername(t *testing.T) {
	buf := &bytes.Buffer{}
	err := layouts.Base("Test", "tok", "testuser").Render(context.Background(), buf)
	if err != nil {
		t.Fatalf("failed to render Base layout: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "testuser") {
		t.Error("expected username in base layout output when username is provided")
	}
}
