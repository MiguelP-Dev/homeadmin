package repositories

import (
	"testing"

	"github.com/homeadmin/internal/database"
)

func setupTestDB(t *testing.T) *UserRepositoryImpl {
	t.Helper()
	db := setupTestDBRaw(t)
	return NewUserRepository(db)
}

func TestUserRepo_Create(t *testing.T) {
	repo := setupTestDB(t)

	user := &database.User{
		Email:        "test@example.com",
		PasswordHash: "hashed-password",
		Role:         "member",
	}

	if err := repo.Create(user); err != nil {
		t.Fatalf("Create returned unexpected error: %v", err)
	}
	if user.ID == 0 {
		t.Fatal("expected user ID to be set after Create")
	}
}

func TestUserRepo_CreateDuplicateEmail(t *testing.T) {
	repo := setupTestDB(t)

	user1 := &database.User{
		Email:        "dup@example.com",
		PasswordHash: "hash1",
		Role:         "member",
	}
	if err := repo.Create(user1); err != nil {
		t.Fatalf("first Create failed: %v", err)
	}

	user2 := &database.User{
		Email:        "dup@example.com",
		PasswordHash: "hash2",
		Role:         "member",
	}
	if err := repo.Create(user2); err == nil {
		t.Fatal("expected error for duplicate email, got nil")
	}
}

func TestUserRepo_FindByEmail_Found(t *testing.T) {
	repo := setupTestDB(t)

	expected := &database.User{
		Email:        "found@example.com",
		PasswordHash: "hash",
		Role:         "admin",
	}
	if err := repo.Create(expected); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	found, err := repo.FindByEmail("found@example.com")
	if err != nil {
		t.Fatalf("FindByEmail returned unexpected error: %v", err)
	}
	if found == nil {
		t.Fatal("expected user to be found, got nil")
	}
	if found.Email != expected.Email {
		t.Errorf("expected email %s, got %s", expected.Email, found.Email)
	}
	if found.Role != "admin" {
		t.Errorf("expected role admin, got %s", found.Role)
	}
}

func TestUserRepo_FindByEmail_NotFound(t *testing.T) {
	repo := setupTestDB(t)

	found, err := repo.FindByEmail("missing@example.com")
	if err != nil {
		t.Fatalf("FindByEmail returned unexpected error: %v", err)
	}
	if found != nil {
		t.Errorf("expected nil for not-found, got user with email %s", found.Email)
	}
}

func TestUserRepo_FindByID_Found(t *testing.T) {
	repo := setupTestDB(t)

	user := &database.User{
		Email:        "byid@example.com",
		PasswordHash: "hash",
		Role:         "member",
	}
	if err := repo.Create(user); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	found, err := repo.FindByID(user.ID)
	if err != nil {
		t.Fatalf("FindByID returned unexpected error: %v", err)
	}
	if found == nil {
		t.Fatal("expected user to be found, got nil")
	}
	if found.ID != user.ID {
		t.Errorf("expected ID %d, got %d", user.ID, found.ID)
	}
	if found.Email != "byid@example.com" {
		t.Errorf("expected email byid@example.com, got %s", found.Email)
	}
}

func TestUserRepo_FindByID_NotFound(t *testing.T) {
	repo := setupTestDB(t)

	found, err := repo.FindByID(999)
	if err != nil {
		t.Fatalf("FindByID returned unexpected error: %v", err)
	}
	if found != nil {
		t.Errorf("expected nil for not-found, got user with ID %d", found.ID)
	}
}

func TestUserRepo_Update(t *testing.T) {
	repo := setupTestDB(t)

	user := &database.User{
		Email:        "update@example.com",
		PasswordHash: "hash1",
		Role:         "member",
	}
	if err := repo.Create(user); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	user.Role = "admin"
	user.PasswordHash = "hash2"
	if err := repo.Update(user); err != nil {
		t.Fatalf("Update returned unexpected error: %v", err)
	}

	found, err := repo.FindByID(user.ID)
	if err != nil {
		t.Fatalf("FindByID after update failed: %v", err)
	}
	if found == nil {
		t.Fatal("expected user to be found after Update")
	}
	if found.Role != "admin" {
		t.Errorf("expected Role=admin, got %s", found.Role)
	}
	if found.PasswordHash != "hash2" {
		t.Errorf("expected PasswordHash=hash2, got %s", found.PasswordHash)
	}
}

func TestUserRepo_Delete(t *testing.T) {
	repo := setupTestDB(t)

	user := &database.User{
		Email:        "delete@example.com",
		PasswordHash: "hash",
		Role:         "member",
	}
	if err := repo.Create(user); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := repo.Delete(user.ID); err != nil {
		t.Fatalf("Delete returned unexpected error: %v", err)
	}

	// Deleted user should still be found (hard delete) — GORM Delete marks deleted_at
	// but GORM's First query filters by deleted_at IS NULL by default
	found, err := repo.FindByID(user.ID)
	if err != nil {
		t.Fatalf("FindByID after Delete failed: %v", err)
	}
	if found != nil {
		t.Error("expected nil for deleted user (GORM default scope excludes soft-deleted)")
	}
}

func TestUserRepo_FindByIDWithHousehold_EagerLoadsHousehold(t *testing.T) {
	db := setupTestDBRaw(t)
	houseRepo := NewHouseholdRepository(db)
	userRepo := NewUserRepository(db)

	hh := &database.Household{Name: "Eager Load Household"}
	if err := houseRepo.Create(hh); err != nil {
		t.Fatalf("Create household failed: %v", err)
	}

	user := &database.User{
		Email:        "eager@example.com",
		PasswordHash: "hash",
		Role:         "member",
		HouseholdID:  &hh.ID,
	}
	if err := userRepo.Create(user); err != nil {
		t.Fatalf("Create user failed: %v", err)
	}

	found, err := userRepo.FindByIDWithHousehold(user.ID)
	if err != nil {
		t.Fatalf("FindByIDWithHousehold returned unexpected error: %v", err)
	}
	if found == nil {
		t.Fatal("expected user to be found, got nil")
	}
	if found.Household == nil {
		t.Fatal("expected Household to be eager-loaded, got nil")
	}
	if found.Household.ID != hh.ID {
		t.Errorf("expected Household.ID %d, got %d", hh.ID, found.Household.ID)
	}
	if found.Household.Name != "Eager Load Household" {
		t.Errorf("expected Household.Name='Eager Load Household', got %s", found.Household.Name)
	}
}

func TestUserRepo_FindByIDWithHousehold_NotFound(t *testing.T) {
	repo := setupTestDB(t)

	found, err := repo.FindByIDWithHousehold(999)
	if err != nil {
		t.Fatalf("FindByIDWithHousehold returned unexpected error: %v", err)
	}
	if found != nil {
		t.Errorf("expected nil for not-found, got user with ID %d", found.ID)
	}
}

// TestUserRepo_ListAllUsers_ReturnsAllWithHousehold verifies ListAllUsers
// (RF-11 / design D9): every user is returned, with the Household relation
// eager-loaded so the site-admin page can render each user's household name.
func TestUserRepo_ListAllUsers_ReturnsAllWithHousehold(t *testing.T) {
	db := setupTestDBRaw(t)
	houseRepo := NewHouseholdRepository(db)
	userRepo := NewUserRepository(db)

	hh := &database.Household{Name: "List All Household"}
	if err := houseRepo.Create(hh); err != nil {
		t.Fatalf("Create household failed: %v", err)
	}
	users := []*database.User{
		{Email: "with@example.com", PasswordHash: "hash", Role: database.RoleMember, HouseholdID: &hh.ID},
		{Email: "without@example.com", PasswordHash: "hash", Role: database.RoleMember},
	}
	for _, u := range users {
		if err := userRepo.Create(u); err != nil {
			t.Fatalf("Create user %s failed: %v", u.Email, err)
		}
	}

	all, err := userRepo.ListAllUsers()
	if err != nil {
		t.Fatalf("ListAllUsers returned unexpected error: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 users, got %d", len(all))
	}

	// Both users present with correct email.
	emails := map[string]bool{}
	for _, u := range all {
		emails[u.Email] = true
	}
	if !emails["with@example.com"] || !emails["without@example.com"] {
		t.Errorf("expected both seeded emails in result, got %v", emails)
	}

	// Household relation eager-loaded for the member; nil for the loner.
	for _, u := range all {
		switch u.Email {
		case "with@example.com":
			if u.Household == nil {
				t.Error("expected Household to be eager-loaded for member, got nil")
			} else if u.Household.Name != "List All Household" {
				t.Errorf("expected Household.Name='List All Household', got %q", u.Household.Name)
			}
		case "without@example.com":
			if u.Household != nil {
				t.Errorf("expected nil Household for user without one, got %q", u.Household.Name)
			}
		}
	}
}

// TestUserRepo_ListAllUsers_Empty verifies the empty result on a fresh database
// (triangulation: the collection is empty because no users exist).
func TestUserRepo_ListAllUsers_Empty(t *testing.T) {
	repo := setupTestDB(t)

	all, err := repo.ListAllUsers()
	if err != nil {
		t.Fatalf("ListAllUsers returned unexpected error: %v", err)
	}
	if len(all) != 0 {
		t.Errorf("expected 0 users on empty db, got %d", len(all))
	}
}
