package repositories

import (
	"github.com/homeadmin/internal/database"
	"gorm.io/gorm"
)

// SavingsRepositoryImpl is the GORM implementation for savings operations.
type SavingsRepositoryImpl struct {
	db *gorm.DB
}

// NewSavingsRepository creates a new GORM-backed SavingsRepository.
func NewSavingsRepository(db *gorm.DB) *SavingsRepositoryImpl {
	return &SavingsRepositoryImpl{db: db}
}

// Create inserts a new savings entry.
func (r *SavingsRepositoryImpl) Create(savings *database.Savings) error {
	return r.db.Create(savings).Error
}

// FindByID looks up a savings entry by primary key. Returns (nil, nil) on not-found.
func (r *SavingsRepositoryImpl) FindByID(id uint) (*database.Savings, error) {
	var s database.Savings
	result := r.db.First(&s, id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, result.Error
	}
	return &s, nil
}

// FindByHousehold returns all savings for a household, ordered by created_at descending.
func (r *SavingsRepositoryImpl) FindByHousehold(householdID uint) ([]database.Savings, error) {
	var savings []database.Savings
	err := r.db.Where("household_id = ? AND deleted_at IS NULL", householdID).
		Order("created_at DESC").
		Find(&savings).Error
	return savings, err
}

// Update persists changes to an existing savings entry.
func (r *SavingsRepositoryImpl) Update(savings *database.Savings) error {
	return r.db.Save(savings).Error
}

// Delete soft-deletes a savings entry by ID.
func (r *SavingsRepositoryImpl) Delete(id uint) error {
	return r.db.Delete(&database.Savings{}, id).Error
}

// GetTotal returns the sum of all savings amounts for a household.
func (r *SavingsRepositoryImpl) GetTotal(householdID uint) (float64, error) {
	var total float64
	err := r.db.Model(&database.Savings{}).
		Where("household_id = ? AND deleted_at IS NULL", householdID).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&total).Error
	return total, err
}
