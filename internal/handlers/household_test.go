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
	createFn        func(userID uint, name string) (*database.Household, error)
	inviteFn        func(userID uint) (string, error)
	joinFn          func(userID uint, code string) (*database.Household, error)
	showFn          func(userID uint) (*services.HouseholdView, error)
	setMemberRoleFn func(ownerID, targetID uint, role string) error
	removeMemberFn  func(ownerID, targetID uint) error
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

func (m *mockHouseholdService) Show(userID uint) (*services.HouseholdView, error) {
	if m.showFn != nil {
		return m.showFn(userID)
	}
	return nil, nil
}

func (m *mockHouseholdService) SetMemberRole(ownerID, targetID uint, role string) error {
	if m.setMemberRoleFn != nil {
		return m.setMemberRoleFn(ownerID, targetID, role)
	}
	return nil
}

func (m *mockHouseholdService) RemoveMember(ownerID, targetID uint) error {
	if m.removeMemberFn != nil {
		return m.removeMemberFn(ownerID, targetID)
	}
	return nil
}

// Verify interface compliance at compile time.
var _ householdServiceInterface = (*mockHouseholdService)(nil)

// --- Test helpers ---

const hhTestJWTSecret = "test-jwt-secret-hh"

func setupHouseholdApp(svc householdServiceInterface) *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: middleware.ErrorHandler,
	})

	handler := NewHouseholdHandler(svc, &mockUserRepo{}, hhTestJWTSecret, 24)

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
	app.Post("/household/members/:id/role", handler.SetMemberRole)
	app.Post("/household/members/:id/remove", handler.RemoveMember)

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
		showFn: func(userID uint) (*services.HouseholdView, error) {
			return nil, nil
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
		showFn: func(userID uint) (*services.HouseholdView, error) {
			return &services.HouseholdView{
				Household:  &database.Household{Name: "My Family"},
				Members:    members,
				ViewerRole: database.RoleOwner,
			}, nil
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
		showFn: func(userID uint) (*services.HouseholdView, error) {
			return &services.HouseholdView{
				Household:  &database.Household{Name: "My Family"},
				Members:    members,
				ViewerRole: database.RoleMember,
			}, nil
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
	userRepo := &mockUserRepo{
		findByIDFn: func(id uint) (*database.User, error) {
			return &database.User{ID: 1, Email: "test@example.com", Role: database.RoleOwner}, nil
		},
	}
	app := fiber.New(fiber.Config{
		ErrorHandler: middleware.ErrorHandler,
	})
	handler := NewHouseholdHandler(svc, userRepo, hhTestJWTSecret, 24)
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userID", uint(1))
		c.Locals("householdID", ptr[uint](1))
		c.Locals("email", "test@example.com")
		c.Locals("csrfToken", "test-csrf-token")
		return c.Next()
	})
	app.Post("/household", handler.Create)

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

	// Decode the JWT cookie and verify claims carry household_id and role=owner.
	claims := decodeJWTCookie(t, resp, hhTestJWTSecret)
	if claims.HouseholdID == nil {
		t.Fatal("expected household_id in JWT claims, got nil")
	}
	if *claims.HouseholdID != hhID {
		t.Errorf("expected household_id=%d in claims, got %d", hhID, *claims.HouseholdID)
	}
	if claims.Role != database.RoleOwner {
		t.Errorf("expected role=%q in claims, got %q", database.RoleOwner, claims.Role)
	}
	// Re-issued token reflects the fresh DB user (design D2): email from the
	// user record, is_admin from User.IsAdmin (slice 5).
	if claims.Email != "test@example.com" {
		t.Errorf("expected Email=test@example.com in claims, got %q", claims.Email)
	}
	if claims.IsAdmin {
		t.Error("expected IsAdmin=false in claims")
	}
}

// TestHouseholdHandler_Create_ReissuesTokenWithIsAdmin verifies the re-issued
// token carries IsAdmin=true when the fresh DB user is a site admin (design D2;
// slice 5 switches the create/join call sites from a hardcoded false).
func TestHouseholdHandler_Create_ReissuesTokenWithIsAdmin(t *testing.T) {
	hhID := uint(7)
	svc := &mockHouseholdService{
		createFn: func(userID uint, name string) (*database.Household, error) {
			return &database.Household{ID: hhID, Name: name}, nil
		},
	}
	userRepo := &mockUserRepo{
		findByIDFn: func(id uint) (*database.User, error) {
			return &database.User{ID: id, Email: "siteadmin@example.com", Role: database.RoleOwner, IsAdmin: true}, nil
		},
	}
	app := fiber.New(fiber.Config{
		ErrorHandler: middleware.ErrorHandler,
	})
	handler := NewHouseholdHandler(svc, userRepo, hhTestJWTSecret, 24)
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userID", uint(1))
		c.Locals("householdID", ptr[uint](1))
		c.Locals("email", "siteadmin@example.com")
		c.Locals("csrfToken", "test-csrf-token")
		return c.Next()
	})
	app.Post("/household", handler.Create)

	form := url.Values{}
	form.Set("name", "Admin Family")

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

	claims := decodeJWTCookie(t, resp, hhTestJWTSecret)
	if !claims.IsAdmin {
		t.Error("expected IsAdmin=true in re-issued token for a site admin")
	}
	if claims.Email != "siteadmin@example.com" {
		t.Errorf("expected Email=siteadmin@example.com in claims, got %q", claims.Email)
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

// --- Invite tests ---

func setupInviteApp(svc householdServiceInterface) *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: middleware.ErrorHandler,
	})

	handler := NewHouseholdHandler(svc, &mockUserRepo{}, hhTestJWTSecret, 24)

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userID", uint(1))
		c.Locals("householdID", ptr[uint](1))
		c.Locals("email", "test@example.com")
		c.Locals("csrfToken", "test-csrf-token")
		return c.Next()
	})

	app.Post("/household/invite", handler.Invite)
	app.Post("/household/join", handler.Join)

	return app
}

// setupInviteAppNoHousehold creates a test app where the user has no household.
func setupInviteAppNoHousehold(svc householdServiceInterface) *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: middleware.ErrorHandler,
	})

	handler := NewHouseholdHandler(svc, &mockUserRepo{}, hhTestJWTSecret, 24)

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userID", uint(1))
		// No householdID set — simulates user without household
		c.Locals("email", "test@example.com")
		c.Locals("csrfToken", "test-csrf-token")
		return c.Next()
	})

	app.Post("/household/invite", handler.Invite)
	app.Post("/household/join", handler.Join)

	return app
}

func TestHouseholdHandler_Invite_NoHousehold(t *testing.T) {
	svc := &mockHouseholdService{
		inviteFn: func(userID uint) (string, error) {
			return "", services.ErrNoHousehold
		},
	}
	app := setupInviteAppNoHousehold(svc)

	req := httptest.NewRequest(http.MethodPost, "/household/invite", nil)
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
	if !strings.Contains(html, "You do not belong to a household") {
		t.Errorf("expected error 'You do not belong to a household', got: %s", html)
	}
}

func TestHouseholdHandler_Invite_NonAdmin(t *testing.T) {
	svc := &mockHouseholdService{
		inviteFn: func(userID uint) (string, error) {
			return "", services.ErrNotAdmin
		},
	}
	app := setupInviteApp(svc)

	req := httptest.NewRequest(http.MethodPost, "/household/invite", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("expected status 403, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	html := string(body)
	if !strings.Contains(html, "Only owners can manage household members") {
		t.Errorf("expected error 'Only owners can manage household members', got: %s", html)
	}
}

func TestHouseholdHandler_Invite_Success(t *testing.T) {
	svc := &mockHouseholdService{
		inviteFn: func(userID uint) (string, error) {
			return "ABC12345", nil
		},
		showFn: func(userID uint) (*services.HouseholdView, error) {
			return &services.HouseholdView{
				Household:  &database.Household{ID: 1, Name: "My Family"},
				Members:    []database.User{{ID: 1, Email: "test@example.com", Role: database.RoleOwner}},
				ViewerRole: database.RoleOwner,
			}, nil
		},
	}
	app := setupInviteApp(svc)

	req := httptest.NewRequest(http.MethodPost, "/household/invite", nil)
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
	if !strings.Contains(html, "ABC12345") {
		t.Errorf("expected invite code 'ABC12345' in response body, got: %s", html)
	}
}

// --- Join tests ---

func TestHouseholdHandler_Join_InvalidCode(t *testing.T) {
	svc := &mockHouseholdService{
		joinFn: func(userID uint, code string) (*database.Household, error) {
			return nil, services.ErrInvalidCode
		},
	}
	app := setupInviteAppNoHousehold(svc)

	form := url.Values{}
	form.Set("code", "XXXXXXX")

	req := httptest.NewRequest(http.MethodPost, "/household/join", strings.NewReader(form.Encode()))
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
	if !strings.Contains(html, "Invalid invite code") {
		t.Errorf("expected error 'Invalid invite code', got: %s", html)
	}
}

func TestHouseholdHandler_Join_ExpiredCode(t *testing.T) {
	svc := &mockHouseholdService{
		joinFn: func(userID uint, code string) (*database.Household, error) {
			return nil, services.ErrExpiredCode
		},
	}
	app := setupInviteAppNoHousehold(svc)

	form := url.Values{}
	form.Set("code", "EXP12345")

	req := httptest.NewRequest(http.MethodPost, "/household/join", strings.NewReader(form.Encode()))
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
	if !strings.Contains(html, "This invite code has expired") {
		t.Errorf("expected error 'This invite code has expired', got: %s", html)
	}
}

func TestHouseholdHandler_Join_UsedCode(t *testing.T) {
	svc := &mockHouseholdService{
		joinFn: func(userID uint, code string) (*database.Household, error) {
			return nil, services.ErrUsedCode
		},
	}
	app := setupInviteAppNoHousehold(svc)

	form := url.Values{}
	form.Set("code", "USED1234")

	req := httptest.NewRequest(http.MethodPost, "/household/join", strings.NewReader(form.Encode()))
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
	if !strings.Contains(html, "This invite code has already been used") {
		t.Errorf("expected error 'This invite code has already been used', got: %s", html)
	}
}

func TestHouseholdHandler_Join_AlreadyInHousehold(t *testing.T) {
	svc := &mockHouseholdService{
		joinFn: func(userID uint, code string) (*database.Household, error) {
			return nil, services.ErrAlreadyHasHousehold
		},
	}
	app := setupInviteApp(svc) // user has householdID set

	form := url.Values{}
	form.Set("code", "ABC12345")

	req := httptest.NewRequest(http.MethodPost, "/household/join", strings.NewReader(form.Encode()))
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
		t.Errorf("expected error 'You already belong to a household', got: %s", html)
	}
}

func TestHouseholdHandler_Join_Success(t *testing.T) {
	hhID := uint(99)
	svc := &mockHouseholdService{
		joinFn: func(userID uint, code string) (*database.Household, error) {
			return &database.Household{ID: hhID, Name: "Joined Family"}, nil
		},
	}
	userRepo := &mockUserRepo{
		findByIDFn: func(id uint) (*database.User, error) {
			return &database.User{ID: 1, Email: "joiner@example.com", Role: "member"}, nil
		},
	}
	app := fiber.New(fiber.Config{
		ErrorHandler: middleware.ErrorHandler,
	})
	handler := NewHouseholdHandler(svc, userRepo, hhTestJWTSecret, 24)
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userID", uint(1))
		c.Locals("email", "joiner@example.com")
		c.Locals("csrfToken", "test-csrf-token")
		return c.Next()
	})
	app.Post("/household/join", handler.Join)

	form := url.Values{}
	form.Set("code", "ABC12345")

	req := httptest.NewRequest(http.MethodPost, "/household/join", strings.NewReader(form.Encode()))
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

	// Decode the JWT cookie and verify claims carry household_id and role=member.
	claims := decodeJWTCookie(t, resp, hhTestJWTSecret)
	if claims.HouseholdID == nil {
		t.Fatal("expected household_id in JWT claims, got nil")
	}
	if *claims.HouseholdID != hhID {
		t.Errorf("expected household_id=%d in claims, got %d", hhID, *claims.HouseholdID)
	}
	if claims.Role != "member" {
		t.Errorf("expected role=%q in claims, got %q", "member", claims.Role)
	}
	// Re-issued token reflects the fresh DB user (design D2).
	if claims.Email != "joiner@example.com" {
		t.Errorf("expected Email=joiner@example.com in claims, got %q", claims.Email)
	}
	if claims.IsAdmin {
		t.Error("expected IsAdmin=false in claims")
	}
}

// --- SetMemberRole (spec: change member role) ---

func TestHouseholdHandler_SetMemberRole(t *testing.T) {
	tests := []struct {
		name       string
		targetPath string
		wantTarget uint
		formRole   string
		setErr     error
		wantStatus int
	}{
		{
			name:       "owner promotes member to admin",
			targetPath: "/household/members/2/role",
			wantTarget: 2,
			formRole:   database.RoleAdmin,
			wantStatus: fiber.StatusFound,
		},
		{
			name:       "member cannot change roles",
			targetPath: "/household/members/2/role",
			wantTarget: 2,
			formRole:   database.RoleAdmin,
			setErr:     services.ErrNotOwner,
			wantStatus: fiber.StatusForbidden,
		},
		{
			name:       "self change rejected",
			targetPath: "/household/members/1/role",
			wantTarget: 1,
			formRole:   database.RoleMember,
			setErr:     services.ErrSelfRoleChange,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "other owner immutable",
			targetPath: "/household/members/4/role",
			wantTarget: 4,
			formRole:   database.RoleMember,
			setErr:     services.ErrOwnerImmutable,
			wantStatus: fiber.StatusForbidden,
		},
		{
			name:       "cross-household target rejected",
			targetPath: "/household/members/3/role",
			wantTarget: 3,
			formRole:   database.RoleAdmin,
			setErr:     services.ErrNotMember,
			wantStatus: fiber.StatusNotFound,
		},
		{
			name:       "invalid role rejected",
			targetPath: "/household/members/2/role",
			wantTarget: 2,
			formRole:   "superadmin",
			setErr:     services.ErrValidation,
			wantStatus: fiber.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotOwner, gotTarget uint
			var gotRole string
			svc := &mockHouseholdService{
				setMemberRoleFn: func(ownerID, targetID uint, role string) error {
					gotOwner, gotTarget, gotRole = ownerID, targetID, role
					return tt.setErr
				},
			}
			app := setupHouseholdApp(svc)

			form := url.Values{}
			form.Set("role", tt.formRole)

			req := httptest.NewRequest(http.MethodPost, tt.targetPath, strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, resp.StatusCode)
			}
			if gotOwner != 1 {
				t.Errorf("SetMemberRole called with ownerID %d, want 1", gotOwner)
			}
			if gotTarget != tt.wantTarget {
				t.Errorf("SetMemberRole called with targetID %d, want %d", gotTarget, tt.wantTarget)
			}
			if gotRole != tt.formRole {
				t.Errorf("SetMemberRole called with role %q, want %q", gotRole, tt.formRole)
			}
			if tt.wantStatus == fiber.StatusFound {
				if location := resp.Header.Get("Location"); location != "/household" {
					t.Errorf("expected redirect to /household, got %q", location)
				}
			}
		})
	}
}

// --- RemoveMember tests (WU-7.3) ---

func TestHouseholdHandler_RemoveMember(t *testing.T) {
	tests := []struct {
		name       string
		targetID   int
		removeErr  error
		wantStatus int
		wantBody   string
	}{
		{
			name:       "owner removes member",
			targetID:   2,
			wantStatus: fiber.StatusFound,
		},
		{
			name:       "self-removal rejected",
			targetID:   1,
			removeErr:  services.ErrSelfRemoval,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "owner immutable",
			targetID:   4,
			removeErr:  services.ErrOwnerImmutable,
			wantStatus: fiber.StatusForbidden,
		},
		{
			name:       "non-owner forbidden",
			targetID:   2,
			removeErr:  services.ErrNotOwner,
			wantStatus: fiber.StatusForbidden,
		},
		{
			name:       "not member",
			targetID:   3,
			removeErr:  services.ErrNotMember,
			wantStatus: fiber.StatusNotFound,
		},
		{
			name:       "invalid target ID",
			targetID:   0,
			wantStatus: fiber.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotOwner, gotTarget uint
			svc := &mockHouseholdService{
				removeMemberFn: func(ownerID, targetID uint) error {
					gotOwner, gotTarget = ownerID, targetID
					return tt.removeErr
				},
			}
			app := setupHouseholdApp(svc)

			path := "/household/members/" + itoa(tt.targetID) + "/remove"
			req := httptest.NewRequest(http.MethodPost, path, nil)
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, resp.StatusCode)
			}

			// For valid target IDs (non-zero), verify the service was called with correct IDs
			if tt.targetID > 0 && tt.wantStatus != fiber.StatusBadRequest {
				if gotOwner != 1 {
					t.Errorf("RemoveMember called with ownerID %d, want 1", gotOwner)
				}
				if gotTarget != uint(tt.targetID) {
					t.Errorf("RemoveMember called with targetID %d, want %d", gotTarget, tt.targetID)
				}
			}

			// Successful removal redirects to /household
			if tt.wantStatus == fiber.StatusFound {
				if location := resp.Header.Get("Location"); location != "/household" {
					t.Errorf("expected redirect to /household, got %q", location)
				}
			}

			// Error responses contain translated error messages
			if tt.wantStatus >= 400 {
				body, _ := io.ReadAll(resp.Body)
				html := string(body)
				if tt.wantBody != "" && !strings.Contains(html, tt.wantBody) {
					t.Errorf("expected error message %q in body, got: %s", tt.wantBody, html)
				}
			}
		})
	}
}

// itoa is a simple int-to-string helper for building test URLs.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + itoa(-n)
	}
	digits := []byte{}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
