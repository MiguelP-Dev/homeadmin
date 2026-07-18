package database

// ExpenseCategories is the fixed list of allowed categories per spec §6.2.
// Kept alphabetically sorted.
var ExpenseCategories = []string{
	"Dining Out",
	"Education",
	"Entertainment",
	"Groceries",
	"Household",
	"Insurance",
	"Other",
	"Personal Care",
	"Rent",
	"Savings",
	"Subscriptions",
	"Transportation",
	"Utilities",
}

// IsValidCategory checks if a category is in the fixed list.
func IsValidCategory(category string) bool {
	for _, c := range ExpenseCategories {
		if c == category {
			return true
		}
	}
	return false
}
