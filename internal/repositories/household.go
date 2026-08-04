package repositories

import (
	"github.com/homeadmin/internal/database"
	"gorm.io/gorm"
)

// HouseholdRepositoryImpl is the GORM implementation of HouseholdRepository.
type HouseholdRepositoryImpl struct {
	db *gorm.DB
}

// NewHouseholdRepository creates a new GORM-backed HouseholdRepository.
func NewHouseholdRepository(db *gorm.DB) *HouseholdRepositoryImpl {
	return &HouseholdRepositoryImpl{db: db}
}

// Create inserts a new household.
func (r *HouseholdRepositoryImpl) Create(household *database.Household) error {
	return r.db.Create(household).Error
}

// FindByID looks up a household by primary key. Returns (nil, nil) on not-found.
func (r *HouseholdRepositoryImpl) FindByID(id uint) (*database.Household, error) {
	var h database.Household
	result := r.db.First(&h, id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, result.Error
	}
	return &h, nil
}

// FindByUserID returns the household a user belongs to via User.HouseholdID.
// Returns (nil, nil) if the user has no household.
func (r *HouseholdRepositoryImpl) FindByUserID(userID uint) (*database.Household, error) {
	var user database.User
	result := r.db.First(&user, userID)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, result.Error
	}
	if user.HouseholdID == nil {
		return nil, nil
	}
	return r.FindByID(*user.HouseholdID)
}

// FindByName looks up a household by exact name match. Returns (nil, nil) on not-found.
func (r *HouseholdRepositoryImpl) FindByName(name string) (*database.Household, error) {
	var hh database.Household
	result := r.db.Where("name = ?", name).First(&hh)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, result.Error
	}
	return &hh, nil
}

// FindByInviteCode looks up an invite code by its code string. Returns (nil, nil) on not-found.
func (r *HouseholdRepositoryImpl) FindByInviteCode(code string) (*database.InviteCode, error) {
	var invite database.InviteCode
	result := r.db.Where("code = ?", code).First(&invite)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, result.Error
	}
	return &invite, nil
}

// CreateInviteCode inserts a new invite code.
func (r *HouseholdRepositoryImpl) CreateInviteCode(invite *database.InviteCode) error {
	return r.db.Create(invite).Error
}

// MarkUsed sets the UsedBy field on an invite code to record who used it.
func (r *HouseholdRepositoryImpl) MarkUsed(inviteID, userID uint) error {
	return r.db.Model(&database.InviteCode{}).Where("id = ?", inviteID).
		Update("used_by", userID).Error
}

// GetMembers returns all users belonging to the specified household.
func (r *HouseholdRepositoryImpl) GetMembers(householdID uint) ([]database.User, error) {
	var users []database.User
	err := r.db.Where("household_id = ?", householdID).Find(&users).Error
	return users, err
}

// Update persists changes to an existing household.
func (r *HouseholdRepositoryImpl) Update(household *database.Household) error {
	return r.db.Save(household).Error
}

// Delete soft-deletes a household by ID.
func (r *HouseholdRepositoryImpl) Delete(id uint) error {
	return r.db.Delete(&database.Household{}, id).Error
}

// AddMember assigns a user to a household by setting User.HouseholdID.
func (r *HouseholdRepositoryImpl) AddMember(householdID, userID uint, role string) error {
	var user database.User
	if err := r.db.First(&user, userID).Error; err != nil {
		return err
	}
	user.HouseholdID = &householdID
	user.Role = role
	return r.db.Save(&user).Error
}

// RemoveMember removes a user from a household by clearing User.HouseholdID.
func (r *HouseholdRepositoryImpl) RemoveMember(householdID, userID uint) error {
	var user database.User
	if err := r.db.First(&user, userID).Error; err != nil {
		return err
	}
	user.HouseholdID = nil
	return r.db.Save(&user).Error
}
