package repositories

import (
	"testing"
	"time"

	"github.com/homeadmin/internal/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// seedSiteOverviewData creates two households, three users, and cross-household
// transactions for join-query tests.
func seedSiteOverviewData(t *testing.T, dbSeed func(expenseRepo *ExpenseRepositoryImpl, savingsRepo *SavingsRepositoryImpl, userRepo *UserRepositoryImpl, houseRepo *HouseholdRepositoryImpl)) {
	t.Helper()
	db := setupTestDBRaw(t)
	expenseRepo := NewExpenseRepository(db)
	savingsRepo := NewSavingsRepository(db)
	userRepo := NewUserRepository(db)
	houseRepo := NewHouseholdRepository(db)

	hhA := &database.Household{Name: "Alpha Home"}
	require.NoError(t, houseRepo.Create(hhA))
	hhB := &database.Household{Name: "Beta Home"}
	require.NoError(t, houseRepo.Create(hhB))

	u1 := &database.User{Email: "owner@test.com", PasswordHash: "hash", Role: "owner"}
	require.NoError(t, userRepo.Create(u1))
	require.NoError(t, houseRepo.AddMember(hhA.ID, u1.ID, "owner"))
	u2 := &database.User{Email: "member@test.com", PasswordHash: "hash", Role: "member"}
	require.NoError(t, userRepo.Create(u2))
	require.NoError(t, houseRepo.AddMember(hhA.ID, u2.ID, "member"))
	u3 := &database.User{Email: "solo@test.com", PasswordHash: "hash", Role: "owner"}
	require.NoError(t, userRepo.Create(u3))
	require.NoError(t, houseRepo.AddMember(hhB.ID, u3.ID, "owner"))

	dbSeed(expenseRepo, savingsRepo, userRepo, houseRepo)
}

func TestExpenseRepo_ListAllWithUsers_JoinsOwnerEmailAndOrdersByHouseholdThenDate(t *testing.T) {
	seedSiteOverviewData(t, func(expenseRepo *ExpenseRepositoryImpl, _ *SavingsRepositoryImpl, userRepo *UserRepositoryImpl, houseRepo *HouseholdRepositoryImpl) {
		hhA, err := houseRepo.FindByName("Alpha Home")
		require.NoError(t, err)
		hhB, err := houseRepo.FindByName("Beta Home")
		require.NoError(t, err)
		owner, err := userRepo.FindByEmail("owner@test.com")
		require.NoError(t, err)
		member, err := userRepo.FindByEmail("member@test.com")
		require.NoError(t, err)
		solo, err := userRepo.FindByEmail("solo@test.com")
		require.NoError(t, err)

		require.NoError(t, expenseRepo.Create(&database.Expense{
			Amount: 100, Description: "Old Rent", Category: "rent", Type: "expense",
			HouseholdID: hhA.ID, CreatedByID: owner.ID,
			Visibility: database.VisibleEditable, Date: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		}))
		require.NoError(t, expenseRepo.Create(&database.Expense{
			Amount: 120, Description: "New Rent", Category: "rent", Type: "expense",
			HouseholdID: hhA.ID, CreatedByID: member.ID,
			Visibility: database.HiddenPrivate, Date: time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC),
		}))
		require.NoError(t, expenseRepo.Create(&database.Expense{
			Amount: 500, Description: "Salary", Category: "other", Type: "income",
			HouseholdID: hhB.ID, CreatedByID: solo.ID,
			Visibility: database.VisibleOnly, Date: time.Date(2026, 7, 5, 0, 0, 0, 0, time.UTC),
		}))

		rows, err := expenseRepo.ListAllWithUsers(database.ExpenseFilters{})
		require.NoError(t, err)
		require.Len(t, rows, 3)

		// Ordered by household ASC then date DESC.
		assert.Equal(t, "New Rent", rows[0].Description)
		assert.Equal(t, "member@test.com", rows[0].OwnerEmail)
		assert.Equal(t, "Old Rent", rows[1].Description)
		assert.Equal(t, "owner@test.com", rows[1].OwnerEmail)
		assert.Equal(t, "Salary", rows[2].Description)
		assert.Equal(t, "solo@test.com", rows[2].OwnerEmail)

		// Site-wide view ignores per-member visibility rules.
		assert.Equal(t, database.HiddenPrivate, rows[0].Visibility)
	})
}

func TestExpenseRepo_ListAllWithUsers_ExcludesSoftDeleted(t *testing.T) {
	seedSiteOverviewData(t, func(expenseRepo *ExpenseRepositoryImpl, _ *SavingsRepositoryImpl, userRepo *UserRepositoryImpl, houseRepo *HouseholdRepositoryImpl) {
		hhA, err := houseRepo.FindByName("Alpha Home")
		require.NoError(t, err)
		owner, err := userRepo.FindByEmail("owner@test.com")
		require.NoError(t, err)

		e := &database.Expense{
			Amount: 42, Description: "Doomed", Category: "other", Type: "expense",
			HouseholdID: hhA.ID, CreatedByID: owner.ID,
			Visibility: database.VisibleEditable, Date: time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC),
		}
		require.NoError(t, expenseRepo.Create(e))
		require.NoError(t, expenseRepo.Delete(e.ID))

		rows, err := expenseRepo.ListAllWithUsers(database.ExpenseFilters{})
		require.NoError(t, err)
		assert.Empty(t, rows)
	})
}

func TestExpenseRepo_ListAllWithUsers_CategoryFilter(t *testing.T) {
	seedSiteOverviewData(t, func(expenseRepo *ExpenseRepositoryImpl, _ *SavingsRepositoryImpl, userRepo *UserRepositoryImpl, houseRepo *HouseholdRepositoryImpl) {
		hhA, err := houseRepo.FindByName("Alpha Home")
		require.NoError(t, err)
		owner, err := userRepo.FindByEmail("owner@test.com")
		require.NoError(t, err)

		require.NoError(t, expenseRepo.Create(&database.Expense{
			Amount: 10, Description: "Pizza", Category: "dining_out", Type: "expense",
			HouseholdID: hhA.ID, CreatedByID: owner.ID,
			Visibility: database.VisibleEditable, Date: time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC),
		}))
		require.NoError(t, expenseRepo.Create(&database.Expense{
			Amount: 900, Description: "Rent", Category: "rent", Type: "expense",
			HouseholdID: hhA.ID, CreatedByID: owner.ID,
			Visibility: database.VisibleEditable, Date: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		}))

		rows, err := expenseRepo.ListAllWithUsers(database.ExpenseFilters{Category: "rent"})
		require.NoError(t, err)
		require.Len(t, rows, 1)
		assert.Equal(t, "Rent", rows[0].Description)
	})
}

func TestSavingsRepo_ListAllWithUsers_JoinsOwnerEmailAndOrdersByHouseholdThenCreatedDesc(t *testing.T) {
	seedSiteOverviewData(t, func(_ *ExpenseRepositoryImpl, savingsRepo *SavingsRepositoryImpl, userRepo *UserRepositoryImpl, houseRepo *HouseholdRepositoryImpl) {
		hhA, err := houseRepo.FindByName("Alpha Home")
		require.NoError(t, err)
		hhB, err := houseRepo.FindByName("Beta Home")
		require.NoError(t, err)
		owner, err := userRepo.FindByEmail("owner@test.com")
		require.NoError(t, err)
		solo, err := userRepo.FindByEmail("solo@test.com")
		require.NoError(t, err)

		first := &database.Savings{Description: "Emergency", Amount: 300, HouseholdID: hhA.ID, CreatedByID: owner.ID}
		require.NoError(t, savingsRepo.Create(first))
		second := &database.Savings{Description: "Vacation", Amount: 150, Target: 500, HouseholdID: hhA.ID, CreatedByID: owner.ID}
		require.NoError(t, savingsRepo.Create(second))
		require.NoError(t, savingsRepo.Create(&database.Savings{Description: "Car", Amount: 1000, HouseholdID: hhB.ID, CreatedByID: solo.ID}))

		rows, err := savingsRepo.ListAllWithUsers()
		require.NoError(t, err)
		require.Len(t, rows, 3)

		// Ordered by household ASC then created_at DESC.
		assert.Equal(t, "Vacation", rows[0].Description)
		assert.Equal(t, "owner@test.com", rows[0].OwnerEmail)
		assert.Equal(t, "Emergency", rows[1].Description)
		assert.Equal(t, "Car", rows[2].Description)
		assert.Equal(t, "solo@test.com", rows[2].OwnerEmail)

		var total float64
		for _, r := range rows {
			total += r.Amount
		}
		assert.InDelta(t, 1450.0, total, 0.001)
	})
}
