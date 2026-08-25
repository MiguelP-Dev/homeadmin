package components

import (
	"github.com/a-h/templ"
	"github.com/homeadmin/internal/database"
)

// confirmAttrs builds the translated onclick confirmation dialog for delete
// forms. Spread the result into the element tag with
// `{ confirmAttrs(msg)... }` so templ renders a real onclick attribute.
// Mirrors pages.confirmAttrs (kept local: pages cannot be imported from
// components without an import cycle).
func confirmAttrs(msg string) templ.Attributes {
	return templ.Attributes(map[string]any{"onclick": "return confirm('" + msg + "')"})
}

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
