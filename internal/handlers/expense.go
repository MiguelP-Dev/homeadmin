package handlers

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/homeadmin/internal/database"
	"github.com/homeadmin/internal/middleware"
	"github.com/homeadmin/internal/services"
)

// expenseServiceInterface defines the expense service methods needed by the handler.
type expenseServiceInterface interface {
	Create(userID, householdID uint, amount float64, description, category string, date time.Time, visibility database.VisibilityType, isFixed bool) error
	FindByHousehold(userID, householdID uint, filters database.ExpenseFilters) ([]database.Expense, error)
	Update(userID, expenseID uint, fields services.ExpenseUpdateFields) error
	Delete(userID, expenseID uint) error
	GetDashboardSummary(userID, householdID uint) (*services.DashboardSummary, error)
}

// ExpenseHandler holds dependencies for expense HTTP handlers.
type ExpenseHandler struct {
	Service expenseServiceInterface
}

// NewExpenseHandler creates a new ExpenseHandler with real service dependencies.
func NewExpenseHandler(svc expenseServiceInterface) *ExpenseHandler {
	return &ExpenseHandler{Service: svc}
}

// Create handles POST /expenses — parses form data and delegates to service.
func (h *ExpenseHandler) Create(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)
	householdID := c.Locals("householdID").(uint)

	amountStr := c.FormValue("amount")
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		return middleware.BadRequest("invalid amount")
	}

	description := c.FormValue("description")
	category := c.FormValue("category")
	dateStr := c.FormValue("date")
	visibilityStr := c.FormValue("visibility")
	isFixedStr := c.FormValue("isFixed")

	var date time.Time
	if dateStr != "" {
		date, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			return middleware.BadRequest("invalid date format, use YYYY-MM-DD")
		}
	} else {
		date = time.Now()
	}

	visibility := database.VisibilityType(visibilityStr)
	if visibility == "" {
		visibility = database.VisibleEditable
	}

	isFixed := isFixedStr == "true" || isFixedStr == "1"

	if err := h.Service.Create(userID, householdID, amount, description, category, date, visibility, isFixed); err != nil {
		if err == services.ErrPermission {
			return middleware.Forbidden(err.Error())
		}
		return middleware.BadRequest(err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "expense created",
	})
}

// List handles GET /expenses — returns all visible expenses for the household.
func (h *ExpenseHandler) List(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)
	householdID := c.Locals("householdID").(uint)

	filters := database.ExpenseFilters{
		Category: c.Query("category"),
	}
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			filters.Limit = l
		}
	}
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil {
			filters.Offset = o
		}
	}

	expenses, err := h.Service.FindByHousehold(userID, householdID, filters)
	if err != nil {
		return middleware.Internal("failed to fetch expenses")
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"expenses": expenses,
	})
}

// Update handles PUT /expenses/:id — applies field changes to an expense.
func (h *ExpenseHandler) Update(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)

	expenseID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return middleware.BadRequest("invalid expense id")
	}

	var fields services.ExpenseUpdateFields

	if v := c.FormValue("amount"); v != "" {
		amount, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return middleware.BadRequest("invalid amount")
		}
		fields.Amount = &amount
	}
	if v := c.FormValue("description"); v != "" {
		fields.Description = &v
	}
	if v := c.FormValue("category"); v != "" {
		fields.Category = &v
	}
	if v := c.FormValue("date"); v != "" {
		date, err := time.Parse("2006-01-02", v)
		if err != nil {
			return middleware.BadRequest("invalid date format, use YYYY-MM-DD")
		}
		fields.Date = &date
	}
	if v := c.FormValue("visibility"); v != "" {
		vis := database.VisibilityType(v)
		fields.Visibility = &vis
	}
	if v := c.FormValue("isFixed"); v != "" {
		isFixed := v == "true" || v == "1"
		fields.IsFixed = &isFixed
	}

	if err := h.Service.Update(userID, uint(expenseID), fields); err != nil {
		if err == services.ErrPermission {
			return middleware.Forbidden(err.Error())
		}
		return middleware.BadRequest(err.Error())
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "expense updated",
	})
}

// Delete handles DELETE /expenses/:id — removes an expense if the user is the creator.
func (h *ExpenseHandler) Delete(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)

	expenseID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return middleware.BadRequest("invalid expense id")
	}

	if err := h.Service.Delete(userID, uint(expenseID)); err != nil {
		if err == services.ErrPermission {
			return middleware.Forbidden(err.Error())
		}
		return middleware.BadRequest(err.Error())
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "expense deleted",
	})
}

// Dashboard handles GET /dashboard — returns an HTML page with monthly summary.
func (h *ExpenseHandler) Dashboard(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)
	householdID := c.Locals("householdID").(uint)

	summary, err := h.Service.GetDashboardSummary(userID, householdID)
	if err != nil {
		return middleware.Internal("failed to load dashboard")
	}

	return c.Status(fiber.StatusOK).SendString(dashboardHTML(summary))
}

// dashboardHTML renders the dashboard as a complete HTML page.
func dashboardHTML(summary *services.DashboardSummary) string {
	now := time.Now()
	monthName := now.Format("January 2006")

	// Category breakdown rows
	catRows := ""
	for _, ct := range summary.CategoryTotals {
		catRows += fmt.Sprintf("<tr><td>%s</td><td>$%.2f</td></tr>\n", ct.Category, ct.Total)
	}
	if catRows == "" {
		catRows = "<tr><td colspan=\"2\">No expenses this month</td></tr>\n"
	}

	// Recent expenses list
	recentItems := ""
	for _, e := range summary.RecentExpenses {
		recentItems += fmt.Sprintf(
			"<li><strong>%s</strong> — $%.2f (%s) <em>%s</em></li>\n",
			e.Description, e.Amount, e.Category, e.Date.Format("Jan 2"),
		)
	}
	if recentItems == "" {
		recentItems = "<li>No recent expenses</li>\n"
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Dashboard — HomeAdmin</title>
    <style>
        body { font-family: system-ui, sans-serif; max-width: 800px; margin: 2rem auto; padding: 0 1rem; color: #333; }
        h1 { border-bottom: 2px solid #e0e0e0; padding-bottom: 0.5rem; }
        .total { font-size: 2rem; font-weight: bold; color: #2d6a4f; margin: 1rem 0; }
        table { width: 100%%; border-collapse: collapse; margin: 1rem 0; }
        th, td { text-align: left; padding: 0.5rem; border-bottom: 1px solid #e0e0e0; }
        th { background: #f5f5f5; }
        ul { list-style: none; padding: 0; }
        li { padding: 0.4rem 0; border-bottom: 1px solid #f0f0f0; }
        a { color: #2d6a4f; text-decoration: none; }
        a:hover { text-decoration: underline; }
    </style>
</head>
<body>
    <h1>Dashboard — %s</h1>
    <div class="total">Monthly Total: $%.2f</div>
    <h2>Category Breakdown</h2>
    <table>
        <thead><tr><th>Category</th><th>Total</th></tr></thead>
        <tbody>%s</tbody>
    </table>
    <h2>Recent Expenses</h2>
    <ul>%s</ul>
    <p><a href="/expenses">&larr; Back to Expenses</a></p>
</body>
</html>`, monthName, summary.MonthlyTotal, catRows, recentItems)
}
