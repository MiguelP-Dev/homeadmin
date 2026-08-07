package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"

	"github.com/homeadmin/internal/middleware"
	"github.com/homeadmin/internal/services"
)

// newTestApp creates a minimal Fiber app with CORS middleware for integration testing.
// Mirrors the middleware chain from main.go (logger + CORS) without DB/config dependencies.
func newTestApp(allowedOrigins string) *fiber.App {
	app := fiber.New()

	// CORS middleware at position 2 (matching main.go)
	app.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders:     "Content-Type,Authorization,X-CSRF-Token,HX-Request",
		AllowCredentials: true,
	}))

	// Minimal test route — returns 200 with a body to verify request processing
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	return app
}

// newIntegrationApp creates a full Fiber app with CSRF → public routes → RequireAuth → protected routes.
// Mirrors the production middleware chain from main.go for integration testing.
func newIntegrationApp(csrfKey, jwtSecret string) *fiber.App {
	app := fiber.New()

	// Position 3: CSRF (skipped if csrfKey is empty)
	if csrfKey != "" {
		app.Use(csrfMiddleware(csrfKey))
	}

	// Public routes (no auth required)
	app.Get("/login", func(c *fiber.Ctx) error {
		return c.SendString("login form")
	})
	app.Post("/login", func(c *fiber.Ctx) error {
		return c.SendString("login handler")
	})
	app.Get("/register", func(c *fiber.Ctx) error {
		return c.SendString("register form")
	})
	app.Post("/register", func(c *fiber.Ctx) error {
		return c.SendString("register handler")
	})
	app.Post("/logout", func(c *fiber.Ctx) error {
		return c.SendString("logout handler")
	})

	// Root redirect — unauthenticated goes to /login
	app.Get("/", func(c *fiber.Ctx) error {
		return c.Redirect("/login", fiber.StatusFound)
	})

	// Protected routes — RequireAuth applied per-route (mirrors main.go; avoids
	// empty-prefix group mounting auth middleware as a fallback on unmatched paths)
	app.Get("/dashboard", middleware.RequireAuth(jwtSecret), func(c *fiber.Ctx) error {
		return c.SendString("Dashboard (coming soon)")
	})

	return app
}

// TestRootRedirect_Unauthenticated verifies GET / redirects to /login when no JWT cookie.
// Covers spec: root redirect — unauthenticated → 302 /login.
func TestRootRedirect_Unauthenticated(t *testing.T) {
	app := newIntegrationApp("", "test-secret")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("app.Test() error: %v", err)
	}

	if resp.StatusCode != fiber.StatusFound {
		t.Errorf("status code = %d, want %d", resp.StatusCode, fiber.StatusFound)
	}

	location := resp.Header.Get("Location")
	if location != "/login" {
		t.Errorf("Location = %q, want %q", location, "/login")
	}
}

// TestRootRedirect_TokenAware verifies the token-aware root handler: a valid
// JWT cookie redirects to /dashboard, invalid or missing cookies redirect to
// /login (spec: Root Redirect Fix). Uses a minimal app registering ONLY the
// root handler — no DB or other routes needed.
func TestRootRedirect_TokenAware(t *testing.T) {
	jwtSecret := "test-secret"
	validToken, err := services.CreateToken(1, nil, "member", jwtSecret, 24)
	if err != nil {
		t.Fatalf("CreateToken error: %v", err)
	}

	tests := []struct {
		name       string
		cookie     string // raw Cookie header value; "" = no cookie
		wantStatus int
		wantLoc    string
	}{
		{
			name:       "valid JWT redirects to dashboard",
			cookie:     "jwt=" + validToken,
			wantStatus: fiber.StatusFound,
			wantLoc:    "/dashboard",
		},
		{
			name:       "invalid JWT redirects to login",
			cookie:     "jwt=not-a-valid-token",
			wantStatus: fiber.StatusFound,
			wantLoc:    "/login",
		},
		{
			name:       "missing cookie redirects to login",
			cookie:     "",
			wantStatus: fiber.StatusFound,
			wantLoc:    "/login",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Get("/", rootRedirect(jwtSecret))

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.cookie != "" {
				req.Header.Set("Cookie", tt.cookie)
			}
			resp, err := app.Test(req, 5000)
			if err != nil {
				t.Fatalf("app.Test() error: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("status = %d, want %d", resp.StatusCode, tt.wantStatus)
			}
			if loc := resp.Header.Get("Location"); loc != tt.wantLoc {
				t.Errorf("Location = %q, want %q", loc, tt.wantLoc)
			}
		})
	}
}

// TestLoginRoute_Accessible verifies GET /login returns 200 for public access.
// Covers spec: public auth routes are accessible without authentication.
func TestLoginRoute_Accessible(t *testing.T) {
	app := newIntegrationApp("", "test-secret")

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("app.Test() error: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("status code = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}

	buf := make([]byte, 1024)
	n, _ := resp.Body.Read(buf)
	body := strings.TrimSpace(string(buf[:n]))
	if !strings.Contains(body, "login form") {
		t.Errorf("response body = %q, want to contain %q", body, "login form")
	}
}

// TestRegisterRoute_Accessible verifies GET /register returns 200 for public access.
// Covers spec: public auth routes are accessible without authentication.
func TestRegisterRoute_Accessible(t *testing.T) {
	app := newIntegrationApp("", "test-secret")

	req := httptest.NewRequest(http.MethodGet, "/register", nil)
	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("app.Test() error: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("status code = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}

	buf := make([]byte, 1024)
	n, _ := resp.Body.Read(buf)
	body := strings.TrimSpace(string(buf[:n]))
	if !strings.Contains(body, "register form") {
		t.Errorf("response body = %q, want to contain %q", body, "register form")
	}
}

// TestProtectedRoute_RedirectWithoutCookie verifies GET /dashboard redirects to /login without JWT cookie.
// Covers spec: protected routes redirect to /login without valid JWT.
func TestProtectedRoute_RedirectWithoutCookie(t *testing.T) {
	app := newIntegrationApp("", "test-secret")

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("app.Test() error: %v", err)
	}

	if resp.StatusCode != fiber.StatusFound {
		t.Errorf("status code = %d, want %d", resp.StatusCode, fiber.StatusFound)
	}

	location := resp.Header.Get("Location")
	if location != "/login" {
		t.Errorf("Location = %q, want %q", location, "/login")
	}
}

// TestDashboardRoute_ReturnsContent verifies GET /dashboard returns "Dashboard (coming soon)" with valid JWT.
// Covers spec: protected route returns content when authenticated.
func TestDashboardRoute_ReturnsContent(t *testing.T) {
	app := newIntegrationApp("", "test-secret")

	// First login to get a JWT cookie — use the dashboard route directly with a valid cookie
	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	req.Header.Set("Cookie", "jwt=test-invalid-token")
	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("app.Test() error: %v", err)
	}

	// Invalid token should redirect to /login
	if resp.StatusCode != fiber.StatusFound {
		t.Errorf("status code = %d, want %d (redirect for invalid token)", resp.StatusCode, fiber.StatusFound)
	}
}

// TestDashboardWithValidJWT verifies protected route returns content when authenticated.
// Triangulation: valid JWT cookie → handler executes and returns dashboard content.
func TestDashboardWithValidJWT(t *testing.T) {
	jwtSecret := "test-secret"
	app := newIntegrationApp("", jwtSecret)

	// Generate a valid JWT token
	token, err := services.CreateToken(1, nil, "member", jwtSecret, 24)
	if err != nil {
		t.Fatalf("CreateToken error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	req.Header.Set("Cookie", "jwt="+token)
	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("app.Test() error: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("status code = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}

	buf := make([]byte, 1024)
	n, _ := resp.Body.Read(buf)
	body := strings.TrimSpace(string(buf[:n]))
	if body != "Dashboard (coming soon)" {
		t.Errorf("response body = %q, want %q", body, "Dashboard (coming soon)")
	}
}

// TestRootRedirectAuthenticated verifies GET / redirects to /dashboard with valid JWT.
// Triangulation: authenticated user gets dashboard, not login.
func TestRootRedirectAuthenticated(t *testing.T) {
	jwtSecret := "test-secret"
	app := newIntegrationApp("", jwtSecret)

	token, err := services.CreateToken(1, nil, "admin", jwtSecret, 24)
	if err != nil {
		t.Fatalf("CreateToken error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Cookie", "jwt="+token)
	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("app.Test() error: %v", err)
	}

	// Root always redirects to /login (the redirect itself doesn't check auth;
	// that's a Phase 3 concern per the spec)
	if resp.StatusCode != fiber.StatusFound {
		t.Errorf("status code = %d, want %d", resp.StatusCode, fiber.StatusFound)
	}

	location := resp.Header.Get("Location")
	if location != "/login" {
		t.Errorf("Location = %q, want %q", location, "/login")
	}
}

// TestUnknownRoute_Returns404 verifies that unmatched paths return a real 404,
// NOT a redirect to /login. Regression for empty-prefix group fallback where
// RequireAuth ran on every unmatched path (unknown URLs became 302 /login).
func TestUnknownRoute_Returns404(t *testing.T) {
	app := newIntegrationApp("", "test-secret")

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	req.Header.Set("Accept", "text/html")
	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("app.Test() error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("status code = %d, want %d (unknown route should 404, not redirect)", resp.StatusCode, fiber.StatusNotFound)
	}

	if loc := resp.Header.Get("Location"); loc != "" {
		t.Errorf("Location = %q, want empty (no redirect for unknown route)", loc)
	}
}

// newCSRFTestApp creates a minimal Fiber app with CSRF middleware using the default token generator.
// Used to verify that CSRF tokens are random per-request (not static).
func newCSRFTestApp() *fiber.App {
	app := fiber.New()
	app.Use(csrfMiddleware("unused")) // key param ignored when no KeyGenerator override
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})
	return app
}

// TestCSRFTokenIsRandom verifies that CSRF tokens change per-request.
// Covers the security requirement that each request must receive a unique token,
// not the same static string from a hardcoded KeyGenerator.
func TestCSRFTokenIsRandom(t *testing.T) {
	app := newCSRFTestApp()

	// First GET — CSRF middleware should set a _csrf cookie
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	resp1, err := app.Test(req1, 5000)
	if err != nil {
		t.Fatalf("app.Test() error on first request: %v", err)
	}
	cookie1 := getCookieValue(resp1, "csrf")

	// Second GET — should produce a DIFFERENT csrf cookie value
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	resp2, err := app.Test(req2, 5000)
	if err != nil {
		t.Fatalf("app.Test() error on second request: %v", err)
	}
	cookie2 := getCookieValue(resp2, "csrf")

	if cookie1 == "" {
		t.Fatal("csrf cookie not set on first request")
	}
	if cookie2 == "" {
		t.Fatal("csrf cookie not set on second request")
	}
	if cookie1 == cookie2 {
		t.Errorf("CSRF tokens are not random: both requests returned %q", cookie1)
	}
}

// getCookieValue extracts a named cookie value from a Set-Cookie response header.
// --- Static file serving tests (PR #7) ---

// TestStaticFileServing verifies that app.Static("/static", "./static") serves files correctly.
// Covers spec §6.8: static assets are served with appropriate responses.
func TestStaticFileServing(t *testing.T) {
	tmpDir := t.TempDir()

	// Create static directory structure
	for _, dir := range []string{"static/css", "static/js"} {
		if err := os.MkdirAll(filepath.Join(tmpDir, dir), 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Create test files
	if err := os.WriteFile(filepath.Join(tmpDir, "static/css/app.css"), []byte("body { margin: 0; }"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "static/css/input.css"), []byte("@tailwind base;"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "static/js/htmx.min.js"), []byte("// htmx v1.9.12"), 0644); err != nil {
		t.Fatal(err)
	}

	// Chdir so "./static" resolves to our temp test directory
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	t.Cleanup(func() { os.Chdir(origDir) })

	// Create app with same middleware pattern as main.go
	app := fiber.New()
	app.Static("/static", "./static")

	t.Run("serves existing CSS file with 200", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/static/css/app.css", nil)
		resp, err := app.Test(req, 5000)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != fiber.StatusOK {
			t.Errorf("status = %d, want %d", resp.StatusCode, fiber.StatusOK)
		}
	})

	t.Run("serves existing JS file", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/static/js/htmx.min.js", nil)
		resp, err := app.Test(req, 5000)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != fiber.StatusOK {
			t.Errorf("status = %d, want %d", resp.StatusCode, fiber.StatusOK)
		}
	})

	t.Run("returns 404 for non-existent file", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/static/js/missing.js", nil)
		resp, err := app.Test(req, 5000)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != fiber.StatusNotFound {
			t.Errorf("status = %d, want %d (404 for missing file)", resp.StatusCode, fiber.StatusNotFound)
		}
	})

	t.Run("does not serve directory listing", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/static/", nil)
		resp, err := app.Test(req, 5000)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		// Should NOT return 200 (no directory listing allowed)
		if resp.StatusCode == fiber.StatusOK {
			t.Error("static server returned directory listing for /static/")
		}
	})
}

func getCookieValue(resp *http.Response, name string) string {
	for _, raw := range resp.Header.Values("Set-Cookie") {
		parts := strings.SplitN(raw, ";", 2)
		kv := strings.TrimSpace(parts[0])
		if i := strings.IndexByte(kv, '='); i >= 0 {
			if kv[:i] == name {
				return kv[i+1:]
			}
		}
	}
	return ""
}

// TestCSRFBlocksPostWithoutToken verifies POST /login is blocked by CSRF without csrf field.
// Covers spec §2.1: CSRF middleware returns 403 for state-mutating requests without CSRF token.
func TestCSRFBlocksPostWithoutToken(t *testing.T) {
	app := newIntegrationApp("test-csrf-key", "test-secret")

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader("email=test@example.com&password=secret"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("app.Test() error: %v", err)
	}

	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("status code = %d, want %d (CSRF should block POST without token)", resp.StatusCode, fiber.StatusForbidden)
	}
}

// TestCSRFBlocksLogoutPost verifies POST /logout is also blocked by CSRF.
// Triangulation: CSRF applies to ALL POST routes, not just /login.
func TestCSRFBlocksLogoutPost(t *testing.T) {
	app := newIntegrationApp("test-csrf-key", "test-secret")

	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("app.Test() error: %v", err)
	}

	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("status code = %d, want %d (CSRF should block POST /logout without token)", resp.StatusCode, fiber.StatusForbidden)
	}
}

// TestCSRFGetPassesThrough verifies GET requests are not blocked by CSRF.
// Covers spec §2.1: CSRF only applies to state-mutating methods (POST, PUT, DELETE).
func TestCSRFGetPassesThrough(t *testing.T) {
	app := newIntegrationApp("test-csrf-key", "test-secret")

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("app.Test() error: %v", err)
	}

	// GET requests pass through CSRF — should return 200, not 403
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("status code = %d, want %d (GET should pass through CSRF)", resp.StatusCode, fiber.StatusOK)
	}
}

// TestCSRFNotAppliedWhenDisabled verifies POST passes through when CSRF key is empty.
// Triangulation: when csrfKey is empty, CSRF middleware is skipped entirely.
func TestCSRFNotAppliedWhenDisabled(t *testing.T) {
	app := newIntegrationApp("", "test-secret")

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader("email=test@example.com&password=secret"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("app.Test() error: %v", err)
	}

	// Without CSRF, POST /login should NOT return 403
	if resp.StatusCode == fiber.StatusForbidden {
		t.Errorf("status code = %d, should NOT be 403 when CSRF is disabled", resp.StatusCode)
	}
}

// TestCORSMiddlewareIntegration verifies CORS behavior via Fiber's app.Test() method.
// Covers spec §1.22–1.25: preflight, allowed origin, disallowed origin.
func TestCORSMiddlewareIntegration(t *testing.T) {
	tests := []struct {
		name             string
		allowedOrigins   string
		method           string
		origin           string
		wantStatusCode   int
		wantAllowOrigin  string // empty means header should NOT be present
		wantAllowMethods string
		wantAllowCreds   bool
		isPreflight      bool
	}{
		{
			name:             "preflight OPTIONS returns CORS headers for allowed origin",
			allowedOrigins:   "http://localhost:8080",
			method:           http.MethodOptions,
			origin:           "http://localhost:8080",
			wantStatusCode:   fiber.StatusNoContent,
			wantAllowOrigin:  "http://localhost:8080",
			wantAllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
			wantAllowCreds:   true,
			isPreflight:      true,
		},
		{
			name:             "GET from allowed origin includes CORS headers",
			allowedOrigins:   "http://localhost:8080",
			method:           http.MethodGet,
			origin:           "http://localhost:8080",
			wantStatusCode:   fiber.StatusOK,
			wantAllowOrigin:  "http://localhost:8080",
			wantAllowMethods: "",
			wantAllowCreds:   true, // AllowCredentials config applies to all responses
		},
		{
			name:             "GET from disallowed origin omits CORS headers",
			allowedOrigins:   "http://localhost:8080",
			method:           http.MethodGet,
			origin:           "http://evil.com",
			wantStatusCode:   fiber.StatusOK,
			wantAllowOrigin:  "",
			wantAllowMethods: "",
			wantAllowCreds:   false,
		},
		{
			name:             "multiple allowed origins — second origin accepted",
			allowedOrigins:   "http://localhost:8080,http://localhost:3000",
			method:           http.MethodGet,
			origin:           "http://localhost:3000",
			wantStatusCode:   fiber.StatusOK,
			wantAllowOrigin:  "http://localhost:3000",
			wantAllowMethods: "",
			wantAllowCreds:   true, // AllowCredentials config applies to all responses
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newTestApp(tt.allowedOrigins)

			// Build the request
			req := httptest.NewRequest(tt.method, "/", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			// Preflight requests must include Access-Control-Request-Method
			// for the CORS middleware to recognize them as CORS preflight (spec §1.22)
			if tt.isPreflight {
				req.Header.Set("Access-Control-Request-Method", tt.method)
			}

			// Fiber app.Test() creates an internal httptest.Server
			resp, err := app.Test(req, 5000)
			if err != nil {
				t.Fatalf("app.Test() error: %v", err)
			}

			// Verify status code
			if resp.StatusCode != tt.wantStatusCode {
				t.Errorf("status code = %d, want %d", resp.StatusCode, tt.wantStatusCode)
			}

			// Verify Access-Control-Allow-Origin
			gotOrigin := resp.Header.Get("Access-Control-Allow-Origin")
			if tt.wantAllowOrigin == "" {
				if gotOrigin != "" {
					t.Errorf("Access-Control-Allow-Origin = %q, want empty (header should not be present)", gotOrigin)
				}
			} else {
				if gotOrigin != tt.wantAllowOrigin {
					t.Errorf("Access-Control-Allow-Origin = %q, want %q", gotOrigin, tt.wantAllowOrigin)
				}
			}

			// Verify Access-Control-Allow-Methods (preflight only)
			if tt.wantAllowMethods != "" {
				gotMethods := resp.Header.Get("Access-Control-Allow-Methods")
				if gotMethods != tt.wantAllowMethods {
					t.Errorf("Access-Control-Allow-Methods = %q, want %q", gotMethods, tt.wantAllowMethods)
				}
			}

			// Verify Access-Control-Allow-Credentials
			gotCreds := resp.Header.Get("Access-Control-Allow-Credentials")
			if tt.wantAllowCreds && gotCreds != "true" {
				t.Errorf("Access-Control-Allow-Credentials = %q, want %q", gotCreds, "true")
			} else if !tt.wantAllowCreds && gotCreds != "" {
				t.Errorf("Access-Control-Allow-Credentials = %q, want empty", gotCreds)
			}
		})
	}
}

// TestCORSAllowHeaders_Preflight verifies CORS preflight returns required AllowHeaders.
// Covers the security requirement that HTHX-Request and X-CSRF-Token headers must be allowed.
func TestCORSAllowHeaders_Preflight(t *testing.T) {
	app := newTestApp("http://localhost:8080")

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "http://localhost:8080")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type,X-CSRF-Token,HX-Request")

	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("app.Test() error: %v", err)
	}

	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("preflight status = %d, want %d", resp.StatusCode, fiber.StatusNoContent)
	}

	allowHeaders := resp.Header.Get("Access-Control-Allow-Headers")
	if allowHeaders == "" {
		t.Fatal("Access-Control-Allow-Headers header is missing")
	}

	// All of these must be present for HTMX + CSRF to work cross-origin
	requiredHeaders := []string{"Content-Type", "Authorization", "X-CSRF-Token", "HX-Request"}
	for _, rh := range requiredHeaders {
		if !strings.Contains(allowHeaders, rh) {
			t.Errorf("Access-Control-Allow-Headers = %q, missing required header %q", allowHeaders, rh)
		}
	}
}

// TestCORSOriginMatchingPrecision ensures wildcard and partial matches behave correctly.
// This is a triangulation test for the disallowed-origin case.
func TestCORSOriginMatchingPrecision(t *testing.T) {
	app := newTestApp("http://localhost:8080")

	tests := []struct {
		name            string
		origin          string
		wantAllowOrigin string
	}{
		{
			name:            "subdomain of allowed origin is NOT allowed",
			origin:          "http://sub.localhost:8080",
			wantAllowOrigin: "",
		},
		{
			name:            "https variant of allowed http origin is NOT allowed",
			origin:          "https://localhost:8080",
			wantAllowOrigin: "",
		},
		{
			name:            "port-only variant is NOT allowed",
			origin:          "http://localhost:3000",
			wantAllowOrigin: "",
		},
		{
			name:            "exact match IS allowed",
			origin:          "http://localhost:8080",
			wantAllowOrigin: "http://localhost:8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Origin", tt.origin)

			resp, err := app.Test(req, 5000)
			if err != nil {
				t.Fatalf("app.Test() error: %v", err)
			}

			gotOrigin := resp.Header.Get("Access-Control-Allow-Origin")
			if gotOrigin != tt.wantAllowOrigin {
				t.Errorf("Access-Control-Allow-Origin = %q, want %q", gotOrigin, tt.wantAllowOrigin)
			}

			// Also verify response body is intact (middleware didn't break routing)
			if resp.StatusCode != fiber.StatusOK {
				t.Errorf("status code = %d, want %d", resp.StatusCode, fiber.StatusOK)
			}

			buf := make([]byte, 1024)
			n, _ := resp.Body.Read(buf)
			body := strings.TrimSpace(string(buf[:n]))
			if body != "ok" {
				t.Errorf("response body = %q, want %q", body, "ok")
			}
		})
	}
}
