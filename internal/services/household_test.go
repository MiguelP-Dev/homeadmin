package services

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/homeadmin/internal/database"
)

// --- Function-field mocks ---

type mockHouseholdRepo struct {
	createFn           func(household *database.Household) error
	findByIDFn         func(id uint) (*database.Household, error)
	findByInviteCodeFn func(code string) (*database.InviteCode, error)
	createInviteCodeFn func(invite *database.InviteCode) error
	getMembersFn       func(householdID uint) ([]database.User, error)
}

func (m *mockHouseholdRepo) Create(household *database.Household) error {
	if m.createFn != nil {
		return m.createFn(household)
	}
	return nil
}

func (m *mockHouseholdRepo) FindByID(id uint) (*database.Household, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(id)
	}
	return nil, nil
}

func (m *mockHouseholdRepo) FindByInviteCode(code string) (*database.InviteCode, error) {
	if m.findByInviteCodeFn != nil {
		return m.findByInviteCodeFn(code)
	}
	return nil, nil
}

func (m *mockHouseholdRepo) CreateInviteCode(invite *database.InviteCode) error {
	if m.createInviteCodeFn != nil {
		return m.createInviteCodeFn(invite)
	}
	return nil
}

func (m *mockHouseholdRepo) GetMembers(householdID uint) ([]database.User, error) {
	if m.getMembersFn != nil {
		return m.getMembersFn(householdID)
	}
	return nil, nil
}

var _ householdRepo = (*mockHouseholdRepo)(nil)

type mockUserRepo struct {
	findByIDFn              func(id uint) (*database.User, error)
	findByIDWithHouseholdFn func(id uint) (*database.User, error)
	updateFn                func(user *database.User) error
}

func (m *mockUserRepo) FindByID(id uint) (*database.User, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(id)
	}
	return nil, nil
}

func (m *mockUserRepo) FindByIDWithHousehold(id uint) (*database.User, error) {
	if m.findByIDWithHouseholdFn != nil {
		return m.findByIDWithHouseholdFn(id)
	}
	return nil, nil
}

func (m *mockUserRepo) Update(user *database.User) error {
	if m.updateFn != nil {
		return m.updateFn(user)
	}
	return nil
}

var _ userRepo = (*mockUserRepo)(nil)

type mockInviteRepo struct {
	markUsedFn func(inviteID, userID uint) error
}

func (m *mockInviteRepo) MarkUsed(inviteID, userID uint) error {
	if m.markUsedFn != nil {
		return m.markUsedFn(inviteID, userID)
	}
	return nil
}

var _ inviteRepo = (*mockInviteRepo)(nil)

func ptr[T any](v T) *T { return &v }

// --- Create (spec: create household) ---

func TestCreate(t *testing.T) {
	tests := []struct {
		name    string
		hhName  string
		user    *database.User
		wantErr error
	}{
		{
			name:    "empty name rejected",
			hhName:  "",
			user:    &database.User{ID: 1},
			wantErr: ErrNameRequired,
		},
		{
			name:    "name over 100 chars rejected",
			hhName:  strings.Repeat("a", 101),
			user:    &database.User{ID: 1},
			wantErr: ErrNameRequired,
		},
		{
			name:    "user already in a household",
			hhName:  "Second Home",
			user:    &database.User{ID: 1, HouseholdID: ptr(uint(7))},
			wantErr: ErrAlreadyHasHousehold,
		},
		{
			name:   "happy path creates household and promotes user to owner",
			hhName: "My Family",
			user:   &database.User{ID: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var created *database.Household
			var updated *database.User
			createCalled := false

			hhRepo := &mockHouseholdRepo{
				createFn: func(hh *database.Household) error {
					createCalled = true
					created = hh
					hh.ID = 10 // GORM populates the primary key on Create
					return nil
				},
			}
			usrRepo := &mockUserRepo{
				findByIDFn: func(id uint) (*database.User, error) {
					return tt.user, nil
				},
				updateFn: func(user *database.User) error {
					updated = user
					return nil
				},
			}
			svc := NewHouseholdService(hhRepo, usrRepo, &mockInviteRepo{})

			hh, err := svc.Create(1, tt.hhName)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Create() error = %v, want %v", err, tt.wantErr)
				}
				if createCalled {
					t.Fatal("Create() persisted a household for invalid input")
				}
				if updated != nil {
					t.Fatal("Create() updated the user for invalid input")
				}
				if hh != nil {
					t.Fatalf("Create() returned household %v, want nil on error", hh)
				}
				return
			}

			if err != nil {
				t.Fatalf("Create() unexpected error: %v", err)
			}
			if !createCalled {
				t.Fatal("Create() did not persist the household")
			}
			if created == nil || created.Name != tt.hhName {
				t.Fatalf("Create() persisted household = %v, want name %q", created, tt.hhName)
			}
			if hh != created {
				t.Errorf("Create() returned %v, want the persisted household %v", hh, created)
			}
			if updated == nil {
				t.Fatal("Create() did not update the user")
			}
			if updated.HouseholdID == nil || *updated.HouseholdID != created.ID {
				t.Errorf("Create() set user HouseholdID = %v, want %d", updated.HouseholdID, created.ID)
			}
			if updated.Role != database.RoleOwner {
				t.Errorf("Create() set user role %q, want %q", updated.Role, database.RoleOwner)
			}
		})
	}
}

// --- Show (spec: show household) ---

func TestShow(t *testing.T) {
	household := &database.Household{ID: 7, Name: "My Family"}

	tests := []struct {
		name          string
		user          *database.User
		members       []database.User
		wantHousehold bool
		wantIsAdmin   bool
		wantMembers   bool
	}{
		{
			name: "user without household gets no data",
			user: &database.User{ID: 1},
		},
		{
			name:          "member sees household without admin flag",
			user:          &database.User{ID: 2, HouseholdID: ptr(uint(7)), Household: household, Role: "member"},
			members:       []database.User{{ID: 2, Role: "member"}},
			wantHousehold: true,
			wantIsAdmin:   false,
			wantMembers:   true,
		},
		{
			name:          "admin sees household with admin flag",
			user:          &database.User{ID: 3, HouseholdID: ptr(uint(7)), Household: household, Role: "admin"},
			members:       []database.User{{ID: 2, Role: "member"}, {ID: 3, Role: "admin"}},
			wantHousehold: true,
			wantIsAdmin:   true,
			wantMembers:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			membersCalled := false
			hhRepo := &mockHouseholdRepo{
				getMembersFn: func(householdID uint) ([]database.User, error) {
					membersCalled = true
					if householdID != 7 {
						t.Errorf("GetMembers called with householdID %d, want 7", householdID)
					}
					return tt.members, nil
				},
			}
			usrRepo := &mockUserRepo{
				findByIDWithHouseholdFn: func(id uint) (*database.User, error) {
					return tt.user, nil
				},
			}
			svc := NewHouseholdService(hhRepo, usrRepo, &mockInviteRepo{})

			hh, members, isAdmin, err := svc.Show(1)
			if err != nil {
				t.Fatalf("Show() unexpected error: %v", err)
			}
			if tt.wantHousehold {
				if hh == nil || hh.ID != household.ID || hh.Name != household.Name {
					t.Errorf("Show() household = %v, want %+v", hh, household)
				}
			} else if hh != nil {
				t.Errorf("Show() household = %v, want nil", hh)
			}
			if tt.wantMembers {
				if len(members) != len(tt.members) {
					t.Fatalf("Show() members = %v, want %d members", members, len(tt.members))
				}
				for i := range tt.members {
					if members[i].ID != tt.members[i].ID || members[i].Role != tt.members[i].Role {
						t.Errorf("Show() members[%d] = %+v, want %+v", i, members[i], tt.members[i])
					}
				}
			} else if members != nil {
				t.Errorf("Show() members = %v, want nil", members)
			}
			if isAdmin != tt.wantIsAdmin {
				t.Errorf("Show() isAdmin = %v, want %v", isAdmin, tt.wantIsAdmin)
			}
			if tt.wantMembers && !membersCalled {
				t.Error("Show() did not fetch the member list")
			}
			if !tt.wantMembers && membersCalled {
				t.Error("Show() fetched members for a user without household")
			}
		})
	}
}

// --- Invite (spec: invite member) ---

func TestInvite(t *testing.T) {
	tests := []struct {
		name          string
		user          *database.User
		wantErr       error
		wantCodeSaved bool
	}{
		{
			name:    "user without household rejected",
			user:    &database.User{ID: 1},
			wantErr: ErrNoHousehold,
		},
		{
			name:    "non-admin member rejected",
			user:    &database.User{ID: 2, HouseholdID: ptr(uint(7)), Role: "member"},
			wantErr: ErrNotAdmin,
		},
		{
			name:          "owner generates invite code",
			user:          &database.User{ID: 3, HouseholdID: ptr(uint(7)), Role: database.RoleOwner},
			wantCodeSaved: true,
		},
		{
			name:          "admin generates invite code",
			user:          &database.User{ID: 4, HouseholdID: ptr(uint(7)), Role: database.RoleAdmin},
			wantCodeSaved: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var saved *database.InviteCode
			hhRepo := &mockHouseholdRepo{
				createInviteCodeFn: func(invite *database.InviteCode) error {
					saved = invite
					return nil
				},
			}
			usrRepo := &mockUserRepo{
				findByIDFn: func(id uint) (*database.User, error) {
					return tt.user, nil
				},
			}
			svc := NewHouseholdService(hhRepo, usrRepo, &mockInviteRepo{})

			code, err := svc.Invite(1)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Invite() error = %v, want %v", err, tt.wantErr)
				}
				if saved != nil {
					t.Fatal("Invite() persisted an invite code for unauthorized user")
				}
				if code != "" {
					t.Fatalf("Invite() returned code %q, want empty on error", code)
				}
				return
			}

			if err != nil {
				t.Fatalf("Invite() unexpected error: %v", err)
			}
			if saved == nil {
				t.Fatal("Invite() did not persist an invite code")
			}
			if len(code) != 8 {
				t.Errorf("Invite() returned code %q with length %d, want 8", code, len(code))
			}
			if code != saved.Code {
				t.Errorf("Invite() returned code %q, but persisted %q", code, saved.Code)
			}
			if saved.HouseholdID != *tt.user.HouseholdID {
				t.Errorf("Invite() persisted householdID %d, want %d", saved.HouseholdID, *tt.user.HouseholdID)
			}
			now := time.Now()
			if saved.ExpiresAt.Before(now.Add(7*24*time.Hour-time.Minute)) ||
				saved.ExpiresAt.After(now.Add(7*24*time.Hour+time.Minute)) {
				t.Errorf("Invite() persisted ExpiresAt %v, want ~7 days from now", saved.ExpiresAt)
			}
		})
	}
}

// --- generateInviteCode (pure function, extract-before-mock) ---

func TestGenerateInviteCode_LengthAndCharset(t *testing.T) {
	for i := 0; i < 100; i++ {
		code, err := generateInviteCode()
		if err != nil {
			t.Fatalf("generateInviteCode() unexpected error: %v", err)
		}
		if len(code) != 8 {
			t.Fatalf("generateInviteCode() returned %q with length %d, want 8", code, len(code))
		}
		for _, r := range code {
			if !((r >= '0' && r <= '9') || (r >= 'A' && r <= 'Z')) {
				t.Fatalf("generateInviteCode() produced char %q outside [0-9A-Z]", r)
			}
		}
	}
}

func TestGenerateInviteCode_NoCollisions(t *testing.T) {
	const n = 100
	seen := make(map[string]bool, n)
	for i := 0; i < n; i++ {
		code, err := generateInviteCode()
		if err != nil {
			t.Fatalf("generateInviteCode() unexpected error: %v", err)
		}
		if seen[code] {
			t.Fatalf("generateInviteCode() returned duplicate code %q across %d calls", code, n)
		}
		seen[code] = true
	}
	if len(seen) != n {
		t.Fatalf("generateInviteCode() produced %d distinct codes, want %d", len(seen), n)
	}
}

// --- Join (spec: join household) ---

func TestJoin(t *testing.T) {
	validInvite := &database.InviteCode{
		ID:          42,
		Code:        "ABC12345",
		HouseholdID: 7,
		ExpiresAt:   time.Now().Add(24 * time.Hour),
	}

	tests := []struct {
		name             string
		user             *database.User
		invite           *database.InviteCode
		wantErr          error
		wantInviteLookup bool
	}{
		{
			name:    "user already in a household",
			user:    &database.User{ID: 1, HouseholdID: ptr(uint(5))},
			invite:  validInvite,
			wantErr: ErrAlreadyHasHousehold,
		},
		{
			name:             "code not found",
			user:             &database.User{ID: 1},
			invite:           nil,
			wantInviteLookup: true,
			wantErr:          ErrInvalidCode,
		},
		{
			name:             "expired code",
			user:             &database.User{ID: 1},
			invite:           &database.InviteCode{ID: 43, Code: "EXPIRED1", HouseholdID: 7, ExpiresAt: time.Now().Add(-time.Hour)},
			wantInviteLookup: true,
			wantErr:          ErrExpiredCode,
		},
		{
			name:             "used code",
			user:             &database.User{ID: 1},
			invite:           &database.InviteCode{ID: 44, Code: "USEDCODE", HouseholdID: 7, ExpiresAt: time.Now().Add(24 * time.Hour), UsedBy: ptr(uint(9))},
			wantInviteLookup: true,
			wantErr:          ErrUsedCode,
		},
		{
			name:             "happy path joins household as member",
			user:             &database.User{ID: 1},
			invite:           validInvite,
			wantInviteLookup: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inviteLookupCalled := false
			hhRepo := &mockHouseholdRepo{
				findByInviteCodeFn: func(code string) (*database.InviteCode, error) {
					inviteLookupCalled = true
					return tt.invite, nil
				},
				findByIDFn: func(id uint) (*database.Household, error) {
					if id != 7 {
						t.Errorf("FindByID called with id %d, want 7", id)
					}
					return &database.Household{ID: 7, Name: "My Family"}, nil
				},
			}
			var updated *database.User
			updateCalled := false
			usrRepo := &mockUserRepo{
				findByIDFn: func(id uint) (*database.User, error) {
					return tt.user, nil
				},
				updateFn: func(user *database.User) error {
					updateCalled = true
					updated = user
					return nil
				},
			}
			var markedInviteID, markedUserID uint
			markCalled := false
			invRepo := &mockInviteRepo{
				markUsedFn: func(inviteID, userID uint) error {
					markCalled = true
					markedInviteID = inviteID
					markedUserID = userID
					return nil
				},
			}
			svc := NewHouseholdService(hhRepo, usrRepo, invRepo)

			hh, err := svc.Join(1, "ABC12345")
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Join() error = %v, want %v", err, tt.wantErr)
				}
				if inviteLookupCalled != tt.wantInviteLookup {
					t.Errorf("Join() invite lookup called = %v, want %v", inviteLookupCalled, tt.wantInviteLookup)
				}
				if updateCalled {
					t.Fatal("Join() updated the user on error")
				}
				if markCalled {
					t.Fatal("Join() marked the code used on error")
				}
				if hh != nil {
					t.Fatalf("Join() returned household %v, want nil on error", hh)
				}
				return
			}

			if err != nil {
				t.Fatalf("Join() unexpected error: %v", err)
			}
			if !inviteLookupCalled {
				t.Fatal("Join() did not look up the invite code")
			}
			if !updateCalled || updated == nil {
				t.Fatal("Join() did not update the user")
			}
			if updated.HouseholdID == nil || *updated.HouseholdID != tt.invite.HouseholdID {
				t.Errorf("Join() set user HouseholdID = %v, want %d", updated.HouseholdID, tt.invite.HouseholdID)
			}
			if updated.Role != database.RoleMember {
				t.Errorf("Join() set user role %q, want %q", updated.Role, database.RoleMember)
			}
			if !markCalled {
				t.Fatal("Join() did not mark the invite code as used")
			}
			if markedInviteID != tt.invite.ID {
				t.Errorf("Join() MarkUsed called with inviteID %d, want %d", markedInviteID, tt.invite.ID)
			}
			if markedUserID != 1 {
				t.Errorf("Join() MarkUsed called with userID %d, want 1", markedUserID)
			}
			if hh == nil || hh.ID != tt.invite.HouseholdID {
				t.Fatalf("Join() returned household %v, want ID %d", hh, tt.invite.HouseholdID)
			}
		})
	}
}
