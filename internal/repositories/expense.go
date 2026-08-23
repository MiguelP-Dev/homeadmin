package repositories

import (
	"time"

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

// visibilityPredicate returns the SQL visibility condition for a household viewer.
// The viewer's household role is resolved by the service layer (the single
// authorization authority) and passed in: hidden_private is only visible to the
// household owner, visible_editable and visible_only are visible to every member.
func visibilityPredicate() string {
	return "((visibility = ? AND ? = ?) OR visibility IN (?, ?))"
}

// FindByHousehold returns expenses visible to the given user within a household.
// Visibility rules: hidden_private expenses are only returned when the viewer is
// the household owner; visible_editable and visible_only are returned for all members.
func (r *ExpenseRepositoryImpl) FindByHousehold(userID, householdID uint, viewerRole string, filters database.ExpenseFilters) ([]database.Expense, error) {
	var expenses []database.Expense

	query := r.db.Where(
		"household_id = ? AND deleted_at IS NULL AND "+visibilityPredicate(),
		householdID,
		database.HiddenPrivate, viewerRole, database.RoleOwner,
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

// MonthlyTotal returns the sum of amounts for expenses visible to the user
// within the household for the given year/month.
func (r *ExpenseRepositoryImpl) MonthlyTotal(userID, householdID uint, viewerRole string, year int, month time.Month) (float64, error) {
	var total float64

	start := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)

	err := r.db.Model(&database.Expense{}).
		Where(
			"household_id = ? AND deleted_at IS NULL AND "+visibilityPredicate()+" AND date >= ? AND date < ?",
			householdID,
			database.HiddenPrivate, viewerRole, database.RoleOwner,
			database.VisibleEditable, database.VisibleOnly,
			start, end,
		).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&total).Error

	return total, err
}

// CategoryBreakdown returns per-category aggregated amounts for the user-visible
// expenses within the household for the given year/month.
func (r *ExpenseRepositoryImpl) CategoryBreakdown(userID, householdID uint, viewerRole string, year int, month time.Month) ([]CategoryTotal, error) {
	var results []CategoryTotal

	start := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)

	err := r.db.Model(&database.Expense{}).
		Select("category, SUM(amount) as total").
		Where(
			"household_id = ? AND deleted_at IS NULL AND "+visibilityPredicate()+" AND date >= ? AND date < ?",
			householdID,
			database.HiddenPrivate, viewerRole, database.RoleOwner,
			database.VisibleEditable, database.VisibleOnly,
			start, end,
		).
		Group("category").
		Order("total DESC").
		Scan(&results).Error

	return results, err
}

// RecentExpenses returns the most recent expenses visible to the user within the household,
// ordered by created_at descending, limited to the given count.
func (r *ExpenseRepositoryImpl) RecentExpenses(userID, householdID uint, viewerRole string, limit int) ([]database.Expense, error) {
	var expenses []database.Expense

	if limit <= 0 {
		limit = 5
	}

	err := r.db.Where(
		"household_id = ? AND deleted_at IS NULL AND "+visibilityPredicate(),
		householdID,
		database.HiddenPrivate, viewerRole, database.RoleOwner,
		database.VisibleEditable, database.VisibleOnly,
	).
		Order("created_at DESC").
		Limit(limit).
		Find(&expenses).Error

	return expenses, err
}

// ListAllWithUsers returns every non-deleted expense across all households
// joined with the creating user's email, ordered by household then date
// descending. Unlike FindByHousehold it applies no per-member visibility
// filtering: site admins see every operation in the site.
func (r *ExpenseRepositoryImpl) ListAllWithUsers(filters database.ExpenseFilters) ([]ExpenseWithUser, error) {
	var expenses []ExpenseWithUser

	query := r.db.Model(&database.Expense{}).
		Select("expenses.*, users.email AS owner_email").
		Joins("JOIN users ON users.id = expenses.created_by_id").
		Where("expenses.deleted_at IS NULL")

	if filters.Category != "" {
		query = query.Where("expenses.category = ?", filters.Category)
	}
	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}
	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}

	err := query.Order("expenses.household_id ASC, expenses.date DESC").Scan(&expenses).Error
	return expenses, err
}
