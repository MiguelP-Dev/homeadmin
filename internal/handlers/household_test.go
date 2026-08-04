package handlers

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/homeadmin/internal/database"
	"github.com/homeadmin/internal/middleware"
	"github.com/homeadmin/internal/services"
)

// --- Mock HouseholdService ---

type mockHouseholdService struct {
	createFn func(userID uint, name string) (*database.Household, error)
	inviteFn func(userID uint) (string, error)
	joinFn   func(userID uint, code string) (*database.Household, error)
	showFn   func(userID uint) (*database.Household, []database.User, bool, error)
}

func (m *mockHouseholdService) Create(userID uint, name string) (*database.Household, error) {
	if m.createFn != nil {
		return m.createFn(userID, name)
	}
	return nil, nil
}

func (m *mockHouseholdService) Invite(userID uint) (string, error) {
	if m.inviteFn != nil {
		return m.inviteFn(userID)
	}
	return "", nil
}

func (m *mockHouseholdService) Join(userID uint, code string) (*database.Household, error) {
	if m.joinFn != nil {
		return m.joinFn(userID, code)
	}
	return nil, nil
}

func (m *mockHouseholdService) Show(userID uint) (*database.Household, []database.User, bool, error) {
	if m.showFn != nil {
		return m.showFn(userID)
	}
	return nil, nil, false, nil
}

// Verify interface compliance at compile time.
var _ householdServiceInterface = (*mockHouseholdService)(nil)

// --- Test helpers ---

const hhTestJWTSecret = "test-jwt-secret-hh"

func setupHouseholdApp(svc householdServiceInterface) *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: middleware.ErrorHandler,
	})

	handler := NewHouseholdHandler(svc, hhTestJWTSecret, 24)

	// Middleware to simulate RequireAuth locals.
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userID", uint(1))
		c.Locals("householdID", ptr[uint](1))
		c.Locals("email", "test@example.com")
		c.Locals("csrfToken", "test-csrf-token")
		return c.Next()
	})

	app.Get("/household", handler.Show)
	app.Post("/household", handler.Create)

	return app
}

// decodeJWTCookie extracts the "jwt" cookie value from the response and
// validates it, returning the claims for assertion.
func decodeJWTCookie(t *testing.T, resp *http.Response, secret string) *services.Claims {
	t.Helper()
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "jwt" {
			claims, err := services.ValidateToken(cookie.Value, secret)
			if err != nil {
				t.Fatalf("failed to validate JWT cookie: %v", err)
			}
			return claims
		}
	}
	t.Fatal("no jwt cookie found in response")
	return nil
}

// --- Show tests ---

func TestHouseholdHandler_Show_NoHousehold(t *testing.T) {
	svc := &mockHouseholdService{
		showFn: func(userID uint) (*database.Household, []database.User, bool, error) {
			return nil, nil, false, nil
		},
	}
	app := setupHouseholdApp(svc)

	req := httptest.NewRequest(http.MethodGet, "/household", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	html := string(body)
	if !strings.Contains(html, "Create a household") {
		t.Error("expected HouseholdSetup to render 'Create a household' form")
	}
	if !strings.Contains(html, "Join with an invite code") {
		t.Error("expected HouseholdSetup to render 'Join with an invite code' form")
	}
}

func TestHouseholdHandler_Show_WithHousehold_Admin(t *testing.T) {
	members := []database.User{
		{Email: "admin@example.com", Role: "admin"},
		{Email: "member@example.com", Role: "member"},
	}
	svc := &mockHouseholdService{
		showFn: func(userID uint) (*database.Household, []database.User, bool, error) {
			return &database.Household{Name: "My Family"}, members, true, nil
		},
	}
	app := setupHouseholdApp(svc)

	req := httptest.NewRequest(http.MethodGet, "/household", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	html := string(body)
	if !strings.Contains(html, "My Family") {
		t.Error("expected HouseholdShow to render household name 'My Family'")
	}
	if !strings.Contains(html, "admin@example.com") {
		t.Error("expected member list to show admin email")
	}
	if !strings.Contains(html, "member@example.com") {
		t.Error("expected member list to show member email")
	}
	if !strings.Contains(html, "Invite member") {
		t.Error("expected invite button for admin user")
	}
}

func TestHouseholdHandler_Show_WithHousehold_Member(t *testing.T) {
	members := []database.User{
		{Email: "admin@example.com", Role: "admin"},
		{Email: "test@example.com", Role: "member"},
	}
	svc := &mockHouseholdService{
		showFn: func(userID uint) (*database.Household, []database.User, bool, error) {
			return &database.Household{Name: "My Family"}, members, false, nil
		},
	}
	app := setupHouseholdApp(svc)

	req := httptest.NewRequest(http.MethodGet, "/household", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	html := string(body)
	if !strings.Contains(html, "My Family") {
		t.Error("expected household name in response")
	}
	if strings.Contains(html, "Invite member") {
		t.Error("member should NOT see invite button")
	}
}

// --- Create tests ---

func TestHouseholdHandler_Create_Success(t *testing.T) {
	hhID := uint(42)
	svc := &mockHouseholdService{
		createFn: func(userID uint, name string) (*database.Household, error) {
			if name == "" {
				return nil, services.ErrNameRequired
			}
			return &database.Household{ID: hhID, Name: name}, nil
		},
	}
	app := setupHouseholdApp(svc)

	form := url.Values{}
	form.Set("name", "My Family")

	req := httptest.NewRequest(http.MethodPost, "/household", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusFound {
		t.Errorf("expected status 302, got %d", resp.StatusCode)
	}

	location := resp.Header.Get("Location")
	if location != "/dashboard" {
		t.Errorf("expected redirect to /dashboard, got %q", location)
	}

	// Decode the JWT cookie and verify claims carry household_id and role=admin.
	claims := decodeJWTCookie(t, resp, hhTestJWTSecret)
	if claims.HouseholdID == nil {
		t.Fatal("expected household_id in JWT claims, got nil")
	}
	if *claims.HouseholdID != hhID {
		t.Errorf("expected household_id=%d in claims, got %d", hhID, *claims.HouseholdID)
	}
	if claims.Role != "admin" {
		t.Errorf("expected role=%q in claims, got %q", "admin", claims.Role)
	}
}

func TestHouseholdHandler_Create_AlreadyHasHousehold(t *testing.T) {
	svc := &mockHouseholdService{
		createFn: func(userID uint, name string) (*database.Household, error) {
			return nil, services.ErrAlreadyHasHousehold
		},
	}
	app := setupHouseholdApp(svc)

	form := url.Values{}
	form.Set("name", "New House")

	req := httptest.NewRequest(http.MethodPost, "/household", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("expected status 400, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	html := string(body)
	if !strings.Contains(html, "You already belong to a household") {
		t.Errorf("expected error message 'You already belong to a household' in response body, got: %s", html)
	}
}

func TestHouseholdHandler_Create_EmptyName(t *testing.T) {
	svc := &mockHouseholdService{
		createFn: func(userID uint, name string) (*database.Household, error) {
			return nil, services.ErrNameRequired
		},
	}
	app := setupHouseholdApp(svc)

	form := url.Values{}
	form.Set("name", "")

	req := httptest.NewRequest(http.MethodPost, "/household", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("expected status 400, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	html := string(body)
	if !strings.Contains(html, "Household name is required") {
		t.Errorf("expected error message 'Household name is required' in response body, got: %s", html)
	}
}
