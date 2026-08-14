package services

import (
	"testing"

	"github.com/homeadmin/internal/database"
	"github.com/stretchr/testify/assert"
)

func TestPromoteAdmin_Success(t *testing.T) {
	users := &mockUserRepoForSiteAdmin{
		byEmail: map[string]*database.User{
			"alice@example.com": {ID: 1, Email: "alice@example.com", IsAdmin: false},
		},
	}
	svc := NewSiteAdminService(users, nil)

	err := svc.PromoteAdmin("alice@example.com")
	assert.NoError(t, err)
	assert.True(t, users.byEmail["alice@example.com"].IsAdmin)
}

func TestPromoteAdmin_UnknownEmail(t *testing.T) {
	users := &mockUserRepoForSiteAdmin{byEmail: map[string]*database.User{}}
	svc := NewSiteAdminService(users, nil)

	err := svc.PromoteAdmin("ghost@example.com")
	assert.Error(t, err)
}

func TestListUsers(t *testing.T) {
	users := &mockUserRepoForSiteAdmin{
		all: []database.User{
			{ID: 1, Email: "a@example.com"},
			{ID: 2, Email: "b@example.com"},
		},
	}
	svc := NewSiteAdminService(users, nil)

	result, err := svc.ListUsers()
	assert.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestListHouseholds(t *testing.T) {
	households := &mockHouseholdRepoForSiteAdmin{
		all: []database.Household{
			{ID: 1, Name: "HH1"},
		},
	}
	svc := NewSiteAdminService(nil, households)

	result, err := svc.ListHouseholds()
	assert.NoError(t, err)
	assert.Len(t, result, 1)
}

// mockUserRepoForSiteAdmin satisfies UserRepository for tests.
type mockUserRepoForSiteAdmin struct {
	byEmail map[string]*database.User
	all     []database.User
}

func (m *mockUserRepoForSiteAdmin) Create(user *database.User) error { return nil }
func (m *mockUserRepoForSiteAdmin) FindByID(id uint) (*database.User, error) {
	for _, u := range m.all {
		if u.ID == id {
			return &u, nil
		}
	}
	return nil, nil
}
func (m *mockUserRepoForSiteAdmin) FindByEmail(email string) (*database.User, error) {
	u, ok := m.byEmail[email]
	if ok {
		return u, nil
	}
	return nil, nil
}
func (m *mockUserRepoForSiteAdmin) FindByIDWithHousehold(id uint) (*database.User, error) { return nil, nil }
func (m *mockUserRepoForSiteAdmin) Update(user *database.User) error {
	if m.byEmail != nil {
		m.byEmail[user.Email] = user
	}
	return nil
}
func (m *mockUserRepoForSiteAdmin) Delete(id uint) error                        { return nil }
func (m *mockUserRepoForSiteAdmin) ListAllUsers() ([]database.User, error)       { return m.all, nil }

// mockHouseholdRepoForSiteAdmin satisfies HouseholdRepository for tests.
type mockHouseholdRepoForSiteAdmin struct {
	all []database.Household
}

func (m *mockHouseholdRepoForSiteAdmin) Create(h *database.Household) error       { return nil }
func (m *mockHouseholdRepoForSiteAdmin) FindByID(id uint) (*database.Household, error) { return nil, nil }
func (m *mockHouseholdRepoForSiteAdmin) FindByUserID(userID uint) (*database.Household, error) { return nil, nil }
func (m *mockHouseholdRepoForSiteAdmin) FindByName(name string) (*database.Household, error) { return nil, nil }
func (m *mockHouseholdRepoForSiteAdmin) FindByInviteCode(code string) (*database.InviteCode, error) { return nil, nil }
func (m *mockHouseholdRepoForSiteAdmin) CreateInviteCode(invite *database.InviteCode) error { return nil }
func (m *mockHouseholdRepoForSiteAdmin) MarkUsed(inviteID, userID uint) error     { return nil }
func (m *mockHouseholdRepoForSiteAdmin) GetMembers(householdID uint) ([]database.User, error) { return nil, nil }
func (m *mockHouseholdRepoForSiteAdmin) Update(h *database.Household) error      { return nil }
func (m *mockHouseholdRepoForSiteAdmin) Delete(id uint) error                    { return nil }
func (m *mockHouseholdRepoForSiteAdmin) AddMember(householdID, userID uint, role string) error { return nil }
func (m *mockHouseholdRepoForSiteAdmin) RemoveMember(householdID, userID uint) error { return nil }
func (m *mockHouseholdRepoForSiteAdmin) ListAllHouseholds() ([]database.Household, error) { return m.all, nil }
