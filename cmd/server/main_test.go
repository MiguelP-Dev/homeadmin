package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"gorm.io/gorm"

	"github.com/homeadmin/internal/database"
	"github.com/homeadmin/internal/handlers"
	"github.com/homeadmin/internal/middleware"
	"github.com/homeadmin/internal/repositories"
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

// newIntegrationApp creates a full Fiber app backed by an in-memory SQLite
// database and the REAL handlers, repos and services, mirroring the production
// middleware chain and route/middleware order from main.go (design §6: the old
// "Dashboard (coming soon)" stub was the exact reason the *uint Locals panic
// shipped — it must never come back).
func newIntegrationApp(t *testing.T, csrfKey, jwtSecret string) *fiber.App {
	t.Helper()
	app, _ := newIntegrationAppWithDB(t, csrfKey, jwtSecret)
	return app
}

// newIntegrationAppWithDB builds the same app as newIntegrationApp but also
// returns the GORM handle so tests can assert persisted state (T5.4).
func newIntegrationAppWithDB(t *testing.T, csrfKey, jwtSecret string) (*fiber.App, *gorm.DB) {
	t.Helper()
	db, err := database.Connect(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("failed to connect to test db: %v", err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatalf("failed to migrate test db: %v", err)
	}

	userRepo := repositories.NewUserRepository(db)
	expenseRepo := repositories.NewExpenseRepository(db)
	authHandler := handlers.NewAuthHandler(userRepo, jwtSecret)
	expenseService := services.NewExpenseService(expenseRepo)
	expenseHandler := handlers.NewExpenseHandler(expenseService)
	householdRepo := repositories.NewHouseholdRepository(db)
	householdService := services.NewHouseholdService(householdRepo, userRepo, householdRepo)
	householdHandler := handlers.NewHouseholdHandler(householdService, userRepo, jwtSecret, 24)

	app := fiber.New(fiber.Config{
		ErrorHandler: middleware.ErrorHandler,
	})

	// CSRF (skipped if csrfKey is empty) — same position as main.go
	if csrfKey != "" {
		app.Use(csrfMiddleware(csrfKey))
	}

	// Public auth routes (no auth required)
	app.Get("/login", authHandler.ShowLogin)
	app.Post("/login", authHandler.Login)
	app.Get("/register", authHandler.ShowRegister)
	app.Post("/register", authHandler.Register)
	app.Post("/logout", authHandler.Logout)

	// Root redirect — token-aware: authenticated users go to /dashboard,
	// everyone else to /login (same handler main.go mounts).
	app.Get("/", rootRedirect(jwtSecret))

	// Protected routes — RequireAuth applied per-route, RequireHousehold after
	// it on household-mandatory routes (mirrors main.go order, design §2).
	app.Get("/dashboard", middleware.RequireAuth(jwtSecret), middleware.RequireHousehold(), expenseHandler.Dashboard)
	app.Get("/expenses", middleware.RequireAuth(jwtSecret), middleware.RequireHousehold(), expenseHandler.List)
	app.Post("/expenses", middleware.RequireAuth(jwtSecret), middleware.RequireHousehold(), expenseHandler.Create)
	app.Get("/expenses/new", middleware.RequireAuth(jwtSecret), middleware.RequireHousehold(), expenseHandler.ShowNew)
	app.Get("/expenses/:id/edit", middleware.RequireAuth(jwtSecret), middleware.RequireHousehold(), expenseHandler.ShowEdit)
	app.Post("/expenses/:id/update", middleware.RequireAuth(jwtSecret), middleware.RequireHousehold(), expenseHandler.Update)
	app.Post("/expenses/:id/delete", middleware.RequireAuth(jwtSecret), middleware.RequireHousehold(), expenseHandler.Delete)

	// Household routes — /household/invite is RequireAuth-only by design (the
	// handler 400s no-household users; batch-7 deviation from design §3).
	app.Get("/household", middleware.RequireAuth(jwtSecret), householdHandler.Show)
	app.Post("/household", middleware.RequireAuth(jwtSecret), householdHandler.Create)
	app.Post("/household/invite", middleware.RequireAuth(jwtSecret), householdHandler.Invite)
	app.Post("/household/join", middleware.RequireAuth(jwtSecret), householdHandler.Join)
	app.Post("/household/members/:id/role", middleware.RequireAuth(jwtSecret), householdHandler.SetMemberRole)

	return app, db
}

// TestExpenses_PutDeleteUnregistered verifies PUT /expenses/:id and
// DELETE /expenses/:id are no longer registered (RF-4): the router answers 404
// for an unauthenticated request — a registered route would have hit RequireAuth
// and redirected (302 /login) instead.
func TestExpenses_PutDeleteUnregistered(t *testing.T) {
	app := newIntegrationApp(t, "", "test-secret")

	for _, method := range []string{http.MethodPut, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/expenses/1", nil)
			resp, err := app.Test(req, 5000)
			if err != nil {
				t.Fatalf("app.Test() error: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != fiber.StatusNotFound {
				t.Errorf("status = %d, want %d (PUT/DELETE must be unregistered)", resp.StatusCode, fiber.StatusNotFound)
			}
		})
	}
}

// TestExpenses_CSRFLessPostForbidden verifies a POST /expenses without a CSRF
// token is rejected with 403 AND no row is persisted (threat matrix, RF-3).
func TestExpenses_CSRFLessPostForbidden(t *testing.T) {
	app, db := newIntegrationAppWithDB(t, "test-csrf-key", "test-secret")

	form := "amount=100&description=CSRF+Test&category=Rent&date=2026-07-27"
	req := httptest.NewRequest(http.MethodPost, "/expenses", strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("app.Test() error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("status = %d, want %d (CSRF must block token-less POST)", resp.StatusCode, fiber.StatusForbidden)
	}

	var count int64
	if err := db.Model(&database.Expense{}).Count(&count).Error; err != nil {
		t.Fatalf("count expenses: %v", err)
	}
	if count != 0 {
		t.Errorf("expense rows = %d, want 0 (no mutation without CSRF)", count)
	}
}

// TestHouseholdRole_CSRFLessPostForbidden verifies a POST
// /household/members/:id/role without a CSRF token is rejected with 403 and no
// role change is persisted (threat matrix, RF-8).
func TestHouseholdRole_CSRFLessPostForbidden(t *testing.T) {
	app, db := newIntegrationAppWithDB(t, "test-csrf-key", "test-secret")

	form := "role=admin"
	req := httptest.NewRequest(http.MethodPost, "/household/members/2/role", strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("app.Test() error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("status = %d, want %d (CSRF must block token-less POST)", resp.StatusCode, fiber.StatusForbidden)
	}

	// No user exists, so no row could have been touched — but assert there is
	// no admin-role user either (no partial mutation).
	var count int64
	if err := db.Model(&database.User{}).Where("role = ?", database.RoleAdmin).Count(&count).Error; err != nil {
		t.Fatalf("count users: %v", err)
	}
	if count != 0 {
		t.Errorf("admin-role users = %d, want 0 (no mutation without CSRF)", count)
	}
}

// TestExpenseHTML_E2E drives the expense HTML flow end-to-end (T2.7): register
// → create household → GET /expenses renders HTML → POST valid create → 303 +
// persisted row → POST update → 303 + persisted change → POST delete → 303 +
// row gone. Invalid create → 422 + no row.
func TestExpenseHTML_E2E(t *testing.T) {
	const jwtSecret = "test-secret"
	app, db := newIntegrationAppWithDB(t, "", jwtSecret)
	const email = "expuser@example.com"

	// Register and create a household (mirrors TestHouseholdE2EFlow setup).
	req := httptest.NewRequest(http.MethodPost, "/register",
		strings.NewReader("name=ExpUser&email="+email+"&password=password123"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("register error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusFound || resp.Header.Get("Location") != "/household" {
		t.Fatalf("register: status %d Location %q, want 302 /household", resp.StatusCode, resp.Header.Get("Location"))
	}
	cookie := getCookieValue(resp, "jwt")
	if cookie == "" {
		t.Fatal("register: no jwt cookie")
	}

	req = httptest.NewRequest(http.MethodPost, "/household", strings.NewReader("name=Expense+Family"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cookie", "jwt="+cookie)
	resp, err = app.Test(req, 5000)
	if err != nil {
		t.Fatalf("household create error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusFound || resp.Header.Get("Location") != "/dashboard" {
		t.Fatalf("household: status %d Location %q, want 302 /dashboard", resp.StatusCode, resp.Header.Get("Location"))
	}
	cookie = getCookieValue(resp, "jwt")

	// GET /expenses renders the HTML list (RF-1).
	req = httptest.NewRequest(http.MethodGet, "/expenses", nil)
	req.Header.Set("Cookie", "jwt="+cookie)
	resp, err = app.Test(req, 5000)
	if err != nil {
		t.Fatalf("GET /expenses error: %v", err)
	}
	listBody, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK || !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/html") {
		t.Fatalf("GET /expenses: status %d CT %q, want 200 text/html", resp.StatusCode, resp.Header.Get("Content-Type"))
	}
	if !strings.Contains(string(listBody), "Expenses") {
		t.Errorf("GET /expenses body does not render the Expenses page")
	}

	// Invalid create → 422, nothing persisted (RF-3 edge).
	req = httptest.NewRequest(http.MethodPost, "/expenses",
		strings.NewReader("amount=100&description=&category=Rent&date=2026-07-27"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cookie", "jwt="+cookie)
	resp, err = app.Test(req, 5000)
	if err != nil {
		t.Fatalf("invalid create error: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		t.Errorf("invalid create: status %d, want 422", resp.StatusCode)
	}
	var count int64
	if err := db.Model(&database.Expense{}).Count(&count).Error; err != nil {
		t.Fatalf("count after invalid create: %v", err)
	}
	if count != 0 {
		t.Errorf("rows after invalid create = %d, want 0", count)
	}

	// Valid create → 303 + persisted row (RF-3 happy path).
	req = httptest.NewRequest(http.MethodPost, "/expenses",
		strings.NewReader("amount=85.50&description=Groceries&category=Groceries&date=2026-07-27&visibility=visible_editable"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cookie", "jwt="+cookie)
	resp, err = app.Test(req, 5000)
	if err != nil {
		t.Fatalf("valid create error: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusSeeOther || resp.Header.Get("Location") != "/expenses" {
		t.Fatalf("valid create: status %d Location %q, want 303 /expenses", resp.StatusCode, resp.Header.Get("Location"))
	}
	var expense database.Expense
	if err := db.First(&expense).Error; err != nil {
		t.Fatalf("created expense not persisted: %v", err)
	}
	if expense.Description != "Groceries" || expense.Amount != 85.50 {
		t.Errorf("persisted expense = %+v, want Groceries 85.50", expense)
	}

	// Update → 303 + persisted change (RF-4a).
	updatePath := fmt.Sprintf("/expenses/%d/update", expense.ID)
	req = httptest.NewRequest(http.MethodPost, updatePath,
		strings.NewReader("description=Weekly+Groceries&category=Groceries"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cookie", "jwt="+cookie)
	resp, err = app.Test(req, 5000)
	if err != nil {
		t.Fatalf("update error: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusSeeOther || resp.Header.Get("Location") != "/expenses" {
		t.Fatalf("update: status %d Location %q, want 303 /expenses", resp.StatusCode, resp.Header.Get("Location"))
	}
	if err := db.First(&expense, expense.ID).Error; err != nil {
		t.Fatalf("reload expense: %v", err)
	}
	if expense.Description != "Weekly Groceries" {
		t.Errorf("updated description = %q, want 'Weekly Groceries'", expense.Description)
	}

	// Delete → 303 + row gone (RF-4b).
	deletePath := fmt.Sprintf("/expenses/%d/delete", expense.ID)
	req = httptest.NewRequest(http.MethodPost, deletePath, nil)
	req.Header.Set("Cookie", "jwt="+cookie)
	resp, err = app.Test(req, 5000)
	if err != nil {
		t.Fatalf("delete error: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusSeeOther || resp.Header.Get("Location") != "/expenses" {
		t.Fatalf("delete: status %d Location %q, want 303 /expenses", resp.StatusCode, resp.Header.Get("Location"))
	}
	if err := db.Model(&database.Expense{}).Count(&count).Error; err != nil {
		t.Fatalf("count after delete: %v", err)
	}
	if count != 0 {
		t.Errorf("rows after delete = %d, want 0", count)
	}
}

// TestRootRedirect_Unauthenticated verifies GET / redirects to /login when no JWT cookie.
// Covers spec: root redirect — unauthenticated → 302 /login.
func TestRootRedirect_Unauthenticated(t *testing.T) {
	app := newIntegrationApp(t, "", "test-secret")

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
	validToken, err := services.CreateToken(1, nil, "member", "user@example.com", false, jwtSecret, 24)
	if err != nil {
		t.Fatalf("CreateToken error: %v", err)
	}
	expiredToken, err := services.CreateToken(1, nil, "member", "user@example.com", false, jwtSecret, -1)
	if err != nil {
		t.Fatalf("CreateToken(expired) error: %v", err)
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
			name:       "expired JWT redirects to login",
			cookie:     "jwt=" + expiredToken,
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
	app := newIntegrationApp(t, "", "test-secret")

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("app.Test() error: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("status code = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	if !strings.Contains(string(body), `action="/login"`) {
		t.Errorf("response body does not contain the login form")
	}
}

// TestRegisterRoute_Accessible verifies GET /register returns 200 for public access.
// Covers spec: public auth routes are accessible without authentication.
func TestRegisterRoute_Accessible(t *testing.T) {
	app := newIntegrationApp(t, "", "test-secret")

	req := httptest.NewRequest(http.MethodGet, "/register", nil)
	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("app.Test() error: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("status code = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	if !strings.Contains(string(body), `action="/register"`) {
		t.Errorf("response body does not contain the register form")
	}
}

// TestProtectedRoute_RedirectWithoutCookie verifies GET /dashboard redirects to /login without JWT cookie.
// Covers spec: protected routes redirect to /login without valid JWT.
func TestProtectedRoute_RedirectWithoutCookie(t *testing.T) {
	app := newIntegrationApp(t, "", "test-secret")

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

// TestDashboardRoute_InvalidTokenRedirects verifies GET /dashboard with an
// invalid JWT redirects to /login (RequireAuth rejects before the handler).
// Covers spec: protected routes redirect to /login without a valid JWT.
func TestDashboardRoute_InvalidTokenRedirects(t *testing.T) {
	app := newIntegrationApp(t, "", "test-secret")

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	req.Header.Set("Cookie", "jwt=test-invalid-token")
	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("app.Test() error: %v", err)
	}
	defer resp.Body.Close()

	// Invalid token should redirect to /login
	if resp.StatusCode != fiber.StatusFound {
		t.Errorf("status code = %d, want %d (redirect for invalid token)", resp.StatusCode, fiber.StatusFound)
	}
	if loc := resp.Header.Get("Location"); loc != "/login" {
		t.Errorf("Location = %q, want %q", loc, "/login")
	}
}

// TestDashboard_WithHouseholdJWT_NoPanic is the regression the old "Dashboard
// (coming soon)" stub hid: a JWT carrying household_id must reach the real
// dashboard handler and render without the *uint Locals panic (PR1, design §6).
func TestDashboard_WithHouseholdJWT_NoPanic(t *testing.T) {
	jwtSecret := "test-secret"
	app := newIntegrationApp(t, "", jwtSecret)

	householdID := uint(1)
	token, err := services.CreateToken(1, &householdID, "member", "user@example.com", false, jwtSecret, 24)
	if err != nil {
		t.Fatalf("CreateToken error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	req.Header.Set("Cookie", "jwt="+token)
	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("app.Test() error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		t.Fatalf("status = %d, want < 500 (handler must not panic)", resp.StatusCode)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	if !strings.Contains(string(body), "Monthly Total:") {
		t.Errorf("response body does not contain the dashboard summary")
	}
}

// TestDashboard_NoHousehold_RedirectsToHousehold verifies RequireHousehold
// blocks users without a household from /dashboard (spec: RequireHousehold).
func TestDashboard_NoHousehold_RedirectsToHousehold(t *testing.T) {
	jwtSecret := "test-secret"
	app := newIntegrationApp(t, "", jwtSecret)

	token, err := services.CreateToken(1, nil, "member", "user@example.com", false, jwtSecret, 24)
	if err != nil {
		t.Fatalf("CreateToken error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	req.Header.Set("Cookie", "jwt="+token)
	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("app.Test() error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, fiber.StatusFound)
	}
	if loc := resp.Header.Get("Location"); loc != "/household" {
		t.Errorf("Location = %q, want %q", loc, "/household")
	}
}

// TestRootRedirect_Integration re-verifies the token-aware root redirect through
// the full app. It SUPERSEDES the stale TestRootRedirectAuthenticated, which
// asserted the pre-change always-/login contract against the stub and would
// have passed even if rootRedirect regressed. Covers spec: Root Redirect Fix.
func TestRootRedirect_Integration(t *testing.T) {
	jwtSecret := "test-secret"
	app := newIntegrationApp(t, "", jwtSecret)

	validToken, err := services.CreateToken(1, nil, "member", "user@example.com", false, jwtSecret, 24)
	if err != nil {
		t.Fatalf("CreateToken error: %v", err)
	}
	expiredToken, err := services.CreateToken(1, nil, "member", "user@example.com", false, jwtSecret, -1)
	if err != nil {
		t.Fatalf("CreateToken(expired) error: %v", err)
	}

	tests := []struct {
		name       string
		cookie     string
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
			name:       "no JWT redirects to login",
			cookie:     "",
			wantStatus: fiber.StatusFound,
			wantLoc:    "/login",
		},
		{
			name:       "invalid JWT redirects to login",
			cookie:     "jwt=not-a-valid-token",
			wantStatus: fiber.StatusFound,
			wantLoc:    "/login",
		},
		{
			name:       "expired JWT redirects to login",
			cookie:     "jwt=" + expiredToken,
			wantStatus: fiber.StatusFound,
			wantLoc:    "/login",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

// TestRegisterRedirectsToHousehold verifies POST /register sends a brand-new
// user (who never belongs to a household) to /household to create or join one.
// Covers spec: Register redirect fix (T4.5).
func TestRegisterRedirectsToHousehold(t *testing.T) {
	app := newIntegrationApp(t, "", "test-secret")

	req := httptest.NewRequest(http.MethodPost, "/register",
		strings.NewReader("name=Alice&email=alice@example.com&password=password123"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("app.Test() error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, fiber.StatusFound)
	}
	if loc := resp.Header.Get("Location"); loc != "/household" {
		t.Errorf("Location = %q, want %q", loc, "/household")
	}
}

// TestAuthNavE2E_EmailInProtectedPage proves the broken-guest-nav fix
// end-to-end (PR1, T1.7): the token issued at registration carries the user's
// email and is_admin claim, and a protected page rendered after login shows the
// logged-in nav with that email instead of the guest nav.
func TestAuthNavE2E_EmailInProtectedPage(t *testing.T) {
	const jwtSecret = "test-secret"
	app := newIntegrationApp(t, "", jwtSecret)
	const email = "navuser@example.com"

	// Register through the real flow — the issued token must carry the email.
	req := httptest.NewRequest(http.MethodPost, "/register",
		strings.NewReader("name=NavUser&email="+email+"&password=password123"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("app.Test() register error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusFound || resp.Header.Get("Location") != "/household" {
		t.Fatalf("register: status %d Location %q, want 302 /household", resp.StatusCode, resp.Header.Get("Location"))
	}
	cookie := getCookieValue(resp, "jwt")
	if cookie == "" {
		t.Fatal("register: no jwt cookie in response")
	}

	claims, err := services.ValidateToken(cookie, jwtSecret)
	if err != nil {
		t.Fatalf("ValidateToken(register cookie): %v", err)
	}
	if claims.Email != email {
		t.Errorf("claims.Email = %q, want %q", claims.Email, email)
	}
	if claims.IsAdmin {
		t.Error("claims.IsAdmin = true, want false for a fresh registration")
	}

	// The protected page must render the logged-in nav with the user's email
	// (the broken-guest-nav fix: RequireAuth now sets the email local).
	req2 := httptest.NewRequest(http.MethodGet, "/household", nil)
	req2.Header.Set("Cookie", "jwt="+cookie)
	resp2, err := app.Test(req2, 5000)
	if err != nil {
		t.Fatalf("app.Test() /household error: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != fiber.StatusOK {
		t.Fatalf("/household status = %d, want %d", resp2.StatusCode, fiber.StatusOK)
	}
	body, err := io.ReadAll(resp2.Body)
	if err != nil {
		t.Fatalf("failed to read /household body: %v", err)
	}
	bodyStr := string(body)
	if !strings.Contains(bodyStr, email) {
		t.Errorf("protected page does not render the user's email %q (guest nav still shown)", email)
	}
	if !strings.Contains(bodyStr, `href="/logout"`) {
		t.Error("protected page does not render the logged-in nav /logout link")
	}
	if strings.Contains(bodyStr, `href="/login"`) {
		t.Error("protected page still renders the guest nav /login link")
	}
}

// inviteCodeRe matches the 8-char [0-9A-Z] invite code rendered on the
// household page after an invite (internal/services/household.go charset).
var inviteCodeRe = regexp.MustCompile(`[0-9A-Z]{8}`)

// TestHouseholdE2EFlow drives the full household onboarding journey through the
// real app: register → create household → invite → join, then asserts the
// persisted DB state. It is the end-to-end wiring proof for T4.7 (routes,
// middleware order, DI). Triangulation: single-use code rejection and the
// non-admin invite 403.
func TestHouseholdE2EFlow(t *testing.T) {
	const jwtSecret = "test-secret"
	app, db := newIntegrationAppWithDB(t, "", jwtSecret)

	// register creates a user via the real auth flow and returns the JWT cookie
	// plus the persisted user ID.
	register := func(name, email, password string) (cookie string, userID uint) {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, "/register",
			strings.NewReader("name="+name+"&email="+email+"&password="+password))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := app.Test(req, 5000)
		if err != nil {
			t.Fatalf("register %s: %v", email, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != fiber.StatusFound || resp.Header.Get("Location") != "/household" {
			t.Fatalf("register %s: status %d Location %q, want 302 /household",
				email, resp.StatusCode, resp.Header.Get("Location"))
		}
		cookie = getCookieValue(resp, "jwt")
		if cookie == "" {
			t.Fatalf("register %s: no jwt cookie in response", email)
		}
		var user database.User
		if err := db.Where("email = ?", email).First(&user).Error; err != nil {
			t.Fatalf("find user %s in db: %v", email, err)
		}
		return cookie, user.ID
	}

	// post sends a form POST with the given jwt cookie and returns the response.
	post := func(path, form, cookie string) *http.Response {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Cookie", "jwt="+cookie)
		resp, err := app.Test(req, 5000)
		if err != nil {
			t.Fatalf("POST %s: %v", path, err)
		}
		return resp
	}

	expectRedirect := func(resp *http.Response, path, wantLoc string) {
		t.Helper()
		defer resp.Body.Close()
		if resp.StatusCode != fiber.StatusFound || resp.Header.Get("Location") != wantLoc {
			t.Fatalf("POST %s: status %d Location %q, want 302 %s",
				path, resp.StatusCode, resp.Header.Get("Location"), wantLoc)
		}
	}

	// A registers and creates "Test Family" → admin.
	cookieA, userA := register("User A", "a@example.com", "password123")
	resp := post("/household", "name=Test+Family", cookieA)
	expectRedirect(resp, "/household", "/dashboard")
	cookieA = getCookieValue(resp, "jwt") // re-issued with household_id + role admin
	if cookieA == "" {
		t.Fatal("no re-issued jwt cookie after household create")
	}

	var household database.Household
	if err := db.Where("name = ?", "Test Family").First(&household).Error; err != nil {
		t.Fatalf("household not persisted: %v", err)
	}
	var adminUser database.User
	if err := db.First(&adminUser, userA).Error; err != nil {
		t.Fatalf("load user A: %v", err)
	}
	if adminUser.HouseholdID == nil || *adminUser.HouseholdID != household.ID {
		t.Errorf("user A HouseholdID = %v, want %d", adminUser.HouseholdID, household.ID)
	}
	if adminUser.Role != database.RoleOwner {
		t.Errorf("user A role = %q, want owner", adminUser.Role)
	}

	// A invites → 200 rendering an 8-char code.
	resp = post("/household/invite", "", cookieA)
	inviteBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read invite response: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("invite: status %d, want 200", resp.StatusCode)
	}
	code := inviteCodeRe.FindString(string(inviteBody))
	if len(code) != 8 {
		t.Fatalf("invite response does not contain an 8-char code; body: %s", inviteBody)
	}

	// B registers and joins with the code → member.
	cookieB, userB := register("User B", "b@example.com", "password123")
	resp = post("/household/join", "code="+code, cookieB)
	expectRedirect(resp, "/household/join", "/dashboard")

	// DB state: B linked to the household as member; invite marked used by B.
	var member database.User
	if err := db.First(&member, userB).Error; err != nil {
		t.Fatalf("load user B: %v", err)
	}
	if member.HouseholdID == nil || *member.HouseholdID != household.ID {
		t.Errorf("user B HouseholdID = %v, want %d", member.HouseholdID, household.ID)
	}
	if member.Role != "member" {
		t.Errorf("user B role = %q, want member", member.Role)
	}
	var invite database.InviteCode
	if err := db.Where("code = ?", code).First(&invite).Error; err != nil {
		t.Fatalf("load invite: %v", err)
	}
	if invite.UsedBy == nil || *invite.UsedBy != userB {
		t.Errorf("invite UsedBy = %v, want %d", invite.UsedBy, userB)
	}

	// TRIANGULATE — error path: a fresh user joining with the used code → 400.
	cookieC, _ := register("User C", "c@example.com", "password123")
	resp = post("/household/join", "code="+code, cookieC)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("join with used code: status %d, want 400", resp.StatusCode)
	}
	resp.Body.Close()

	// TRIANGULATE — non-admin cannot invite: C joins via a fresh invite, then
	// tries to invite → 403 (spec: Only admins can invite).
	resp = post("/household/invite", "", cookieA)
	freshBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read second invite response: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("second invite: status %d, want 200", resp.StatusCode)
	}
	code2 := inviteCodeRe.FindString(string(freshBody))
	if len(code2) != 8 {
		t.Fatalf("second invite response does not contain an 8-char code")
	}
	resp = post("/household/join", "code="+code2, cookieC)
	expectRedirect(resp, "/household/join", "/dashboard")
	cookieC = getCookieValue(resp, "jwt")

	resp = post("/household/invite", "", cookieC)
	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("member invite: status %d, want 403", resp.StatusCode)
	}
	resp.Body.Close()
}

// TestHouseholdRole_E2E drives the role-change flow end-to-end (T3.8): a
// member's role-change POST is rejected with 403; the owner promotes a member
// to admin → 303 and the role is persisted; the owner's own role is immutable.
func TestHouseholdRole_E2E(t *testing.T) {
	const jwtSecret = "test-secret"
	app, db := newIntegrationAppWithDB(t, "", jwtSecret)

	// register creates a user via the real auth flow and returns the JWT cookie
	// plus the persisted user ID.
	register := func(name, email, password string) (cookie string, userID uint) {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, "/register",
			strings.NewReader("name="+name+"&email="+email+"&password="+password))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := app.Test(req, 5000)
		if err != nil {
			t.Fatalf("register %s: %v", email, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != fiber.StatusFound || resp.Header.Get("Location") != "/household" {
			t.Fatalf("register %s: status %d Location %q, want 302 /household",
				email, resp.StatusCode, resp.Header.Get("Location"))
		}
		cookie = getCookieValue(resp, "jwt")
		if cookie == "" {
			t.Fatalf("register %s: no jwt cookie in response", email)
		}
		var user database.User
		if err := db.Where("email = ?", email).First(&user).Error; err != nil {
			t.Fatalf("find user %s in db: %v", email, err)
		}
		return cookie, user.ID
	}

	// post sends a form POST with the given jwt cookie and returns the response.
	post := func(path, form, cookie string) *http.Response {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Cookie", "jwt="+cookie)
		resp, err := app.Test(req, 5000)
		if err != nil {
			t.Fatalf("POST %s: %v", path, err)
		}
		return resp
	}

	// A registers and creates "Role Family" → owner.
	cookieA, userA := register("Owner A", "owner@example.com", "password123")
	resp := post("/household", "name=Role+Family", cookieA)
	if resp.StatusCode != fiber.StatusFound || resp.Header.Get("Location") != "/dashboard" {
		t.Fatalf("create household: status %d Location %q, want 302 /dashboard",
			resp.StatusCode, resp.Header.Get("Location"))
	}
	resp.Body.Close()

	// B registers and joins via invite → member.
	cookieB, userB := register("Member B", "member@example.com", "password123")
	resp = post("/household/invite", "", cookieA)
	inviteBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read invite response: %v", err)
	}
	resp.Body.Close()
	code := inviteCodeRe.FindString(string(inviteBody))
	if len(code) != 8 {
		t.Fatalf("invite response does not contain an 8-char code; body: %s", inviteBody)
	}
	resp = post("/household/join", "code="+code, cookieB)
	if resp.StatusCode != fiber.StatusFound || resp.Header.Get("Location") != "/dashboard" {
		t.Fatalf("join: status %d Location %q, want 302 /dashboard",
			resp.StatusCode, resp.Header.Get("Location"))
	}
	resp.Body.Close()

	// TRIANGULATE — member B tries to change roles → 403, role unchanged.
	resp = post(fmt.Sprintf("/household/members/%d/role", userB), "role=admin", cookieB)
	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("member role change: status %d, want 403", resp.StatusCode)
	}
	resp.Body.Close()
	var stillMember database.User
	if err := db.First(&stillMember, userB).Error; err != nil {
		t.Fatalf("load user B: %v", err)
	}
	if stillMember.Role != database.RoleMember {
		t.Errorf("user B role = %q, want member (must stay unchanged)", stillMember.Role)
	}

	// Owner A promotes B member → admin → 303 and role persisted.
	resp = post(fmt.Sprintf("/household/members/%d/role", userB), "role=admin", cookieA)
	if resp.StatusCode != fiber.StatusFound || resp.Header.Get("Location") != "/household" {
		t.Fatalf("promote: status %d Location %q, want 302 /household",
			resp.StatusCode, resp.Header.Get("Location"))
	}
	resp.Body.Close()

	var promoted database.User
	if err := db.First(&promoted, userB).Error; err != nil {
		t.Fatalf("load user B: %v", err)
	}
	if promoted.Role != database.RoleAdmin {
		t.Errorf("user B role = %q, want admin after promotion", promoted.Role)
	}

	// TRIANGULATE — B is admin but not owner; demoting owner A still 403.
	resp = post(fmt.Sprintf("/household/members/%d/role", userA), "role=member", cookieB)
	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("non-owner role change: status %d, want 403", resp.StatusCode)
	}
	resp.Body.Close()

	var owner database.User
	if err := db.First(&owner, userA).Error; err != nil {
		t.Fatalf("load user A: %v", err)
	}
	if owner.Role != database.RoleOwner {
		t.Errorf("user A role = %q, want owner (owner must never change)", owner.Role)
	}
}

// TestUnknownRoute_Returns404 verifies that unmatched paths return a real 404,
// NOT a redirect to /login. Regression for empty-prefix group fallback where
// RequireAuth ran on every unmatched path (unknown URLs became 302 /login).
func TestUnknownRoute_Returns404(t *testing.T) {
	app := newIntegrationApp(t, "", "test-secret")

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
	app := newIntegrationApp(t, "test-csrf-key", "test-secret")

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
	app := newIntegrationApp(t, "test-csrf-key", "test-secret")

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
	app := newIntegrationApp(t, "test-csrf-key", "test-secret")

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
	app := newIntegrationApp(t, "", "test-secret")

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
