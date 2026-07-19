package repositories

import "github.com/homeadmin/internal/database"

// UserRepository defines data-access operations for User entities.
// Service layers depend on this interface, not on GORM directly.
type UserRepository interface {
	Create(user *database.User) error
	FindByID(id uint) (*database.User, error)
	FindByEmail(email string) (*database.User, error)
	Update(user *database.User) error
	Delete(id uint) error
}

// HouseholdRepository defines data-access operations for Household entities.
type HouseholdRepository interface {
	Create(household *database.Household) error
	FindByID(id uint) (*database.Household, error)
	FindByUserID(userID uint) (*database.Household, error)
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
}
