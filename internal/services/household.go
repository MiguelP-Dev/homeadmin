package services

import (
	"crypto/rand"
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

// Invite generates a single-use 8-char invite code for the user's household.
// Only admins can generate codes. Codes expire after 7 days.
func (s *HouseholdService) Invite(userID uint) (string, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return "", err
	}
	if user == nil {
		return "", errors.New("user not found")
	}
	if user.HouseholdID == nil {
		return "", ErrNoHousehold
	}
	if user.Role != "admin" {
		return "", ErrNotAdmin
	}

	code, err := generateInviteCode()
	if err != nil {
		return "", err
	}

	invite := &database.InviteCode{
		Code:        code,
		HouseholdID: *user.HouseholdID,
		ExpiresAt:   time.Now().Add(inviteCodeTTL),
	}
	if err := s.houseRepo.CreateInviteCode(invite); err != nil {
		return "", err
	}

	return code, nil
}

// generateInviteCode creates a cryptographically random 8-char code from [0-9A-Z].
func generateInviteCode() (string, error) {
	b := make([]byte, inviteCodeLength)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	for i := range b {
		b[i] = inviteCodeCharset[b[i]%byte(len(inviteCodeCharset))]
	}
	return string(b), nil
}

// Join links a user to a household via a valid invite code.
// Validates: user has no household, code exists, not expired, not used.
func (s *HouseholdService) Join(userID uint, code string) (*database.Household, error) {
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

	invite, err := s.houseRepo.FindByInviteCode(code)
	if err != nil {
		return nil, err
	}
	if invite == nil {
		return nil, ErrInvalidCode
	}
	if time.Now().After(invite.ExpiresAt) {
		return nil, ErrExpiredCode
	}
	if invite.UsedBy != nil {
		return nil, ErrUsedCode
	}

	user.HouseholdID = &invite.HouseholdID
	user.Role = "member"
	if err := s.userRepo.Update(user); err != nil {
		return nil, err
	}

	if err := s.inviteRepo.MarkUsed(invite.ID, userID); err != nil {
		return nil, err
	}

	household, err := s.houseRepo.FindByID(invite.HouseholdID)
	if err != nil {
		return nil, err
	}
	if household == nil {
		return nil, errors.New("household not found")
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
