package services

import (
	"errors"
	"testing"
	"time"

	"github.com/homeadmin/internal/database"
	"github.com/homeadmin/internal/repositories"
)

// --- Mock ExpenseRepository ---

type mockExpenseRepo struct {
	createFn             func(expense *database.Expense) error
	findByIDFn           func(id uint) (*database.Expense, error)
	findByHouseholdFn    func(userID, householdID uint, filters database.ExpenseFilters) ([]database.Expense, error)
	updateFn             func(expense *database.Expense) error
	deleteFn             func(id uint) error
	monthlyTotalFn       func(userID, householdID uint, year int, month time.Month) (float64, error)
	categoryBreakdownFn  func(userID, householdID uint, year int, month time.Month) ([]repositories.CategoryTotal, error)
	recentExpensesFn     func(userID, householdID uint, limit int) ([]database.Expense, error)
}

func (m *mockExpenseRepo) Create(expense *database.Expense) error {
	if m.createFn != nil {
		return m.createFn(expense)
	}
	return nil
}

func (m *mockExpenseRepo) FindByID(id uint) (*database.Expense, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(id)
	}
	return nil, nil
}

func (m *mockExpenseRepo) FindByHousehold(userID, householdID uint, filters database.ExpenseFilters) ([]database.Expense, error) {
	if m.findByHouseholdFn != nil {
		return m.findByHouseholdFn(userID, householdID, filters)
	}
	return nil, nil
}

func (m *mockExpenseRepo) Update(expense *database.Expense) error {
	if m.updateFn != nil {
		return m.updateFn(expense)
	}
	return nil
}

func (m *mockExpenseRepo) Delete(id uint) error {
	if m.deleteFn != nil {
		return m.deleteFn(id)
	}
	return nil
}

func (m *mockExpenseRepo) MonthlyTotal(userID, householdID uint, year int, month time.Month) (float64, error) {
	if m.monthlyTotalFn != nil {
		return m.monthlyTotalFn(userID, householdID, year, month)
	}
	return 0, nil
}

func (m *mockExpenseRepo) CategoryBreakdown(userID, householdID uint, year int, month time.Month) ([]repositories.CategoryTotal, error) {
	if m.categoryBreakdownFn != nil {
		return m.categoryBreakdownFn(userID, householdID, year, month)
	}
	return nil, nil
}

func (m *mockExpenseRepo) RecentExpenses(userID, householdID uint, limit int) ([]database.Expense, error) {
	if m.recentExpensesFn != nil {
		return m.recentExpensesFn(userID, householdID, limit)
	}
	return nil, nil
}

// Verify interface compliance at compile time
var _ ExpenseRepository = (*mockExpenseRepo)(nil)

// --- Helper ---

func fixedDate() time.Time {
	return time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
}

// --- Create tests ---

func TestCreate_EmptyDescription(t *testing.T) {
	repo := &mockExpenseRepo{
		createFn: func(e *database.Expense) error {
			t.Error("Create should not be called when description is empty")
			return nil
		},
	}
	svc := NewExpenseService(repo)

	err := svc.Create(1, 1, 100.0, "", "Rent", fixedDate(), database.VisibleEditable, false)
	if err == nil {
		t.Fatal("expected error for empty description")
	}
	if !errors.Is(err, ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

func TestCreate_NegativeAmount(t *testing.T) {
	repo := &mockExpenseRepo{
		createFn: func(e *database.Expense) error {
			t.Error("Create should not be called when amount is negative")
			return nil
		},
	}
	svc := NewExpenseService(repo)

	err := svc.Create(1, 1, -50.0, "Groceries", "Groceries", fixedDate(), database.VisibleEditable, false)
	if err == nil {
		t.Fatal("expected error for negative amount")
	}
	if !errors.Is(err, ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

func TestCreate_InvalidCategory(t *testing.T) {
	repo := &mockExpenseRepo{
		createFn: func(e *database.Expense) error {
			t.Error("Create should not be called when category is invalid")
			return nil
		},
	}
	svc := NewExpenseService(repo)

	err := svc.Create(1, 1, 100.0, "Rent", "CatCosts", fixedDate(), database.VisibleEditable, false)
	if err == nil {
		t.Fatal("expected error for invalid category")
	}
	if !errors.Is(err, ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

func TestCreate_Success(t *testing.T) {
	var saved *database.Expense
	repo := &mockExpenseRepo{
		createFn: func(e *database.Expense) error {
			saved = e
			return nil
		},
	}
	svc := NewExpenseService(repo)

	err := svc.Create(1, 1, 1500.0, "Monthly Rent", "Rent", fixedDate(), database.VisibleEditable, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if saved == nil {
		t.Fatal("expected repo.Create to be called")
	}
	if saved.Description != "Monthly Rent" {
		t.Errorf("expected description 'Monthly Rent', got '%s'", saved.Description)
	}
	if saved.Amount != 1500.0 {
		t.Errorf("expected amount 1500, got %f", saved.Amount)
	}
	if saved.Category != "Rent" {
		t.Errorf("expected category 'Rent', got '%s'", saved.Category)
	}
	if saved.CreatedByID != 1 {
		t.Errorf("expected created_by_id 1, got %d", saved.CreatedByID)
	}
	if saved.HouseholdID != 1 {
		t.Errorf("expected household_id 1, got %d", saved.HouseholdID)
	}
	if saved.Visibility != database.VisibleEditable {
		t.Errorf("expected visibility 'visible_editable', got '%s'", saved.Visibility)
	}
	if !saved.IsFixed {
		t.Error("expected is_fixed true")
	}
}

// --- Update tests ---

func TestUpdate_AllowCreatorVisibleOnly(t *testing.T) {
	var updated *database.Expense
	repo := &mockExpenseRepo{
		findByIDFn: func(id uint) (*database.Expense, error) {
			return &database.Expense{
				ID:          1,
				CreatedByID: 1,
				Visibility:  database.VisibleOnly,
				Description: "Old desc",
				Amount:      100.0,
				Category:    "Rent",
			}, nil
		},
		updateFn: func(e *database.Expense) error {
			updated = e
			return nil
		},
	}
	svc := NewExpenseService(repo)

	err := svc.Update(1, 1, ExpenseUpdateFields{Description: strPtr("New desc")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated == nil {
		t.Fatal("expected repo.Update to be called")
	}
	if updated.Description != "New desc" {
		t.Errorf("expected description 'New desc', got '%s'", updated.Description)
	}
}

func TestUpdate_BlockNonCreatorVisibleOnly(t *testing.T) {
	repo := &mockExpenseRepo{
		findByIDFn: func(id uint) (*database.Expense, error) {
			return &database.Expense{
				ID:          1,
				CreatedByID: 1,
				Visibility:  database.VisibleOnly,
				Description: "Old desc",
				Amount:      100.0,
				Category:    "Rent",
			}, nil
		},
		updateFn: func(e *database.Expense) error {
			t.Error("Update should not be called for non-creator on visible_only")
			return nil
		},
	}
	svc := NewExpenseService(repo)

	err := svc.Update(2, 1, ExpenseUpdateFields{Description: strPtr("Hacked")})
	if err == nil {
		t.Fatal("expected permission error")
	}
	if !errors.Is(err, ErrPermission) {
		t.Errorf("expected ErrPermission, got %v", err)
	}
}

func TestUpdate_AllowAnyMemberVisibleEditable(t *testing.T) {
	var updated *database.Expense
	repo := &mockExpenseRepo{
		findByIDFn: func(id uint) (*database.Expense, error) {
			return &database.Expense{
				ID:          1,
				CreatedByID: 1,
				Visibility:  database.VisibleEditable,
				Description: "Old desc",
				Amount:      100.0,
				Category:    "Rent",
			}, nil
		},
		updateFn: func(e *database.Expense) error {
			updated = e
			return nil
		},
	}
	svc := NewExpenseService(repo)

	// User 2 is NOT the creator but visible_editable allows any member
	err := svc.Update(2, 1, ExpenseUpdateFields{Description: strPtr("Updated by member")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated == nil {
		t.Fatal("expected repo.Update to be called")
	}
	if updated.Description != "Updated by member" {
		t.Errorf("expected description 'Updated by member', got '%s'", updated.Description)
	}
}

// --- Delete tests ---

func TestDelete_AllowCreator(t *testing.T) {
	var deletedID uint
	repo := &mockExpenseRepo{
		findByIDFn: func(id uint) (*database.Expense, error) {
			return &database.Expense{
				ID:          1,
				CreatedByID: 1,
				Visibility:  database.VisibleEditable,
			}, nil
		},
		deleteFn: func(id uint) error {
			deletedID = id
			return nil
		},
	}
	svc := NewExpenseService(repo)

	err := svc.Delete(1, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deletedID != 1 {
		t.Errorf("expected delete called with ID 1, got %d", deletedID)
	}
}

func TestDelete_BlockNonCreator(t *testing.T) {
	repo := &mockExpenseRepo{
		findByIDFn: func(id uint) (*database.Expense, error) {
			return &database.Expense{
				ID:          1,
				CreatedByID: 1,
				Visibility:  database.VisibleEditable,
			}, nil
		},
		deleteFn: func(id uint) error {
			t.Error("Delete should not be called for non-creator")
			return nil
		},
	}
	svc := NewExpenseService(repo)

	err := svc.Delete(2, 1)
	if err == nil {
		t.Fatal("expected permission error")
	}
	if !errors.Is(err, ErrPermission) {
		t.Errorf("expected ErrPermission, got %v", err)
	}
}

// --- Helper ---

func strPtr(s string) *string {
	return &s
}

// --- Dashboard tests ---

func TestGetDashboardSummary_Success(t *testing.T) {
	repo := &mockExpenseRepo{
		monthlyTotalFn: func(userID, householdID uint, year int, month time.Month) (float64, error) {
			return 350.50, nil
		},
		categoryBreakdownFn: func(userID, householdID uint, year int, month time.Month) ([]repositories.CategoryTotal, error) {
			return []repositories.CategoryTotal{
				{Category: "Groceries", Total: 200.00},
				{Category: "Rent", Total: 150.50},
			}, nil
		},
		recentExpensesFn: func(userID, householdID uint, limit int) ([]database.Expense, error) {
			return []database.Expense{
				{ID: 1, Description: "Groceries", Amount: 50, Category: "Groceries"},
				{ID: 2, Description: "Rent", Amount: 150, Category: "Rent"},
			}, nil
		},
	}
	svc := NewExpenseService(repo)

	summary, err := svc.GetDashboardSummary(1, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary.MonthlyTotal != 350.50 {
		t.Errorf("expected MonthlyTotal = 350.50, got %.2f", summary.MonthlyTotal)
	}
	if len(summary.CategoryTotals) != 2 {
		t.Errorf("expected 2 category totals, got %d", len(summary.CategoryTotals))
	}
	if len(summary.RecentExpenses) != 2 {
		t.Errorf("expected 2 recent expenses, got %d", len(summary.RecentExpenses))
	}
}

func TestGetDashboardSummary_Empty(t *testing.T) {
	repo := &mockExpenseRepo{
		monthlyTotalFn: func(userID, householdID uint, year int, month time.Month) (float64, error) {
			return 0, nil
		},
		categoryBreakdownFn: func(userID, householdID uint, year int, month time.Month) ([]repositories.CategoryTotal, error) {
			return []repositories.CategoryTotal{}, nil
		},
		recentExpensesFn: func(userID, householdID uint, limit int) ([]database.Expense, error) {
			return []database.Expense{}, nil
		},
	}
	svc := NewExpenseService(repo)

	summary, err := svc.GetDashboardSummary(1, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary.MonthlyTotal != 0 {
		t.Errorf("expected MonthlyTotal = 0, got %.2f", summary.MonthlyTotal)
	}
	if len(summary.CategoryTotals) != 0 {
		t.Errorf("expected 0 category totals, got %d", len(summary.CategoryTotals))
	}
	if len(summary.RecentExpenses) != 0 {
		t.Errorf("expected 0 recent expenses, got %d", len(summary.RecentExpenses))
	}
}
