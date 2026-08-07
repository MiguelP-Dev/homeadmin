package repositories

import (
	"fmt"
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

	// u1 (owner) lists — should see: u1 hidden + u1 visible_editable + u1 visible_only + u2 visible_editable = 4
	// u1 should NOT see u2's hidden_private (if any existed)
	results, err := repo.FindByHousehold(u1ID, hhID, database.RoleOwner, database.ExpenseFilters{})
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

// --- Task 4.1: non-owner viewers never see hidden_private ---

func TestExpenseRepo_FindByHousehold_NonOwnerExcludesHidden(t *testing.T) {
	repo, userRepo, houseRepo := setupExpenseTestDB(t)
	hhID, u1ID, u2ID := seedHouseholdAndUsers(t, userRepo, houseRepo)

	// u2 creates a hidden_private expense — a member viewer must not see it.
	createTestExpense(t, repo, &database.Expense{
		Amount:      99.00,
		Description: "u2 private",
		Category:    "Other",
		HouseholdID: hhID,
		CreatedByID: u2ID,
		Visibility:  database.HiddenPrivate,
		Date:        time.Date(2026, 7, 19, 0, 0, 0, 0, time.UTC),
	})
	// u1 creates a hidden_private expense — a member viewer must not see even
	// their own hidden_private (only the owner may view those).
	createTestExpense(t, repo, &database.Expense{
		Amount:      10.00,
		Description: "u1 private",
		Category:    "Other",
		HouseholdID: hhID,
		CreatedByID: u1ID,
		Visibility:  database.HiddenPrivate,
		Date:        time.Date(2026, 7, 19, 0, 0, 0, 0, time.UTC),
	})
	// u1 also creates a visible expense — proves the query runs and filters.
	createTestExpense(t, repo, &database.Expense{
		Amount:      5.00,
		Description: "u1 visible",
		Category:    "Other",
		HouseholdID: hhID,
		CreatedByID: u1ID,
		Visibility:  database.VisibleEditable,
		Date:        time.Date(2026, 7, 19, 0, 0, 0, 0, time.UTC),
	})

	results, err := repo.FindByHousehold(u1ID, hhID, database.RoleMember, database.ExpenseFilters{})
	if err != nil {
		t.Fatalf("FindByHousehold returned unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 expense for member viewer, got %d", len(results))
	}
	if len(results) > 0 && results[0].Description != "u1 visible" {
		t.Errorf("expected 'u1 visible', got %s", results[0].Description)
	}
}

// seedViewerRoleExpenses creates hidden_private, visible_only and visible_editable
// expenses by u2 for viewer-role matrix tests.
func seedViewerRoleExpenses(t *testing.T, repo *ExpenseRepositoryImpl, hhID, u2ID uint) {
	t.Helper()
	createTestExpense(t, repo, &database.Expense{
		Amount: 500.00, Description: "u2 hidden", Category: "Other",
		HouseholdID: hhID, CreatedByID: u2ID, Visibility: database.HiddenPrivate,
		Date: time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC),
	})
	createTestExpense(t, repo, &database.Expense{
		Amount: 40.00, Description: "u2 visible_only", Category: "Other",
		HouseholdID: hhID, CreatedByID: u2ID, Visibility: database.VisibleOnly,
		Date: time.Date(2026, 7, 11, 0, 0, 0, 0, time.UTC),
	})
	createTestExpense(t, repo, &database.Expense{
		Amount: 60.00, Description: "u2 visible_editable", Category: "Other",
		HouseholdID: hhID, CreatedByID: u2ID, Visibility: database.VisibleEditable,
		Date: time.Date(2026, 7, 12, 0, 0, 0, 0, time.UTC),
	})
}

func TestExpenseRepo_ViewerRoleMatrix(t *testing.T) {
	repo, userRepo, houseRepo := setupExpenseTestDB(t)
	hhID, u1ID, u2ID := seedHouseholdAndUsers(t, userRepo, houseRepo)
	seedViewerRoleExpenses(t, repo, hhID, u2ID)

	// u1 (owner) sees everything, including u2's hidden_private.
	results, err := repo.FindByHousehold(u1ID, hhID, database.RoleOwner, database.ExpenseFilters{})
	if err != nil {
		t.Fatalf("owner FindByHousehold: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("owner should see all 3 expenses, got %d", len(results))
	}

	// u1 as member sees visible_only + visible_editable, never hidden_private.
	results, err = repo.FindByHousehold(u1ID, hhID, database.RoleMember, database.ExpenseFilters{})
	if err != nil {
		t.Fatalf("member FindByHousehold: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("member should see 2 expenses, got %d", len(results))
	}
	for _, e := range results {
		if e.Description == "u2 hidden" {
			t.Error("member viewer must not see hidden_private expenses")
		}
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
	results, err := repo.FindByHousehold(u1ID, hhID, database.RoleOwner, database.ExpenseFilters{Category: "Rent"})
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
	results, err = repo.FindByHousehold(u1ID, hhID, database.RoleOwner, database.ExpenseFilters{Category: "Groceries"})
	if err != nil {
		t.Fatalf("FindByHousehold returned unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 Groceries expenses, got %d", len(results))
	}
}

// --- Dashboard: MonthlyTotal ---

func TestExpenseRepo_MonthlyTotal(t *testing.T) {
	repo, userRepo, houseRepo := setupExpenseTestDB(t)
	hhID, u1ID, _ := seedHouseholdAndUsers(t, userRepo, houseRepo)

	// Seed expenses in July 2026
	createTestExpense(t, repo, &database.Expense{
		Amount: 100.00, Description: "Rent", Category: "Rent",
		HouseholdID: hhID, CreatedByID: u1ID, Visibility: database.VisibleEditable,
		Date: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
	})
	createTestExpense(t, repo, &database.Expense{
		Amount: 50.00, Description: "Groceries", Category: "Groceries",
		HouseholdID: hhID, CreatedByID: u1ID, Visibility: database.VisibleEditable,
		Date: time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC),
	})
	createTestExpense(t, repo, &database.Expense{
		Amount: 25.00, Description: "Transport", Category: "Transportation",
		HouseholdID: hhID, CreatedByID: u1ID, Visibility: database.VisibleOnly,
		Date: time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC),
	})
	// Expense in August — should NOT be included
	createTestExpense(t, repo, &database.Expense{
		Amount: 999.00, Description: "Next month", Category: "Other",
		HouseholdID: hhID, CreatedByID: u1ID, Visibility: database.VisibleEditable,
		Date: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
	})

	total, err := repo.MonthlyTotal(u1ID, hhID, database.RoleOwner, 2026, time.July)
	if err != nil {
		t.Fatalf("MonthlyTotal returned unexpected error: %v", err)
	}
	expected := 175.00
	if total != expected {
		t.Errorf("expected MonthlyTotal = %.2f, got %.2f", expected, total)
	}
}

func TestExpenseRepo_MonthlyTotal_VisibilityFilter(t *testing.T) {
	repo, userRepo, houseRepo := setupExpenseTestDB(t)
	hhID, u1ID, u2ID := seedHouseholdAndUsers(t, userRepo, houseRepo)

	// u2 creates hidden_private expense — a member viewer must not see it in total
	createTestExpense(t, repo, &database.Expense{
		Amount: 500.00, Description: "u2 private", Category: "Other",
		HouseholdID: hhID, CreatedByID: u2ID, Visibility: database.HiddenPrivate,
		Date: time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC),
	})
	// u1 creates hidden_private — a member viewer must not see even their own
	createTestExpense(t, repo, &database.Expense{
		Amount: 50.00, Description: "u1 private", Category: "Personal Care",
		HouseholdID: hhID, CreatedByID: u1ID, Visibility: database.HiddenPrivate,
		Date: time.Date(2026, 7, 12, 0, 0, 0, 0, time.UTC),
	})
	// Shared visible expense
	createTestExpense(t, repo, &database.Expense{
		Amount: 100.00, Description: "Shared", Category: "Groceries",
		HouseholdID: hhID, CreatedByID: u2ID, Visibility: database.VisibleEditable,
		Date: time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC),
	})

	total, err := repo.MonthlyTotal(u1ID, hhID, database.RoleMember, 2026, time.July)
	if err != nil {
		t.Fatalf("MonthlyTotal returned unexpected error: %v", err)
	}
	// Member viewer: only the shared visible expense counts = 100, NOT 650
	expected := 100.00
	if total != expected {
		t.Errorf("expected MonthlyTotal = %.2f (visibility filtered), got %.2f", expected, total)
	}
}

// --- Dashboard: CategoryBreakdown ---

func TestExpenseRepo_CategoryBreakdown(t *testing.T) {
	repo, userRepo, houseRepo := setupExpenseTestDB(t)
	hhID, u1ID, _ := seedHouseholdAndUsers(t, userRepo, houseRepo)

	createTestExpense(t, repo, &database.Expense{
		Amount: 1200.00, Description: "Rent", Category: "Rent",
		HouseholdID: hhID, CreatedByID: u1ID, Visibility: database.VisibleEditable,
		Date: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
	})
	createTestExpense(t, repo, &database.Expense{
		Amount: 30.00, Description: "Groceries 1", Category: "Groceries",
		HouseholdID: hhID, CreatedByID: u1ID, Visibility: database.VisibleEditable,
		Date: time.Date(2026, 7, 5, 0, 0, 0, 0, time.UTC),
	})
	createTestExpense(t, repo, &database.Expense{
		Amount: 45.00, Description: "Groceries 2", Category: "Groceries",
		HouseholdID: hhID, CreatedByID: u1ID, Visibility: database.VisibleEditable,
		Date: time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC),
	})
	createTestExpense(t, repo, &database.Expense{
		Amount: 20.00, Description: "Transport", Category: "Transportation",
		HouseholdID: hhID, CreatedByID: u1ID, Visibility: database.VisibleEditable,
		Date: time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC),
	})

	breakdown, err := repo.CategoryBreakdown(u1ID, hhID, database.RoleOwner, 2026, time.July)
	if err != nil {
		t.Fatalf("CategoryBreakdown returned unexpected error: %v", err)
	}
	if len(breakdown) != 3 {
		t.Fatalf("expected 3 categories, got %d", len(breakdown))
	}

	// Build map for assertion
	catMap := make(map[string]float64)
	for _, ct := range breakdown {
		catMap[ct.Category] = ct.Total
	}
	if catMap["Rent"] != 1200.00 {
		t.Errorf("expected Rent total = 1200.00, got %.2f", catMap["Rent"])
	}
	if catMap["Groceries"] != 75.00 {
		t.Errorf("expected Groceries total = 75.00, got %.2f", catMap["Groceries"])
	}
	if catMap["Transportation"] != 20.00 {
		t.Errorf("expected Transportation total = 20.00, got %.2f", catMap["Transportation"])
	}
}

// --- Dashboard: RecentExpenses ---

func TestExpenseRepo_RecentExpenses(t *testing.T) {
	repo, userRepo, houseRepo := setupExpenseTestDB(t)
	hhID, u1ID, _ := seedHouseholdAndUsers(t, userRepo, houseRepo)

	// Seed 10 expenses across July 2026
	for i := 1; i <= 10; i++ {
		createTestExpense(t, repo, &database.Expense{
			Amount: float64(i * 10), Description: fmt.Sprintf("Expense %d", i), Category: "Groceries",
			HouseholdID: hhID, CreatedByID: u1ID, Visibility: database.VisibleEditable,
			Date: time.Date(2026, 7, i, 0, 0, 0, 0, time.UTC),
		})
	}

	recent, err := repo.RecentExpenses(u1ID, hhID, database.RoleOwner, 5)
	if err != nil {
		t.Fatalf("RecentExpenses returned unexpected error: %v", err)
	}
	if len(recent) != 5 {
		t.Errorf("expected 5 recent expenses, got %d", len(recent))
	}

	// Most recent first — Expense 10 should be first
	if len(recent) > 0 && recent[0].Description != "Expense 10" {
		t.Errorf("expected most recent 'Expense 10', got '%s'", recent[0].Description)
	}
}

func TestExpenseRepo_RecentExpenses_VisibilityFilter(t *testing.T) {
	repo, userRepo, houseRepo := setupExpenseTestDB(t)
	hhID, u1ID, u2ID := seedHouseholdAndUsers(t, userRepo, houseRepo)

	// u2 creates hidden_private expense — should not appear for u1
	createTestExpense(t, repo, &database.Expense{
		Amount: 999.00, Description: "u2 hidden", Category: "Other",
		HouseholdID: hhID, CreatedByID: u2ID, Visibility: database.HiddenPrivate,
		Date: time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC),
	})
	// u1 creates a visible expense
	createTestExpense(t, repo, &database.Expense{
		Amount: 100.00, Description: "u1 visible", Category: "Groceries",
		HouseholdID: hhID, CreatedByID: u1ID, Visibility: database.VisibleEditable,
		Date: time.Date(2026, 7, 26, 0, 0, 0, 0, time.UTC),
	})

	recent, err := repo.RecentExpenses(u1ID, hhID, database.RoleMember, 10)
	if err != nil {
		t.Fatalf("RecentExpenses returned unexpected error: %v", err)
	}
	for _, e := range recent {
		if e.Description == "u2 hidden" {
			t.Error("expected u2's hidden_private expense to be excluded from u1's recent")
		}
	}
}
