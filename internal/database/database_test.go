package database

import (
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
