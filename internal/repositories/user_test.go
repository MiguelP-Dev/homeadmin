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
