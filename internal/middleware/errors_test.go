package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestAppError(t *testing.T) {
	err := &AppError{Status: 404, Message: "not found"}
	if err.Error() != "not found" {
		t.Errorf("Error() = %q, want %q", err.Error(), "not found")
	}
	if err.Status != 404 {
		t.Errorf("Status = %d, want 404", err.Status)
	}
}

func TestSentinelConstructors(t *testing.T) {
	tests := []struct {
		name     string
		fn       func(string) *AppError
		wantStat int
		wantMsg  string
	}{
		{"NotFound", NotFound, 404, "not found"},
		{"BadRequest", BadRequest, 400, "bad request"},
		{"Forbidden", Forbidden, 403, "forbidden"},
		{"Internal", Internal, 500, "internal error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn(tt.wantMsg)
			if err.Status != tt.wantStat {
				t.Errorf("Status = %d, want %d", err.Status, tt.wantStat)
			}
			if err.Error() != tt.wantMsg {
				t.Errorf("Error() = %q, want %q", err.Error(), tt.wantMsg)
			}
		})
	}
}

func newTestApp() *fiber.App {
	return fiber.New(fiber.Config{
		ErrorHandler: ErrorHandler,
	})
}

func TestErrorHandlerJSON(t *testing.T) {
	app := newTestApp()
	app.Get("/test", func(c *fiber.Ctx) error {
		return NotFound("expense not found")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}

	body := make([]byte, 4096)
	n, _ := resp.Body.Read(body)
	bodyStr := string(body[:n])

	if !containsStr(bodyStr, `"error"`) {
		t.Errorf("response body missing error field: %s", bodyStr)
	}
	if !containsStr(bodyStr, "expense not found") {
		t.Errorf("response body missing message: %s", bodyStr)
	}

	ct := resp.Header.Get("Content-Type")
	if !containsStr(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}

func TestErrorHandlerHTML(t *testing.T) {
	app := newTestApp()
	app.Get("/test", func(c *fiber.Ctx) error {
		return NotFound("expense not found")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept", "text/html")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}

	body := make([]byte, 4096)
	n, _ := resp.Body.Read(body)
	bodyStr := string(body[:n])

	if !containsStr(bodyStr, "<html") {
		t.Errorf("response body missing <html tag: %s", bodyStr)
	}
	if !containsStr(bodyStr, "expense not found") {
		t.Errorf("response body missing error message: %s", bodyStr)
	}
}

func TestErrorHandlerDefaultAccept(t *testing.T) {
	app := newTestApp()
	app.Get("/test", func(c *fiber.Ctx) error {
		return BadRequest("invalid input")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	// No Accept header — should default to HTML (browser default)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}

	body := make([]byte, 4096)
	n, _ := resp.Body.Read(body)
	bodyStr := string(body[:n])

	if !containsStr(bodyStr, "<html") {
		t.Errorf("default Accept should render HTML, got: %s", bodyStr)
	}
}

func TestErrorHandlerHtmxSetsTriggerHeader(t *testing.T) {
	app := newTestApp()
	app.Get("/test", func(c *fiber.Ctx) error {
		return BadRequest("invalid email format")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("HX-Request", "true")
	req.Header.Set("Accept", "text/html")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}

	trigger := resp.Header.Get("HX-Trigger")
	if trigger == "" {
		t.Fatal("expected HX-Trigger header on HTMX request, got empty")
	}
	if !containsStr(trigger, "invalid email format") {
		t.Errorf("HX-Trigger missing error message: %s", trigger)
	}
	if !containsStr(trigger, `"level":"error"`) {
		t.Errorf("HX-Trigger missing level field: %s", trigger)
	}
}

func TestErrorHandlerNonHtmxNoTriggerHeader(t *testing.T) {
	app := newTestApp()
	app.Get("/test", func(c *fiber.Ctx) error {
		return NotFound("not found")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept", "application/json")
	// No HX-Request header — regular request
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}

	trigger := resp.Header.Get("HX-Trigger")
	if trigger != "" {
		t.Errorf("HX-Trigger header should NOT be set for non-HTMX requests, got: %s", trigger)
	}
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
