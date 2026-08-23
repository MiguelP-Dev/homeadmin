package components

import "github.com/homeadmin/internal/database"

// RoleBadgeClass returns Tailwind CSS classes for a household role badge chip.
func RoleBadgeClass(role string) string {
	switch role {
	case database.RoleOwner:
		return "bg-blue-100 text-blue-700"
	case database.RoleAdmin:
		return "bg-purple-100 text-purple-700"
	default:
		return "bg-gray-200 text-gray-700"
	}
}

// netBadgeClass colors the all-time net pill by sign.
func netBadgeClass(net float64) string {
	if net >= 0 {
		return "bg-green-100 text-green-700"
	}
	return "bg-red-100 text-red-700"
}
