package handlers

import (
	"net/http"
	"net/http/httptest"
	"io"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/homeadmin/internal/database"
	"github.com/homeadmin/internal/services"
	"github.com/stretchr/testify/assert"
)

func TestAdminHandler_Show_RendersPage(t *testing.T) {
	users := []database.User{
		{ID: 1, Email: "admin@example.com", IsAdmin: true},
	}
	households := []database.Household{
		{ID: 1, Name: "HH1", Members: []database.User{{ID: 1, Email: "admin@example.com"}}},
	}
	svc := services.NewSiteAdminService(&mockUserRepoForAdminTest{users: users}, &mockHouseholdRepoForAdminTest{households: households}, nil, nil)
	handler := NewAdminHandler(svc)

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("email", "admin@example.com")
		c.Locals("isAdmin", true)
		c.Locals("csrfToken", "tok")
		return c.Next()
	})
	app.Get("/admin", handler.Show)

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "Admin")
}

func TestAdminHandler_Show_RendersUsersTable(t *testing.T) {
	users := []database.User{
		{ID: 1, Email: "a@example.com", Role: "owner", IsAdmin: true},
	}
	households := []database.Household{}
	svc := services.NewSiteAdminService(&mockUserRepoForAdminTest{users: users}, &mockHouseholdRepoForAdminTest{households: households}, nil, nil)
	handler := NewAdminHandler(svc)

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("email", "admin@example.com")
		c.Locals("isAdmin", true)
		c.Locals("csrfToken", "tok")
		return c.Next()
	})
	app.Get("/admin", handler.Show)

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "a@example.com")
}

func TestAdminHandler_Show_RendersHouseholdsTable(t *testing.T) {
	users := []database.User{}
	households := []database.Household{
		{ID: 1, Name: "Family", Members: []database.User{}},
	}
	svc := services.NewSiteAdminService(&mockUserRepoForAdminTest{users: users}, &mockHouseholdRepoForAdminTest{households: households}, nil, nil)
	handler := NewAdminHandler(svc)

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("email", "admin@example.com")
		c.Locals("isAdmin", true)
		c.Locals("csrfToken", "tok")
		return c.Next()
	})
	app.Get("/admin", handler.Show)

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "Family")
}

type mockUserRepoForAdminTest struct {
	users []database.User
}

func (m *mockUserRepoForAdminTest) Create(user *database.User) error                          { return nil }
func (m *mockUserRepoForAdminTest) CountAndCreate(user *database.User) error                  { return nil }
func (m *mockUserRepoForAdminTest) FindByID(id uint) (*database.User, error)                  { return nil, nil }
func (m *mockUserRepoForAdminTest) FindByEmail(email string) (*database.User, error)           { return nil, nil }
func (m *mockUserRepoForAdminTest) FindByIDWithHousehold(id uint) (*database.User, error)      { return nil, nil }
func (m *mockUserRepoForAdminTest) Update(user *database.User) error                          { return nil }
func (m *mockUserRepoForAdminTest) Delete(id uint) error                                      { return nil }
func (m *mockUserRepoForAdminTest) ListAllUsers() ([]database.User, error)                     { return m.users, nil }

type mockHouseholdRepoForAdminTest struct {
	households []database.Household
}

func (m *mockHouseholdRepoForAdminTest) Create(h *database.Household) error                  { return nil }
func (m *mockHouseholdRepoForAdminTest) FindByID(id uint) (*database.Household, error)        { return nil, nil }
func (m *mockHouseholdRepoForAdminTest) FindByUserID(userID uint) (*database.Household, error) { return nil, nil }
func (m *mockHouseholdRepoForAdminTest) FindByName(name string) (*database.Household, error)  { return nil, nil }
func (m *mockHouseholdRepoForAdminTest) FindByInviteCode(code string) (*database.InviteCode, error) { return nil, nil }
func (m *mockHouseholdRepoForAdminTest) CreateInviteCode(invite *database.InviteCode) error   { return nil }
func (m *mockHouseholdRepoForAdminTest) MarkUsed(inviteID, userID uint) error                 { return nil }
func (m *mockHouseholdRepoForAdminTest) GetMembers(householdID uint) ([]database.User, error) { return nil, nil }
func (m *mockHouseholdRepoForAdminTest) Update(h *database.Household) error                  { return nil }
func (m *mockHouseholdRepoForAdminTest) Delete(id uint) error                                { return nil }
func (m *mockHouseholdRepoForAdminTest) AddMember(householdID, userID uint, role string) error { return nil }
func (m *mockHouseholdRepoForAdminTest) RemoveMember(householdID, userID uint) error          { return nil }
func (m *mockHouseholdRepoForAdminTest) ListAllHouseholds() ([]database.Household, error)     { return m.households, nil }
