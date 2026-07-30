package handlers

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/homeadmin/internal/database"
	"github.com/homeadmin/internal/middleware"
	"github.com/homeadmin/internal/repositories"
	"github.com/homeadmin/internal/services"
)

// --- Mock ExpenseService ---

type mockExpenseService struct {
	createFn              func(userID, householdID uint, amount float64, description, category string, date time.Time, visibility database.VisibilityType, isFixed bool) error
	findByHouseholdFn     func(userID, householdID uint, filters database.ExpenseFilters) ([]database.Expense, error)
	updateFn              func(userID, expenseID uint, fields services.ExpenseUpdateFields) error
	deleteFn              func(userID, expenseID uint) error
	getDashboardSummaryFn func(userID, householdID uint) (*services.DashboardSummary, error)
}

func (m *mockExpenseService) Create(userID, householdID uint, amount float64, description, category string, date time.Time, visibility database.VisibilityType, isFixed bool) error {
	if m.createFn != nil {
		return m.createFn(userID, householdID, amount, description, category, date, visibility, isFixed)
	}
	return nil
}

func (m *mockExpenseService) FindByHousehold(userID, householdID uint, filters database.ExpenseFilters) ([]database.Expense, error) {
	if m.findByHouseholdFn != nil {
		return m.findByHouseholdFn(userID, householdID, filters)
	}
	return nil, nil
}

func (m *mockExpenseService) Update(userID, expenseID uint, fields services.ExpenseUpdateFields) error {
	if m.updateFn != nil {
		return m.updateFn(userID, expenseID, fields)
	}
	return nil
}

func (m *mockExpenseService) Delete(userID, expenseID uint) error {
	if m.deleteFn != nil {
		return m.deleteFn(userID, expenseID)
	}
	return nil
}

func (m *mockExpenseService) GetDashboardSummary(userID, householdID uint) (*services.DashboardSummary, error) {
	if m.getDashboardSummaryFn != nil {
		return m.getDashboardSummaryFn(userID, householdID)
	}
	return &services.DashboardSummary{}, nil
}

// Verify interface compliance at compile time
var _ expenseServiceInterface = (*mockExpenseService)(nil)

// --- Test helpers ---

func setupExpenseApp(svc expenseServiceInterface) *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: middleware.ErrorHandler,
	})
	handler := NewExpenseHandler(svc)

	// Middleware to simulate JWT-validated locals (normally set by RequireAuth)
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userID", uint(1))
		c.Locals("householdID", uint(1))
		c.Locals("email", "test@example.com")
		c.Locals("csrfToken", "test-csrf-token")
		return c.Next()
	})

	app.Post("/expenses", handler.Create)
	app.Get("/expenses", handler.List)
	app.Put("/expenses/:id", handler.Update)
	app.Delete("/expenses/:id", handler.Delete)
	app.Get("/dashboard", handler.Dashboard)

	return app
}

// --- Create tests ---

func TestCreateHandler_ValidationError(t *testing.T) {
	svc := &mockExpenseService{
		createFn: func(userID, householdID uint, amount float64, description, category string, date time.Time, visibility database.VisibilityType, isFixed bool) error {
			return services.ErrValidation
		},
	}
	app := setupExpenseApp(svc)

	form := url.Values{}
	form.Set("amount", "100")
	form.Set("description", "")
	form.Set("category", "Rent")
	form.Set("date", "2026-07-27")
	form.Set("visibility", "visible_editable")

	req := httptest.NewRequest(http.MethodPost, "/expenses", strings.NewReader(form.Encode()))
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
	if !strings.Contains(string(body), "error") {
		t.Error("expected error message in response body")
	}
}

func TestCreateHandler_PermissionDenied(t *testing.T) {
	svc := &mockExpenseService{
		createFn: func(userID, householdID uint, amount float64, description, category string, date time.Time, visibility database.VisibilityType, isFixed bool) error {
			return services.ErrPermission
		},
	}
	app := setupExpenseApp(svc)

	form := url.Values{}
	form.Set("amount", "100")
	form.Set("description", "Some expense")
	form.Set("category", "rent")
	form.Set("date", "2026-07-27")
	form.Set("visibility", "visible_editable")

	req := httptest.NewRequest(http.MethodPost, "/expenses", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("expected status 403 for permission denied, got %d", resp.StatusCode)
	}
}

func TestCreateHandler_InvalidAmount(t *testing.T) {
	svc := &mockExpenseService{}
	app := setupExpenseApp(svc)

	form := url.Values{}
	form.Set("amount", "not-a-number")
	form.Set("description", "Some expense")
	form.Set("category", "Rent")
	form.Set("date", "2026-07-27")

	req := httptest.NewRequest(http.MethodPost, "/expenses", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("expected status 400 for invalid amount, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "invalid amount") {
		t.Errorf("expected 'invalid amount' in response, got: %s", string(body))
	}
}

func TestCreateHandler_Success(t *testing.T) {
	var savedDesc string
	svc := &mockExpenseService{
		createFn: func(userID, householdID uint, amount float64, description, category string, date time.Time, visibility database.VisibilityType, isFixed bool) error {
			savedDesc = description
			return nil
		},
	}
	app := setupExpenseApp(svc)

	form := url.Values{}
	form.Set("amount", "250.50")
	form.Set("description", "Groceries")
	form.Set("category", "food")
	form.Set("date", "2026-07-27")
	form.Set("visibility", "visible_editable")
	form.Set("isFixed", "false")

	req := httptest.NewRequest(http.MethodPost, "/expenses", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusCreated {
		t.Errorf("expected status 201, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	if !strings.Contains(bodyStr, "expense created") {
		t.Errorf("expected 'expense created' in response, got: %s", bodyStr)
	}
	if savedDesc != "Groceries" {
		t.Errorf("expected service called with 'Groceries', got '%s'", savedDesc)
	}
}

// --- List tests ---

func TestListHandler_Success(t *testing.T) {
	svc := &mockExpenseService{
		findByHouseholdFn: func(userID, householdID uint, filters database.ExpenseFilters) ([]database.Expense, error) {
			return []database.Expense{
				{ID: 1, Description: "Rent", Amount: 1500, Category: "Rent"},
				{ID: 2, Description: "Groceries", Amount: 250, Category: "Groceries"},
			}, nil
		},
	}
	app := setupExpenseApp(svc)

	req := httptest.NewRequest(http.MethodGet, "/expenses", nil)
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
	if !strings.Contains(bodyStr, "Rent") {
		t.Error("expected 'Rent' in response")
	}
	if !strings.Contains(bodyStr, "Groceries") {
		t.Error("expected 'Groceries' in response")
	}
}

func TestListHandler_WithFilters(t *testing.T) {
	svc := &mockExpenseService{
		findByHouseholdFn: func(userID, householdID uint, filters database.ExpenseFilters) ([]database.Expense, error) {
			if filters.Category != "food" {
				t.Errorf("expected category filter 'food', got '%s'", filters.Category)
			}
			if filters.Limit != 10 {
				t.Errorf("expected limit 10, got %d", filters.Limit)
			}
			if filters.Offset != 5 {
				t.Errorf("expected offset 5, got %d", filters.Offset)
			}
			return []database.Expense{
				{ID: 1, Description: "Lunch", Amount: 25, Category: "food"},
			}, nil
		},
	}
	app := setupExpenseApp(svc)

	req := httptest.NewRequest(http.MethodGet, "/expenses?category=food&limit=10&offset=5", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestListHandler_ServiceError(t *testing.T) {
	svc := &mockExpenseService{
		findByHouseholdFn: func(userID, householdID uint, filters database.ExpenseFilters) ([]database.Expense, error) {
			return nil, fmt.Errorf("database connection failed")
		},
	}
	app := setupExpenseApp(svc)

	req := httptest.NewRequest(http.MethodGet, "/expenses", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Errorf("expected status 500 for service error, got %d", resp.StatusCode)
	}
}

func TestListHandler_Empty(t *testing.T) {
	svc := &mockExpenseService{
		findByHouseholdFn: func(userID, householdID uint, filters database.ExpenseFilters) ([]database.Expense, error) {
			return []database.Expense{}, nil
		},
	}
	app := setupExpenseApp(svc)

	req := httptest.NewRequest(http.MethodGet, "/expenses", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

// --- Update tests ---

func TestUpdateHandler_InvalidID(t *testing.T) {
	svc := &mockExpenseService{}
	app := setupExpenseApp(svc)

	form := url.Values{}
	form.Set("description", "Should not update")
	req := httptest.NewRequest(http.MethodPut, "/expenses/abc", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("expected status 400 for invalid ID, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "invalid expense id") {
		t.Errorf("expected 'invalid expense id' in response, got: %s", string(body))
	}
}

func TestUpdateHandler_PermissionDenied(t *testing.T) {
	svc := &mockExpenseService{
		updateFn: func(userID, expenseID uint, fields services.ExpenseUpdateFields) error {
			return services.ErrPermission
		},
	}
	app := setupExpenseApp(svc)

	form := url.Values{}
	form.Set("description", "Hacked")
	req := httptest.NewRequest(http.MethodPut, "/expenses/1", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("expected status 403, got %d", resp.StatusCode)
	}
}

func TestUpdateHandler_ValidationError(t *testing.T) {
	svc := &mockExpenseService{
		updateFn: func(userID, expenseID uint, fields services.ExpenseUpdateFields) error {
			return services.ErrValidation
		},
	}
	app := setupExpenseApp(svc)

	form := url.Values{}
	form.Set("category", "InvalidCat")
	req := httptest.NewRequest(http.MethodPut, "/expenses/1", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("expected status 400, got %d", resp.StatusCode)
	}
}

func TestUpdateHandler_Success(t *testing.T) {
	svc := &mockExpenseService{
		updateFn: func(userID, expenseID uint, fields services.ExpenseUpdateFields) error {
			return nil
		},
	}
	app := setupExpenseApp(svc)

	form := url.Values{}
	form.Set("description", "Updated desc")
	req := httptest.NewRequest(http.MethodPut, "/expenses/1", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

// --- Delete tests ---

func TestDeleteHandler_Success(t *testing.T) {
	svc := &mockExpenseService{
		deleteFn: func(userID, expenseID uint) error {
			return nil
		},
	}
	app := setupExpenseApp(svc)

	req := httptest.NewRequest(http.MethodDelete, "/expenses/1", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestDeleteHandler_InvalidID(t *testing.T) {
	svc := &mockExpenseService{}
	app := setupExpenseApp(svc)

	req := httptest.NewRequest(http.MethodDelete, "/expenses/abc", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("expected status 400 for invalid ID, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "invalid expense id") {
		t.Errorf("expected 'invalid expense id' in response, got: %s", string(body))
	}
}

func TestDeleteHandler_NotFound(t *testing.T) {
	svc := &mockExpenseService{
		deleteFn: func(userID, expenseID uint) error {
			return services.ErrNotFound
		},
	}
	app := setupExpenseApp(svc)

	req := httptest.NewRequest(http.MethodDelete, "/expenses/999", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("expected status 400 for not found, got %d", resp.StatusCode)
	}
}

func TestDeleteHandler_PermissionDenied(t *testing.T) {
	svc := &mockExpenseService{
		deleteFn: func(userID, expenseID uint) error {
			return services.ErrPermission
		},
	}
	app := setupExpenseApp(svc)

	req := httptest.NewRequest(http.MethodDelete, "/expenses/1", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("expected status 403, got %d", resp.StatusCode)
	}
}

// --- Dashboard tests ---

func TestDashboardHandler_Success(t *testing.T) {
	svc := &mockExpenseService{
		getDashboardSummaryFn: func(userID, householdID uint) (*services.DashboardSummary, error) {
			return &services.DashboardSummary{
				MonthlyTotal: 350.50,
				CategoryTotals: []repositories.CategoryTotal{
					{Category: "Groceries", Total: 200.00},
					{Category: "Rent", Total: 150.50},
				},
				RecentExpenses: []database.Expense{
					{ID: 1, Description: "Groceries", Amount: 50, Category: "Groceries", Date: time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC)},
				},
			}, nil
		},
	}
	app := setupExpenseApp(svc)

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
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
	if !strings.Contains(bodyStr, "350.50") {
		t.Error("expected monthly total 350.50 in HTML")
	}
	if !strings.Contains(bodyStr, "Groceries") {
		t.Error("expected 'Groceries' in category breakdown")
	}
	if !strings.Contains(bodyStr, "Dashboard") {
		t.Error("expected 'Dashboard' in page title")
	}
	if !strings.Contains(bodyStr, "/expenses") {
		t.Error("expected link back to /expenses")
	}
}

func TestDashboardHandler_Empty(t *testing.T) {
	svc := &mockExpenseService{
		getDashboardSummaryFn: func(userID, householdID uint) (*services.DashboardSummary, error) {
			return &services.DashboardSummary{
				MonthlyTotal:   0,
				CategoryTotals: []repositories.CategoryTotal{},
				RecentExpenses: []database.Expense{},
			}, nil
		},
	}
	app := setupExpenseApp(svc)

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
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
	if !strings.Contains(bodyStr, "0.00") {
		t.Error("expected 0.00 for empty monthly total")
	}
	if !strings.Contains(bodyStr, "No expenses this month") {
		t.Error("expected 'No expenses this month' empty state")
	}
	if !strings.Contains(bodyStr, "No recent expenses") {
		t.Error("expected 'No recent expenses' empty state")
	}
}

func TestDashboardHandler_ServiceError(t *testing.T) {
	svc := &mockExpenseService{
		getDashboardSummaryFn: func(userID, householdID uint) (*services.DashboardSummary, error) {
			return nil, fmt.Errorf("database connection lost")
		},
	}
	app := setupExpenseApp(svc)

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", resp.StatusCode)
	}
}
