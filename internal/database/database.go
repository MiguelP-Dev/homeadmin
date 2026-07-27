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

// Migrate runs AutoMigrate for all application models.
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&Household{}, &User{}, &Expense{})
}
