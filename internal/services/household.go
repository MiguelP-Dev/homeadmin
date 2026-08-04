package services

import (
	"errors"
	"time"

	"github.com/homeadmin/internal/database"
)

// Sentinel errors for household service operations.
var (
	ErrAlreadyHasHousehold = errors.New("user already has a household")
	ErrNoHousehold         = errors.New("user has no household")
	ErrNotAdmin            = errors.New("only admins can invite")
	ErrInvalidCode         = errors.New("invalid invite code")
	ErrExpiredCode         = errors.New("invite code has expired")
	ErrUsedCode            = errors.New("invite code has already been used")
	ErrNameRequired        = errors.New("household name is required")
)

// householdRepo is the household data-access surface the service depends on.
type householdRepo interface {
	Create(household *database.Household) error
	FindByID(id uint) (*database.Household, error)
	FindByInviteCode(code string) (*database.InviteCode, error)
	CreateInviteCode(invite *database.InviteCode) error
	GetMembers(householdID uint) ([]database.User, error)
}

// userRepo is the user data-access surface the service depends on.
type userRepo interface {
	FindByID(id uint) (*database.User, error)
	FindByIDWithHousehold(id uint) (*database.User, error)
	Update(user *database.User) error
}

// inviteRepo is the invite-code data-access surface the service depends on.
type inviteRepo interface {
	MarkUsed(inviteID, userID uint) error
}

const (
	maxHouseholdNameLength = 100
	inviteCodeLength       = 8
	inviteCodeTTL          = 7 * 24 * time.Hour
	inviteCodeCharset      = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

// HouseholdService provides business logic for household creation, invites, and joins.
type HouseholdService struct {
	houseRepo  householdRepo
	userRepo   userRepo
	inviteRepo inviteRepo
}

// NewHouseholdService creates a new HouseholdService with injected dependencies.
func NewHouseholdService(houseRepo householdRepo, userRepo userRepo, inviteRepo inviteRepo) *HouseholdService {
	return &HouseholdService{
		houseRepo:  houseRepo,
		userRepo:   userRepo,
		inviteRepo: inviteRepo,
	}
}

// Create creates a new household and makes the user its admin.
// Validates: name 1-100 chars, user must not already belong to a household.
func (s *HouseholdService) Create(userID uint, name string) (*database.Household, error) {
	if name == "" || len(name) > maxHouseholdNameLength {
		return nil, ErrNameRequired
	}

	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("user not found")
	}
	if user.HouseholdID != nil {
		return nil, ErrAlreadyHasHousehold
	}

	household := &database.Household{Name: name}
	if err := s.houseRepo.Create(household); err != nil {
		return nil, err
	}

	user.HouseholdID = &household.ID
	user.Role = "admin"
	if err := s.userRepo.Update(user); err != nil {
		return nil, err
	}

	return household, nil
}

// Show returns the user's household, its members, and whether the user is admin.
// A user without a household gets (nil, nil, false, nil).
func (s *HouseholdService) Show(userID uint) (*database.Household, []database.User, bool, error) {
	user, err := s.userRepo.FindByIDWithHousehold(userID)
	if err != nil {
		return nil, nil, false, err
	}
	if user == nil || user.HouseholdID == nil || user.Household == nil {
		return nil, nil, false, nil
	}

	members, err := s.houseRepo.GetMembers(*user.HouseholdID)
	if err != nil {
		return nil, nil, false, err
	}

	return user.Household, members, user.Role == "admin", nil
}
