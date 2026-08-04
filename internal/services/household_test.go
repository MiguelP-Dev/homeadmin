package services

import (
	"errors"
	"strings"
	"testing"

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
			name:   "happy path creates household and promotes user to admin",
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
			if updated.Role != "admin" {
				t.Errorf("Create() set user role %q, want %q", updated.Role, "admin")
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
