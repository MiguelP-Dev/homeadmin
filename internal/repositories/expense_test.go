package repositories

import (
	"testing"
	"time"

	"github.com/homeadmin/internal/database"
)

// setupExpenseTestDB creates an expense repo backed by an in-memory SQLite DB.
func setupExpenseTestDB(t *testing.T) (*ExpenseRepositoryImpl, *UserRepositoryImpl, *HouseholdRepositoryImpl) {
	t.Helper()
	db := setupTestDBRaw(t)
	return NewExpenseRepository(db), NewUserRepository(db), NewHouseholdRepository(db)
}

// seedHouseholdAndUsers creates a household and two users for visibility tests.
func seedHouseholdAndUsers(t *testing.T, userRepo *UserRepositoryImpl, houseRepo *HouseholdRepositoryImpl) (uint, uint, uint) {
	t.Helper()
	hh := &database.Household{Name: "Test Household"}
	if err := houseRepo.Create(hh); err != nil {
		t.Fatalf("seed household: %v", err)
	}
	u1 := &database.User{Email: "user1@test.com", PasswordHash: "hash", Role: "member"}
	if err := userRepo.Create(u1); err != nil {
		t.Fatalf("seed user1: %v", err)
	}
	houseRepo.AddMember(hh.ID, u1.ID, "member")

	u2 := &database.User{Email: "user2@test.com", PasswordHash: "hash", Role: "member"}
	if err := userRepo.Create(u2); err != nil {
		t.Fatalf("seed user2: %v", err)
	}
	houseRepo.AddMember(hh.ID, u2.ID, "member")

	return hh.ID, u1.ID, u2.ID
}

// createTestExpense inserts an expense directly via the repo and returns it.
func createTestExpense(t *testing.T, repo *ExpenseRepositoryImpl, e *database.Expense) {
	t.Helper()
	if err := repo.Create(e); err != nil {
		t.Fatalf("createTestExpense: %v", err)
	}
	if e.ID == 0 {
		t.Fatal("expected expense ID to be set after Create")
	}
}

// --- Task 2.1: CRUD tests ---

func TestExpenseRepo_Create(t *testing.T) {
	repo, userRepo, houseRepo := setupExpenseTestDB(t)
	hhID, u1ID, _ := seedHouseholdAndUsers(t, userRepo, houseRepo)

	expense := &database.Expense{
		Amount:      25.50,
		Description: "Groceries",
		Category:    "Groceries",
		HouseholdID: hhID,
		CreatedByID: u1ID,
		Visibility:  database.VisibleEditable,
		Date:        time.Date(2026, 7, 19, 0, 0, 0, 0, time.UTC),
	}

	if err := repo.Create(expense); err != nil {
		t.Fatalf("Create returned unexpected error: %v", err)
	}
	if expense.ID == 0 {
		t.Fatal("expected expense ID to be set after Create")
	}
}

func TestExpenseRepo_FindByID_Found(t *testing.T) {
	repo, userRepo, houseRepo := setupExpenseTestDB(t)
	hhID, u1ID, _ := seedHouseholdAndUsers(t, userRepo, houseRepo)

	created := &database.Expense{
		Amount:      100.00,
		Description: "Rent",
		Category:    "Rent",
		HouseholdID: hhID,
		CreatedByID: u1ID,
		Visibility:  database.VisibleEditable,
		Date:        time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
	}
	createTestExpense(t, repo, created)

	found, err := repo.FindByID(created.ID)
	if err != nil {
		t.Fatalf("FindByID returned unexpected error: %v", err)
	}
	if found == nil {
		t.Fatal("expected expense to be found, got nil")
	}
	if found.Description != "Rent" {
		t.Errorf("expected Description='Rent', got %s", found.Description)
	}
	if found.Amount != 100.00 {
		t.Errorf("expected Amount=100.00, got %f", found.Amount)
	}
}

func TestExpenseRepo_FindByID_NotFound(t *testing.T) {
	repo, _, _ := setupExpenseTestDB(t)

	found, err := repo.FindByID(999)
	if err != nil {
		t.Fatalf("FindByID returned unexpected error: %v", err)
	}
	if found != nil {
		t.Errorf("expected nil for not-found, got expense with ID %d", found.ID)
	}
}

func TestExpenseRepo_Update(t *testing.T) {
	repo, userRepo, houseRepo := setupExpenseTestDB(t)
	hhID, u1ID, _ := seedHouseholdAndUsers(t, userRepo, houseRepo)

	created := &database.Expense{
		Amount:      50.00,
		Description: "Old Desc",
		Category:    "Groceries",
		HouseholdID: hhID,
		CreatedByID: u1ID,
		Visibility:  database.VisibleEditable,
		Date:        time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC),
	}
	createTestExpense(t, repo, created)

	created.Description = "New Desc"
	created.Amount = 75.00
	if err := repo.Update(created); err != nil {
		t.Fatalf("Update returned unexpected error: %v", err)
	}

	found, _ := repo.FindByID(created.ID)
	if found == nil {
		t.Fatal("expected expense to be found after Update")
	}
	if found.Description != "New Desc" {
		t.Errorf("expected Description='New Desc', got %s", found.Description)
	}
	if found.Amount != 75.00 {
		t.Errorf("expected Amount=75.00, got %f", found.Amount)
	}
}

func TestExpenseRepo_Delete(t *testing.T) {
	repo, userRepo, houseRepo := setupExpenseTestDB(t)
	hhID, u1ID, _ := seedHouseholdAndUsers(t, userRepo, houseRepo)

	created := &database.Expense{
		Amount:      30.00,
		Description: "To Delete",
		Category:    "Other",
		HouseholdID: hhID,
		CreatedByID: u1ID,
		Visibility:  database.VisibleEditable,
		Date:        time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC),
	}
	createTestExpense(t, repo, created)

	if err := repo.Delete(created.ID); err != nil {
		t.Fatalf("Delete returned unexpected error: %v", err)
	}

	// Soft-deleted expense should not be found
	found, err := repo.FindByID(created.ID)
	if err != nil {
		t.Fatalf("FindByID returned unexpected error after Delete: %v", err)
	}
	if found != nil {
		t.Error("expected nil for soft-deleted expense, got non-nil")
	}
}

// --- Task 2.2: FindByHousehold returns user's hidden_private + all visible_* ---

func TestExpenseRepo_FindByHousehold_VisibilityStates(t *testing.T) {
	repo, userRepo, houseRepo := setupExpenseTestDB(t)
	hhID, u1ID, _ := seedHouseholdAndUsers(t, userRepo, houseRepo)

	// u1 creates 4 expenses with different visibility states
	expenses := []struct {
		desc       string
		visibility database.VisibilityType
	}{
		{"u1 hidden", database.HiddenPrivate},
		{"u1 visible_editable", database.VisibleEditable},
		{"u1 visible_only", database.VisibleOnly},
		{"u2 visible_editable", database.VisibleEditable},
	}
	for i, e := range expenses {
		createdBy := u1ID
		if i == 3 {
			// u2's expense — need to look up u2
			var users []database.User
			houseRepo.db.Where("email = ?", "user2@test.com").Find(&users)
			if len(users) == 0 {
				t.Fatal("user2 not found")
			}
			createdBy = users[0].ID
		}
		createTestExpense(t, repo, &database.Expense{
			Amount:      float64(10 * (i + 1)),
			Description: e.desc,
			Category:    "Groceries",
			HouseholdID: hhID,
			CreatedByID: createdBy,
			Visibility:  e.visibility,
			Date:        time.Date(2026, 7, 19, 0, 0, 0, 0, time.UTC),
		})
	}

	// u1 lists — should see: u1 hidden + u1 visible_editable + u1 visible_only + u2 visible_editable = 4
	// u1 should NOT see u2's hidden_private (if any existed)
	results, err := repo.FindByHousehold(u1ID, hhID, database.ExpenseFilters{})
	if err != nil {
		t.Fatalf("FindByHousehold returned unexpected error: %v", err)
	}
	if len(results) != 4 {
		t.Errorf("expected 4 expenses visible to u1, got %d", len(results))
	}

	descriptions := make(map[string]bool)
	for _, r := range results {
		descriptions[r.Description] = true
	}
	for _, expected := range []string{"u1 hidden", "u1 visible_editable", "u1 visible_only", "u2 visible_editable"} {
		if !descriptions[expected] {
			t.Errorf("expected expense %q in results but not found", expected)
		}
	}
}

// --- Task 2.3: FindByHousehold excludes other users' hidden_private ---

func TestExpenseRepo_FindByHousehold_ExcludesOthersPrivate(t *testing.T) {
	repo, userRepo, houseRepo := setupExpenseTestDB(t)
	hhID, u1ID, u2ID := seedHouseholdAndUsers(t, userRepo, houseRepo)

	// u2 creates a hidden_private expense — u1 should NOT see it
	createTestExpense(t, repo, &database.Expense{
		Amount:      99.00,
		Description: "u2 private",
		Category:    "Other",
		HouseholdID: hhID,
		CreatedByID: u2ID,
		Visibility:  database.HiddenPrivate,
		Date:        time.Date(2026, 7, 19, 0, 0, 0, 0, time.UTC),
	})

	// u1 also creates a hidden_private expense — u1 SHOULD see it
	createTestExpense(t, repo, &database.Expense{
		Amount:      10.00,
		Description: "u1 private",
		Category:    "Other",
		HouseholdID: hhID,
		CreatedByID: u1ID,
		Visibility:  database.HiddenPrivate,
		Date:        time.Date(2026, 7, 19, 0, 0, 0, 0, time.UTC),
	})

	results, err := repo.FindByHousehold(u1ID, hhID, database.ExpenseFilters{})
	if err != nil {
		t.Fatalf("FindByHousehold returned unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 expense (only u1's private), got %d", len(results))
	}
	if len(results) > 0 && results[0].Description != "u1 private" {
		t.Errorf("expected 'u1 private', got %s", results[0].Description)
	}
}

// --- Task 2.4: FindByHousehold with Category filter ---

func TestExpenseRepo_FindByHousehold_CategoryFilter(t *testing.T) {
	repo, userRepo, houseRepo := setupExpenseTestDB(t)
	hhID, u1ID, _ := seedHouseholdAndUsers(t, userRepo, houseRepo)

	createTestExpense(t, repo, &database.Expense{
		Amount:      25.00,
		Description: "Groceries item",
		Category:    "Groceries",
		HouseholdID: hhID,
		CreatedByID: u1ID,
		Visibility:  database.VisibleEditable,
		Date:        time.Date(2026, 7, 19, 0, 0, 0, 0, time.UTC),
	})
	createTestExpense(t, repo, &database.Expense{
		Amount:      1200.00,
		Description: "Monthly rent",
		Category:    "Rent",
		HouseholdID: hhID,
		CreatedByID: u1ID,
		Visibility:  database.VisibleEditable,
		Date:        time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
	})
	createTestExpense(t, repo, &database.Expense{
		Amount:      50.00,
		Description: "More groceries",
		Category:    "Groceries",
		HouseholdID: hhID,
		CreatedByID: u1ID,
		Visibility:  database.VisibleOnly,
		Date:        time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC),
	})

	// Filter by Rent — should return 1
	results, err := repo.FindByHousehold(u1ID, hhID, database.ExpenseFilters{Category: "Rent"})
	if err != nil {
		t.Fatalf("FindByHousehold returned unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 Rent expense, got %d", len(results))
	}
	if len(results) > 0 && results[0].Category != "Rent" {
		t.Errorf("expected Category='Rent', got %s", results[0].Category)
	}

	// Filter by Groceries — should return 2
	results, err = repo.FindByHousehold(u1ID, hhID, database.ExpenseFilters{Category: "Groceries"})
	if err != nil {
		t.Fatalf("FindByHousehold returned unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 Groceries expenses, got %d", len(results))
	}
}
