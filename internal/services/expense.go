package services

import (
	"errors"
	"time"

	"github.com/homeadmin/internal/database"
	"github.com/homeadmin/internal/repositories"
)

// Sentinel errors for expense service.
var (
	ErrValidation = errors.New("validation failed")
	ErrPermission = errors.New("permission denied")
	ErrNotFound   = errors.New("expense not found")
)

// ExpenseRepository is the data-access interface the service depends on.
type ExpenseRepository interface {
	Create(expense *database.Expense) error
	FindByID(id uint) (*database.Expense, error)
	FindByHousehold(userID, householdID uint, filters database.ExpenseFilters) ([]database.Expense, error)
	Update(expense *database.Expense) error
	Delete(id uint) error
	MonthlyTotal(userID, householdID uint, year int, month time.Month) (float64, error)
	CategoryBreakdown(userID, householdID uint, year int, month time.Month) ([]repositories.CategoryTotal, error)
	RecentExpenses(userID, householdID uint, limit int) ([]database.Expense, error)
}

// ExpenseUpdateFields carries optional field updates for an expense.
type ExpenseUpdateFields struct {
	Amount      *float64
	Description *string
	Category    *string
	Date        *time.Time
	Visibility  *database.VisibilityType
	IsFixed     *bool
}

// ExpenseService provides business logic for expenses.
type ExpenseService struct {
	repo ExpenseRepository
}

// NewExpenseService creates a new ExpenseService with the given repository.
func NewExpenseService(repo ExpenseRepository) *ExpenseService {
	return &ExpenseService{repo: repo}
}

// Create validates and persists a new expense.
func (s *ExpenseService) Create(userID, householdID uint, amount float64, description, category string, date time.Time, visibility database.VisibilityType, isFixed bool) error {
	if description == "" {
		return ErrValidation
	}
	if amount < 0 {
		return ErrValidation
	}
	if !database.IsValidCategory(category) {
		return ErrValidation
	}

	expense := &database.Expense{
		Amount:      amount,
		Description: description,
		Category:    category,
		HouseholdID: householdID,
		CreatedByID: userID,
		Visibility:  visibility,
		IsFixed:     isFixed,
		Date:        date,
	}
	return s.repo.Create(expense)
}

// Update applies field changes to an expense if the user has permission.
// Permission: creator OR visible_editable visibility.
func (s *ExpenseService) Update(userID, expenseID uint, fields ExpenseUpdateFields) error {
	expense, err := s.repo.FindByID(expenseID)
	if err != nil {
		return err
	}
	if expense == nil {
		return ErrNotFound
	}

	if !canEditExpense(userID, expense) {
		return ErrPermission
	}

	if fields.Amount != nil {
		expense.Amount = *fields.Amount
	}
	if fields.Description != nil {
		expense.Description = *fields.Description
	}
	if fields.Category != nil {
		if !database.IsValidCategory(*fields.Category) {
			return ErrValidation
		}
		expense.Category = *fields.Category
	}
	if fields.Date != nil {
		expense.Date = *fields.Date
	}
	if fields.Visibility != nil {
		expense.Visibility = *fields.Visibility
	}
	if fields.IsFixed != nil {
		expense.IsFixed = *fields.IsFixed
	}

	return s.repo.Update(expense)
}

// Delete removes an expense if the user is the creator.
func (s *ExpenseService) Delete(userID, expenseID uint) error {
	expense, err := s.repo.FindByID(expenseID)
	if err != nil {
		return err
	}
	if expense == nil {
		return ErrNotFound
	}

	if expense.CreatedByID != userID {
		return ErrPermission
	}

	return s.repo.Delete(expenseID)
}

// FindByHousehold returns expenses visible to the user within a household.
func (s *ExpenseService) FindByHousehold(userID, householdID uint, filters database.ExpenseFilters) ([]database.Expense, error) {
	return s.repo.FindByHousehold(userID, householdID, filters)
}

// canEditExpense checks if a user can edit an expense based on ownership and visibility.
func canEditExpense(userID uint, expense *database.Expense) bool {
	if expense.CreatedByID == userID {
		return true
	}
	return expense.Visibility == database.VisibleEditable
}

// DashboardSummary aggregates monthly totals, category breakdown, and recent expenses.
type DashboardSummary struct {
	MonthlyTotal   float64
	CategoryTotals []repositories.CategoryTotal
	RecentExpenses []database.Expense
}

// GetDashboardSummary returns an aggregated summary for the current month's expenses.
func (s *ExpenseService) GetDashboardSummary(userID, householdID uint) (*DashboardSummary, error) {
	now := time.Now()

	total, err := s.repo.MonthlyTotal(userID, householdID, now.Year(), now.Month())
	if err != nil {
		return nil, err
	}

	categories, err := s.repo.CategoryBreakdown(userID, householdID, now.Year(), now.Month())
	if err != nil {
		return nil, err
	}

	recent, err := s.repo.RecentExpenses(userID, householdID, 5)
	if err != nil {
		return nil, err
	}

	return &DashboardSummary{
		MonthlyTotal:   total,
		CategoryTotals: categories,
		RecentExpenses: recent,
	}, nil
}
