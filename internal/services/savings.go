package services

import "github.com/homeadmin/internal/database"

// SavingsRepository defines the data-access interface for savings.
type SavingsRepository interface {
	Create(savings *database.Savings) error
	FindByID(id uint) (*database.Savings, error)
	FindByHousehold(householdID uint) ([]database.Savings, error)
	Update(savings *database.Savings) error
	Delete(id uint) error
	GetTotal(householdID uint) (float64, error)
}

// SavingsService provides business logic for savings.
type SavingsService struct {
	repo  SavingsRepository
	users userRepo
}

// NewSavingsService creates a new SavingsService.
func NewSavingsService(repo SavingsRepository, users userRepo) *SavingsService {
	return &SavingsService{repo: repo, users: users}
}

// Create persists a new savings entry.
func (s *SavingsService) Create(userID, householdID uint, description string, amount, target float64) error {
	if description == "" {
		return ErrValidation
	}
	if amount < 0 {
		return ErrValidation
	}

	entry := &database.Savings{
		Description: description,
		Amount:      amount,
		Target:      target,
		HouseholdID: householdID,
		CreatedByID: userID,
	}
	return s.repo.Create(entry)
}

// FindByHousehold returns all savings for a household.
func (s *SavingsService) FindByHousehold(householdID uint) ([]database.Savings, error) {
	return s.repo.FindByHousehold(householdID)
}

// FindByID returns a savings entry if it belongs to the household.
func (s *SavingsService) FindByID(householdID, savingsID uint) (*database.Savings, error) {
	entry, err := s.repo.FindByID(savingsID)
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return nil, ErrNotFound
	}
	if entry.HouseholdID != householdID {
		return nil, ErrNotFound
	}
	return entry, nil
}

// Delete removes a savings entry if it belongs to the household.
func (s *SavingsService) Delete(householdID, savingsID uint) error {
	entry, err := s.repo.FindByID(savingsID)
	if err != nil {
		return err
	}
	if entry == nil {
		return ErrNotFound
	}
	if entry.HouseholdID != householdID {
		return ErrPermission
	}
	return s.repo.Delete(savingsID)
}

// GetTotal returns the sum of all savings for a household.
func (s *SavingsService) GetTotal(householdID uint) (float64, error) {
	return s.repo.GetTotal(householdID)
}
