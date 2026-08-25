package handlers

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/homeadmin/internal/database"
	"github.com/homeadmin/internal/middleware"
	"github.com/homeadmin/internal/repositories"
	"github.com/homeadmin/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// overviewFixture returns two household blocks with cross-household data used
// to prove the global views render everything, not just one household.
func overviewFixture() []services.HouseholdBlock {
	thisMonth := time.Now().UTC()
	return []services.HouseholdBlock{
		{
			Household: database.Household{ID: 1, Name: "Alpha Home"},
			Members: []services.HouseholdMember{
				{Email: "alpha-owner@test.com", Role: "owner"},
				{Email: "alpha-member@test.com", Role: "member"},
			},
			Expenses: []repositories.ExpenseWithUser{
				{
					Expense: database.Expense{
						ID: 1, Amount: 1200, Description: "Alpha Rent", Category: "rent",
						Type: database.TransactionTypeExpense,
						Date: time.Date(thisMonth.Year(), thisMonth.Month(), 2, 0, 0, 0, 0, time.UTC),
					},
					OwnerEmail: "alpha-owner@test.com",
				},
			},
			Savings: []repositories.SavingsWithUser{
				{Savings: database.Savings{ID: 1, Description: "Alpha Fund", Amount: 300}, OwnerEmail: "alpha-owner@test.com"},
			},
			MonthlyIncome:  500,
			MonthlyExpense: 1200,
			AllTimeNet:     -700,
			SavingsTotal:   300,
		},
		{
			Household: database.Household{ID: 2, Name: "Beta Home"},
			Members: []services.HouseholdMember{
				{Email: "beta-owner@test.com", Role: "owner"},
			},
			Expenses: []repositories.ExpenseWithUser{
				{
					Expense: database.Expense{
						ID: 2, Amount: 900, Description: "Beta Salary", Category: "other",
						Type: database.TransactionTypeIncome,
						Date: time.Date(thisMonth.Year(), thisMonth.Month(), 3, 0, 0, 0, 0, time.UTC),
					},
					OwnerEmail: "beta-owner@test.com",
				},
			},
			MonthlyIncome: 900,
		},
	}
}

func setupAdminBranchApp(handler fiber.Handler, locals map[string]any) *fiber.App {
	app := fiber.New(fiber.Config{ErrorHandler: middleware.ErrorHandler})
	app.Use(func(c *fiber.Ctx) error {
		for k, v := range locals {
			c.Locals(k, v)
		}
		return c.Next()
	})
	app.Get("/route", handler)
	return app
}

func getBody(t *testing.T, app *fiber.App) (int, string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/route", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, string(body)
}

func TestExpenseList_AdminRendersGlobalGroupedView(t *testing.T) {
	svc := &fakeSiteOverview{blocks: overviewFixture()}
	expenseSvc := &mockExpenseService{}
	handler := NewExpenseHandler(expenseSvc)
	handler.SiteAdmin = svc

	app := setupAdminBranchApp(handler.List, map[string]any{
		"userID":      uint(9),
		"householdID": (*uint)(nil),
		"email":       "siteadmin@example.com",
		"isAdmin":     true,
		"csrfToken":   "tok",
		"lang":        "en",
	})

	status, body := getBody(t, app)
	require.Equal(t, fiber.StatusOK, status)

	// Both households appear with their members' emails and transactions.
	assert.Contains(t, body, "All Households")
	assert.Contains(t, body, "Alpha Home")
	assert.Contains(t, body, "Beta Home")
	assert.Contains(t, body, "alpha-owner@test.com")
	assert.Contains(t, body, "beta-owner@test.com")
	assert.Contains(t, body, "Alpha Rent")
	assert.Contains(t, body, "Beta Salary")
	assert.Contains(t, body, "Alpha Fund")
}

func TestExpenseList_AdminWithoutHouseholdStillRendersGlobal(t *testing.T) {
	// Same as above but without any householdID local at all: the middleware
	// bypass plus handler branch mean no BadRequest for admins.
	svc := &fakeSiteOverview{blocks: overviewFixture()}
	handler := NewExpenseHandler(&mockExpenseService{})
	handler.SiteAdmin = svc

	app := setupAdminBranchApp(handler.List, map[string]any{
		"userID":    uint(9),
		"email":     "siteadmin@example.com",
		"isAdmin":   true,
		"csrfToken": "tok",
		"lang":      "en",
	})

	status, _ := getBody(t, app)
	assert.Equal(t, fiber.StatusOK, status)
}

func TestExpenseList_MemberStaysHouseholdScoped(t *testing.T) {
	var requestedHousehold uint
	svc := &mockExpenseService{
		findByHouseholdFn: func(userID, householdID uint, filters database.ExpenseFilters) ([]database.Expense, error) {
			requestedHousehold = householdID
			return []database.Expense{{ID: 5, Description: "Only Mine", Amount: 10, Category: "other", Date: time.Now()}}, nil
		},
	}
	handler := NewExpenseHandler(svc)
	handler.SiteAdmin = &fakeSiteOverview{blocks: overviewFixture()}

	hh := uint(42)
	app := setupAdminBranchApp(handler.List, map[string]any{
		"userID":      uint(2),
		"householdID": &hh,
		"email":       "member@example.com",
		"isAdmin":     false,
		"csrfToken":   "tok",
		"lang":        "en",
	})

	status, body := getBody(t, app)
	require.Equal(t, fiber.StatusOK, status)

	// Scoped service was queried with the member's household...
	assert.Equal(t, uint(42), requestedHousehold)
	assert.Contains(t, body, "Only Mine")
	// ...and none of the global blocks leaked into the page.
	assert.NotContains(t, body, "Alpha Home")
	assert.NotContains(t, body, "Beta Salary")
	assert.Equal(t, "Expenses", strings.TrimSpace("Expenses"))
}

func TestExpenseCreate_AdminWithoutHouseholdStillRejected(t *testing.T) {
	// Writes stay household-scoped even for admins (documented behavior).
	handler := NewExpenseHandler(&mockExpenseService{})
	handler.SiteAdmin = &fakeSiteOverview{blocks: overviewFixture()}

	app := fiber.New(fiber.Config{ErrorHandler: middleware.ErrorHandler})
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userID", uint(9))
		c.Locals("householdID", (*uint)(nil))
		c.Locals("isAdmin", true)
		return c.Next()
	})
	app.Post("/expenses", handler.Create)

	req := httptest.NewRequest(http.MethodPost, "/expenses", strings.NewReader("amount=10&description=x&category=other"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestDashboard_AdminRendersGlobalView(t *testing.T) {
	svc := &fakeSiteOverview{blocks: overviewFixture()}
	handler := NewExpenseHandler(&mockExpenseService{})
	handler.SiteAdmin = svc

	app := setupAdminBranchApp(handler.Dashboard, map[string]any{
		"userID":      uint(9),
		"householdID": (*uint)(nil),
		"email":       "siteadmin@example.com",
		"role":        "member",
		"isAdmin":     true,
		"csrfToken":   "tok",
		"lang":        "en",
	})

	status, body := getBody(t, app)
	require.Equal(t, fiber.StatusOK, status)
	assert.Contains(t, body, "All Households")
	assert.Contains(t, body, "Alpha Home")
	assert.Contains(t, body, "Beta Home")
}

func TestDashboard_MemberStaysHouseholdScoped(t *testing.T) {
	svc := &mockExpenseService{
		getDashboardSummaryFn: func(userID, householdID uint) (*services.DashboardSummary, error) {
			return &services.DashboardSummary{TotalIncome: 111}, nil
		},
	}
	handler := NewExpenseHandler(svc)
	handler.SiteAdmin = &fakeSiteOverview{blocks: overviewFixture()}

	hh := uint(7)
	app := setupAdminBranchApp(handler.Dashboard, map[string]any{
		"userID":      uint(2),
		"householdID": &hh,
		"email":       "member@example.com",
		"role":        "owner",
		"isAdmin":     false,
		"csrfToken":   "tok",
		"lang":        "en",
	})

	status, body := getBody(t, app)
	require.Equal(t, fiber.StatusOK, status)
	assert.Contains(t, body, "$111.00")
	assert.NotContains(t, body, "Alpha Home")
}

func TestSavingsList_AdminRendersGlobalView(t *testing.T) {
	svc := &fakeSiteOverview{blocks: overviewFixture()}
	handler := NewSavingsHandler(nil)
	handler.SiteAdmin = svc

	app := setupAdminBranchApp(handler.List, map[string]any{
		"userID":      uint(9),
		"householdID": (*uint)(nil),
		"email":       "siteadmin@example.com",
		"isAdmin":     true,
		"csrfToken":   "tok",
		"lang":        "en",
	})

	status, body := getBody(t, app)
	require.Equal(t, fiber.StatusOK, status)
	assert.Contains(t, body, "All Households")
	assert.Contains(t, body, "Alpha Fund")
	assert.Contains(t, body, "beta-owner@test.com")
}

func TestSavingsList_MemberStaysHouseholdScoped(t *testing.T) {
	repo := &mockSavingsRepoForHandlers{}
	savingsSvc := services.NewSavingsService(repo, nil)
	handler := NewSavingsHandler(savingsSvc)
	handler.SiteAdmin = &fakeSiteOverview{blocks: overviewFixture()}

	hh := uint(42)
	app := setupAdminBranchApp(handler.List, map[string]any{
		"userID":      uint(2),
		"householdID": &hh,
		"email":       "member@example.com",
		"isAdmin":     false,
		"csrfToken":   "tok",
		"lang":        "en",
	})

	status, body := getBody(t, app)
	require.Equal(t, fiber.StatusOK, status)
	assert.NotContains(t, body, "Alpha Fund")
}

func TestExpenseList_AdminWithBrokenOverviewDependencyReturns500(t *testing.T) {
	handler := NewExpenseHandler(&mockExpenseService{})
	handler.SiteAdmin = &fakeSiteOverview{err: assert.AnError}

	app := setupAdminBranchApp(handler.List, map[string]any{
		"userID":    uint(9),
		"isAdmin":   true,
		"csrfToken": "tok",
		"lang":      "en",
	})

	status, _ := getBody(t, app)
	assert.Equal(t, fiber.StatusInternalServerError, status)
}
