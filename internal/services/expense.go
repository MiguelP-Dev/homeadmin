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
// Queries take the resolved viewer role so the repository applies visibility
// filtering without owning any authorization logic.
type ExpenseRepository interface {
	Create(expense *database.Expense) error
	FindByID(id uint) (*database.Expense, error)
	FindByHousehold(userID, householdID uint, viewerRole string, filters database.ExpenseFilters) ([]database.Expense, error)
	Update(expense *database.Expense) error
	Delete(id uint) error
	MonthlyTotal(userID, householdID uint, viewerRole string, year int, month time.Month) (float64, error)
	CategoryBreakdown(userID, householdID uint, viewerRole string, year int, month time.Month) ([]repositories.CategoryTotal, error)
	RecentExpenses(userID, householdID uint, viewerRole string, limit int) ([]database.Expense, error)
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
// It is the single authorization authority: it resolves the viewer's household
// role via userRepo and passes it down so the repository can filter by it.
type ExpenseService struct {
	repo  ExpenseRepository
	users userRepo
}

// NewExpenseService creates a new ExpenseService with the given repositories.
func NewExpenseService(repo ExpenseRepository, users userRepo) *ExpenseService {
	return &ExpenseService{repo: repo, users: users}
}

// viewerRole resolves the user's household role for authorization decisions.
// A missing user defaults to member (least privilege, fail closed).
func (s *ExpenseService) viewerRole(userID uint) (string, error) {
	user, err := s.users.FindByID(userID)
	if err != nil {
		return "", err
	}
	if user == nil || user.Role == "" {
		return database.RoleMember, nil
	}
	return user.Role, nil
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

// Update applies field changes to an expense if the user may edit it.
// Permission follows the visibility edit matrix: visible_editable is editable
// by any member, visible_only by the creator, hidden_private only by the
// creating household owner.
func (s *ExpenseService) Update(userID, expenseID uint, fields ExpenseUpdateFields) error {
	expense, err := s.repo.FindByID(expenseID)
	if err != nil {
		return err
	}
	if expense == nil {
		return ErrNotFound
	}

	role, err := s.viewerRole(userID)
	if err != nil {
		return err
	}
	if !canEditExpense(userID, role, expense) {
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

// Delete removes an expense if the user may edit it (same matrix as Update).
func (s *ExpenseService) Delete(userID, expenseID uint) error {
	expense, err := s.repo.FindByID(expenseID)
	if err != nil {
		return err
	}
	if expense == nil {
		return ErrNotFound
	}

	role, err := s.viewerRole(userID)
	if err != nil {
		return err
	}
	if !canEditExpense(userID, role, expense) {
		return ErrPermission
	}

	return s.repo.Delete(expenseID)
}

// FindByHousehold returns expenses visible to the user within a household.
func (s *ExpenseService) FindByHousehold(userID, householdID uint, filters database.ExpenseFilters) ([]database.Expense, error) {
	role, err := s.viewerRole(userID)
	if err != nil {
		return nil, err
	}
	return s.repo.FindByHousehold(userID, householdID, role, filters)
}

// FindByID returns an expense if the user may view it within their household.
// Permission: the expense must belong to the household; hidden_private expenses
// are viewable only by the household owner, while visible_editable and
// visible_only are viewable by any household member.
func (s *ExpenseService) FindByID(userID, householdID, expenseID uint) (*database.Expense, error) {
	expense, err := s.repo.FindByID(expenseID)
	if err != nil {
		return nil, err
	}
	if expense == nil {
		return nil, ErrNotFound
	}
	if expense.HouseholdID != householdID {
		return nil, ErrNotFound
	}
	role, err := s.viewerRole(userID)
	if err != nil {
		return nil, err
	}
	if !canViewExpense(role, expense) {
		return nil, ErrPermission
	}
	return expense, nil
}

// canEditExpense checks if a user can edit an expense based on their household
// role, ownership, and visibility. A privileged member (non-owner) never edits
// hidden_private expenses, even their own — only the household owner can.
func canEditExpense(userID uint, role string, expense *database.Expense) bool {
	if expense.Visibility == database.HiddenPrivate {
		return expense.CreatedByID == userID && role == database.RoleOwner
	}
	if expense.CreatedByID == userID {
		return true
	}
	return expense.Visibility == database.VisibleEditable
}

// canViewExpense checks if a user can view an expense based on their household
// role and its visibility. hidden_private is visible to the household owner
// only; every member sees visible_* expenses.
func canViewExpense(role string, expense *database.Expense) bool {
	if expense.Visibility == database.HiddenPrivate {
		return role == database.RoleOwner
	}
	return true
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

	role, err := s.viewerRole(userID)
	if err != nil {
		return nil, err
	}

	total, err := s.repo.MonthlyTotal(userID, householdID, role, now.Year(), now.Month())
	if err != nil {
		return nil, err
	}

	categories, err := s.repo.CategoryBreakdown(userID, householdID, role, now.Year(), now.Month())
	if err != nil {
		return nil, err
	}

	recent, err := s.repo.RecentExpenses(userID, householdID, role, 5)
	if err != nil {
		return nil, err
	}

	return &DashboardSummary{
		MonthlyTotal:   total,
		CategoryTotals: categories,
		RecentExpenses: recent,
	}, nil
}
