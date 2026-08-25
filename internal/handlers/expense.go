package handlers

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/a-h/templ"
	"github.com/gofiber/fiber/v2"
	"github.com/homeadmin/internal/database"
	"github.com/homeadmin/internal/i18n"
	"github.com/homeadmin/internal/middleware"
	"github.com/homeadmin/internal/services"
	"github.com/homeadmin/internal/templates/layouts"
	"github.com/homeadmin/internal/templates/pages"
)

// expenseServiceInterface defines the expense service methods needed by the handler.
type expenseServiceInterface interface {
	Create(userID, householdID uint, amount float64, description, category string, date time.Time, visibility database.VisibilityType, isFixed bool, txType string) error
	FindByID(userID, householdID, expenseID uint) (*database.Expense, error)
	FindByHousehold(userID, householdID uint, filters database.ExpenseFilters) ([]database.Expense, error)
	Update(userID, expenseID uint, fields services.ExpenseUpdateFields) error
	Delete(userID, expenseID uint) error
	GetDashboardSummary(userID, householdID uint) (*services.DashboardSummary, error)
}

// ExpenseHandler holds dependencies for expense HTTP handlers.
type ExpenseHandler struct {
	Service expenseServiceInterface
	// SiteAdmin is optional; when set, site admins (IsAdmin claim) get
	// cross-household global views instead of the household-scoped list.
	SiteAdmin siteOverviewService
}

// NewExpenseHandler creates a new ExpenseHandler with real service dependencies.
func NewExpenseHandler(svc expenseServiceInterface) *ExpenseHandler {
	return &ExpenseHandler{Service: svc}
}

// unprocessableKeyed re-maps a validation failure onto 422 while preserving
// its i18n Key, so the failure localizes at render time instead of being
// flattened through err.Error() (which drops the Key).
func unprocessableKeyed(err error) error {
	var appErr *middleware.AppError
	if errors.As(err, &appErr) && appErr.Key != "" {
		appErr.Status = fiber.StatusUnprocessableEntity
		return appErr
	}
	return middleware.Unprocessable(err.Error())
}

// Create handles POST /expenses — parses form data and delegates to service.
// Valid submissions redirect (303) to /expenses; invalid payloads re-render the
// form with errors at 422 without persisting (RF-3).
//
// Writes stay household-scoped even for site admins: RequireHousehold lets
// admins through to browse the global views, but an admin without a household
// keeps hitting the BadRequest below until they create or join a household.
func (h *ExpenseHandler) Create(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)
	householdID, ok := c.Locals("householdID").(*uint)
	if !ok || householdID == nil {
		return middleware.Keyed(fiber.StatusBadRequest, "expense.household_required")
	}
	hhID := *householdID

	amountStr := c.FormValue("amount")
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		return middleware.Keyed(fiber.StatusUnprocessableEntity, "expense.invalid_amount")
	}

	description := c.FormValue("description")
	category := c.FormValue("category")
	txType := c.FormValue("type")

	// Validate inputs before touching the service layer.
	if err := middleware.Validate(
		middleware.ValidateRequired(description, "description"),
		middleware.ValidateMaxLength(description, "description", 255),
		middleware.ValidatePositive(amount, "amount"),
		middleware.ValidateRequired(category, "category"),
		middleware.ValidateIn(category, "category", database.AllCategories(txType)),
	); err != nil {
		return unprocessableKeyed(err)
	}

	dateStr := c.FormValue("date")
	visibilityStr := c.FormValue("visibility")
	isFixedStr := c.FormValue("isFixed")

	var date time.Time
	if dateStr != "" {
		date, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			return middleware.Keyed(fiber.StatusUnprocessableEntity, "expense.invalid_date")
		}
	} else {
		date = time.Now()
	}

	visibility := database.VisibilityType(visibilityStr)
	if visibility == "" {
		visibility = database.VisibleEditable
	}

	isFixed := isFixedStr == "true" || isFixedStr == "1"

	if err := h.Service.Create(userID, hhID, amount, description, category, date, visibility, isFixed, txType); err != nil {
		if err == services.ErrPermission {
			return middleware.Keyed(fiber.StatusForbidden, "expense.permission")
		}
		if err == services.ErrValidation {
			return middleware.Keyed(fiber.StatusUnprocessableEntity, "expense.validation_failed")
		}
		return middleware.Internal("failed to create expense")
	}

	return c.Redirect("/expenses", fiber.StatusSeeOther)
}

// List handles GET /expenses — site admins get the global cross-household
// view; everyone else gets all visible expenses for their household.
func (h *ExpenseHandler) List(c *fiber.Ctx) error {
	if isAdmin, _ := c.Locals("isAdmin").(bool); isAdmin {
		return h.listGlobal(c)
	}

	userID := c.Locals("userID").(uint)
	householdID, ok := c.Locals("householdID").(*uint)
	if !ok || householdID == nil {
		return middleware.Keyed(fiber.StatusBadRequest, "expense.household_required")
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
	email, _ := c.Locals("email").(string)
	isAdmin, _ := c.Locals("isAdmin").(bool)
	lang, _ := c.Locals("lang").(string)
	if lang == "" {
		lang = "en"
	}
	activePath := c.Path()

	component := pages.Expenses(expenses, lang, csrfToken)
	page := layouts.Base(pageTitle(lang, "title.expenses"), csrfToken, email, isAdmin, lang, activePath)
	c.Type("html")
	ctx := templ.WithChildren(c.Context(), component)
	return page.Render(ctx, c.Response().BodyWriter())
}

// listGlobal renders the site-admin view: every household's transactions
// grouped by household with owner emails. Reachable even when the admin has
// no household (RequireHousehold bypass).
func (h *ExpenseHandler) listGlobal(c *fiber.Ctx) error {
	blocks, err := fetchSiteOverview(h.SiteAdmin)
	if err != nil {
		return middleware.Internal("failed to load site-wide expenses")
	}

	csrfToken, _ := c.Locals("csrfToken").(string)
	email, _ := c.Locals("email").(string)
	isAdmin, _ := c.Locals("isAdmin").(bool)
	lang, _ := c.Locals("lang").(string)
	if lang == "" {
		lang = "en"
	}
	activePath := c.Path()

	component := pages.ExpensesGlobal(blocks, lang)
	page := layouts.Base(pageTitle(lang, "title.expenses"), csrfToken, email, isAdmin, lang, activePath)
	c.Type("html")
	ctx := templ.WithChildren(c.Context(), component)
	return page.Render(ctx, c.Response().BodyWriter())
}

// ShowNew handles GET /expenses/new — renders the create-expense form.
func (h *ExpenseHandler) ShowNew(c *fiber.Ctx) error {
	csrfToken, _ := c.Locals("csrfToken").(string)
	email, _ := c.Locals("email").(string)
	isAdmin, _ := c.Locals("isAdmin").(bool)
	lang, _ := c.Locals("lang").(string)
	if lang == "" {
		lang = "en"
	}
	activePath := c.Path()

	component := pages.ExpenseForm(csrfToken, "/expenses", i18n.T(lang, "expenses.create"), "", pages.ExpenseFormValuesFrom(nil), lang)
	page := layouts.Base(pageTitle(lang, "expenses.create"), csrfToken, email, isAdmin, lang, activePath)
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
		return middleware.Keyed(fiber.StatusBadRequest, "expense.household_required")
	}
	hhID := *householdID

	expenseID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return middleware.BadRequest("invalid expense id")
	}

	expense, err := h.Service.FindByID(userID, hhID, uint(expenseID))
	if err != nil {
		if err == services.ErrPermission {
			return middleware.Keyed(fiber.StatusForbidden, "expense.permission")
		}
		if err == services.ErrNotFound {
			return middleware.Keyed(fiber.StatusNotFound, "expense.not_found")
		}
		return middleware.Internal("failed to load expense")
	}

	csrfToken, _ := c.Locals("csrfToken").(string)
	email, _ := c.Locals("email").(string)
	isAdmin, _ := c.Locals("isAdmin").(bool)
	lang, _ := c.Locals("lang").(string)
	if lang == "" {
		lang = "en"
	}
	activePath := c.Path()

	component := pages.ExpenseForm(csrfToken, fmt.Sprintf("/expenses/%d/update", expenseID), i18n.T(lang, "expenses.update"), "", pages.ExpenseFormValuesFrom(expense), lang)
	page := layouts.Base(pageTitle(lang, "expenses.edit"), csrfToken, email, isAdmin, lang, activePath)
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
			return middleware.Keyed(fiber.StatusUnprocessableEntity, "expense.invalid_amount")
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
			return middleware.Keyed(fiber.StatusUnprocessableEntity, "expense.invalid_date")
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
	if v := c.FormValue("type"); v != "" {
		fields.Type = &v
	}

	if err := h.Service.Update(userID, uint(expenseID), fields); err != nil {
		if err == services.ErrPermission {
			return middleware.Keyed(fiber.StatusForbidden, "expense.permission")
		}
		if err == services.ErrValidation {
			return middleware.Keyed(fiber.StatusUnprocessableEntity, "expense.validation_failed")
		}
		if err == services.ErrNotFound {
			return middleware.Keyed(fiber.StatusNotFound, "expense.not_found")
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
			return middleware.Keyed(fiber.StatusForbidden, "expense.permission")
		}
		if err == services.ErrNotFound {
			return middleware.Keyed(fiber.StatusNotFound, "expense.not_found")
		}
		return middleware.Internal("failed to delete expense")
	}

	return c.Redirect("/expenses", fiber.StatusSeeOther)
}

// Dashboard handles GET /dashboard — site admins get the global cross-household
// summary; everyone else gets their household's monthly summary via templ.
func (h *ExpenseHandler) Dashboard(c *fiber.Ctx) error {
	if isAdmin, _ := c.Locals("isAdmin").(bool); isAdmin {
		return h.dashboardGlobal(c)
	}

	userID := c.Locals("userID").(uint)
	householdID, ok := c.Locals("householdID").(*uint)
	if !ok || householdID == nil {
		return middleware.Keyed(fiber.StatusBadRequest, "expense.household_required")
	}
	hhID := *householdID

	summary, err := h.Service.GetDashboardSummary(userID, hhID)
	if err != nil {
		return middleware.Internal("failed to load dashboard")
	}

	username, _ := c.Locals("email").(string)
	viewerRole, _ := c.Locals("role").(string)
	isAdmin, _ := c.Locals("isAdmin").(bool)
	csrfToken, _ := c.Locals("csrfToken").(string)
	lang, _ := c.Locals("lang").(string)
	if lang == "" {
		lang = "en"
	}
	activePath := c.Path()

	component := pages.Dashboard(summary, viewerRole, lang, csrfToken)
	page := layouts.Base(pageTitle(lang, "title.dashboard"), csrfToken, username, isAdmin, lang, activePath)
	c.Type("html")
	ctx := templ.WithChildren(c.Context(), component)
	return page.Render(ctx, c.Response().BodyWriter())
}

// dashboardGlobal renders the site-admin dashboard: every household's
// operations grouped by household with owner emails and aggregates.
func (h *ExpenseHandler) dashboardGlobal(c *fiber.Ctx) error {
	blocks, err := fetchSiteOverview(h.SiteAdmin)
	if err != nil {
		return middleware.Internal("failed to load site-wide dashboard")
	}

	username, _ := c.Locals("email").(string)
	isAdmin, _ := c.Locals("isAdmin").(bool)
	csrfToken, _ := c.Locals("csrfToken").(string)
	lang, _ := c.Locals("lang").(string)
	if lang == "" {
		lang = "en"
	}
	activePath := c.Path()

	component := pages.DashboardGlobal(blocks, lang)
	page := layouts.Base(pageTitle(lang, "title.dashboard"), csrfToken, username, isAdmin, lang, activePath)
	c.Type("html")
	ctx := templ.WithChildren(c.Context(), component)
	return page.Render(ctx, c.Response().BodyWriter())
}
