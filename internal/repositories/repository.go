package repositories

import (
	"time"

	"github.com/homeadmin/internal/database"
)

// CategoryTotal holds aggregated expenses by category.
type CategoryTotal struct {
	Category string
	Total    float64
}

// UserRepository defines data-access operations for User entities.
// Service layers depend on this interface, not on GORM directly.
type UserRepository interface {
	Create(user *database.User) error
	FindByID(id uint) (*database.User, error)
	FindByEmail(email string) (*database.User, error)
	FindByIDWithHousehold(id uint) (*database.User, error)
	Update(user *database.User) error
	Delete(id uint) error
}

// HouseholdRepository defines data-access operations for Household entities.
type HouseholdRepository interface {
	Create(household *database.Household) error
	FindByID(id uint) (*database.Household, error)
	FindByUserID(userID uint) (*database.Household, error)
	FindByName(name string) (*database.Household, error)
	FindByInviteCode(code string) (*database.InviteCode, error)
	CreateInviteCode(invite *database.InviteCode) error
	MarkUsed(inviteID, userID uint) error
	GetMembers(householdID uint) ([]database.User, error)
	Update(household *database.Household) error
	Delete(id uint) error
	AddMember(householdID, userID uint, role string) error
	RemoveMember(householdID, userID uint) error
}

// ExpenseRepository defines data-access operations for Expense entities.
type ExpenseRepository interface {
	Create(expense *database.Expense) error
	FindByID(id uint) (*database.Expense, error)
	FindByHousehold(userID, householdID uint, filters database.ExpenseFilters) ([]database.Expense, error)
	Update(expense *database.Expense) error
	Delete(id uint) error

	// Dashboard aggregation methods.
	MonthlyTotal(userID, householdID uint, year int, month time.Month) (float64, error)
	CategoryBreakdown(userID, householdID uint, year int, month time.Month) ([]CategoryTotal, error)
	RecentExpenses(userID, householdID uint, limit int) ([]database.Expense, error)
}
