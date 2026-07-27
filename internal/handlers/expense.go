package handlers

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/homeadmin/internal/database"
	"github.com/homeadmin/internal/services"
)

// expenseServiceInterface defines the expense service methods needed by the handler.
type expenseServiceInterface interface {
	Create(userID, householdID uint, amount float64, description, category string, date time.Time, visibility database.VisibilityType, isFixed bool) error
	FindByHousehold(userID, householdID uint, filters database.ExpenseFilters) ([]database.Expense, error)
	Update(userID, expenseID uint, fields services.ExpenseUpdateFields) error
	Delete(userID, expenseID uint) error
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
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid amount"})
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
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid date format, use YYYY-MM-DD"})
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
		status := fiber.StatusBadRequest
		if err == services.ErrPermission {
			status = fiber.StatusForbidden
		}
		return c.Status(status).JSON(fiber.Map{"error": err.Error()})
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
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch expenses"})
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
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid expense id"})
	}

	var fields services.ExpenseUpdateFields

	if v := c.FormValue("amount"); v != "" {
		amount, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid amount"})
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
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid date format, use YYYY-MM-DD"})
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
		status := fiber.StatusBadRequest
		if err == services.ErrPermission {
			status = fiber.StatusForbidden
		}
		return c.Status(status).JSON(fiber.Map{"error": err.Error()})
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
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid expense id"})
	}

	if err := h.Service.Delete(userID, uint(expenseID)); err != nil {
		status := fiber.StatusBadRequest
		if err == services.ErrPermission {
			status = fiber.StatusForbidden
		}
		return c.Status(status).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "expense deleted",
	})
}
