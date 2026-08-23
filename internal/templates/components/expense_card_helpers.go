package components

import "github.com/homeadmin/internal/database"

// visibilityBadgeClass returns Tailwind CSS classes for the expense visibility badge.
func visibilityBadgeClass(visibility database.VisibilityType) string {
	switch visibility {
	case database.HiddenPrivate:
		return "bg-red-100 text-red-700 dark:bg-red-900 dark:text-red-300"
	case database.VisibleOnly:
		return "bg-yellow-100 text-yellow-700 dark:bg-yellow-900 dark:text-yellow-300"
	default:
		return "bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300"
	}
}

// TypeBadgeClass returns Tailwind CSS classes for the transaction type badge.
func TypeBadgeClass(txType string) string {
	switch txType {
	case database.TransactionTypeIncome:
		return "bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300"
	default:
		return "bg-red-100 text-red-700 dark:bg-red-900 dark:text-red-300"
	}
}
