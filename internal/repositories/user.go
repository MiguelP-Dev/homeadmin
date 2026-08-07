package repositories

import (
	"github.com/homeadmin/internal/database"
	"gorm.io/gorm"
)

// UserRepositoryImpl is the GORM implementation of UserRepository.
type UserRepositoryImpl struct {
	db *gorm.DB
}

// NewUserRepository creates a new GORM-backed UserRepository.
func NewUserRepository(db *gorm.DB) *UserRepositoryImpl {
	return &UserRepositoryImpl{db: db}
}

// Create inserts a new user. Returns error on duplicate email.
func (r *UserRepositoryImpl) Create(user *database.User) error {
	return r.db.Create(user).Error
}

// FindByID looks up a user by primary key. Returns (nil, nil) on not-found.
func (r *UserRepositoryImpl) FindByID(id uint) (*database.User, error) {
	var user database.User
	result := r.db.First(&user, id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, result.Error
	}
	return &user, nil
}

// FindByEmail looks up a user by unique email index. Returns (nil, nil) on not-found.
func (r *UserRepositoryImpl) FindByEmail(email string) (*database.User, error) {
	var user database.User
	result := r.db.Where("email = ?", email).First(&user)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, result.Error
	}
	return &user, nil
}

// FindByIDWithHousehold looks up a user by ID and eager-loads the Household relation.
// Returns (nil, nil) on not-found.
func (r *UserRepositoryImpl) FindByIDWithHousehold(id uint) (*database.User, error) {
	var user database.User
	result := r.db.Preload("Household").First(&user, id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, result.Error
	}
	return &user, nil
}

// ListAllUsers returns every user with the Household relation eager-loaded,
// ordered by email. Site-admin listing (RF-11).
func (r *UserRepositoryImpl) ListAllUsers() ([]database.User, error) {
	var users []database.User
	result := r.db.Preload("Household").Order("email").Find(&users)
	if result.Error != nil {
		return nil, result.Error
	}
	return users, nil
}

// Update persists changes to an existing user.
func (r *UserRepositoryImpl) Update(user *database.User) error {
	return r.db.Save(user).Error
}

// Delete removes a user by ID.
func (r *UserRepositoryImpl) Delete(id uint) error {
	return r.db.Delete(&database.User{}, id).Error
}
