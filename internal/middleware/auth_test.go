package middleware

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/homeadmin/internal/services"
)

const testSecret = "test-jwt-secret"

// setupTestApp creates a Fiber app with RequireAuth middleware and a test handler
// that echoes back the Locals values set by the middleware.
func setupTestApp() *fiber.App {
	app := fiber.New()

	app.Get("/protected", RequireAuth(testSecret), func(c *fiber.Ctx) error {
		userID, _ := c.Locals("userID").(uint)
		householdID, _ := c.Locals("householdID").(*uint)
		role, _ := c.Locals("role").(string)
		email, _ := c.Locals("email").(string)
		isAdmin, _ := c.Locals("isAdmin").(bool)
		return c.SendString(fmt.Sprintf("userID=%d,role=%s,hasHousehold=%v,email=%s,isAdmin=%v", userID, role, householdID != nil, email, isAdmin))
	})

	return app
}

// createValidToken generates a JWT cookie value for testing.
func createValidToken(userID uint, householdID *uint, role, email string, isAdmin bool) string {
	token, _ := services.CreateToken(userID, householdID, role, email, isAdmin, testSecret, 24)
	return token
}

func TestRequireAuth_ValidToken(t *testing.T) {
	app := setupTestApp()

	var householdID uint = 42
	token := createValidToken(1, &householdID, "admin", "admin@example.com", false)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "jwt", Value: token})

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	// Verify the middleware set correct Locals by checking the echoed response
	if bodyStr != "userID=1,role=admin,hasHousehold=true,email=admin@example.com,isAdmin=false" {
		t.Errorf("unexpected response body: %s", bodyStr)
	}
}

func TestRequireAuth_ValidToken_NilHousehold(t *testing.T) {
	app := setupTestApp()

	token := createValidToken(3, nil, "member", "member@example.com", false)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "jwt", Value: token})

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	expected := "userID=3,role=member,hasHousehold=false,email=member@example.com,isAdmin=false"
	if string(body) != expected {
		t.Errorf("expected %q, got %q", expected, string(body))
	}
}

func TestRequireAuth_ValidToken_IsAdmin(t *testing.T) {
	app := setupTestApp()

	token := createValidToken(7, nil, "admin", "siteadmin@example.com", true)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "jwt", Value: token})

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	expected := "userID=7,role=admin,hasHousehold=false,email=siteadmin@example.com,isAdmin=true"
	if string(body) != expected {
		t.Errorf("expected %q, got %q", expected, string(body))
	}
}

func TestRequireAuth_MissingCookie(t *testing.T) {
	app := setupTestApp()

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	// No cookie set

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusFound {
		t.Errorf("expected redirect 302, got %d", resp.StatusCode)
	}

	location := resp.Header.Get("Location")
	if location != "/login" {
		t.Errorf("expected redirect to /login, got %s", location)
	}
}

func TestRequireAuth_InvalidToken(t *testing.T) {
	app := setupTestApp()

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "jwt", Value: "totally-invalid-token-value"})

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusFound {
		t.Errorf("expected redirect 302, got %d", resp.StatusCode)
	}

	location := resp.Header.Get("Location")
	if location != "/login" {
		t.Errorf("expected redirect to /login, got %s", location)
	}
}

func TestRequireAuth_EmptyCookie(t *testing.T) {
	app := setupTestApp()

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "jwt", Value: ""})

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusFound {
		t.Errorf("expected redirect 302, got %d", resp.StatusCode)
	}

	location := resp.Header.Get("Location")
	if location != "/login" {
		t.Errorf("expected redirect to /login, got %s", location)
	}
}

func TestRequireAuth_ExpiredToken(t *testing.T) {
	app := setupTestApp()

	// Create token with 0 hours expiration — it's already expired
	token, _ := services.CreateToken(1, nil, "member", "member@example.com", false, testSecret, 0)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "jwt", Value: token})

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusFound {
		t.Errorf("expected redirect 302, got %d", resp.StatusCode)
	}

	location := resp.Header.Get("Location")
	if location != "/login" {
		t.Errorf("expected redirect to /login, got %s", location)
	}
}

func TestRequireAuth_WrongSecret(t *testing.T) {
	app := setupTestApp()

	// Create token with a different secret than what the middleware expects
	token, _ := services.CreateToken(1, nil, "member", "member@example.com", false, "wrong-secret", 24)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "jwt", Value: token})

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusFound {
		t.Errorf("expected redirect 302, got %d", resp.StatusCode)
	}

	location := resp.Header.Get("Location")
	if location != "/login" {
		t.Errorf("expected redirect to /login, got %s", location)
	}
}

// setupHouseholdApp builds a Fiber app whose routes are guarded by
// RequireHousehold, with a probe handler echoing the householdID value when
// the middleware lets the request through. locals simulates what RequireAuth
// stores in c.Locals (claims.HouseholdID is *uint — services/auth.go).
func setupHouseholdApp(locals map[string]any) *fiber.App {
	app := fiber.New()

	app.Use(func(c *fiber.Ctx) error {
		for k, v := range locals {
			c.Locals(k, v)
		}
		return c.Next()
	})

	app.Get("/protected", RequireHousehold(), func(c *fiber.Ctx) error {
		hhID, _ := c.Locals("householdID").(*uint)
		return c.SendString(fmt.Sprintf("household=%d", *hhID))
	})

	return app
}

// setupSiteAdminApp builds a Fiber app whose routes are guarded by
// RequireSiteAdmin, with a probe handler echoing the isAdmin claim when the
// middleware lets the request through. The centralized ErrorHandler is wired
// so the 403 Forbidden path renders through the same pipeline as production.
func setupSiteAdminApp() *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: ErrorHandler,
	})

	app.Get("/protected", RequireSiteAdmin(testSecret), func(c *fiber.Ctx) error {
		isAdmin, _ := c.Locals("isAdmin").(bool)
		return c.SendString(fmt.Sprintf("isAdmin=%v", isAdmin))
	})

	return app
}

func TestRequireSiteAdmin(t *testing.T) {
	adminToken := createValidToken(1, nil, "owner", "siteadmin@example.com", true)
	memberToken := createValidToken(2, nil, "member", "member@example.com", false)

	tests := []struct {
		name         string
		cookie       *http.Cookie
		wantStatus   int
		wantLocation string
		wantBody     string
	}{
		// Unauthenticated: no cookie or a malformed one -> 302 /login.
		{"no cookie", nil, fiber.StatusFound, "/login", ""},
		{"empty cookie", &http.Cookie{Name: "jwt", Value: ""}, fiber.StatusFound, "/login", ""},
		{"invalid cookie", &http.Cookie{Name: "jwt", Value: "not-a-real-token"}, fiber.StatusFound, "/login", ""},
		{"expired token", &http.Cookie{Name: "jwt", Value: func() string {
			tok, _ := services.CreateToken(1, nil, "owner", "siteadmin@example.com", true, testSecret, 0)
			return tok
		}()}, fiber.StatusFound, "/login", ""},
		// Authenticated but not a site admin: forbidden (no ordering
		// dependency — the middleware parses the jwt cookie itself).
		{"authenticated but not admin", &http.Cookie{Name: "jwt", Value: memberToken}, fiber.StatusForbidden, "", ""},
		// Site admin: allowed through.
		{"site admin", &http.Cookie{Name: "jwt", Value: adminToken}, fiber.StatusOK, "", "isAdmin=true"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := setupSiteAdminApp()

			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if tt.cookie != nil {
				req.AddCookie(tt.cookie)
			}
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, resp.StatusCode)
			}

			if tt.wantLocation != "" {
				if loc := resp.Header.Get("Location"); loc != tt.wantLocation {
					t.Errorf("expected redirect to %s, got %s", tt.wantLocation, loc)
				}
			}

			if tt.wantBody != "" {
				body, _ := io.ReadAll(resp.Body)
				if string(body) != tt.wantBody {
					t.Errorf("expected body %q, got %q", tt.wantBody, string(body))
				}
			}
		})
	}
}

func TestRequireHousehold(t *testing.T) {
	hhID := uint(42)
	zero := uint(0)
	large := uint(999)

	tests := []struct {
		name         string
		locals       map[string]any
		wantStatus   int
		wantLocation string
		wantBody     string
	}{
		{"no householdID local", map[string]any{}, fiber.StatusFound, "/household", ""},
		{"nil interface value", map[string]any{"householdID": nil}, fiber.StatusFound, "/household", ""},
		{"typed nil *uint pointer", map[string]any{"householdID": (*uint)(nil)}, fiber.StatusFound, "/household", ""},
		{"wrong type uint value", map[string]any{"householdID": uint(42)}, fiber.StatusFound, "/household", ""},
		{"wrong type string value", map[string]any{"householdID": "42"}, fiber.StatusFound, "/household", ""},
		{"other auth locals but no householdID", map[string]any{"userID": uint(1), "role": "member"}, fiber.StatusFound, "/household", ""},
		{"non-nil household pointer", map[string]any{"householdID": &hhID}, fiber.StatusOK, "", "household=42"},
		{"non-nil pointer to zero", map[string]any{"householdID": &zero}, fiber.StatusOK, "", "household=0"},
		{"non-nil pointer to large value", map[string]any{"householdID": &large}, fiber.StatusOK, "", "household=999"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := setupHouseholdApp(tt.locals)

			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, resp.StatusCode)
			}

			if tt.wantLocation != "" {
				if loc := resp.Header.Get("Location"); loc != tt.wantLocation {
					t.Errorf("expected redirect to %s, got %s", tt.wantLocation, loc)
				}
			}

			if tt.wantBody != "" {
				body, _ := io.ReadAll(resp.Body)
				if string(body) != tt.wantBody {
					t.Errorf("expected body %q, got %q", tt.wantBody, string(body))
				}
			}
		})
	}
}
