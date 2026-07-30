package components

import "github.com/homeadmin/internal/database"

// visibilityBadgeClass returns Tailwind CSS classes for the expense visibility badge.
func visibilityBadgeClass(visibility database.VisibilityType) string {
	switch visibility {
	case database.HiddenPrivate:
		return "bg-red-100 text-red-700"
	case database.VisibleOnly:
		return "bg-yellow-100 text-yellow-700"
	default:
		return "bg-green-100 text-green-700"
	}
}
