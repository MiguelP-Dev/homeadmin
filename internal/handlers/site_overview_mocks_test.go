package handlers

import (
	"time"

	"github.com/homeadmin/internal/database"
	"github.com/homeadmin/internal/repositories"
	"github.com/homeadmin/internal/services"
)

// fakeSiteOverview returns canned household blocks for handler branch tests.
type fakeSiteOverview struct {
	blocks []services.HouseholdBlock
	err    error
}

func (f *fakeSiteOverview) SiteAdminOverview() ([]services.HouseholdBlock, error) {
	return f.blocks, f.err
}

// mockExpenseRepoForHandlers satisfies repositories.ExpenseRepository.
type mockExpenseRepoForHandlers struct {
	all []repositories.ExpenseWithUser
}

func (m *mockExpenseRepoForHandlers) Create(expense *database.Expense) error     { return nil }
func (m *mockExpenseRepoForHandlers) FindByID(id uint) (*database.Expense, error) { return nil, nil }
func (m *mockExpenseRepoForHandlers) FindByHousehold(userID, householdID uint, viewerRole string, filters database.ExpenseFilters) ([]database.Expense, error) {
	return nil, nil
}
func (m *mockExpenseRepoForHandlers) Update(expense *database.Expense) error { return nil }
func (m *mockExpenseRepoForHandlers) Delete(id uint) error                   { return nil }
func (m *mockExpenseRepoForHandlers) MonthlyTotal(userID, householdID uint, viewerRole string, year int, month time.Month) (float64, error) {
	return 0, nil
}
func (m *mockExpenseRepoForHandlers) CategoryBreakdown(userID, householdID uint, viewerRole string, year int, month time.Month) ([]repositories.CategoryTotal, error) {
	return nil, nil
}
func (m *mockExpenseRepoForHandlers) RecentExpenses(userID, householdID uint, viewerRole string, limit int) ([]database.Expense, error) {
	return nil, nil
}
func (m *mockExpenseRepoForHandlers) ListAllWithUsers(filters database.ExpenseFilters) ([]repositories.ExpenseWithUser, error) {
	return m.all, nil
}

// mockSavingsRepoForHandlers satisfies services.SavingsRepository.
type mockSavingsRepoForHandlers struct {
	all []repositories.SavingsWithUser
}

func (m *mockSavingsRepoForHandlers) Create(savings *database.Savings) error      { return nil }
func (m *mockSavingsRepoForHandlers) FindByID(id uint) (*database.Savings, error) { return nil, nil }
func (m *mockSavingsRepoForHandlers) FindByHousehold(householdID uint) ([]database.Savings, error) {
	return nil, nil
}
func (m *mockSavingsRepoForHandlers) Update(savings *database.Savings) error { return nil }
func (m *mockSavingsRepoForHandlers) Delete(id uint) error                   { return nil }
func (m *mockSavingsRepoForHandlers) GetTotal(householdID uint) (float64, error) {
	return 0, nil
}
func (m *mockSavingsRepoForHandlers) ListAllWithUsers() ([]repositories.SavingsWithUser, error) {
	return m.all, nil
}

// compile-time checks
var (
	_ repositories.ExpenseRepository = (*mockExpenseRepoForHandlers)(nil)
	_ services.SavingsRepository     = (*mockSavingsRepoForHandlers)(nil)
	_ siteOverviewService            = (*fakeSiteOverview)(nil)
)
