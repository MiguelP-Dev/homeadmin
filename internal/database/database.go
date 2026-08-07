package database

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Connect opens a GORM connection with the given DSN and configures the connection pool.
// Uses the default sqlite driver for backward compatibility.
func Connect(dsn string) (*gorm.DB, error) {
	return ConnectWithDriver(dsn, "sqlite")
}

// ConnectWithDriver opens a GORM connection using the specified driver and configures the connection pool.
// Supported drivers: "sqlite", "postgres".
func ConnectWithDriver(dsn string, driver string) (*gorm.DB, error) {
	var dialector gorm.Dialector

	switch driver {
	case "sqlite":
		dialector = sqlite.Open(dsn)
	case "postgres":
		dialector = postgres.Open(dsn)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s (supported: sqlite, postgres)", driver)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	// Connection pool configuration per spec §3.3
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)

	return db, nil
}

// Migrate runs AutoMigrate for all application models, then applies data
// migrations (migrateLegacyRoles).
func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&Household{}, &User{}, &Expense{}, &InviteCode{}); err != nil {
		return err
	}
	return migrateLegacyRoles(db)
}

// migrateLegacyRoles maps the legacy household role "admin" (previously held
// only by household creators) to the three-tier role "owner" (RF-7). It is
// idempotent: once every admin row has been mapped, a re-run updates zero rows.
func migrateLegacyRoles(db *gorm.DB) error {
	return db.Model(&User{}).Where("role = ?", RoleAdmin).Update("role", RoleOwner).Error
}
