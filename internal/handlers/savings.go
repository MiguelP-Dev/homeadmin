package handlers

import (
	"strconv"

	"github.com/a-h/templ"
	"github.com/gofiber/fiber/v2"
	"github.com/homeadmin/internal/middleware"
	"github.com/homeadmin/internal/services"
	"github.com/homeadmin/internal/templates/layouts"
	"github.com/homeadmin/internal/templates/pages"
)

// SavingsHandler holds dependencies for savings HTTP handlers.
type SavingsHandler struct {
	Service *services.SavingsService
}

// NewSavingsHandler creates a new SavingsHandler.
func NewSavingsHandler(svc *services.SavingsService) *SavingsHandler {
	return &SavingsHandler{Service: svc}
}

// List handles GET /savings — returns all savings for the household.
func (h *SavingsHandler) List(c *fiber.Ctx) error {
	householdID, ok := c.Locals("householdID").(*uint)
	if !ok || householdID == nil {
		return middleware.BadRequest("household required")
	}
	hhID := *householdID

	savings, err := h.Service.FindByHousehold(hhID)
	if err != nil {
		return middleware.Internal("failed to fetch savings")
	}

	total, err := h.Service.GetTotal(hhID)
	if err != nil {
		return middleware.Internal("failed to compute savings total")
	}

	csrfToken, _ := c.Locals("csrfToken").(string)
	email, _ := c.Locals("email").(string)
	isAdmin, _ := c.Locals("isAdmin").(bool)
	lang, _ := c.Locals("lang").(string)
	if lang == "" {
		lang = "en"
	}
	activePath := c.Path()

	component := pages.Savings(savings, total, lang, csrfToken)
	page := layouts.Base("Savings — HomeAdmin", csrfToken, email, isAdmin, lang, activePath)
	c.Type("html")
	ctx := templ.WithChildren(c.Context(), component)
	return page.Render(ctx, c.Response().BodyWriter())
}

// ShowNew handles GET /savings/new — renders the create-savings form.
func (h *SavingsHandler) ShowNew(c *fiber.Ctx) error {
	csrfToken, _ := c.Locals("csrfToken").(string)
	email, _ := c.Locals("email").(string)
	isAdmin, _ := c.Locals("isAdmin").(bool)
	lang, _ := c.Locals("lang").(string)
	if lang == "" {
		lang = "en"
	}
	activePath := c.Path()

	component := pages.SavingsForm(csrfToken, "/savings", "Create Savings", "", lang)
	page := layouts.Base("Create Savings — HomeAdmin", csrfToken, email, isAdmin, lang, activePath)
	c.Type("html")
	ctx := templ.WithChildren(c.Context(), component)
	return page.Render(ctx, c.Response().BodyWriter())
}

// Create handles POST /savings — creates a new savings entry.
func (h *SavingsHandler) Create(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)
	householdID, ok := c.Locals("householdID").(*uint)
	if !ok || householdID == nil {
		return middleware.BadRequest("household required")
	}
	hhID := *householdID

	description := c.FormValue("description")
	amountStr := c.FormValue("amount")
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		return middleware.Unprocessable("invalid amount")
	}

	targetStr := c.FormValue("target")
	var target float64
	if targetStr != "" {
		target, err = strconv.ParseFloat(targetStr, 64)
		if err != nil {
			return middleware.Unprocessable("invalid target")
		}
	}

	if err := h.Service.Create(userID, hhID, description, amount, target); err != nil {
		if err == services.ErrValidation {
			return middleware.Unprocessable(err.Error())
		}
		return middleware.Internal("failed to create savings")
	}

	return c.Redirect("/savings", fiber.StatusSeeOther)
}

// Delete handles POST /savings/:id/delete — removes a savings entry.
func (h *SavingsHandler) Delete(c *fiber.Ctx) error {
	householdID, ok := c.Locals("householdID").(*uint)
	if !ok || householdID == nil {
		return middleware.BadRequest("household required")
	}
	hhID := *householdID

	savingsID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return middleware.BadRequest("invalid savings id")
	}

	if err := h.Service.Delete(hhID, uint(savingsID)); err != nil {
		if err == services.ErrPermission {
			return middleware.Forbidden(err.Error())
		}
		if err == services.ErrNotFound {
			return middleware.NotFound("savings not found")
		}
		return middleware.Internal("failed to delete savings")
	}

	return c.Redirect("/savings", fiber.StatusSeeOther)
}
