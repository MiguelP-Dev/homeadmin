package main

import (
	"testing"

	"github.com/homeadmin/internal/database"
	"github.com/homeadmin/internal/repositories"
)

// setupTestUserRepo connects an in-memory SQLite database, runs Migrate, and
// returns a real GORM-backed UserRepository — the same data path production
// uses, so persistence and zero-rows-changed are exercised for real.
func setupTestUserRepo(t *testing.T) repositories.UserRepository {
	t.Helper()
	db, err := database.Connect("file::memory:")
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}
	return repositories.NewUserRepository(db)
}

func createUser(t *testing.T, repo repositories.UserRepository, email string) {
	t.Helper()
	u := &database.User{Email: email, PasswordHash: "hash", Role: database.RoleMember}
	if err := repo.Create(u); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
}

// TestPromoteUser_KnownEmail flips IsAdmin=true and leaves everything else
// untouched (RF-10).
func TestPromoteUser_KnownEmail(t *testing.T) {
	repo := setupTestUserRepo(t)
	createUser(t, repo, "known@example.com")

	if err := promoteUser(repo, "known@example.com"); err != nil {
		t.Fatalf("promoteUser returned unexpected error: %v", err)
	}

	got, err := repo.FindByEmail("known@example.com")
	if err != nil || got == nil {
		t.Fatalf("failed to reload promoted user: %v", err)
	}
	if !got.IsAdmin {
		t.Error("expected IsAdmin=true after promotion")
	}
	if got.Role != database.RoleMember {
		t.Errorf("expected role unchanged, got %q", got.Role)
	}
	if got.Email != "known@example.com" {
		t.Errorf("expected email unchanged, got %q", got.Email)
	}
}

// TestPromoteUser_UnknownEmail returns an error and changes zero rows
// (threat-matrix row: unknown email must not exit 0 nor flip anyone).
func TestPromoteUser_UnknownEmail(t *testing.T) {
	repo := setupTestUserRepo(t)
	createUser(t, repo, "other@example.com")

	err := promoteUser(repo, "unknown@example.com")
	if err == nil {
		t.Fatal("expected error for unknown email, got nil")
	}

	// Zero rows changed: the existing user must remain non-admin.
	all, err := repo.ListAllUsers()
	if err != nil {
		t.Fatalf("ListAllUsers failed: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected 1 user, got %d", len(all))
	}
	if all[0].IsAdmin {
		t.Error("existing user must not be promoted by a failed run")
	}
}

// TestPromoteUser_Idempotent: promoting an already-promoted user is a no-op
// success — safe to re-run in automation.
func TestPromoteUser_Idempotent(t *testing.T) {
	repo := setupTestUserRepo(t)
	createUser(t, repo, "again@example.com")

	if err := promoteUser(repo, "again@example.com"); err != nil {
		t.Fatalf("first promote failed: %v", err)
	}
	if err := promoteUser(repo, "again@example.com"); err != nil {
		t.Fatalf("second promote failed: %v", err)
	}

	got, err := repo.FindByEmail("again@example.com")
	if err != nil || got == nil {
		t.Fatalf("failed to reload promoted user: %v", err)
	}
	if !got.IsAdmin {
		t.Error("expected user to remain admin after re-promotion")
	}
}
