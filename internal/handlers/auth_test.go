package handlers

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/homeadmin/internal/database"
	"github.com/homeadmin/internal/middleware"
	"github.com/homeadmin/internal/repositories"
	"github.com/homeadmin/internal/services"
)

// --- Mock UserRepository ---

type mockUserRepo struct {
	createFn         func(user *database.User) error
	countAndCreateFn func(user *database.User) error
	findByEmailFn    func(email string) (*database.User, error)
	findByIDFn       func(id uint) (*database.User, error)
	updateFn         func(user *database.User) error
	listAllUsersFn   func() ([]database.User, error)
}

func (m *mockUserRepo) Create(user *database.User) error {
	if m.createFn != nil {
		return m.createFn(user)
	}
	return nil
}

func (m *mockUserRepo) CountAndCreate(user *database.User) error {
	if m.countAndCreateFn != nil {
		return m.countAndCreateFn(user)
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

func (m *mockUserRepo) Update(user *database.User) error {
	if m.updateFn != nil {
		return m.updateFn(user)
	}
	return nil
}
func (m *mockUserRepo) Delete(id uint) error { return nil }
func (m *mockUserRepo) FindByIDWithHousehold(id uint) (*database.User, error) {
	return nil, nil
}

func (m *mockUserRepo) ListAllUsers() ([]database.User, error) {
	if m.listAllUsersFn != nil {
		return m.listAllUsersFn()
	}
	return nil, nil
}

// Verify interface compliance at compile time
var _ repositories.UserRepository = (*mockUserRepo)(nil)

// --- Test helpers ---

const testJWTSecret = "handler-test-secret"

func setupHandlerApp(repo repositories.UserRepository) *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: middleware.ErrorHandler,
	})
	handler := NewAuthHandler(repo, testJWTSecret)

	app.Get("/login", handler.ShowLogin)
	app.Post("/login", handler.Login)
	app.Get("/register", handler.ShowRegister)
	app.Post("/register", handler.Register)
	app.Post("/logout", handler.Logout)
	app.Post("/settings/lang", handler.LangSwitch)

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
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", ct)
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
	form.Set("name", "Test User")

	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	// Should redirect to /household — new users always have a nil household
	// (mirrors Login's nil-household branch).
	if resp.StatusCode != fiber.StatusFound {
		t.Errorf("expected redirect 302, got %d", resp.StatusCode)
	}
	location := resp.Header.Get("Location")
	if location != "/household" {
		t.Errorf("expected redirect to /household, got %s", location)
	}

	// Should set JWT cookie with the standard attributes (approval: attrs must
	// stay unchanged when the handler refactored to the shared SetJWTCookie).
	found := false
	for _, c := range resp.Cookies() {
		if c.Name == "jwt" && c.Value != "" {
			found = true
			if !c.HttpOnly {
				t.Error("expected JWT cookie to be HttpOnly")
			}
			if c.SameSite != http.SameSiteStrictMode {
				t.Errorf("expected JWT cookie SameSite Strict, got %v", c.SameSite)
			}
			if c.Path != "/" {
				t.Errorf("expected JWT cookie Path /, got %q", c.Path)
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
	form.Set("name", "Existing User")

	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	// Duplicate email check returns 200 with HTML (user-facing error)
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

	// Submit with empty email and missing name
	form := url.Values{}
	form.Set("email", "")
	form.Set("password", "securepass123")
	form.Set("name", "")

	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	// Centralized validation returns 400
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("expected status 400 for validation error, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	if !strings.Contains(bodyStr, "Please correct the errors below") {
		t.Errorf("expected validation summary message, got: %s", bodyStr)
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
	form.Set("name", "Test User")

	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	// Centralized validation returns 400
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("expected status 400 for weak password validation, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	if !strings.Contains(bodyStr, "Please correct the errors below") {
		t.Errorf("expected validation summary message, got: %s", bodyStr)
	}

	// Should NOT set JWT cookie
	for _, c := range resp.Cookies() {
		if c.Name == "jwt" {
			t.Error("should not set JWT cookie for weak password")
		}
	}
}

// --- Token claims tests (RF-12) ---

func TestRegister_IssuesTokenWithEmailAndIsAdmin(t *testing.T) {
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
	form.Set("name", "Test User")

	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusFound {
		t.Fatalf("expected redirect 302, got %d", resp.StatusCode)
	}

	claims := decodeJWTCookie(t, resp, testJWTSecret)
	if claims.Email != "new@example.com" {
		t.Errorf("expected token Email=new@example.com, got %q", claims.Email)
	}
	if claims.IsAdmin {
		t.Error("expected token IsAdmin=false for a fresh registration")
	}
}

func TestLogin_IssuesTokenWithEmailAndIsAdmin(t *testing.T) {
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
		t.Fatalf("expected redirect 302, got %d", resp.StatusCode)
	}

	claims := decodeJWTCookie(t, resp, testJWTSecret)
	if claims.Email != "user@example.com" {
		t.Errorf("expected token Email=user@example.com, got %q", claims.Email)
	}
	if claims.IsAdmin {
		t.Error("expected token IsAdmin=false for a plain member login")
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

// --- LangSwitch tests (WU-5) ---

// langSwitchApp creates a test app with the LangSwitch route and the given mock repo.
func langSwitchApp(repo repositories.UserRepository) *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: middleware.ErrorHandler,
	})
	handler := NewAuthHandler(repo, testJWTSecret)
	app.Post("/settings/lang", handler.LangSwitch)
	return app
}

// createValidJWT creates a valid JWT cookie string for testing.
func createValidJWT(t *testing.T, userID uint, lang string) string {
	t.Helper()
	var householdID uint = 1
	token, err := services.CreateToken(userID, &householdID, "member", "test@example.com", lang, false, testJWTSecret, 24)
	if err != nil {
		t.Fatalf("failed to create test JWT: %v", err)
	}
	return token
}

func TestLangSwitch_Success_SwitchToEs(t *testing.T) {
	var capturedUser *database.User
	repo := &mockUserRepo{
		findByIDFn: func(id uint) (*database.User, error) {
			return &database.User{
				ID:    1,
				Email: "test@example.com",
				Lang:  "en",
				Role:  "member",
			}, nil
		},
		updateFn: func(user *database.User) error {
			capturedUser = user
			return nil
		},
	}
	app := langSwitchApp(repo)

	form := url.Values{}
	form.Set("lang", "es")
	form.Set("csrf", "test-csrf-token")

	req := httptest.NewRequest(http.MethodPost, "/settings/lang", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "jwt", Value: createValidJWT(t, 1, "en")})

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	// Should redirect (302)
	if resp.StatusCode != fiber.StatusFound {
		t.Errorf("expected redirect 302, got %d", resp.StatusCode)
	}

	// Should default redirect to /dashboard
	location := resp.Header.Get("Location")
	if location != "/dashboard" {
		t.Errorf("expected redirect to /dashboard, got %s", location)
	}

	// Should update user.Lang to "es" in DB
	if capturedUser == nil {
		t.Fatal("expected Update to be called on user repo")
	}
	if capturedUser.Lang != "es" {
		t.Errorf("expected user.Lang = 'es', got %q", capturedUser.Lang)
	}

	// Should set new JWT cookie with lang=es in claims
	claims := decodeJWTCookie(t, resp, testJWTSecret)
	if claims.Lang != "es" {
		t.Errorf("expected JWT claim Lang='es', got %q", claims.Lang)
	}
}

func TestLangSwitch_Success_SwitchToEn(t *testing.T) {
	var capturedUser *database.User
	repo := &mockUserRepo{
		findByIDFn: func(id uint) (*database.User, error) {
			return &database.User{
				ID:    1,
				Email: "test@example.com",
				Lang:  "es",
				Role:  "member",
			}, nil
		},
		updateFn: func(user *database.User) error {
			capturedUser = user
			return nil
		},
	}
	app := langSwitchApp(repo)

	form := url.Values{}
	form.Set("lang", "en")
	form.Set("csrf", "test-csrf-token")

	req := httptest.NewRequest(http.MethodPost, "/settings/lang", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "jwt", Value: createValidJWT(t, 1, "es")})

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusFound {
		t.Errorf("expected redirect 302, got %d", resp.StatusCode)
	}

	if capturedUser == nil {
		t.Fatal("expected Update to be called")
	}
	if capturedUser.Lang != "en" {
		t.Errorf("expected user.Lang = 'en', got %q", capturedUser.Lang)
	}

	claims := decodeJWTCookie(t, resp, testJWTSecret)
	if claims.Lang != "en" {
		t.Errorf("expected JWT claim Lang='en', got %q", claims.Lang)
	}
}

func TestLangSwitch_InvalidLang_DefaultsToEn(t *testing.T) {
	var capturedUser *database.User
	repo := &mockUserRepo{
		findByIDFn: func(id uint) (*database.User, error) {
			return &database.User{
				ID:    1,
				Email: "test@example.com",
				Lang:  "en",
				Role:  "member",
			}, nil
		},
		updateFn: func(user *database.User) error {
			capturedUser = user
			return nil
		},
	}
	app := langSwitchApp(repo)

	form := url.Values{}
	form.Set("lang", "fr") // invalid — not "en" or "es"
	form.Set("csrf", "test-csrf-token")

	req := httptest.NewRequest(http.MethodPost, "/settings/lang", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "jwt", Value: createValidJWT(t, 1, "en")})

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusFound {
		t.Errorf("expected redirect 302, got %d", resp.StatusCode)
	}

	// Invalid lang should fall back to "en"
	if capturedUser == nil {
		t.Fatal("expected Update to be called")
	}
	if capturedUser.Lang != "en" {
		t.Errorf("expected user.Lang = 'en' (fallback), got %q", capturedUser.Lang)
	}

	claims := decodeJWTCookie(t, resp, testJWTSecret)
	if claims.Lang != "en" {
		t.Errorf("expected JWT claim Lang='en' (fallback), got %q", claims.Lang)
	}
}

func TestLangSwitch_EmptyLang_DefaultsToEn(t *testing.T) {
	var capturedUser *database.User
	repo := &mockUserRepo{
		findByIDFn: func(id uint) (*database.User, error) {
			return &database.User{
				ID:    1,
				Email: "test@example.com",
				Lang:  "es",
				Role:  "member",
			}, nil
		},
		updateFn: func(user *database.User) error {
			capturedUser = user
			return nil
		},
	}
	app := langSwitchApp(repo)

	form := url.Values{}
	form.Set("csrf", "test-csrf-token")
	// lang is omitted — should default to "en"

	req := httptest.NewRequest(http.MethodPost, "/settings/lang", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "jwt", Value: createValidJWT(t, 1, "es")})

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	if capturedUser == nil {
		t.Fatal("expected Update to be called")
	}
	if capturedUser.Lang != "en" {
		t.Errorf("expected user.Lang = 'en' (empty fallback), got %q", capturedUser.Lang)
	}
}

func TestLangSwitch_NoJWT_RedirectsToLogin(t *testing.T) {
	repo := &mockUserRepo{}
	app := langSwitchApp(repo)

	form := url.Values{}
	form.Set("lang", "es")
	form.Set("csrf", "test-csrf-token")

	req := httptest.NewRequest(http.MethodPost, "/settings/lang", strings.NewReader(form.Encode()))
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
	if location != "/login" {
		t.Errorf("expected redirect to /login, got %s", location)
	}
}

func TestLangSwitch_InvalidJWT_RedirectsToLogin(t *testing.T) {
	repo := &mockUserRepo{}
	app := langSwitchApp(repo)

	form := url.Values{}
	form.Set("lang", "es")
	form.Set("csrf", "test-csrf-token")

	req := httptest.NewRequest(http.MethodPost, "/settings/lang", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "jwt", Value: "invalid.jwt.token"})

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

func TestLangSwitch_UserNotFound_RedirectsToLogin(t *testing.T) {
	repo := &mockUserRepo{
		findByIDFn: func(id uint) (*database.User, error) {
			return nil, nil // user not found
		},
	}
	app := langSwitchApp(repo)

	form := url.Values{}
	form.Set("lang", "es")
	form.Set("csrf", "test-csrf-token")

	req := httptest.NewRequest(http.MethodPost, "/settings/lang", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "jwt", Value: createValidJWT(t, 1, "en")})

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

func TestLangSwitch_PreservesRefererPath(t *testing.T) {
	repo := &mockUserRepo{
		findByIDFn: func(id uint) (*database.User, error) {
			return &database.User{
				ID:    1,
				Email: "test@example.com",
				Lang:  "en",
				Role:  "member",
			}, nil
		},
		updateFn: func(user *database.User) error {
			return nil
		},
	}
	app := langSwitchApp(repo)

	form := url.Values{}
	form.Set("lang", "es")
	form.Set("csrf", "test-csrf-token")

	req := httptest.NewRequest(http.MethodPost, "/settings/lang", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", "https://example.com/expenses?month=1")
	req.AddCookie(&http.Cookie{Name: "jwt", Value: createValidJWT(t, 1, "en")})

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusFound {
		t.Errorf("expected redirect 302, got %d", resp.StatusCode)
	}
	location := resp.Header.Get("Location")
	if location != "/expenses?month=1" {
		t.Errorf("expected redirect to /expenses?month=1, got %s", location)
	}
}

func TestLangSwitch_ProtocolRelativeReferer_FallsBackToDashboard(t *testing.T) {
	repo := &mockUserRepo{
		findByIDFn: func(id uint) (*database.User, error) {
			return &database.User{
				ID:    1,
				Email: "test@example.com",
				Lang:  "en",
				Role:  "member",
			}, nil
		},
		updateFn: func(user *database.User) error {
			return nil
		},
	}
	app := langSwitchApp(repo)

	form := url.Values{}
	form.Set("lang", "es")
	form.Set("csrf", "test-csrf-token")

	req := httptest.NewRequest(http.MethodPost, "/settings/lang", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", "//evil.example.com/steal")
	req.AddCookie(&http.Cookie{Name: "jwt", Value: createValidJWT(t, 1, "en")})

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusFound {
		t.Errorf("expected redirect 302, got %d", resp.StatusCode)
	}
	location := resp.Header.Get("Location")
	// Protocol-relative Referer with "//" prefix should be rejected → fallback
	if location != "/dashboard" {
		t.Errorf("expected redirect to /dashboard (protocol-relative blocked), got %s", location)
	}
}

func TestLangSwitch_UpdateError_Returns500(t *testing.T) {
	repo := &mockUserRepo{
		findByIDFn: func(id uint) (*database.User, error) {
			return &database.User{
				ID:    1,
				Email: "test@example.com",
				Lang:  "en",
				Role:  "member",
			}, nil
		},
		updateFn: func(user *database.User) error {
			return fmt.Errorf("database error")
		},
	}
	app := langSwitchApp(repo)

	form := url.Values{}
	form.Set("lang", "es")
	form.Set("csrf", "test-csrf-token")

	req := httptest.NewRequest(http.MethodPost, "/settings/lang", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "jwt", Value: createValidJWT(t, 1, "en")})

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", resp.StatusCode)
	}
}

// --- Anonymous lang switcher (auth-page nav) ---

// TestResolveAnonLang covers the visitor language resolution precedence:
// valid "lang" cookie values win, everything else falls back to English.
// Once logged in, RequireAuth overrides this via the JWT claim.
func TestResolveAnonLang(t *testing.T) {
	tests := []struct {
		name   string
		cookie string // raw Cookie header value; "" = no cookie
		want   string
	}{
		{name: "no cookie defaults to en", cookie: "", want: "en"},
		{name: "es cookie resolves es", cookie: "lang=es", want: "es"},
		{name: "en cookie resolves en", cookie: "lang=en", want: "en"},
		{name: "unsupported value falls back to en", cookie: "lang=fr", want: "en"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got string
			app := fiber.New()
			app.Get("/probe", func(c *fiber.Ctx) error {
				got = ResolveAnonLang(c)
				return nil
			})

			req := httptest.NewRequest(http.MethodGet, "/probe", nil)
			if tt.cookie != "" {
				req.Header.Set("Cookie", tt.cookie)
			}
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test failed: %v", err)
			}
			defer resp.Body.Close()

			if got != tt.want {
				t.Errorf("ResolveAnonLang = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestShowLogin_RendersAnonLangSwitcher verifies an anonymous GET /login shows
// both EN|ES forms so visitors can pick a language before having an account.
func TestShowLogin_RendersAnonLangSwitcher(t *testing.T) {
	app := setupHandlerApp(&mockUserRepo{})

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	if got := strings.Count(bodyStr, `action="/settings/lang"`); got != 2 {
		t.Errorf("login page renders %d lang switcher forms, want 2", got)
	}
	if !strings.Contains(bodyStr, `value="en"`) || !strings.Contains(bodyStr, `value="es"`) {
		t.Error("login page lang switcher missing the EN/ES hidden inputs")
	}
}

// TestShowRegister_RendersAnonLangSwitcher verifies the same for /register.
func TestShowRegister_RendersAnonLangSwitcher(t *testing.T) {
	app := setupHandlerApp(&mockUserRepo{})

	req := httptest.NewRequest(http.MethodGet, "/register", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	if got := strings.Count(bodyStr, `action="/settings/lang"`); got != 2 {
		t.Errorf("register page renders %d lang switcher forms, want 2", got)
	}
}

// TestShowLogin_AnonLangCookie_RendersSpanish verifies the cookie chosen via
// the anonymous switcher drives the rendered language on /login: document
// lang attribute plus exactly the ES pill highlighted.
func TestShowLogin_AnonLangCookie_RendersSpanish(t *testing.T) {
	app := setupHandlerApp(&mockUserRepo{})

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	req.Header.Set("Cookie", "lang=es")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	if !strings.Contains(bodyStr, `<html lang="es"`) {
		t.Error("expected <html lang=\"es\"> on login page with lang=es cookie")
	}
	if !strings.Contains(bodyStr, `class="bg-blue-600 text-white px-2 py-0.5">ES</button>`) {
		t.Error("expected ES switcher button to be highlighted as active")
	}
	if strings.Contains(bodyStr, `class="bg-blue-600 text-white px-2 py-0.5">EN</button>`) {
		t.Error("EN switcher button must not be active while lang=es")
	}
}

// TestShowRegister_AnonLangCookie_RendersSpanish verifies the same resolution
// on /register.
func TestShowRegister_AnonLangCookie_RendersSpanish(t *testing.T) {
	app := setupHandlerApp(&mockUserRepo{})

	req := httptest.NewRequest(http.MethodGet, "/register", nil)
	req.Header.Set("Cookie", "lang=es")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	if !strings.Contains(bodyStr, `<html lang="es"`) {
		t.Error("expected <html lang=\"es\"> on register page with lang=es cookie")
	}
}

// TestLogin_AnonLangCookie_RendersSpanishLabels verifies the login page body
// itself is translated when the visitor's lang cookie is es (root-cause fix:
// labels were previously hardcoded English).
func TestLogin_AnonLangCookie_RendersSpanishLabels(t *testing.T) {
	app := setupHandlerApp(&mockUserRepo{})

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	req.Header.Set("Cookie", "lang=es")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	for _, want := range []string{"Iniciar sesión", "Correo electrónico:", "Contraseña:", "¿No tienes una cuenta? Regístrate"} {
		if !strings.Contains(bodyStr, want) {
			t.Errorf("expected Spanish label %q on login page with lang=es cookie", want)
		}
	}
	if strings.Contains(bodyStr, ">Login</h1>") || strings.Contains(bodyStr, "have an account? Register") {
		t.Error("login page still renders hardcoded English copy with lang=es")
	}
}

// TestRegister_AnonLangCookie_RendersSpanishLabels verifies the same for /register.
func TestRegister_AnonLangCookie_RendersSpanishLabels(t *testing.T) {
	app := setupHandlerApp(&mockUserRepo{})

	req := httptest.NewRequest(http.MethodGet, "/register", nil)
	req.Header.Set("Cookie", "lang=es")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	for _, want := range []string{"Registrarse", "Nombre:", "¿Ya tienes una cuenta? Inicia sesión"} {
		if !strings.Contains(bodyStr, want) {
			t.Errorf("expected Spanish label %q on register page with lang=es cookie", want)
		}
	}
	if strings.Contains(bodyStr, ">Register</h1>") || strings.Contains(bodyStr, "have an account? Login") {
		t.Error("register page still renders hardcoded English copy with lang=es")
	}
}

// TestLogin_FailedLogin_TranslatedErrorPerLang verifies the inline
// invalid-credentials message follows the visitor's language: English by
// default, Spanish with the lang=es cookie.
func TestLogin_FailedLogin_TranslatedErrorPerLang(t *testing.T) {
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

	postLogin := func(cookie string) string {
		t.Helper()
		form := url.Values{}
		form.Set("email", "user@example.com")
		form.Set("password", "wrongpassword")
		req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		if cookie != "" {
			req.Header.Set("Cookie", cookie)
		}
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("app.Test failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != fiber.StatusOK {
			t.Fatalf("expected status 200 for failed login, got %d", resp.StatusCode)
		}
		body, _ := io.ReadAll(resp.Body)
		return string(body)
	}

	enBody := postLogin("")
	if !strings.Contains(enBody, "Invalid credentials") {
		t.Error("expected 'Invalid credentials' in English failed-login render")
	}
	if strings.Contains(enBody, "Credenciales inválidas") {
		t.Error("English render must not contain the Spanish message")
	}

	esBody := postLogin("lang=es")
	if !strings.Contains(esBody, "Credenciales inválidas") {
		t.Error("expected 'Credenciales inválidas' in Spanish failed-login render")
	}
	if strings.Contains(esBody, "Invalid credentials") {
		t.Error("Spanish render must not contain the English message")
	}
}

// TestRegister_DuplicateEmail_TranslatedErrorPerLang verifies the duplicate
// email inline message follows the visitor's language.
func TestRegister_DuplicateEmail_TranslatedErrorPerLang(t *testing.T) {
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
	}
	app := setupHandlerApp(repo)

	postRegister := func(cookie string) string {
		t.Helper()
		form := url.Values{}
		form.Set("email", "existing@example.com")
		form.Set("password", "securepass123")
		form.Set("name", "Existing User")
		req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		if cookie != "" {
			req.Header.Set("Cookie", cookie)
		}
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("app.Test failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != fiber.StatusOK {
			t.Fatalf("expected status 200 for duplicate email, got %d", resp.StatusCode)
		}
		body, _ := io.ReadAll(resp.Body)
		return string(body)
	}

	enBody := postRegister("")
	if !strings.Contains(enBody, "Email already registered") {
		t.Error("expected 'Email already registered' in English duplicate-email render")
	}

	esBody := postRegister("lang=es")
	if !strings.Contains(esBody, "El correo ya está registrado") {
		t.Error("expected 'El correo ya está registrado' in Spanish duplicate-email render")
	}
}
