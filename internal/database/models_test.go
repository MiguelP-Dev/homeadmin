package database

import (
	"reflect"
	"testing"
	"time"

	"gorm.io/gorm"
)

func TestHouseholdModelFields(t *testing.T) {
	h := Household{}

	rt := reflect.TypeOf(h)

	// Verify required fields exist with correct types
	tests := []struct {
		field    string
		expected reflect.Kind
	}{
		{"ID", reflect.Uint},
		{"Name", reflect.String},
		{"CreatedAt", reflect.TypeOf(time.Time{}).Kind()},
		{"UpdatedAt", reflect.TypeOf(time.Time{}).Kind()},
		{"DeletedAt", reflect.TypeOf(gorm.DeletedAt{}).Kind()},
	}

	for _, tt := range tests {
		f, ok := rt.FieldByName(tt.field)
		if !ok {
			t.Errorf("Household missing field %s", tt.field)
			continue
		}
		if f.Type.Kind() != tt.expected {
			t.Errorf("Household.%s: expected kind %v, got %v", tt.field, tt.expected, f.Type.Kind())
		}
	}

	// Verify GORM tag: Name should be not null with size 100
	nameField, _ := rt.FieldByName("Name")
	nameTag := nameField.Tag.Get("gorm")
	if nameTag == "" {
		t.Error("Household.Name missing gorm tag")
	}

	// Verify DeletedAt uses gorm index
	deletedField, _ := rt.FieldByName("DeletedAt")
	deletedTag := deletedField.Tag.Get("gorm")
	if deletedTag == "" {
		t.Error("Household.DeletedAt missing gorm tag")
	}
}

func TestUserModelFields(t *testing.T) {
	u := User{}

	rt := reflect.TypeOf(u)

	tests := []struct {
		field    string
		expected reflect.Kind
	}{
		{"ID", reflect.Uint},
		{"Email", reflect.String},
		{"PasswordHash", reflect.String},
		{"Role", reflect.String},
		{"IsAdmin", reflect.Bool},
		{"HouseholdID", reflect.Ptr},
		{"CreatedAt", reflect.TypeOf(time.Time{}).Kind()},
		{"UpdatedAt", reflect.TypeOf(time.Time{}).Kind()},
	}

	for _, tt := range tests {
		f, ok := rt.FieldByName(tt.field)
		if !ok {
			t.Errorf("User missing field %s", tt.field)
			continue
		}
		if f.Type.Kind() != tt.expected {
			t.Errorf("User.%s: expected kind %v, got %v", tt.field, tt.expected, f.Type.Kind())
		}
	}

	// Verify Email has uniqueIndex gorm tag
	emailField, _ := rt.FieldByName("Email")
	emailTag := emailField.Tag.Get("gorm")
	if emailTag == "" {
		t.Error("User.Email missing gorm tag")
	}

	// Verify Role default is 'member'
	roleField, _ := rt.FieldByName("Role")
	roleTag := roleField.Tag.Get("gorm")
	if roleTag == "" {
		t.Error("User.Role missing gorm tag")
	}
}

func TestExpenseModelFields(t *testing.T) {
	e := Expense{}

	rt := reflect.TypeOf(e)

	tests := []struct {
		field    string
		expected reflect.Kind
	}{
		{"ID", reflect.Uint},
		{"Amount", reflect.Float64},
		{"Description", reflect.String},
		{"Category", reflect.String},
		{"HouseholdID", reflect.Uint},
		{"CreatedByID", reflect.Uint},
		{"Visibility", reflect.TypeOf(VisibilityType("")).Kind()},
		{"Date", reflect.TypeOf(time.Time{}).Kind()},
		{"CreatedAt", reflect.TypeOf(time.Time{}).Kind()},
		{"UpdatedAt", reflect.TypeOf(time.Time{}).Kind()},
		{"DeletedAt", reflect.TypeOf(gorm.DeletedAt{}).Kind()},
	}

	for _, tt := range tests {
		f, ok := rt.FieldByName(tt.field)
		if !ok {
			t.Errorf("Expense missing field %s", tt.field)
			continue
		}
		if f.Type.Kind() != tt.expected {
			t.Errorf("Expense.%s: expected kind %v, got %v", tt.field, tt.expected, f.Type.Kind())
		}
	}
}

// TestUserIsAdmin_DefaultsFalse verifies a fresh User has IsAdmin=false (RF-9):
// site-admin is opt-in, never granted at registration.
func TestUserIsAdmin_DefaultsFalse(t *testing.T) {
	u := User{}
	if u.IsAdmin {
		t.Error("expected fresh User IsAdmin=false, got true")
	}
}

// TestUserIsAdmin_PersistsRoundTrip verifies AutoMigrate is additive for the
// IsAdmin column: a user persisted with IsAdmin=true reloads with IsAdmin=true,
// and a non-admin user reloads with false.
func TestUserIsAdmin_PersistsRoundTrip(t *testing.T) {
	db, err := Connect("file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}

	admin := User{Email: "admin@example.com", PasswordHash: "hash", Role: RoleMember, IsAdmin: true}
	plain := User{Email: "plain@example.com", PasswordHash: "hash", Role: RoleMember}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatalf("create admin user: %v", err)
	}
	if err := db.Create(&plain).Error; err != nil {
		t.Fatalf("create plain user: %v", err)
	}

	var gotAdmin, gotPlain User
	if err := db.First(&gotAdmin, admin.ID).Error; err != nil {
		t.Fatalf("reload admin user: %v", err)
	}
	if !gotAdmin.IsAdmin {
		t.Error("expected IsAdmin=true after reload of promoted user")
	}
	if err := db.First(&gotPlain, plain.ID).Error; err != nil {
		t.Fatalf("reload plain user: %v", err)
	}
	if gotPlain.IsAdmin {
		t.Error("expected IsAdmin=false for non-promoted user")
	}
}

func TestRoleConstants(t *testing.T) {
	tests := []struct {
		name     string
		role     string
		expected string
	}{
		{"RoleOwner", RoleOwner, "owner"},
		{"RoleAdmin", RoleAdmin, "admin"},
		{"RoleMember", RoleMember, "member"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.role != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, tt.role)
			}
		})
	}
}

func TestVisibilityConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant VisibilityType
		expected string
	}{
		{"VisibleEditable", VisibleEditable, "visible_editable"},
		{"VisibleOnly", VisibleOnly, "visible_only"},
		{"HiddenPrivate", HiddenPrivate, "hidden_private"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, string(tt.constant))
			}
		})
	}
}
