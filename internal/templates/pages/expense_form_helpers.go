package pages

import (
	"fmt"

	"github.com/homeadmin/internal/database"
)

// ExpenseFormValues carries the display values pre-filled into the expense
// form. The zero value renders an empty create form with the default
// visible_editable visibility.
type ExpenseFormValues struct {
	Amount      string
	Description string
	Category    string
	Date        string
	Visibility  database.VisibilityType
	IsFixed     bool
}

// ExpenseFormValuesFrom converts an existing expense into form values; a nil
// expense yields the empty create-form defaults.
func ExpenseFormValuesFrom(expense *database.Expense) ExpenseFormValues {
	values := ExpenseFormValues{Visibility: database.VisibleEditable}
	if expense == nil {
		return values
	}
	values.Amount = fmt.Sprintf("%.2f", expense.Amount)
	values.Description = expense.Description
	values.Category = expense.Category
	values.Date = expense.Date.Format("2006-01-02")
	values.Visibility = expense.Visibility
	values.IsFixed = expense.IsFixed
	return values
}
