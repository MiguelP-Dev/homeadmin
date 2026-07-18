package database

import (
	"sort"
	"testing"
)

func TestCategoryCount(t *testing.T) {
	// spec §6.2: 13 fixed expense categories
	expected := 13
	if len(ExpenseCategories) != expected {
		t.Errorf("expected %d categories, got %d", expected, len(ExpenseCategories))
	}
}

func TestIsValidCategory(t *testing.T) {
	// spec §1.20–1.21: valid categories return true, invalid return false
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"Rent is valid", "Rent", true},
		{"Utilities is valid", "Utilities", true},
		{"Groceries is valid", "Groceries", true},
		{"Dining Out is valid", "Dining Out", true},
		{"Transportation is valid", "Transportation", true},
		{"Entertainment is valid", "Entertainment", true},
		{"Subscriptions is valid", "Subscriptions", true},
		{"Insurance is valid", "Insurance", true},
		{"Household is valid", "Household", true},
		{"Personal Care is valid", "Personal Care", true},
		{"Education is valid", "Education", true},
		{"Savings is valid", "Savings", true},
		{"Other is valid", "Other", true},
		{"empty string is invalid", "", false},
		{"lowercase rent is invalid", "rent", false},
		{"unknown category is invalid", "Crypto", false},
		{"partial match is invalid", "Rent ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidCategory(tt.input)
			if got != tt.expected {
				t.Errorf("IsValidCategory(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestCategoriesAreSorted(t *testing.T) {
	// spec §1.21 (optional, good to have): categories are alphabetically sorted
	if !sort.StringsAreSorted(ExpenseCategories) {
		t.Errorf("ExpenseCategories is not sorted alphabetically; got: %v", ExpenseCategories)
	}
}
