package pages

// monthlyNetClass colors a monthly-net value green when non-negative,
// red when negative.
func monthlyNetClass(net float64) string {
	if net >= 0 {
		return "text-green-600 dark:text-green-400"
	}
	return "text-red-600 dark:text-red-400"
}
