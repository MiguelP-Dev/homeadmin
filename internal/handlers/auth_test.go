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
	"github.com/homeadmin/internal/repositories"
	"github.com/homeadmin/internal/services"
)

// --- Mock UserRepository ---

type mockUserRepo struct {
	createFn      func(user *database.User) error
	findByEmailFn func(email string) (*database.User, error)
	findByIDFn    func(id uint) (*database.User, error)
}

func (m *mockUserRepo) Create(user *database.User) error {
	if m.createFn != nil {
		return m.createFn(user)
	}
	return nil
}

func (m *mockUserRepo) FindByEmail(email string) (*database.User, error) {
	if m.findByEmailFn != nil {
		return m.findByEmailFn(email)
	}
	return nil, nil
}

func (m *mockUserRepo) FindByID(id uint) (*database.User, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(id)
	}
	return nil, nil
}

func (m *mockUserRepo) Update(user *database.User) error { return nil }
func (m *mockUserRepo) Delete(id uint) error             { return nil }

// Verify interface compliance at compile time
var _ repositories.UserRepository = (*mockUserRepo)(nil)

// --- Test helpers ---

const testJWTSecret = "handler-test-secret"

func setupHandlerApp(repo repositories.UserRepository) *fiber.App {
	app := fiber.New()
	handler := NewAuthHandler(repo, testJWTSecret)

	app.Get("/login", handler.ShowLogin)
	app.Post("/login", handler.Login)
	app.Get("/register", handler.ShowRegister)
	app.Post("/register", handler.Register)
	app.Post("/logout", handler.Logout)

	return app
}

// --- ShowLogin tests (spec §1.38) ---

func TestShowLogin(t *testing.T) {
	app := setupHandlerApp(&mockUserRepo{})

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
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
	if !strings.Contains(bodyStr, "<form") {
		t.Error("expected HTML form in response body")
	}
	if !strings.Contains(bodyStr, "email") {
		t.Error("expected email field in login form")
	}
	if !strings.Contains(bodyStr, "password") {
		t.Error("expected password field in login form")
	}
}

// --- ShowRegister tests (spec §1.39) ---

func TestShowRegister(t *testing.T) {
	app := setupHandlerApp(&mockUserRepo{})

	req := httptest.NewRequest(http.MethodGet, "/register", nil)
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
	if !strings.Contains(bodyStr, "<form") {
		t.Error("expected HTML form in response body")
	}
	if !strings.Contains(bodyStr, "email") {
		t.Error("expected email field in register form")
	}
	if !strings.Contains(bodyStr, "password") {
		t.Error("expected password field in register form")
	}
}

// --- Login tests (spec §1.40-1.42) ---

func TestLogin_Success_NoHousehold(t *testing.T) {
	hash, _ := services.HashPassword("pass1234")
	repo := &mockUserRepo{
		findByEmailFn: func(email string) (*database.User, error) {
			return &database.User{
				ID:           1,
				Email:        email,
				PasswordHash: hash,
				Role:         "member",
			}, nil
		},
	}
	app := setupHandlerApp(repo)

	form := url.Values{}
	form.Set("email", "user@example.com")
	form.Set("password", "pass1234")

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	// Should redirect (302) to /household when no household_id
	if resp.StatusCode != fiber.StatusFound {
		t.Errorf("expected redirect 302, got %d", resp.StatusCode)
	}
	location := resp.Header.Get("Location")
	if location != "/household" {
		t.Errorf("expected redirect to /household, got %s", location)
	}

	// Should set JWT cookie
	cookies := resp.Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == "jwt" && c.Value != "" {
			found = true
			if !c.HttpOnly {
				t.Error("expected JWT cookie to be HttpOnly")
			}
			break
		}
	}
	if !found {
		t.Error("expected JWT cookie to be set")
	}
}

func TestLogin_Success_WithHousehold(t *testing.T) {
	hash, _ := services.HashPassword("pass1234")
	var householdID uint = 5
	repo := &mockUserRepo{
		findByEmailFn: func(email string) (*database.User, error) {
			return &database.User{
				ID:           2,
				Email:        email,
				PasswordHash: hash,
				Role:         "member",
				HouseholdID:  &householdID,
			}, nil
		},
	}
	app := setupHandlerApp(repo)

	form := url.Values{}
	form.Set("email", "user@example.com")
	form.Set("password", "pass1234")

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusFound {
		t.Errorf("expected redirect 302, got %d", resp.StatusCode)
	}
	location := resp.Header.Get("Location")
	if location != "/dashboard" {
		t.Errorf("expected redirect to /dashboard, got %s", location)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	hash, _ := services.HashPassword("correctpass")
	repo := &mockUserRepo{
		findByEmailFn: func(email string) (*database.User, error) {
			return &database.User{
				ID:           1,
				Email:        email,
				PasswordHash: hash,
				Role:         "member",
			}, nil
		},
	}
	app := setupHandlerApp(repo)

	form := url.Values{}
	form.Set("email", "user@example.com")
	form.Set("password", "wrongpassword")

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	// Should return 200 (not redirect) with error message
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected status 200 for failed login, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "Invalid credentials") {
		t.Error("expected 'Invalid credentials' message in response")
	}

	// Should NOT set JWT cookie
	for _, c := range resp.Cookies() {
		if c.Name == "jwt" {
			t.Error("should not set JWT cookie on failed login")
		}
	}
}

func TestLogin_NonexistentEmail(t *testing.T) {
	repo := &mockUserRepo{
		findByEmailFn: func(email string) (*database.User, error) {
			return nil, nil // user not found
		},
	}
	app := setupHandlerApp(repo)

	form := url.Values{}
	form.Set("email", "nobody@example.com")
	form.Set("password", "somepassword")

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	// Same response as wrong password — no email enumeration
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "Invalid credentials") {
		t.Error("expected 'Invalid credentials' message (same as wrong password, no enumeration)")
	}
}

// --- Register tests (spec §1.43-1.45) ---

func TestRegister_Success(t *testing.T) {
	repo := &mockUserRepo{
		createFn: func(user *database.User) error {
			user.ID = 10 // simulate auto-generated ID
			return nil
		},
		findByEmailFn: func(email string) (*database.User, error) {
			return nil, nil // no existing user
		},
	}
	app := setupHandlerApp(repo)

	form := url.Values{}
	form.Set("email", "new@example.com")
	form.Set("password", "securepass123")

	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	// Should redirect to /dashboard
	if resp.StatusCode != fiber.StatusFound {
		t.Errorf("expected redirect 302, got %d", resp.StatusCode)
	}
	location := resp.Header.Get("Location")
	if location != "/dashboard" {
		t.Errorf("expected redirect to /dashboard, got %s", location)
	}

	// Should set JWT cookie
	found := false
	for _, c := range resp.Cookies() {
		if c.Name == "jwt" && c.Value != "" {
			found = true
			if !c.HttpOnly {
				t.Error("expected JWT cookie to be HttpOnly")
			}
			break
		}
	}
	if !found {
		t.Error("expected JWT cookie to be set")
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	existingUser := &database.User{
		ID:           1,
		Email:        "existing@example.com",
		PasswordHash: "somehash",
		Role:         "member",
	}
	repo := &mockUserRepo{
		findByEmailFn: func(email string) (*database.User, error) {
			if email == "existing@example.com" {
				return existingUser, nil
			}
			return nil, nil
		},
		createFn: func(user *database.User) error {
			t.Error("Create should not be called for duplicate email")
			return nil
		},
	}
	app := setupHandlerApp(repo)

	form := url.Values{}
	form.Set("email", "existing@example.com")
	form.Set("password", "securepass123")

	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "Email already registered") {
		t.Error("expected 'Email already registered' message")
	}
}

func TestRegister_MissingFields(t *testing.T) {
	repo := &mockUserRepo{}
	app := setupHandlerApp(repo)

	// Submit with empty email
	form := url.Values{}
	form.Set("email", "")
	form.Set("password", "securepass123")

	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	// Should return 200 with validation error (not 302 redirect)
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected status 200 for validation error, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	if !strings.Contains(bodyStr, "Email and password are required") {
		t.Error("expected 'Email and password are required' message")
	}
}

func TestRegister_WeakPassword(t *testing.T) {
	repo := &mockUserRepo{
		createFn: func(user *database.User) error {
			t.Error("Create should not be called for weak password")
			return nil
		},
	}
	app := setupHandlerApp(repo)

	form := url.Values{}
	form.Set("email", "new@example.com")
	form.Set("password", "short")

	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected status 200 for weak password validation, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	if !strings.Contains(bodyStr, "Password must be at least 8 characters") {
		t.Error("expected 'Password must be at least 8 characters' message")
	}

	// Should NOT set JWT cookie
	for _, c := range resp.Cookies() {
		if c.Name == "jwt" {
			t.Error("should not set JWT cookie for weak password")
		}
	}
}

// --- Logout tests (spec §1.46) ---

func TestLogout(t *testing.T) {
	app := setupHandlerApp(&mockUserRepo{})

	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	// Should redirect to /login
	if resp.StatusCode != fiber.StatusFound {
		t.Errorf("expected redirect 302, got %d", resp.StatusCode)
	}
	location := resp.Header.Get("Location")
	if location != "/login" {
		t.Errorf("expected redirect to /login, got %s", location)
	}

	// Should clear JWT cookie (MaxAge=0)
	found := false
	for _, c := range resp.Cookies() {
		if c.Name == "jwt" {
			found = true
			if c.MaxAge != 0 {
				t.Errorf("expected cookie MaxAge=0 to clear, got %d", c.MaxAge)
			}
			break
		}
	}
	if !found {
		t.Error("expected jwt cookie to be present (cleared) in response")
	}
}
