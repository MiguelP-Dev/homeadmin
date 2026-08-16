package database

// ExpenseCategories is the fixed list of allowed expense categories.
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

// IncomeCategories is the fixed list of allowed income categories.
var IncomeCategories = []string{
	"Freelance",
	"Gifts",
	"Investments",
	"Other Income",
	"Refunds",
	"Rental Income",
	"Salary",
	"Side Hustle",
}

// AllCategories returns categories for a given transaction type.
func AllCategories(txType string) []string {
	if txType == "income" {
		return IncomeCategories
	}
	return ExpenseCategories
}

// IsValidCategory checks if a category is in the fixed list.
func IsValidCategory(category string) bool {
	for _, c := range ExpenseCategories {
		if c == category {
			return true
		}
	}
	for _, c := range IncomeCategories {
		if c == category {
			return true
		}
	}
	return false
}
