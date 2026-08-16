package pages

import "fmt"

// uintToString converts a uint to its string representation for use in templ URLs.
func uintToString(v uint) string {
	return fmt.Sprintf("%d", v)
}
