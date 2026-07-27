package repositories

import (
	"github.com/homeadmin/internal/database"
	"gorm.io/gorm"
)

// ExpenseRepositoryImpl is the GORM implementation of ExpenseRepository.
type ExpenseRepositoryImpl struct {
	db *gorm.DB
}

// NewExpenseRepository creates a new GORM-backed ExpenseRepository.
func NewExpenseRepository(db *gorm.DB) *ExpenseRepositoryImpl {
	return &ExpenseRepositoryImpl{db: db}
}

// Create inserts a new expense.
func (r *ExpenseRepositoryImpl) Create(expense *database.Expense) error {
	return r.db.Create(expense).Error
}

// FindByID looks up an expense by primary key. Returns (nil, nil) on not-found.
func (r *ExpenseRepositoryImpl) FindByID(id uint) (*database.Expense, error) {
	var e database.Expense
	result := r.db.First(&e, id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, result.Error
	}
	return &e, nil
}

// FindByHousehold returns expenses visible to the given user within a household.
// Visibility rules: hidden_private expenses are only returned if created_by = userID.
// visible_editable and visible_only expenses are returned for all household members.
func (r *ExpenseRepositoryImpl) FindByHousehold(userID, householdID uint, filters database.ExpenseFilters) ([]database.Expense, error) {
	var expenses []database.Expense

	query := r.db.Where(
		"household_id = ? AND deleted_at IS NULL AND ((visibility = ? AND created_by_id = ?) OR visibility IN (?, ?))",
		householdID,
		database.HiddenPrivate, userID,
		database.VisibleEditable, database.VisibleOnly,
	)

	if filters.Category != "" {
		query = query.Where("category = ?", filters.Category)
	}
	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}
	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}

	err := query.Order("date DESC").Find(&expenses).Error
	return expenses, err
}

// Update persists changes to an existing expense.
func (r *ExpenseRepositoryImpl) Update(expense *database.Expense) error {
	return r.db.Save(expense).Error
}

// Delete soft-deletes an expense by ID.
func (r *ExpenseRepositoryImpl) Delete(id uint) error {
	return r.db.Delete(&database.Expense{}, id).Error
}
