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
		return c.SendString(fmt.Sprintf("userID=%d,role=%s,hasHousehold=%v", userID, role, householdID != nil))
	})

	return app
}

// createValidToken generates a JWT cookie value for testing.
func createValidToken(userID uint, householdID *uint, role string) string {
	token, _ := services.CreateToken(userID, householdID, role, testSecret, 24)
	return token
}

func TestRequireAuth_ValidToken(t *testing.T) {
	app := setupTestApp()

	var householdID uint = 42
	token := createValidToken(1, &householdID, "admin")

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
	if bodyStr != "userID=1,role=admin,hasHousehold=true" {
		t.Errorf("unexpected response body: %s", bodyStr)
	}
}

func TestRequireAuth_ValidToken_NilHousehold(t *testing.T) {
	app := setupTestApp()

	token := createValidToken(3, nil, "member")

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
	expected := "userID=3,role=member,hasHousehold=false"
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
	token, _ := services.CreateToken(1, nil, "member", testSecret, 0)

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
	token, _ := services.CreateToken(1, nil, "member", "wrong-secret", 24)

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
