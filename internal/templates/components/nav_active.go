package components

import "strings"

// IsActive reports whether the current request path belongs to a nav section.
// /dashboard matches exactly; section links match by prefix so nested routes
// like /expenses/new keep their section highlighted.
func IsActive(activePath string, section string) bool {
	if activePath == section {
		return true
	}
	return section != "/dashboard" && strings.HasPrefix(activePath, section+"/")
}
