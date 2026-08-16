package handlers

import (
	"errors"
	"fmt"

	"github.com/a-h/templ"
	"github.com/gofiber/fiber/v2"
	"github.com/homeadmin/internal/database"
	"github.com/homeadmin/internal/middleware"
	"github.com/homeadmin/internal/repositories"
	"github.com/homeadmin/internal/services"
	"github.com/homeadmin/internal/templates/layouts"
	"github.com/homeadmin/internal/templates/pages"
)

// householdServiceInterface defines the household service methods needed by the handler.
type householdServiceInterface interface {
	Create(userID uint, name string) (*database.Household, error)
	Invite(userID uint) (string, error)
	Join(userID uint, code string) (*database.Household, error)
	Show(userID uint) (*services.HouseholdView, error)
	SetMemberRole(ownerID, targetID uint, role string) error
}

// HouseholdHandler handles household management HTTP routes.
type HouseholdHandler struct {
	service            householdServiceInterface
	userRepo           repositories.UserRepository
	jwtSecret          string
	jwtExpirationHours int
}

// NewHouseholdHandler creates a new HouseholdHandler with injected dependencies.
func NewHouseholdHandler(service householdServiceInterface, userRepo repositories.UserRepository, jwtSecret string, jwtExpirationHours int) *HouseholdHandler {
	return &HouseholdHandler{
		service:            service,
		userRepo:           userRepo,
		jwtSecret:          jwtSecret,
		jwtExpirationHours: jwtExpirationHours,
	}
}

// Show handles GET /household — renders the household page.
// Users with a household see their household info; users without see the setup page.
func (h *HouseholdHandler) Show(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)
	csrfToken, _ := c.Locals("csrfToken").(string)
	email, _ := c.Locals("email").(string)
	isAdmin, _ := c.Locals("isAdmin").(bool)

	view, err := h.service.Show(userID)
	if err != nil {
		return middleware.Internal("failed to load household")
	}

	c.Type("html")

	if view == nil {
		component := pages.HouseholdSetup(csrfToken, "")
		page := layouts.Base("Household — HomeAdmin", csrfToken, email, isAdmin)
		ctx := templ.WithChildren(c.Context(), component)
		return page.Render(ctx, c.Response().BodyWriter())
	}

	component := pages.HouseholdShow(view.Household, view.Members, view.ViewerRole, csrfToken, "")
	page := layouts.Base("Household — HomeAdmin", csrfToken, email, isAdmin)
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

	// Re-issue JWT with household_id and role=owner from the fresh DB user
	// (design D2) so the token carries the current household, role, email,
	// and site-admin flag (user.IsAdmin, slice 5).
	user, err := h.userRepo.FindByID(userID)
	if err != nil || user == nil {
		return middleware.Internal("failed to load user for token")
	}
	token, err := services.CreateToken(user.ID, &hh.ID, user.Role, user.Email, user.Lang, user.IsAdmin, h.jwtSecret, h.jwtExpirationHours)
	if err != nil {
		return middleware.Internal("failed to issue token")
	}

	SetJWTCookie(c, token)

	return c.Status(fiber.StatusFound).Redirect("/dashboard")
}

// Invite handles POST /household/invite — generates an invite code for the household.
func (h *HouseholdHandler) Invite(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)
	csrfToken, _ := c.Locals("csrfToken").(string)
	email, _ := c.Locals("email").(string)
	isAdmin, _ := c.Locals("isAdmin").(bool)

	code, err := h.service.Invite(userID)
	if err != nil {
		if errors.Is(err, services.ErrNoHousehold) {
			return middleware.BadRequest("You must belong to a household")
		}
		if errors.Is(err, services.ErrNotAdmin) {
			return middleware.Forbidden("Only admins can invite")
		}
		return middleware.Internal("failed to generate invite code")
	}

	// Re-fetch household to render HouseholdShow with the invite code.
	view, err := h.service.Show(userID)
	if err != nil || view == nil {
		return middleware.Internal("failed to load household")
	}

	component := pages.HouseholdShow(view.Household, view.Members, view.ViewerRole, csrfToken, code)
	page := layouts.Base("Household — HomeAdmin", csrfToken, email, isAdmin)
	ctx := templ.WithChildren(c.Context(), component)
	c.Type("html")
	return page.Render(ctx, c.Response().BodyWriter())
}

// SetMemberRole handles POST /household/members/:id/role — owner-only role
// changes. The owner can promote/demote admins and members, but never change
// their own role nor another owner's (RF-8).
func (h *HouseholdHandler) SetMemberRole(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)

	targetID, err := c.ParamsInt("id")
	if err != nil || targetID <= 0 {
		return middleware.BadRequest("invalid member id")
	}

	role := c.FormValue("role")
	if err := h.service.SetMemberRole(userID, uint(targetID), role); err != nil {
		switch {
		case errors.Is(err, services.ErrNotOwner), errors.Is(err, services.ErrOwnerImmutable):
			return middleware.Forbidden("You cannot change this member's role")
		case errors.Is(err, services.ErrSelfRoleChange):
			return middleware.BadRequest("You cannot change your own role")
		case errors.Is(err, services.ErrNotMember):
			return middleware.NotFound("User is not a member of this household")
		default:
			return middleware.BadRequest("Invalid role")
		}
	}

	return c.Status(fiber.StatusFound).Redirect("/household")
}

// Join handles POST /household/join — joins an existing household via invite code.
func (h *HouseholdHandler) Join(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)
	code := c.FormValue("code")

	hh, err := h.service.Join(userID, code)
	if err != nil {
		if errors.Is(err, services.ErrAlreadyHasHousehold) {
			return middleware.BadRequest("You already belong to a household")
		}
		if errors.Is(err, services.ErrInvalidCode) {
			return middleware.BadRequest("Invalid invite code")
		}
		if errors.Is(err, services.ErrExpiredCode) {
			return middleware.BadRequest("Invite code has expired")
		}
		if errors.Is(err, services.ErrUsedCode) {
			return middleware.BadRequest("Invite code has already been used")
		}
		return middleware.Internal("failed to join household")
	}

	// Re-issue JWT with household_id and role=member from the fresh DB user
	// (design D2) so the token carries the current household, role, email,
	// and site-admin flag (user.IsAdmin, slice 5).
	user, err := h.userRepo.FindByID(userID)
	if err != nil || user == nil {
		return middleware.Internal("failed to load user for token")
	}
	token, err := services.CreateToken(user.ID, &hh.ID, user.Role, user.Email, user.Lang, user.IsAdmin, h.jwtSecret, h.jwtExpirationHours)
	if err != nil {
		return middleware.Internal("failed to issue token")
	}

	SetJWTCookie(c, token)

	return c.Status(fiber.StatusFound).Redirect("/dashboard")
}
