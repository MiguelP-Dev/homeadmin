package database

import (
	"os"
	"testing"
)

func TestConnectSuccess(t *testing.T) {
	db, err := Connect("file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if db == nil {
		t.Fatal("expected non-nil *gorm.DB")
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get underlying sql.DB: %v", err)
	}
	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("failed to ping database: %v", err)
	}
}

func TestConnectFailure(t *testing.T) {
	db, err := Connect("invalid://not-a-real-dsn")
	if err == nil {
		t.Fatal("expected error for invalid DSN, got nil")
	}
	if db != nil {
		t.Fatal("expected nil *gorm.DB on error")
	}
}

func TestMigrateIdempotent(t *testing.T) {
	db, err := Connect("file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}

	// First migration — should succeed
	if err := Migrate(db); err != nil {
		t.Fatalf("first Migrate failed: %v", err)
	}

	// Second migration — must not error (idempotent)
	if err := Migrate(db); err != nil {
		t.Fatalf("second Migrate failed (not idempotent): %v", err)
	}

	// Verify tables exist
	tables := []string{"households", "users", "expenses"}
	for _, table := range tables {
		if !db.Migrator().HasTable(table) {
			t.Errorf("expected table %s to exist after migration", table)
		}
	}
}

func TestMigrateInviteCodesTable(t *testing.T) {
	db, err := Connect("file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}

	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}

	if !db.Migrator().HasTable("invite_codes") {
		t.Fatal("expected invite_codes table to exist after migration")
	}

	columns := []string{"id", "code", "household_id", "expires_at", "used_by", "created_at"}
	for _, col := range columns {
		if !db.Migrator().HasColumn("invite_codes", col) {
			t.Errorf("expected invite_codes column %q to exist", col)
		}
	}
}

func TestMigrateLegacyRoles(t *testing.T) {
	db, err := Connect("file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}

	// Seed a pre-change household: creator with legacy role=admin, one member.
	hh := Household{Name: "Legacy Family"}
	if err := db.Create(&hh).Error; err != nil {
		t.Fatalf("create household: %v", err)
	}
	creator := User{Email: "creator@example.com", PasswordHash: "hash", Role: RoleAdmin, HouseholdID: &hh.ID}
	member := User{Email: "member@example.com", PasswordHash: "hash", Role: RoleMember, HouseholdID: &hh.ID}
	if err := db.Create(&creator).Error; err != nil {
		t.Fatalf("create creator: %v", err)
	}
	if err := db.Create(&member).Error; err != nil {
		t.Fatalf("create member: %v", err)
	}

	// First Migrate run maps admin → owner.
	if err := Migrate(db); err != nil {
		t.Fatalf("first legacy Migrate failed: %v", err)
	}
	var gotCreator, gotMember User
	if err := db.First(&gotCreator, creator.ID).Error; err != nil {
		t.Fatalf("reload creator: %v", err)
	}
	if gotCreator.Role != RoleOwner {
		t.Errorf("creator role after migrate = %q, want %q", gotCreator.Role, RoleOwner)
	}
	if err := db.First(&gotMember, member.ID).Error; err != nil {
		t.Fatalf("reload member: %v", err)
	}
	if gotMember.Role != RoleMember {
		t.Errorf("member role after migrate = %q, want %q (untouched)", gotMember.Role, RoleMember)
	}

	// Second run is a no-op (idempotent).
	if err := Migrate(db); err != nil {
		t.Fatalf("second legacy Migrate failed: %v", err)
	}
	if err := db.First(&gotCreator, creator.ID).Error; err != nil {
		t.Fatalf("reload creator (2nd): %v", err)
	}
	if gotCreator.Role != RoleOwner {
		t.Errorf("creator role after second migrate = %q, want %q (idempotent)", gotCreator.Role, RoleOwner)
	}
}

func TestConnectWithDriverSQLite(t *testing.T) {
	// Test that SQLite driver works when specified
	os.Setenv("DB_DRIVER", "sqlite")
	defer os.Unsetenv("DB_DRIVER")

	db, err := ConnectWithDriver("file::memory:?cache=shared", "sqlite")
	if err != nil {
		t.Fatalf("expected no error with sqlite driver, got: %v", err)
	}
	if db == nil {
		t.Fatal("expected non-nil *gorm.DB")
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get underlying sql.DB: %v", err)
	}
	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("failed to ping database: %v", err)
	}
}

func TestConnectWithDriverInvalid(t *testing.T) {
	// Test that invalid driver returns an error
	db, err := ConnectWithDriver("file::memory:?cache=shared", "invalid_driver")
	if err == nil {
		t.Fatal("expected error for invalid driver, got nil")
	}
	if db != nil {
		t.Fatal("expected nil *gorm.DB on error")
	}
}

func TestConnectWithDriverUnsupported(t *testing.T) {
	// Test that unsupported driver returns a clear error message
	db, err := ConnectWithDriver("file::memory:?cache=shared", "mysql")
	if err == nil {
		t.Fatal("expected error for unsupported driver, got nil")
	}
	if db != nil {
		t.Fatal("expected nil *gorm.DB on error")
	}
	if err.Error() != "unsupported database driver: mysql (supported: sqlite, postgres)" {
		t.Errorf("expected clear error message, got: %v", err)
	}
}
