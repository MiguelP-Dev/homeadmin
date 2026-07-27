package repositories

import (
	"path/filepath"
	"testing"

	"github.com/homeadmin/internal/database"
	"gorm.io/gorm"
)

// setupTestDBRaw creates an isolated SQLite database per test,
// runs migrations, and returns the raw *gorm.DB for use across multiple repositories.
func setupTestDBRaw(t *testing.T) *gorm.DB {
	t.Helper()
	// Each test gets its own file-based DB in a temp directory for full isolation
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := database.Connect(dbPath)
	if err != nil {
		t.Fatalf("failed to connect to test db: %v", err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatalf("failed to migrate test db: %v", err)
	}
	return db
}
