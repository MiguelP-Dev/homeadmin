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
	ErrNotOwner            = errors.New("only the owner can change member roles")
	ErrSelfRoleChange      = errors.New("the owner cannot change their own role")
	ErrOwnerImmutable      = errors.New("the owner role cannot be changed")
	ErrNotMember           = errors.New("user is not a member of this household")
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
	AddMember(householdID, userID uint, role string) error
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

// Create creates a new household and makes the user its owner.
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
	user.Role = database.RoleOwner
	if err := s.userRepo.Update(user); err != nil {
		return nil, err
	}

	return household, nil
}

// Invite generates a single-use 8-char invite code for the user's household.
// Only owners and admins can generate codes. Codes expire after 7 days.
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
	if user.Role != database.RoleOwner && user.Role != database.RoleAdmin {
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
	user.Role = database.RoleMember
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

// HouseholdView is the data the household page renders: the household, its
// members, and the requesting user's role within it (D6).
type HouseholdView struct {
	Household  *database.Household
	Members    []database.User
	ViewerRole string
}

// Show returns the user's household, its members, and the user's role.
// A user without a household gets (nil, nil).
func (s *HouseholdService) Show(userID uint) (*HouseholdView, error) {
	user, err := s.userRepo.FindByIDWithHousehold(userID)
	if err != nil {
		return nil, err
	}
	if user == nil || user.HouseholdID == nil || user.Household == nil {
		return nil, nil
	}

	members, err := s.houseRepo.GetMembers(*user.HouseholdID)
	if err != nil {
		return nil, err
	}

	return &HouseholdView{
		Household:  user.Household,
		Members:    members,
		ViewerRole: user.Role,
	}, nil
}

// SetMemberRole changes a member's role (admin|member). Only the household
// owner may do it, never on themselves and never on another owner (RF-8).
func (s *HouseholdService) SetMemberRole(ownerID, targetID uint, role string) error {
	owner, err := s.userRepo.FindByID(ownerID)
	if err != nil {
		return err
	}
	if owner == nil || owner.Role != database.RoleOwner {
		return ErrNotOwner
	}

	if role != database.RoleAdmin && role != database.RoleMember {
		return ErrValidation
	}

	target, err := s.userRepo.FindByID(targetID)
	if err != nil {
		return err
	}
	if target == nil || target.HouseholdID == nil || owner.HouseholdID == nil || *target.HouseholdID != *owner.HouseholdID {
		return ErrNotMember
	}
	if targetID == ownerID {
		return ErrSelfRoleChange
	}
	if target.Role == database.RoleOwner {
		return ErrOwnerImmutable
	}

	return s.houseRepo.AddMember(*target.HouseholdID, targetID, role)
}
