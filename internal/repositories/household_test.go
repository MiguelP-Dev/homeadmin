package repositories

import (
	"testing"

	"github.com/homeadmin/internal/database"
)

func setupHouseholdTestDB(t *testing.T) (*HouseholdRepositoryImpl, *UserRepositoryImpl) {
	t.Helper()
	db := setupTestDBRaw(t)
	return NewHouseholdRepository(db), NewUserRepository(db)
}

func TestHouseholdRepo_Create(t *testing.T) {
	houseRepo, _ := setupHouseholdTestDB(t)

	hh := &database.Household{Name: "Test Household"}
	if err := houseRepo.Create(hh); err != nil {
		t.Fatalf("Create returned unexpected error: %v", err)
	}
	if hh.ID == 0 {
		t.Fatal("expected household ID to be set after Create")
	}
}

func TestHouseholdRepo_FindByID_Found(t *testing.T) {
	houseRepo, _ := setupHouseholdTestDB(t)

	hh := &database.Household{Name: "Find Me"}
	if err := houseRepo.Create(hh); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	found, err := houseRepo.FindByID(hh.ID)
	if err != nil {
		t.Fatalf("FindByID returned unexpected error: %v", err)
	}
	if found == nil {
		t.Fatal("expected household to be found, got nil")
	}
	if found.Name != "Find Me" {
		t.Errorf("expected Name='Find Me', got %s", found.Name)
	}
}

func TestHouseholdRepo_FindByID_NotFound(t *testing.T) {
	houseRepo, _ := setupHouseholdTestDB(t)

	found, err := houseRepo.FindByID(999)
	if err != nil {
		t.Fatalf("FindByID returned unexpected error: %v", err)
	}
	if found != nil {
		t.Errorf("expected nil for not-found, got household with ID %d", found.ID)
	}
}

func TestHouseholdRepo_FindByUserID_Found(t *testing.T) {
	houseRepo, userRepo := setupHouseholdTestDB(t)

	hh := &database.Household{Name: "User's Household"}
	if err := houseRepo.Create(hh); err != nil {
		t.Fatalf("Create household failed: %v", err)
	}

	user := &database.User{
		Email:        "member@example.com",
		PasswordHash: "hash",
		Role:         "member",
		HouseholdID:  &hh.ID,
	}
	if err := userRepo.Create(user); err != nil {
		t.Fatalf("Create user failed: %v", err)
	}

	found, err := houseRepo.FindByUserID(user.ID)
	if err != nil {
		t.Fatalf("FindByUserID returned unexpected error: %v", err)
	}
	if found == nil {
		t.Fatal("expected household to be found, got nil")
	}
	if found.Name != "User's Household" {
		t.Errorf("expected Name='User's Household', got %s", found.Name)
	}
}

func TestHouseholdRepo_FindByUserID_NoHousehold(t *testing.T) {
	houseRepo, userRepo := setupHouseholdTestDB(t)

	user := &database.User{
		Email:        "nohouse@example.com",
		PasswordHash: "hash",
		Role:         "member",
	}
	if err := userRepo.Create(user); err != nil {
		t.Fatalf("Create user failed: %v", err)
	}

	found, err := houseRepo.FindByUserID(user.ID)
	if err != nil {
		t.Fatalf("FindByUserID returned unexpected error: %v", err)
	}
	if found != nil {
		t.Errorf("expected nil for user with no household, got %v", found.ID)
	}
}

func TestHouseholdRepo_Update(t *testing.T) {
	houseRepo, _ := setupHouseholdTestDB(t)

	hh := &database.Household{Name: "Original Name"}
	if err := houseRepo.Create(hh); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	hh.Name = "Updated Name"
	if err := houseRepo.Update(hh); err != nil {
		t.Fatalf("Update returned unexpected error: %v", err)
	}

	found, _ := houseRepo.FindByID(hh.ID)
	if found == nil {
		t.Fatal("expected household to be found after Update")
	}
	if found.Name != "Updated Name" {
		t.Errorf("expected Name='Updated Name', got %s", found.Name)
	}
}

func TestHouseholdRepo_Delete(t *testing.T) {
	houseRepo, _ := setupHouseholdTestDB(t)

	hh := &database.Household{Name: "To Delete"}
	if err := houseRepo.Create(hh); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := houseRepo.Delete(hh.ID); err != nil {
		t.Fatalf("Delete returned unexpected error: %v", err)
	}

	// Soft-deleted household should not be found by default GORM scope
	found, err := houseRepo.FindByID(hh.ID)
	if err != nil {
		t.Fatalf("FindByID after Delete failed: %v", err)
	}
	if found != nil {
		t.Error("expected nil for soft-deleted household")
	}
}

func TestHouseholdRepo_RemoveMember(t *testing.T) {
	houseRepo, userRepo := setupHouseholdTestDB(t)

	hh := &database.Household{Name: "House for Remove"}
	if err := houseRepo.Create(hh); err != nil {
		t.Fatalf("Create household failed: %v", err)
	}

	user := &database.User{
		Email:        "removeme@example.com",
		PasswordHash: "hash",
		Role:         "member",
	}
	if err := userRepo.Create(user); err != nil {
		t.Fatalf("Create user failed: %v", err)
	}

	// Add member first
	if err := houseRepo.AddMember(hh.ID, user.ID, "member"); err != nil {
		t.Fatalf("AddMember failed: %v", err)
	}

	// Then remove
	if err := houseRepo.RemoveMember(hh.ID, user.ID); err != nil {
		t.Fatalf("RemoveMember returned unexpected error: %v", err)
	}

	// Verify user's HouseholdID is nil
	updated, err := userRepo.FindByID(user.ID)
	if err != nil {
		t.Fatalf("FindByID after RemoveMember failed: %v", err)
	}
	if updated.HouseholdID != nil {
		t.Error("expected HouseholdID to be nil after RemoveMember")
	}
}
