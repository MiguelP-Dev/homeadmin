package services

import (
	"testing"
	"time"

	"github.com/homeadmin/internal/database"
	"github.com/homeadmin/internal/repositories"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPromoteAdmin_Success(t *testing.T) {
	users := &mockUserRepoForSiteAdmin{
		byEmail: map[string]*database.User{
			"alice@example.com": {ID: 1, Email: "alice@example.com", IsAdmin: false},
		},
	}
	svc := NewSiteAdminService(users, nil, nil, nil)

	err := svc.PromoteAdmin("alice@example.com")
	assert.NoError(t, err)
	assert.True(t, users.byEmail["alice@example.com"].IsAdmin)
}

func TestPromoteAdmin_UnknownEmail(t *testing.T) {
	users := &mockUserRepoForSiteAdmin{byEmail: map[string]*database.User{}}
	svc := NewSiteAdminService(users, nil, nil, nil)

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
	svc := NewSiteAdminService(users, nil, nil, nil)

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
	svc := NewSiteAdminService(nil, households, nil, nil)

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
func (m *mockUserRepoForSiteAdmin) CountAndCreate(user *database.User) error { return nil }
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

// mockExpenseRepoForSiteAdmin satisfies ExpenseRepository for overview tests.
type mockExpenseRepoForSiteAdmin struct {
	all []repositories.ExpenseWithUser
}

func (m *mockExpenseRepoForSiteAdmin) Create(expense *database.Expense) error                { return nil }
func (m *mockExpenseRepoForSiteAdmin) FindByID(id uint) (*database.Expense, error)            { return nil, nil }
func (m *mockExpenseRepoForSiteAdmin) FindByHousehold(userID, householdID uint, viewerRole string, filters database.ExpenseFilters) ([]database.Expense, error) {
	return nil, nil
}
func (m *mockExpenseRepoForSiteAdmin) Update(expense *database.Expense) error                 { return nil }
func (m *mockExpenseRepoForSiteAdmin) Delete(id uint) error                                   { return nil }
func (m *mockExpenseRepoForSiteAdmin) MonthlyTotal(userID, householdID uint, viewerRole string, year int, month time.Month) (float64, error) {
	return 0, nil
}
func (m *mockExpenseRepoForSiteAdmin) CategoryBreakdown(userID, householdID uint, viewerRole string, year int, month time.Month) ([]repositories.CategoryTotal, error) {
	return nil, nil
}
func (m *mockExpenseRepoForSiteAdmin) RecentExpenses(userID, householdID uint, viewerRole string, limit int) ([]database.Expense, error) {
	return nil, nil
}
func (m *mockExpenseRepoForSiteAdmin) ListAllWithUsers(filters database.ExpenseFilters) ([]repositories.ExpenseWithUser, error) {
	return m.all, nil
}

// mockSavingsRepoForSiteAdmin satisfies SavingsRepository for overview tests.
type mockSavingsRepoForSiteAdmin struct {
	all []repositories.SavingsWithUser
}

func (m *mockSavingsRepoForSiteAdmin) Create(savings *database.Savings) error      { return nil }
func (m *mockSavingsRepoForSiteAdmin) FindByID(id uint) (*database.Savings, error) { return nil, nil }
func (m *mockSavingsRepoForSiteAdmin) FindByHousehold(householdID uint) ([]database.Savings, error) {
	return nil, nil
}
func (m *mockSavingsRepoForSiteAdmin) Update(savings *database.Savings) error { return nil }
func (m *mockSavingsRepoForSiteAdmin) Delete(id uint) error                   { return nil }
func (m *mockSavingsRepoForSiteAdmin) GetTotal(householdID uint) (float64, error) {
	return 0, nil
}
func (m *mockSavingsRepoForSiteAdmin) ListAllWithUsers() ([]repositories.SavingsWithUser, error) {
	return m.all, nil
}

func TestSiteAdminOverview_GroupsByHouseholdWithAggregates(t *testing.T) {
	now := time.Now().UTC()
	thisMonth := time.Date(now.Year(), now.Month(), 10, 0, 0, 0, 0, time.UTC)
	lastMonth := thisMonth.AddDate(0, -1, 0)

	households := &mockHouseholdRepoForSiteAdmin{
		all: []database.Household{
			{ID: 1, Name: "Alpha", Members: []database.User{
				{ID: 10, Email: "a-owner@test.com", Role: "owner"},
				{ID: 11, Email: "a-member@test.com", Role: "member"},
			}},
			{ID: 2, Name: "Beta", Members: []database.User{
				{ID: 20, Email: "b-owner@test.com", Role: "owner"},
			}},
		},
	}
	expenses := &mockExpenseRepoForSiteAdmin{
		all: []repositories.ExpenseWithUser{
			{OwnerEmail: "b-owner@test.com", Expense: database.Expense{ID: 3, Amount: 500, Type: database.TransactionTypeIncome, HouseholdID: 2, Date: thisMonth}},
			{OwnerEmail: "a-member@test.com", Expense: database.Expense{ID: 2, Amount: 80, Type: database.TransactionTypeExpense, HouseholdID: 1, Date: thisMonth}},
			{OwnerEmail: "a-owner@test.com", Expense: database.Expense{ID: 1, Amount: 1000, Type: database.TransactionTypeIncome, HouseholdID: 1, Date: lastMonth}},
			{OwnerEmail: "a-owner@test.com", Expense: database.Expense{ID: 4, Amount: 300, Type: database.TransactionTypeExpense, HouseholdID: 1, Date: thisMonth}},
		},
	}
	savings := &mockSavingsRepoForSiteAdmin{
		all: []repositories.SavingsWithUser{
			{OwnerEmail: "a-owner@test.com", Savings: database.Savings{ID: 1, Amount: 250, HouseholdID: 1}},
			{OwnerEmail: "b-owner@test.com", Savings: database.Savings{ID: 2, Amount: 75, HouseholdID: 2}},
		},
	}

	svc := NewSiteAdminService(nil, households, expenses, savings)

	blocks, err := svc.SiteAdminOverview()
	require.NoError(t, err)
	require.Len(t, blocks, 2)

	alpha := blocks[0]
	assert.Equal(t, "Alpha", alpha.Household.Name)
	require.Len(t, alpha.Members, 2)
	assert.Equal(t, HouseholdMember{Email: "a-owner@test.com", Role: "owner"}, alpha.Members[0])
	require.Len(t, alpha.Expenses, 3)
	require.Len(t, alpha.Savings, 1)
	// Monthly: +0 income, -(80+300); All-time net: +1000 -380 = 620.
	assert.InDelta(t, 0.0, alpha.MonthlyIncome, 0.001)
	assert.InDelta(t, 380.0, alpha.MonthlyExpense, 0.001)
	assert.InDelta(t, 620.0, alpha.AllTimeNet, 0.001)
	assert.InDelta(t, 250.0, alpha.SavingsTotal, 0.001)

	beta := blocks[1]
	assert.Equal(t, "Beta", beta.Household.Name)
	require.Len(t, beta.Expenses, 1)
	assert.InDelta(t, 500.0, beta.MonthlyIncome, 0.001)
	assert.InDelta(t, 500.0, beta.AllTimeNet, 0.001)
	assert.InDelta(t, 75.0, beta.SavingsTotal, 0.001)
}

func TestSiteAdminOverview_EmptySiteReturnsEmptySlice(t *testing.T) {
	svc := NewSiteAdminService(nil,
		&mockHouseholdRepoForSiteAdmin{},
		&mockExpenseRepoForSiteAdmin{},
		&mockSavingsRepoForSiteAdmin{},
	)

	blocks, err := svc.SiteAdminOverview()
	require.NoError(t, err)
	assert.Empty(t, blocks)
}

func TestBuildAdminSummary_TotalsAndRows(t *testing.T) {
	blocks := []HouseholdBlock{
		{
			Household:      database.Household{ID: 1, Name: "Alpha"},
			Members:        []HouseholdMember{{Email: "a@test.com", Role: "owner"}},
			MonthlyIncome:  400,
			MonthlyExpense: 150,
			SavingsTotal:   100,
			Expenses: []repositories.ExpenseWithUser{
				{Expense: database.Expense{Amount: 400, Type: database.TransactionTypeIncome}},
				{Expense: database.Expense{Amount: 60, Type: database.TransactionTypeExpense}},
			},
		},
		{
			Household:     database.Household{ID: 2, Name: "Beta"},
			SavingsTotal:  50,
			Expenses: []repositories.ExpenseWithUser{
				{Expense: database.Expense{Amount: 200, Type: database.TransactionTypeIncome}},
			},
		},
	}

	summary := BuildAdminSummary(blocks)
	assert.Equal(t, 2, summary.Households)
	assert.Equal(t, 1, summary.Users)
	assert.Equal(t, 3, summary.Transactions)
	assert.InDelta(t, 600.0, summary.TotalIncome, 0.001)
	assert.InDelta(t, 150.0, summary.TotalSavings, 0.001)
	require.Len(t, summary.Rows, 2)
	assert.Equal(t, AdminHouseholdRow{ID: 1, Name: "Alpha", Members: 1, MonthlyNet: 250}, summary.Rows[0])
	assert.Equal(t, AdminHouseholdRow{ID: 2, Name: "Beta", Members: 0, MonthlyNet: 0}, summary.Rows[1])
}
