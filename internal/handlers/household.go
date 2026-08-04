package handlers

import (
	"errors"
	"fmt"

	"github.com/a-h/templ"
	"github.com/gofiber/fiber/v2"
	"github.com/homeadmin/internal/database"
	"github.com/homeadmin/internal/middleware"
	"github.com/homeadmin/internal/services"
	"github.com/homeadmin/internal/templates/layouts"
	"github.com/homeadmin/internal/templates/pages"
)

// householdServiceInterface defines the household service methods needed by the handler.
type householdServiceInterface interface {
	Create(userID uint, name string) (*database.Household, error)
	Invite(userID uint) (string, error)
	Join(userID uint, code string) (*database.Household, error)
	Show(userID uint) (*database.Household, []database.User, bool, error)
}

// HouseholdHandler handles household management HTTP routes.
type HouseholdHandler struct {
	service            householdServiceInterface
	jwtSecret          string
	jwtExpirationHours int
}

// NewHouseholdHandler creates a new HouseholdHandler with injected dependencies.
func NewHouseholdHandler(service householdServiceInterface, jwtSecret string, jwtExpirationHours int) *HouseholdHandler {
	return &HouseholdHandler{
		service:            service,
		jwtSecret:          jwtSecret,
		jwtExpirationHours: jwtExpirationHours,
	}
}

// Show handles GET /household — renders the household page.
// Users with a household see their household info; users without see the setup page.
func (h *HouseholdHandler) Show(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)
	csrfToken, _ := c.Locals("csrfToken").(string)
	username, _ := c.Locals("email").(string)

	hh, members, isAdmin, err := h.service.Show(userID)
	if err != nil {
		return middleware.Internal("failed to load household")
	}

	c.Type("html")

	if hh == nil {
		component := pages.HouseholdSetup(csrfToken, "")
		page := layouts.Base("Household — HomeAdmin", csrfToken, username)
		ctx := templ.WithChildren(c.Context(), component)
		return page.Render(ctx, c.Response().BodyWriter())
	}

	component := pages.HouseholdShow(hh, members, isAdmin, csrfToken, "")
	page := layouts.Base("Household — HomeAdmin", csrfToken, username)
	ctx := templ.WithChildren(c.Context(), component)
	return page.Render(ctx, c.Response().BodyWriter())
}

// Create handles POST /household — creates a new household for the authenticated user.
func (h *HouseholdHandler) Create(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)
	name := c.FormValue("name")

	hh, err := h.service.Create(userID, name)
	if err != nil {
		if errors.Is(err, services.ErrAlreadyHasHousehold) {
			return middleware.BadRequest("You already belong to a household")
		}
		if errors.Is(err, services.ErrNameRequired) {
			return middleware.BadRequest("Household name is required")
		}
		return middleware.Internal(fmt.Sprintf("failed to create household: %v", err))
	}

	// Re-issue JWT with household_id and role=admin.
	token, err := services.CreateToken(userID, &hh.ID, "admin", h.jwtSecret, h.jwtExpirationHours)
	if err != nil {
		return middleware.Internal("failed to issue token")
	}

	SetJWTCookie(c, token)

	return c.Status(fiber.StatusFound).Redirect("/dashboard")
}
