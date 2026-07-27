package handlers

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/homeadmin/internal/database"
	"github.com/homeadmin/internal/services"
)

// --- Mock ExpenseService ---

type mockExpenseService struct {
	createFn          func(userID, householdID uint, amount float64, description, category string, date time.Time, visibility database.VisibilityType, isFixed bool) error
	findByHouseholdFn func(userID, householdID uint, filters database.ExpenseFilters) ([]database.Expense, error)
	updateFn          func(userID, expenseID uint, fields services.ExpenseUpdateFields) error
	deleteFn          func(userID, expenseID uint) error
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

// Verify interface compliance at compile time
var _ expenseServiceInterface = (*mockExpenseService)(nil)

// --- Test helpers ---

func setupExpenseApp(svc expenseServiceInterface) *fiber.App {
	app := fiber.New()
	handler := NewExpenseHandler(svc)

	// Middleware to simulate JWT-validated locals (normally set by RequireAuth)
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userID", uint(1))
		c.Locals("householdID", uint(1))
		return c.Next()
	})

	app.Post("/expenses", handler.Create)
	app.Get("/expenses", handler.List)
	app.Put("/expenses/:id", handler.Update)
	app.Delete("/expenses/:id", handler.Delete)

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
