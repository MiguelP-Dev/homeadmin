package handlers

import (
	"fmt"
	"strconv"
	"time"

	"github.com/a-h/templ"
	"github.com/gofiber/fiber/v2"
	"github.com/homeadmin/internal/database"
	"github.com/homeadmin/internal/middleware"
	"github.com/homeadmin/internal/services"
	"github.com/homeadmin/internal/templates/layouts"
	"github.com/homeadmin/internal/templates/pages"
)

// expenseServiceInterface defines the expense service methods needed by the handler.
type expenseServiceInterface interface {
	Create(userID, householdID uint, amount float64, description, category string, date time.Time, visibility database.VisibilityType, isFixed bool) error
	FindByID(userID, householdID, expenseID uint) (*database.Expense, error)
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
// Valid submissions redirect (303) to /expenses; invalid payloads re-render the
// form with errors at 422 without persisting (RF-3).
func (h *ExpenseHandler) Create(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)
	householdID, ok := c.Locals("householdID").(*uint)
	if !ok || householdID == nil {
		return middleware.BadRequest("household required")
	}
	hhID := *householdID

	amountStr := c.FormValue("amount")
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		return middleware.Unprocessable("invalid amount")
	}

	description := c.FormValue("description")
	category := c.FormValue("category")

	// Validate inputs before touching the service layer.
	if err := middleware.Validate(
		middleware.ValidateRequired(description, "description"),
		middleware.ValidateMaxLength(description, "description", 255),
		middleware.ValidatePositive(amount, "amount"),
		middleware.ValidateRequired(category, "category"),
		middleware.ValidateIn(category, "category", database.ExpenseCategories),
	); err != nil {
		return middleware.Unprocessable(err.Error())
	}

	dateStr := c.FormValue("date")
	visibilityStr := c.FormValue("visibility")
	isFixedStr := c.FormValue("isFixed")

	var date time.Time
	if dateStr != "" {
		date, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			return middleware.Unprocessable("invalid date format, use YYYY-MM-DD")
		}
	} else {
		date = time.Now()
	}

	visibility := database.VisibilityType(visibilityStr)
	if visibility == "" {
		visibility = database.VisibleEditable
	}

	isFixed := isFixedStr == "true" || isFixedStr == "1"

	if err := h.Service.Create(userID, hhID, amount, description, category, date, visibility, isFixed); err != nil {
		if err == services.ErrPermission {
			return middleware.Forbidden(err.Error())
		}
		if err == services.ErrValidation {
			return middleware.Unprocessable(err.Error())
		}
		return middleware.Internal("failed to create expense")
	}

	return c.Redirect("/expenses", fiber.StatusSeeOther)
}

// List handles GET /expenses — returns all visible expenses for the household.
func (h *ExpenseHandler) List(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)
	householdID, ok := c.Locals("householdID").(*uint)
	if !ok || householdID == nil {
		return middleware.BadRequest("household required")
	}
	hhID := *householdID

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

	expenses, err := h.Service.FindByHousehold(userID, hhID, filters)
	if err != nil {
		return middleware.Internal("failed to fetch expenses")
	}

	csrfToken, _ := c.Locals("csrfToken").(string)
	username, _ := c.Locals("email").(string)

	component := pages.Expenses(expenses)
	page := layouts.Base("Expenses — HomeAdmin", csrfToken, username)
	c.Type("html")
	ctx := templ.WithChildren(c.Context(), component)
	return page.Render(ctx, c.Response().BodyWriter())
}

// ShowNew handles GET /expenses/new — renders the create-expense form.
func (h *ExpenseHandler) ShowNew(c *fiber.Ctx) error {
	csrfToken, _ := c.Locals("csrfToken").(string)
	username, _ := c.Locals("email").(string)

	component := pages.ExpenseForm(csrfToken, "/expenses", "Create Expense", "", pages.ExpenseFormValuesFrom(nil))
	page := layouts.Base("Create Expense — HomeAdmin", csrfToken, username)
	c.Type("html")
	ctx := templ.WithChildren(c.Context(), component)
	return page.Render(ctx, c.Response().BodyWriter())
}

// ShowEdit handles GET /expenses/:id/edit — renders the edit form for an
// expense the user may view within their household.
func (h *ExpenseHandler) ShowEdit(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)
	householdID, ok := c.Locals("householdID").(*uint)
	if !ok || householdID == nil {
		return middleware.BadRequest("household required")
	}
	hhID := *householdID

	expenseID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return middleware.BadRequest("invalid expense id")
	}

	expense, err := h.Service.FindByID(userID, hhID, uint(expenseID))
	if err != nil {
		if err == services.ErrPermission {
			return middleware.Forbidden(err.Error())
		}
		if err == services.ErrNotFound {
			return middleware.NotFound("expense not found")
		}
		return middleware.Internal("failed to load expense")
	}

	csrfToken, _ := c.Locals("csrfToken").(string)
	username, _ := c.Locals("email").(string)

	component := pages.ExpenseForm(csrfToken, fmt.Sprintf("/expenses/%d/update", expenseID), "Update Expense", "", pages.ExpenseFormValuesFrom(expense))
	page := layouts.Base("Edit Expense — HomeAdmin", csrfToken, username)
	c.Type("html")
	ctx := templ.WithChildren(c.Context(), component)
	return page.Render(ctx, c.Response().BodyWriter())
}

// Update handles POST /expenses/:id/update — applies field changes to an
// expense. Success redirects (303) to /expenses; invalid payloads re-render
// with errors at 422 (RF-4a).
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
			return middleware.Unprocessable("invalid amount")
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
			return middleware.Unprocessable("invalid date format, use YYYY-MM-DD")
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
		if err == services.ErrValidation {
			return middleware.Unprocessable(err.Error())
		}
		if err == services.ErrNotFound {
			return middleware.NotFound("expense not found")
		}
		return middleware.Internal("failed to update expense")
	}

	return c.Redirect("/expenses", fiber.StatusSeeOther)
}

// Delete handles POST /expenses/:id/delete — removes an expense if the user is
// the creator. Success redirects (303) to /expenses (RF-4b).
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
		if err == services.ErrNotFound {
			return middleware.NotFound("expense not found")
		}
		return middleware.Internal("failed to delete expense")
	}

	return c.Redirect("/expenses", fiber.StatusSeeOther)
}

// Dashboard handles GET /dashboard — returns an HTML page with monthly summary via templ.
func (h *ExpenseHandler) Dashboard(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)
	householdID, ok := c.Locals("householdID").(*uint)
	if !ok || householdID == nil {
		return middleware.BadRequest("household required")
	}
	hhID := *householdID

	summary, err := h.Service.GetDashboardSummary(userID, hhID)
	if err != nil {
		return middleware.Internal("failed to load dashboard")
	}

	username, _ := c.Locals("email").(string)
	csrfToken, _ := c.Locals("csrfToken").(string)

	component := pages.Dashboard(summary)
	page := layouts.Base("Dashboard — HomeAdmin", csrfToken, username)
	c.Type("html")
	ctx := templ.WithChildren(c.Context(), component)
	return page.Render(ctx, c.Response().BodyWriter())
}
