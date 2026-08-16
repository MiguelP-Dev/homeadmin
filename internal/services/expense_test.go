package services

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/homeadmin/internal/database"
	"github.com/homeadmin/internal/repositories"
)

// --- Mock ExpenseRepository ---

type mockExpenseRepo struct {
	createFn            func(expense *database.Expense) error
	findByIDFn          func(id uint) (*database.Expense, error)
	findByHouseholdFn   func(userID, householdID uint, viewerRole string, filters database.ExpenseFilters) ([]database.Expense, error)
	updateFn            func(expense *database.Expense) error
	deleteFn            func(id uint) error
	monthlyTotalFn      func(userID, householdID uint, viewerRole string, year int, month time.Month) (float64, error)
	categoryBreakdownFn func(userID, householdID uint, viewerRole string, year int, month time.Month) ([]repositories.CategoryTotal, error)
	recentExpensesFn    func(userID, householdID uint, viewerRole string, limit int) ([]database.Expense, error)
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

func (m *mockExpenseRepo) FindByHousehold(userID, householdID uint, viewerRole string, filters database.ExpenseFilters) ([]database.Expense, error) {
	if m.findByHouseholdFn != nil {
		return m.findByHouseholdFn(userID, householdID, viewerRole, filters)
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

func (m *mockExpenseRepo) MonthlyTotal(userID, householdID uint, viewerRole string, year int, month time.Month) (float64, error) {
	if m.monthlyTotalFn != nil {
		return m.monthlyTotalFn(userID, householdID, viewerRole, year, month)
	}
	return 0, nil
}

func (m *mockExpenseRepo) CategoryBreakdown(userID, householdID uint, viewerRole string, year int, month time.Month) ([]repositories.CategoryTotal, error) {
	if m.categoryBreakdownFn != nil {
		return m.categoryBreakdownFn(userID, householdID, viewerRole, year, month)
	}
	return nil, nil
}

func (m *mockExpenseRepo) RecentExpenses(userID, householdID uint, viewerRole string, limit int) ([]database.Expense, error) {
	if m.recentExpensesFn != nil {
		return m.recentExpensesFn(userID, householdID, viewerRole, limit)
	}
	return nil, nil
}

// Verify interface compliance at compile time
var _ ExpenseRepository = (*mockExpenseRepo)(nil)

// --- Helper ---

func fixedDate() time.Time {
	return time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
}

// testUserRepo returns a mock user repo whose FindByID resolves the given role.
func testUserRepo(role string) *mockUserRepo {
	return &mockUserRepo{
		findByIDFn: func(id uint) (*database.User, error) {
			return &database.User{ID: id, Role: role}, nil
		},
	}
}

// --- Create tests ---

func TestCreate_EmptyDescription(t *testing.T) {
	repo := &mockExpenseRepo{
		createFn: func(e *database.Expense) error {
			t.Error("Create should not be called when description is empty")
			return nil
		},
	}
	svc := NewExpenseService(repo, testUserRepo(database.RoleMember))

	err := svc.Create(1, 1, 100.0, "", "Rent", fixedDate(), database.VisibleEditable, false, database.TransactionTypeExpense)
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
	svc := NewExpenseService(repo, testUserRepo(database.RoleMember))

	err := svc.Create(1, 1, -50.0, "Groceries", "Groceries", fixedDate(), database.VisibleEditable, false, database.TransactionTypeExpense)
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
	svc := NewExpenseService(repo, testUserRepo(database.RoleMember))

	err := svc.Create(1, 1, 100.0, "Rent", "CatCosts", fixedDate(), database.VisibleEditable, false, database.TransactionTypeExpense)
	if err == nil {
		t.Fatal("expected error for invalid category")
	}
	if !errors.Is(err, ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

func TestCreate_RepoError(t *testing.T) {
	repo := &mockExpenseRepo{
		createFn: func(e *database.Expense) error {
			return errors.New("db connection lost")
		},
	}
	svc := NewExpenseService(repo, testUserRepo(database.RoleMember))

	err := svc.Create(1, 1, 100.0, "Groceries", "Groceries", fixedDate(), database.VisibleEditable, false, database.TransactionTypeExpense)
	if err == nil {
		t.Fatal("expected error when repo.Create fails")
	}
	if err.Error() != "db connection lost" {
		t.Errorf("expected 'db connection lost', got '%v'", err)
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
	svc := NewExpenseService(repo, testUserRepo(database.RoleMember))

	err := svc.Create(1, 1, 1500.0, "Monthly Rent", "Rent", fixedDate(), database.VisibleEditable, true, database.TransactionTypeExpense)
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
	svc := NewExpenseService(repo, testUserRepo(database.RoleMember))

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
	svc := NewExpenseService(repo, testUserRepo(database.RoleMember))

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
	svc := NewExpenseService(repo, testUserRepo(database.RoleMember))

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
	svc := NewExpenseService(repo, testUserRepo(database.RoleMember))

	err := svc.Delete(1, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deletedID != 1 {
		t.Errorf("expected delete called with ID 1, got %d", deletedID)
	}
}

func TestDelete_BlockNonCreatorVisibleOnly(t *testing.T) {
	repo := &mockExpenseRepo{
		findByIDFn: func(id uint) (*database.Expense, error) {
			return &database.Expense{
				ID:          1,
				CreatedByID: 1,
				Visibility:  database.VisibleOnly,
			}, nil
		},
		deleteFn: func(id uint) error {
			t.Error("Delete should not be called for non-creator on visible_only")
			return nil
		},
	}
	svc := NewExpenseService(repo, testUserRepo(database.RoleMember))

	err := svc.Delete(2, 1)
	if err == nil {
		t.Fatal("expected permission error")
	}
	if !errors.Is(err, ErrPermission) {
		t.Errorf("expected ErrPermission, got %v", err)
	}
}

// --- FindByHousehold tests ---

func TestFindByHousehold_CallsRepo(t *testing.T) {
	var calledUserID, calledHouseholdID uint
	var calledFilters database.ExpenseFilters
	repo := &mockExpenseRepo{
		findByHouseholdFn: func(userID, householdID uint, viewerRole string, filters database.ExpenseFilters) ([]database.Expense, error) {
			calledUserID = userID
			calledHouseholdID = householdID
			calledFilters = filters
			return []database.Expense{
				{ID: 1, Description: "Test", Amount: 100, Category: "Rent"},
			}, nil
		},
	}
	svc := NewExpenseService(repo, testUserRepo(database.RoleMember))

	filters := database.ExpenseFilters{Category: "Rent"}
	results, err := svc.FindByHousehold(1, 2, filters)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if calledUserID != 1 {
		t.Errorf("expected userID=1, got %d", calledUserID)
	}
	if calledHouseholdID != 2 {
		t.Errorf("expected householdID=2, got %d", calledHouseholdID)
	}
	if calledFilters.Category != "Rent" {
		t.Errorf("expected filters.Category='Rent', got '%s'", calledFilters.Category)
	}
}

// --- FindByID tests ---

func TestFindByID_Success(t *testing.T) {
	repo := &mockExpenseRepo{
		findByIDFn: func(id uint) (*database.Expense, error) {
			return &database.Expense{
				ID: 1, HouseholdID: 1, CreatedByID: 1, Visibility: database.VisibleEditable,
				Description: "Rent", Amount: 1500, Category: "Rent",
			}, nil
		},
	}
	svc := NewExpenseService(repo, testUserRepo(database.RoleMember))

	expense, err := svc.FindByID(2, 1, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if expense == nil {
		t.Fatal("expected an expense to be returned")
	}
	if expense.Description != "Rent" {
		t.Errorf("expected description 'Rent', got '%s'", expense.Description)
	}
}

func TestFindByID_HiddenPrivateBlocksNonCreator(t *testing.T) {
	repo := &mockExpenseRepo{
		findByIDFn: func(id uint) (*database.Expense, error) {
			return &database.Expense{
				ID: 1, HouseholdID: 1, CreatedByID: 1, Visibility: database.HiddenPrivate,
			}, nil
		},
	}
	svc := NewExpenseService(repo, testUserRepo(database.RoleMember))

	_, err := svc.FindByID(2, 1, 1)
	if !errors.Is(err, ErrPermission) {
		t.Errorf("expected ErrPermission for hidden_private expense, got %v", err)
	}
}

func TestFindByID_NotFound(t *testing.T) {
	repo := &mockExpenseRepo{
		findByIDFn: func(id uint) (*database.Expense, error) {
			return nil, nil
		},
	}
	svc := NewExpenseService(repo, testUserRepo(database.RoleMember))

	_, err := svc.FindByID(1, 1, 999)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestFindByID_WrongHouseholdDenied(t *testing.T) {
	repo := &mockExpenseRepo{
		findByIDFn: func(id uint) (*database.Expense, error) {
			return &database.Expense{
				ID: 1, HouseholdID: 99, CreatedByID: 1, Visibility: database.VisibleEditable,
			}, nil
		},
	}
	svc := NewExpenseService(repo, testUserRepo(database.RoleMember))

	_, err := svc.FindByID(1, 1, 1)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound for expense outside the household, got %v", err)
	}
}

func TestFindByID_RepoError(t *testing.T) {
	repo := &mockExpenseRepo{
		findByIDFn: func(id uint) (*database.Expense, error) {
			return nil, errors.New("db error")
		},
	}
	svc := NewExpenseService(repo, testUserRepo(database.RoleMember))

	_, err := svc.FindByID(1, 1, 1)
	if err == nil || err.Error() != "db error" {
		t.Errorf("expected 'db error', got %v", err)
	}
}

// --- Delete not-found tests ---

func TestDelete_RepoError(t *testing.T) {
	repo := &mockExpenseRepo{
		findByIDFn: func(id uint) (*database.Expense, error) {
			return nil, errors.New("db error")
		},
	}
	svc := NewExpenseService(repo, testUserRepo(database.RoleMember))

	err := svc.Delete(1, 999)
	if err == nil {
		t.Fatal("expected error when repo.FindByID fails")
	}
	if err.Error() != "db error" {
		t.Errorf("expected 'db error', got '%v'", err)
	}
}

func TestDelete_NotFound(t *testing.T) {
	repo := &mockExpenseRepo{
		findByIDFn: func(id uint) (*database.Expense, error) {
			return nil, nil // expense not found
		},
		deleteFn: func(id uint) error {
			t.Error("Delete should not be called when expense is not found")
			return nil
		},
	}
	svc := NewExpenseService(repo, testUserRepo(database.RoleMember))

	err := svc.Delete(1, 999)
	if err == nil {
		t.Fatal("expected error for not-found expense")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// --- Update not-found tests ---

func TestUpdate_NotFound(t *testing.T) {
	repo := &mockExpenseRepo{
		findByIDFn: func(id uint) (*database.Expense, error) {
			return nil, nil // expense not found
		},
		updateFn: func(e *database.Expense) error {
			t.Error("Update should not be called when expense is not found")
			return nil
		},
	}
	svc := NewExpenseService(repo, testUserRepo(database.RoleMember))

	err := svc.Update(1, 999, ExpenseUpdateFields{Description: strPtr("Should not update")})
	if err == nil {
		t.Fatal("expected error for not-found expense")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUpdate_AllFields(t *testing.T) {
	var updated *database.Expense
	date := time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC)
	isFixed := true
	vis := database.VisibleOnly
	repo := &mockExpenseRepo{
		findByIDFn: func(id uint) (*database.Expense, error) {
			return &database.Expense{
				ID: 1, CreatedByID: 1, Visibility: database.VisibleEditable,
				Description: "Old", Amount: 100, Category: "Rent",
			}, nil
		},
		updateFn: func(e *database.Expense) error {
			updated = e
			return nil
		},
	}
	svc := NewExpenseService(repo, testUserRepo(database.RoleMember))

	amount := 250.50
	err := svc.Update(1, 1, ExpenseUpdateFields{
		Amount:      &amount,
		Description: strPtr("New desc"),
		Category:    strPtr("Groceries"),
		Date:        &date,
		Visibility:  &vis,
		IsFixed:     &isFixed,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated == nil {
		t.Fatal("expected repo.Update to be called")
	}
	if updated.Amount != 250.50 {
		t.Errorf("expected Amount=250.50, got %f", updated.Amount)
	}
	if updated.Description != "New desc" {
		t.Errorf("expected Description='New desc', got '%s'", updated.Description)
	}
	if updated.Category != "Groceries" {
		t.Errorf("expected Category='Groceries', got '%s'", updated.Category)
	}
	if !updated.Date.Equal(date) {
		t.Errorf("expected Date=%v, got %v", date, updated.Date)
	}
	if updated.Visibility != vis {
		t.Errorf("expected Visibility=%s, got %s", vis, updated.Visibility)
	}
	if updated.IsFixed != isFixed {
		t.Errorf("expected IsFixed=true, got %v", updated.IsFixed)
	}
}

func TestUpdate_InvalidCategory(t *testing.T) {
	repo := &mockExpenseRepo{
		findByIDFn: func(id uint) (*database.Expense, error) {
			return &database.Expense{
				ID: 1, CreatedByID: 1, Visibility: database.VisibleEditable,
				Description: "Old", Amount: 100, Category: "Rent",
			}, nil
		},
		updateFn: func(e *database.Expense) error {
			t.Error("Update should not be called with invalid category")
			return nil
		},
	}
	svc := NewExpenseService(repo, testUserRepo(database.RoleMember))

	err := svc.Update(1, 1, ExpenseUpdateFields{Category: strPtr("InvalidCat")})
	if err == nil {
		t.Fatal("expected validation error for invalid category")
	}
	if !errors.Is(err, ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

// --- Dashboard error propagation tests ---

func TestGetDashboardSummary_MonthlyTotalError(t *testing.T) {
	repo := &mockExpenseRepo{
		monthlyTotalFn: func(userID, householdID uint, viewerRole string, year int, month time.Month) (float64, error) {
			return 0, errors.New("db error")
		},
	}
	svc := NewExpenseService(repo, testUserRepo(database.RoleMember))

	_, err := svc.GetDashboardSummary(1, 1)
	if err == nil {
		t.Fatal("expected error when MonthlyTotal fails")
	}
}

func TestGetDashboardSummary_CategoryBreakdownError(t *testing.T) {
	repo := &mockExpenseRepo{
		monthlyTotalFn: func(userID, householdID uint, viewerRole string, year int, month time.Month) (float64, error) {
			return 0, nil
		},
		categoryBreakdownFn: func(userID, householdID uint, viewerRole string, year int, month time.Month) ([]repositories.CategoryTotal, error) {
			return nil, errors.New("category db error")
		},
	}
	svc := NewExpenseService(repo, testUserRepo(database.RoleMember))

	_, err := svc.GetDashboardSummary(1, 1)
	if err == nil {
		t.Fatal("expected error when CategoryBreakdown fails")
	}
}

// --- Helper ---

func strPtr(s string) *string {
	return &s
}

// --- Dashboard tests ---

func TestGetDashboardSummary_Success(t *testing.T) {
	repo := &mockExpenseRepo{
		monthlyTotalFn: func(userID, householdID uint, viewerRole string, year int, month time.Month) (float64, error) {
			return 350.50, nil
		},
		categoryBreakdownFn: func(userID, householdID uint, viewerRole string, year int, month time.Month) ([]repositories.CategoryTotal, error) {
			return []repositories.CategoryTotal{
				{Category: "Groceries", Total: 200.00},
				{Category: "Rent", Total: 150.50},
			}, nil
		},
		recentExpensesFn: func(userID, householdID uint, viewerRole string, limit int) ([]database.Expense, error) {
			return []database.Expense{
				{ID: 1, Description: "Groceries", Amount: 50, Category: "Groceries"},
				{ID: 2, Description: "Rent", Amount: 150, Category: "Rent"},
			}, nil
		},
	}
	svc := NewExpenseService(repo, testUserRepo(database.RoleMember))

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
		monthlyTotalFn: func(userID, householdID uint, viewerRole string, year int, month time.Month) (float64, error) {
			return 0, nil
		},
		categoryBreakdownFn: func(userID, householdID uint, viewerRole string, year int, month time.Month) ([]repositories.CategoryTotal, error) {
			return []repositories.CategoryTotal{}, nil
		},
		recentExpensesFn: func(userID, householdID uint, viewerRole string, limit int) ([]database.Expense, error) {
			return []database.Expense{}, nil
		},
	}
	svc := NewExpenseService(repo, testUserRepo(database.RoleMember))

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

// --- Task 4.2: service resolves the viewer role and forwards it ---

func TestService_ForwardsResolvedViewerRole(t *testing.T) {
	var got []string
	repo := &mockExpenseRepo{
		findByHouseholdFn: func(_ uint, _ uint, viewerRole string, _ database.ExpenseFilters) ([]database.Expense, error) {
			got = append(got, "list:"+viewerRole)
			return nil, nil
		},
		monthlyTotalFn: func(_ uint, _ uint, viewerRole string, _ int, _ time.Month) (float64, error) {
			got = append(got, "total:"+viewerRole)
			return 0, nil
		},
		categoryBreakdownFn: func(_ uint, _ uint, viewerRole string, _ int, _ time.Month) ([]repositories.CategoryTotal, error) {
			got = append(got, "breakdown:"+viewerRole)
			return nil, nil
		},
		recentExpensesFn: func(_ uint, _ uint, viewerRole string, _ int) ([]database.Expense, error) {
			got = append(got, "recent:"+viewerRole)
			return nil, nil
		},
	}
	svc := NewExpenseService(repo, testUserRepo(database.RoleOwner))

	if _, err := svc.FindByHousehold(7, 2, database.ExpenseFilters{}); err != nil {
		t.Fatalf("FindByHousehold: %v", err)
	}
	if _, err := svc.GetDashboardSummary(7, 2); err != nil {
		t.Fatalf("GetDashboardSummary: %v", err)
	}
	want := []string{"list:owner", "total:owner", "breakdown:owner", "recent:owner"}
	if !slices.Equal(got, want) {
		t.Errorf("viewer role forwarding = %v, want %v", got, want)
	}
}

func TestFindByHousehold_MissingUserDefaultsToMember(t *testing.T) {
	repo := &mockExpenseRepo{
		findByHouseholdFn: func(_ uint, _ uint, viewerRole string, _ database.ExpenseFilters) ([]database.Expense, error) {
			if viewerRole != database.RoleMember {
				t.Errorf("expected fail-closed viewerRole %q for missing user, got %q", database.RoleMember, viewerRole)
			}
			return nil, nil
		},
	}
	svc := NewExpenseService(repo, &mockUserRepo{}) // FindByID returns nil user

	if _, err := svc.FindByHousehold(7, 2, database.ExpenseFilters{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Task 4.3: edit matrix (canEditExpense) and delete ---

func TestCanEditExpense_Matrix(t *testing.T) {
	tests := []struct {
		name       string
		role       string
		visibility database.VisibilityType
		creator    uint // expense creator id
		actor      uint // user attempting the edit
		want       bool
	}{
		{"visible_editable any member", database.RoleMember, database.VisibleEditable, 1, 2, true},
		{"visible_editable admin", database.RoleAdmin, database.VisibleEditable, 1, 2, true},
		{"visible_only creator", database.RoleMember, database.VisibleOnly, 1, 1, true},
		{"visible_only non-creator", database.RoleMember, database.VisibleOnly, 1, 2, false},
		{"hidden_private owner creator", database.RoleOwner, database.HiddenPrivate, 1, 1, true},
		{"hidden_private owner non-creator", database.RoleOwner, database.HiddenPrivate, 1, 2, false},
		{"hidden_private admin denied (hard rule)", database.RoleAdmin, database.HiddenPrivate, 1, 1, false},
		{"hidden_private member denied", database.RoleMember, database.HiddenPrivate, 1, 1, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expense := &database.Expense{CreatedByID: tt.creator, Visibility: tt.visibility}
			if got := canEditExpense(tt.actor, tt.role, expense); got != tt.want {
				t.Errorf("canEditExpense(%d, %q, %s) = %v, want %v", tt.actor, tt.role, tt.visibility, got, tt.want)
			}
		})
	}
}

// --- Task 4.4: view matrix (canViewExpense) ---

func TestCanViewExpense_Matrix(t *testing.T) {
	tests := []struct {
		name       string
		role       string
		visibility database.VisibilityType
		want       bool
	}{
		{"hidden_private owner", database.RoleOwner, database.HiddenPrivate, true},
		{"hidden_private admin denied", database.RoleAdmin, database.HiddenPrivate, false},
		{"hidden_private member denied", database.RoleMember, database.HiddenPrivate, false},
		{"visible_editable member", database.RoleMember, database.VisibleEditable, true},
		{"visible_only member", database.RoleMember, database.VisibleOnly, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expense := &database.Expense{Visibility: tt.visibility}
			if got := canViewExpense(tt.role, expense); got != tt.want {
				t.Errorf("canViewExpense(%q, %s) = %v, want %v", tt.role, tt.visibility, got, tt.want)
			}
		})
	}
}

func TestUpdate_HiddenPrivateCreatorMemberBlocked(t *testing.T) {
	repo := &mockExpenseRepo{
		findByIDFn: func(id uint) (*database.Expense, error) {
			return &database.Expense{ID: 1, CreatedByID: 1, Visibility: database.HiddenPrivate}, nil
		},
		updateFn: func(e *database.Expense) error {
			t.Error("Update must not run for a member creator of hidden_private")
			return nil
		},
	}
	// Creator is user 1 but only holds the member role — hard rule: never edit.
	svc := NewExpenseService(repo, testUserRepo(database.RoleMember))

	err := svc.Update(1, 1, ExpenseUpdateFields{Description: strPtr("Nope")})
	if !errors.Is(err, ErrPermission) {
		t.Errorf("expected ErrPermission, got %v", err)
	}
}

func TestUpdate_HiddenPrivateOwnerCreatorAllowed(t *testing.T) {
	var updated *database.Expense
	repo := &mockExpenseRepo{
		findByIDFn: func(id uint) (*database.Expense, error) {
			return &database.Expense{ID: 1, CreatedByID: 1, Visibility: database.HiddenPrivate}, nil
		},
		updateFn: func(e *database.Expense) error {
			updated = e
			return nil
		},
	}
	svc := NewExpenseService(repo, testUserRepo(database.RoleOwner))

	err := svc.Update(1, 1, ExpenseUpdateFields{Description: strPtr("OK")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated == nil || updated.Description != "OK" {
		t.Fatal("expected the hidden_private expense to be updated by its owner")
	}
}

func TestDelete_AllowNonCreatorVisibleEditable(t *testing.T) {
	var deletedID uint
	repo := &mockExpenseRepo{
		findByIDFn: func(id uint) (*database.Expense, error) {
			return &database.Expense{ID: 1, CreatedByID: 1, Visibility: database.VisibleEditable}, nil
		},
		deleteFn: func(id uint) error {
			deletedID = id
			return nil
		},
	}
	svc := NewExpenseService(repo, testUserRepo(database.RoleMember))

	// Delete follows the edit matrix: visible_editable is editable by any member.
	err := svc.Delete(2, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deletedID != 1 {
		t.Errorf("expected delete called with ID 1, got %d", deletedID)
	}
}

// --- Task 4.4: FindByID honors the owner-only view rule ---

func TestFindByID_HiddenPrivateOwnerCanViewOthers(t *testing.T) {
	repo := &mockExpenseRepo{
		findByIDFn: func(id uint) (*database.Expense, error) {
			return &database.Expense{
				ID: 1, HouseholdID: 1, CreatedByID: 1, Visibility: database.HiddenPrivate,
			}, nil
		},
	}
	// User 2 is not the creator, but the household owner may view hidden_private.
	svc := NewExpenseService(repo, testUserRepo(database.RoleOwner))

	expense, err := svc.FindByID(2, 1, 1)
	if err != nil {
		t.Fatalf("expected owner to view the hidden_private expense, got error: %v", err)
	}
	if expense == nil || expense.ID != 1 {
		t.Fatal("expected the expense to be returned")
	}
}
