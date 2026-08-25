package middleware

import (
	"encoding/json"
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

	if ct := resp.Header.Get("Content-Type"); len(ct) < 9 || ct[:9] != "text/html" {
		t.Errorf("Content-Type = %q, want text/html", ct)
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

// TestErrorHandlerHtmxTriggerQuoteSafe proves the HX-Trigger payload stays
// valid JSON when the message contains quotes: raw fmt.Sprintf interpolation
// used to emit invalid JSON for such messages; json.Marshal escapes them.
func TestErrorHandlerHtmxTriggerQuoteSafe(t *testing.T) {
	app := newTestApp()
	app.Get("/test", func(c *fiber.Ctx) error {
		return BadRequest(`say "hello" now`)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("HX-Request", "true")
	req.Header.Set("Accept", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}

	trigger := resp.Header.Get("HX-Trigger")
	if trigger == "" {
		t.Fatal("expected HX-Trigger header on HTMX request, got empty")
	}
	var decoded struct {
		Message string `json:"message"`
		Level   string `json:"level"`
	}
	if err := json.Unmarshal([]byte(trigger), &decoded); err != nil {
		t.Fatalf("HX-Trigger is not valid JSON (%v): %s", err, trigger)
	}
	if decoded.Message != `say "hello" now` {
		t.Errorf("HX-Trigger message = %q, want %q", decoded.Message, `say "hello" now`)
	}
	if decoded.Level != "error" {
		t.Errorf("HX-Trigger level = %q, want %q", decoded.Level, "error")
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

// TestErrorHandlerFiberNotFound verifies Fiber's own ErrNotFound maps to a 404
// (not the generic 500 fallback) and is served as HTML with the right Content-Type.
func TestErrorHandlerFiberNotFound(t *testing.T) {
	app := newTestApp()
	app.Get("/known", func(c *fiber.Ctx) error { return c.SendString("ok") })

	req := httptest.NewRequest("GET", "/unknown", nil)
	req.Header.Set("Accept", "text/html")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404 for unknown route", resp.StatusCode)
	}

	if ct := resp.Header.Get("Content-Type"); len(ct) < 9 || ct[:9] != "text/html" {
		t.Errorf("Content-Type = %q, want text/html", ct)
	}
}

// --- WU-3: Keyed error tests ---

func TestKeyedErrorSetsKeyAndArgs(t *testing.T) {
	err := Keyed(400, "household.already_has")
	if err.Status != 400 {
		t.Errorf("Status = %d, want 400", err.Status)
	}
	if err.Key != "household.already_has" {
		t.Errorf("Key = %q, want %q", err.Key, "household.already_has")
	}
	if err.Message != "" {
		t.Errorf("Message should be empty for keyed error, got %q", err.Message)
	}
}

func TestKeyedErrorWithArgs(t *testing.T) {
	err := Keyed(400, "validation.required", "email")
	if len(err.Args) != 1 || err.Args[0] != "email" {
		t.Errorf("Args = %v, want [email]", err.Args)
	}
}

func TestErrorHandlerKeyedES_HTML(t *testing.T) {
	app := newTestApp()
	app.Get("/test", func(c *fiber.Ctx) error {
		c.Locals("lang", "es")
		return Keyed(400, "household.already_has")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept", "text/html")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}

	body := make([]byte, 8192)
	n, _ := resp.Body.Read(body)
	bodyStr := string(body[:n])

	if !containsStr(bodyStr, "Ya perteneces a un hogar") {
		t.Errorf("HTML body missing Spanish translation: %s", bodyStr)
	}
	if !containsStr(bodyStr, `lang="es"`) {
		t.Errorf("HTML body missing lang=\"es\": %s", bodyStr)
	}
}

func TestErrorHandlerKeyedEN_HTML(t *testing.T) {
	app := newTestApp()
	app.Get("/test", func(c *fiber.Ctx) error {
		c.Locals("lang", "en")
		return Keyed(400, "household.already_has")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept", "text/html")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}

	body := make([]byte, 8192)
	n, _ := resp.Body.Read(body)
	bodyStr := string(body[:n])

	if !containsStr(bodyStr, "You already belong to a household") {
		t.Errorf("HTML body missing English translation: %s", bodyStr)
	}
}

func TestErrorHandlerRawMessagePassThrough(t *testing.T) {
	app := newTestApp()
	app.Get("/test", func(c *fiber.Ctx) error {
		c.Locals("lang", "es")
		return NotFound("custom raw message")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}

	body := make([]byte, 4096)
	n, _ := resp.Body.Read(body)
	bodyStr := string(body[:n])

	if !containsStr(bodyStr, "custom raw message") {
		t.Errorf("JSON body should contain raw message, got: %s", bodyStr)
	}
}

func TestErrorHandlerKeyedES_HTMXToast(t *testing.T) {
	app := newTestApp()
	app.Get("/test", func(c *fiber.Ctx) error {
		c.Locals("lang", "es")
		return Keyed(400, "household.already_has")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("HX-Request", "true")
	req.Header.Set("Accept", "text/html")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}

	trigger := resp.Header.Get("HX-Trigger")
	if !containsStr(trigger, "Ya perteneces a un hogar") {
		t.Errorf("HX-Trigger missing Spanish toast: %s", trigger)
	}
}

func TestErrorHandlerKeyedWithArgs_HTML(t *testing.T) {
	app := newTestApp()
	app.Get("/test", func(c *fiber.Ctx) error {
		c.Locals("lang", "en")
		return Keyed(400, "validation.required", "email")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept", "text/html")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}

	body := make([]byte, 8192)
	n, _ := resp.Body.Read(body)
	bodyStr := string(body[:n])

	// "The email field is required." — interpolated from validation.required
	if !containsStr(bodyStr, "email") {
		t.Errorf("HTML body missing interpolated arg 'email': %s", bodyStr)
	}
}

func TestErrorHandlerDefaultLangFallback(t *testing.T) {
	app := newTestApp()
	app.Get("/test", func(c *fiber.Ctx) error {
		// No lang set — should fallback to "en"
		return Keyed(400, "household.already_has")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept", "text/html")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}

	body := make([]byte, 8192)
	n, _ := resp.Body.Read(body)
	bodyStr := string(body[:n])

	if !containsStr(bodyStr, "You already belong to a household") {
		t.Errorf("HTML body missing English translation (fallback): %s", bodyStr)
	}
}

// --- Triangulation: validate key translation round-trips for JSON too ---

func TestErrorHandlerKeyedES_JSON(t *testing.T) {
	app := newTestApp()
	app.Get("/test", func(c *fiber.Ctx) error {
		c.Locals("lang", "es")
		return Keyed(400, "household.already_has")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}

	body := make([]byte, 4096)
	n, _ := resp.Body.Read(body)
	bodyStr := string(body[:n])

	if !containsStr(bodyStr, "Ya perteneces a un hogar") {
		t.Errorf("JSON body missing Spanish translation: %s", bodyStr)
	}
}

func TestErrorHandlerKeyedEN_JSON(t *testing.T) {
	app := newTestApp()
	app.Get("/test", func(c *fiber.Ctx) error {
		c.Locals("lang", "en")
		return Keyed(400, "household.already_has")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}

	body := make([]byte, 4096)
	n, _ := resp.Body.Read(body)
	bodyStr := string(body[:n])

	if !containsStr(bodyStr, "You already belong to a household") {
		t.Errorf("JSON body missing English translation: %s", bodyStr)
	}
}

// Triangulate: errorPageHTML with lang="en" has lang="en"
func TestErrorHandlerKeyedEN_LangAttr(t *testing.T) {
	app := newTestApp()
	app.Get("/test", func(c *fiber.Ctx) error {
		c.Locals("lang", "en")
		return Keyed(400, "household.already_has")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept", "text/html")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}

	body := make([]byte, 8192)
	n, _ := resp.Body.Read(body)
	bodyStr := string(body[:n])

	if !containsStr(bodyStr, `lang="en"`) {
		t.Errorf("HTML body missing lang=\"en\": %s", bodyStr)
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
