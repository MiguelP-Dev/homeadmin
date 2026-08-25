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
	createFn              func(userID, householdID uint, amount float64, description, category string, date time.Time, visibility database.VisibilityType, isFixed bool, txType string) error
	findByIDFn            func(userID, householdID, expenseID uint) (*database.Expense, error)
	findByHouseholdFn     func(userID, householdID uint, filters database.ExpenseFilters) ([]database.Expense, error)
	updateFn              func(userID, expenseID uint, fields services.ExpenseUpdateFields) error
	deleteFn              func(userID, expenseID uint) error
	getDashboardSummaryFn func(userID, householdID uint) (*services.DashboardSummary, error)
}

func (m *mockExpenseService) Create(userID, householdID uint, amount float64, description, category string, date time.Time, visibility database.VisibilityType, isFixed bool, txType string) error {
	if m.createFn != nil {
		return m.createFn(userID, householdID, amount, description, category, date, visibility, isFixed, txType)
	}
	return nil
}

func (m *mockExpenseService) FindByID(userID, householdID, expenseID uint) (*database.Expense, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(userID, householdID, expenseID)
	}
	return nil, services.ErrNotFound
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

// ptr returns a pointer to v (Go 1.18+ generics).
func ptr[T any](v T) *T { return &v }

func setupExpenseApp(svc expenseServiceInterface) *fiber.App {
	return setupExpenseAppWithHousehold(svc, ptr[uint](1))
}

func setupExpenseAppWithHousehold(svc expenseServiceInterface, householdID *uint) *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: middleware.ErrorHandler,
	})
	handler := NewExpenseHandler(svc)

	// Middleware to simulate JWT-validated locals (normally set by RequireAuth).
	// RequireAuth stores claims.HouseholdID, which is *uint (services/auth.go),
	// so the middleware MUST mirror that type or handler assertions misrepresent
	// production behavior.
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userID", uint(1))
		c.Locals("householdID", householdID)
		c.Locals("email", "test@example.com")
		c.Locals("csrfToken", "test-csrf-token")
		return c.Next()
	})

	app.Post("/expenses", handler.Create)
	app.Get("/expenses", handler.List)
	app.Get("/expenses/new", handler.ShowNew)
	app.Get("/expenses/:id/edit", handler.ShowEdit)
	app.Post("/expenses/:id/update", handler.Update)
	app.Post("/expenses/:id/delete", handler.Delete)
	app.Get("/dashboard", handler.Dashboard)

	return app
}

// --- Create tests ---

func TestCreateHandler_ValidationError(t *testing.T) {
	svc := &mockExpenseService{
		createFn: func(userID, householdID uint, amount float64, description, category string, date time.Time, visibility database.VisibilityType, isFixed bool, txType string) error {
			return services.ErrValidation
		},
	}
	app := setupExpenseApp(svc)

	// Valid form: the service-level validation error is what must surface as 422.
	form := url.Values{}
	form.Set("amount", "100")
	form.Set("description", "Rent")
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

	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		t.Errorf("expected status 422, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	// Service-level ErrValidation maps to the keyed expense.validation_failed
	// error, rendered through ErrorHandler as the localized English message.
	if !strings.Contains(string(body), "Please correct the errors below and try again.") {
		t.Errorf("expected localized validation_failed message in response body, got: %s", string(body))
	}
}

func TestCreateHandler_PermissionDenied(t *testing.T) {
	svc := &mockExpenseService{
		createFn: func(userID, householdID uint, amount float64, description, category string, date time.Time, visibility database.VisibilityType, isFixed bool, txType string) error {
			return services.ErrPermission
		},
	}
	app := setupExpenseApp(svc)

	form := url.Values{}
	form.Set("amount", "100")
	form.Set("description", "Some expense")
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

	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		t.Errorf("expected status 422 for invalid amount, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	// Keyed expense.invalid_amount renders as the localized English message.
	if !strings.Contains(string(body), "Invalid amount.") {
		t.Errorf("expected localized 'Invalid amount.' in response, got: %s", string(body))
	}
}

func TestCreateHandler_Success(t *testing.T) {
	var savedDesc string
	svc := &mockExpenseService{
		createFn: func(userID, householdID uint, amount float64, description, category string, date time.Time, visibility database.VisibilityType, isFixed bool, txType string) error {
			savedDesc = description
			return nil
		},
	}
	app := setupExpenseApp(svc)

	form := url.Values{}
	form.Set("amount", "250.50")
	form.Set("description", "Groceries")
	form.Set("category", "Groceries")
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

	if resp.StatusCode != fiber.StatusSeeOther {
		t.Errorf("expected status 303, got %d", resp.StatusCode)
	}
	if loc := resp.Header.Get("Location"); loc != "/expenses" {
		t.Errorf("Location = %q, want /expenses", loc)
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
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("Content-Type = %q, want text/html (list must be HTML, not JSON)", ct)
	}

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	if !strings.Contains(bodyStr, "Rent") {
		t.Error("expected 'Rent' in response")
	}
	if !strings.Contains(bodyStr, "Groceries") {
		t.Error("expected 'Groceries' in response")
	}
	if strings.Contains(bodyStr, `"expenses"`) {
		t.Error("response must not contain a JSON body")
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
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("Content-Type = %q, want text/html (list must be HTML, not JSON)", ct)
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

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "No expenses yet") {
		t.Error("expected 'No expenses yet' empty state in HTML list")
	}
}

// --- ShowNew / ShowEdit tests ---

func TestShowNewHandler_RendersForm(t *testing.T) {
	app := setupExpenseApp(&mockExpenseService{})

	req := httptest.NewRequest(http.MethodGet, "/expenses/new", nil)
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
	if !strings.Contains(bodyStr, `action="/expenses"`) {
		t.Error("expected create form to POST to /expenses")
	}
	if !strings.Contains(bodyStr, "Create Expense") {
		t.Error("expected 'Create Expense' heading/button")
	}
	if !strings.Contains(bodyStr, `name="csrf"`) {
		t.Error("expected a csrf input in the form")
	}
}

func TestShowEditHandler_RendersForm(t *testing.T) {
	svc := &mockExpenseService{
		findByIDFn: func(userID, householdID, expenseID uint) (*database.Expense, error) {
			if expenseID != 7 {
				t.Errorf("expected expenseID 7, got %d", expenseID)
			}
			return &database.Expense{
				ID: 7, Description: "Groceries", Amount: 85.50, Category: "Groceries",
				Visibility: database.VisibleEditable,
			}, nil
		},
	}
	app := setupExpenseApp(svc)

	req := httptest.NewRequest(http.MethodGet, "/expenses/7/edit", nil)
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
	if !strings.Contains(bodyStr, `action="/expenses/7/update"`) {
		t.Error("expected edit form to POST to /expenses/7/update")
	}
	if !strings.Contains(bodyStr, "Update Expense") {
		t.Error("expected 'Update Expense' heading/button")
	}
	if !strings.Contains(bodyStr, `value="Groceries"`) {
		t.Error("expected pre-filled description")
	}
	if !strings.Contains(bodyStr, `name="csrf"`) {
		t.Error("expected a csrf input in the form")
	}
}

func TestShowEditHandler_NotFound(t *testing.T) {
	svc := &mockExpenseService{
		findByIDFn: func(userID, householdID, expenseID uint) (*database.Expense, error) {
			return nil, services.ErrNotFound
		},
	}
	app := setupExpenseApp(svc)

	req := httptest.NewRequest(http.MethodGet, "/expenses/999/edit", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("expected status 404 for unknown expense, got %d", resp.StatusCode)
	}
}

func TestShowEditHandler_Forbidden(t *testing.T) {
	svc := &mockExpenseService{
		findByIDFn: func(userID, householdID, expenseID uint) (*database.Expense, error) {
			return nil, services.ErrPermission
		},
	}
	app := setupExpenseApp(svc)

	req := httptest.NewRequest(http.MethodGet, "/expenses/1/edit", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("expected status 403 for hidden expense, got %d", resp.StatusCode)
	}
}

func TestShowEditHandler_InvalidID(t *testing.T) {
	app := setupExpenseApp(&mockExpenseService{})

	req := httptest.NewRequest(http.MethodGet, "/expenses/abc/edit", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("expected status 400 for invalid id, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "invalid expense id") {
		t.Errorf("expected 'invalid expense id' in response, got: %s", string(body))
	}
}

// --- Update tests ---

func TestUpdateHandler_InvalidID(t *testing.T) {
	svc := &mockExpenseService{}
	app := setupExpenseApp(svc)

	form := url.Values{}
	form.Set("description", "Should not update")
	req := httptest.NewRequest(http.MethodPost, "/expenses/abc/update", strings.NewReader(form.Encode()))
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
	req := httptest.NewRequest(http.MethodPost, "/expenses/1/update", strings.NewReader(form.Encode()))
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
	req := httptest.NewRequest(http.MethodPost, "/expenses/1/update", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		t.Errorf("expected status 422, got %d", resp.StatusCode)
	}
}

func TestUpdateHandler_Success(t *testing.T) {
	var gotExpenseID uint
	var gotFields services.ExpenseUpdateFields
	svc := &mockExpenseService{
		updateFn: func(userID, expenseID uint, fields services.ExpenseUpdateFields) error {
			gotExpenseID = expenseID
			gotFields = fields
			return nil
		},
	}
	app := setupExpenseApp(svc)

	form := url.Values{}
	form.Set("description", "Updated desc")
	form.Set("category", "Rent")
	req := httptest.NewRequest(http.MethodPost, "/expenses/1/update", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusSeeOther {
		t.Errorf("expected status 303, got %d", resp.StatusCode)
	}
	if loc := resp.Header.Get("Location"); loc != "/expenses" {
		t.Errorf("Location = %q, want /expenses", loc)
	}
	if gotExpenseID != 1 {
		t.Errorf("expected service called with expenseID 1, got %d", gotExpenseID)
	}
	if gotFields.Description == nil || *gotFields.Description != "Updated desc" {
		t.Errorf("expected description 'Updated desc' in update fields, got %+v", gotFields.Description)
	}
	if gotFields.Category == nil || *gotFields.Category != "Rent" {
		t.Errorf("expected category 'Rent' in update fields, got %+v", gotFields.Category)
	}
}

// --- Delete tests ---

func TestDeleteHandler_Success(t *testing.T) {
	var gotExpenseID uint
	svc := &mockExpenseService{
		deleteFn: func(userID, expenseID uint) error {
			gotExpenseID = expenseID
			return nil
		},
	}
	app := setupExpenseApp(svc)

	req := httptest.NewRequest(http.MethodPost, "/expenses/1/delete", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusSeeOther {
		t.Errorf("expected status 303, got %d", resp.StatusCode)
	}
	if loc := resp.Header.Get("Location"); loc != "/expenses" {
		t.Errorf("Location = %q, want /expenses", loc)
	}
	if gotExpenseID != 1 {
		t.Errorf("expected service called with expenseID 1, got %d", gotExpenseID)
	}
}

func TestDeleteHandler_InvalidID(t *testing.T) {
	svc := &mockExpenseService{}
	app := setupExpenseApp(svc)

	req := httptest.NewRequest(http.MethodPost, "/expenses/abc/delete", nil)
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

	req := httptest.NewRequest(http.MethodPost, "/expenses/999/delete", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("expected status 404 for not found, got %d", resp.StatusCode)
	}
}

func TestDeleteHandler_PermissionDenied(t *testing.T) {
	svc := &mockExpenseService{
		deleteFn: func(userID, expenseID uint) error {
			return services.ErrPermission
		},
	}
	app := setupExpenseApp(svc)

	req := httptest.NewRequest(http.MethodPost, "/expenses/1/delete", nil)
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
				MonthlyTotal:  350.50,
				TotalIncome:   500.00,
				TotalExpenses: 149.50,
				Balance:       350.50,
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
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", ct)
	}

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	if !strings.Contains(bodyStr, "350.50") {
		t.Error("expected balance 350.50 in HTML")
	}
	if !strings.Contains(bodyStr, "500.00") {
		t.Error("expected income total 500.00 in HTML")
	}
	if !strings.Contains(bodyStr, "149.50") {
		t.Error("expected expense total 149.50 in HTML")
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
				TotalIncome:    0,
				TotalExpenses:  0,
				Balance:        0,
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
		t.Error("expected 0.00 for empty totals")
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

// --- Household Locals regression tests (T1.1) ---
//
// RequireAuth stores claims.HouseholdID, which is *uint (services/auth.go),
// in c.Locals("householdID"). Handlers previously asserted .(uint), which
// PANICS on the real *uint value — every authenticated request to
// Create/List/Dashboard crashed. The fixed behavior: typed-nil *uint →
// 400 "household required" (never 5xx); non-nil *uint → normal success with
// the extracted value passed to the service.

func validExpenseForm() url.Values {
	form := url.Values{}
	form.Set("amount", "250.50")
	form.Set("description", "Groceries")
	form.Set("category", "Groceries")
	form.Set("date", "2026-07-27")
	form.Set("visibility", "visible_editable")
	return form
}

func TestExpenseHandlers_NilHouseholdLocals_RequiresHousehold(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		form   url.Values
	}{
		{"Create", http.MethodPost, "/expenses", validExpenseForm()},
		{"List", http.MethodGet, "/expenses", nil},
		{"Dashboard", http.MethodGet, "/dashboard", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mirror RequireAuth storing claims.HouseholdID = *uint(nil)
			// for a JWT whose household_id claim is null/absent.
			app := setupExpenseAppWithHousehold(&mockExpenseService{}, nil)

			var req *http.Request
			if tt.form != nil {
				req = httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.form.Encode()))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			} else {
				req = httptest.NewRequest(tt.method, tt.path, nil)
			}

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode >= fiber.StatusInternalServerError {
				t.Fatalf("expected no 5xx for nil household, got %d", resp.StatusCode)
			}
			if resp.StatusCode != fiber.StatusBadRequest {
				t.Errorf("expected 400 for nil household, got %d", resp.StatusCode)
			}

			body, _ := io.ReadAll(resp.Body)
			// Keyed expense.household_required renders as the localized
			// English message instead of the former raw "household required".
			if !strings.Contains(string(body), "You must join a household before creating expenses.") {
				t.Errorf("expected localized household_required message in body, got: %s", string(body))
			}
		})
	}
}

func TestExpenseHandlers_NonNilHouseholdLocals_Succeed(t *testing.T) {
	t.Run("Create", func(t *testing.T) {
		var gotHouseholdID uint
		svc := &mockExpenseService{
			createFn: func(userID, householdID uint, amount float64, description, category string, date time.Time, visibility database.VisibilityType, isFixed bool, txType string) error {
				gotHouseholdID = householdID
				return nil
			},
		}
		app := setupExpenseAppWithHousehold(svc, ptr[uint](1))

		req := httptest.NewRequest(http.MethodPost, "/expenses", strings.NewReader(validExpenseForm().Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("app.Test failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != fiber.StatusSeeOther {
			t.Errorf("expected status 303, got %d", resp.StatusCode)
		}
		if gotHouseholdID != 1 {
			t.Errorf("expected service householdID 1, got %d", gotHouseholdID)
		}
	})

	t.Run("List", func(t *testing.T) {
		var gotHouseholdID uint
		svc := &mockExpenseService{
			findByHouseholdFn: func(userID, householdID uint, filters database.ExpenseFilters) ([]database.Expense, error) {
				gotHouseholdID = householdID
				return []database.Expense{{ID: 1, Description: "Rent", Amount: 1500, Category: "Rent"}}, nil
			},
		}
		app := setupExpenseAppWithHousehold(svc, ptr[uint](1))

		req := httptest.NewRequest(http.MethodGet, "/expenses", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("app.Test failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != fiber.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
		if gotHouseholdID != 1 {
			t.Errorf("expected service householdID 1, got %d", gotHouseholdID)
		}

		body, _ := io.ReadAll(resp.Body)
		if !strings.Contains(string(body), "Rent") {
			t.Errorf("expected 'Rent' in response, got: %s", string(body))
		}
	})

	t.Run("Dashboard", func(t *testing.T) {
		var gotHouseholdID uint
		svc := &mockExpenseService{
			getDashboardSummaryFn: func(userID, householdID uint) (*services.DashboardSummary, error) {
				gotHouseholdID = householdID
				return &services.DashboardSummary{
					MonthlyTotal:   350.50,
					TotalIncome:    500.00,
					TotalExpenses:  149.50,
					Balance:        350.50,
					CategoryTotals: []repositories.CategoryTotal{},
					RecentExpenses: []database.Expense{},
				}, nil
			},
		}
		app := setupExpenseAppWithHousehold(svc, ptr[uint](1))

		req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("app.Test failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != fiber.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
		if gotHouseholdID != 1 {
			t.Errorf("expected service householdID 1, got %d", gotHouseholdID)
		}

		body, _ := io.ReadAll(resp.Body)
		if !strings.Contains(string(body), "350.50") {
			t.Errorf("expected balance 350.50 in HTML, got: %s", string(body))
		}
	})
}
