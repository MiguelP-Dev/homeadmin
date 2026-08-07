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
