package database

import (
	"time"

	"gorm.io/gorm"
)

// InviteCode represents a single-use invite token for household join flows.
type InviteCode struct {
	ID          uint      `gorm:"primaryKey"`
	Code        string    `gorm:"size:8;uniqueIndex;not null"`
	HouseholdID uint      `gorm:"not null"`
	ExpiresAt   time.Time `gorm:"not null"`
	UsedBy      *uint     `gorm:"default:null"`
	CreatedAt   time.Time
}

// Household represents a shared group (family, roommates)
type Household struct {
	ID        uint           `gorm:"primaryKey"`
	Name      string         `gorm:"size:100;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// Household roles for User.Role (three-tier: owner / admin / member).
const (
	RoleOwner  = "owner"
	RoleAdmin  = "admin"
	RoleMember = "member"
)

// User represents an authenticated user
type User struct {
	ID           uint           `gorm:"primaryKey"`
	Email        string         `gorm:"size:255;uniqueIndex;not null"`
	PasswordHash string         `gorm:"size:255;not null"`
	Role         string         `gorm:"size:20;default:'member'"`
	HouseholdID  *uint          `gorm:"default:null"`
	Household    *Household     `gorm:"foreignKey:HouseholdID"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// VisibilityType controls who can see/edit an expense
type VisibilityType string

const (
	VisibleEditable VisibilityType = "visible_editable"
	VisibleOnly     VisibilityType = "visible_only"
	HiddenPrivate   VisibilityType = "hidden_private"
)

// Expense tracks fixed and variable payments
type Expense struct {
	ID          uint           `gorm:"primaryKey"`
	Amount      float64        `gorm:"type:decimal(10,2);not null"`
	Description string         `gorm:"size:255;not null"`
	Category    string         `gorm:"size:50;not null"`
	HouseholdID uint           `gorm:"not null"`
	Household   Household      `gorm:"foreignKey:HouseholdID"`
	CreatedByID uint           `gorm:"not null"`
	CreatedBy   User           `gorm:"foreignKey:CreatedByID"`
	Visibility  VisibilityType `gorm:"type:varchar(20);default:'visible_editable'"`
	IsFixed     bool           `gorm:"default:false"`
	Date        time.Time      `gorm:"not null"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

// ExpenseFilters defines query filters for listing expenses.
type ExpenseFilters struct {
	Category string
	Limit    int
	Offset   int
}
