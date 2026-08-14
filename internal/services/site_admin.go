package services

import (
	"errors"

	"github.com/homeadmin/internal/database"
	"github.com/homeadmin/internal/repositories"
)

// SiteAdminService provides site-wide administration operations (RF-11).
type SiteAdminService struct {
	users    repositories.UserRepository
	household repositories.HouseholdRepository
}

// NewSiteAdminService creates a new SiteAdminService.
func NewSiteAdminService(users repositories.UserRepository, household repositories.HouseholdRepository) *SiteAdminService {
	return &SiteAdminService{users: users, household: household}
}

// PromoteAdmin sets IsAdmin=true on the user identified by email.
// Returns ErrNotFound if no user matches the email.
func (s *SiteAdminService) PromoteAdmin(email string) error {
	user, err := s.users.FindByEmail(email)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("user not found")
	}
	user.IsAdmin = true
	return s.users.Update(user)
}

// ListUsers returns all registered users.
func (s *SiteAdminService) ListUsers() ([]database.User, error) {
	return s.users.ListAllUsers()
}

// ListHouseholds returns all households with members eager-loaded.
func (s *SiteAdminService) ListHouseholds() ([]database.Household, error) {
	return s.household.ListAllHouseholds()
}
