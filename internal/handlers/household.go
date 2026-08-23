package handlers

import (
	"errors"

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
	RemoveMember(ownerID, targetID uint) error
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
	lang, _ := c.Locals("lang").(string)
	if lang == "" {
		lang = "en"
	}
	activePath := c.Path()

	view, err := h.service.Show(userID)
	if err != nil {
		return middleware.Keyed(500, "error.internal_server")
	}

	c.Type("html")

	if view == nil {
		component := pages.HouseholdSetup(csrfToken, "", lang)
		page := layouts.Base("Household — HomeAdmin", csrfToken, email, isAdmin, lang, activePath)
		ctx := templ.WithChildren(c.Context(), component)
		return page.Render(ctx, c.Response().BodyWriter())
	}

	component := pages.HouseholdShow(view.Household, view.Members, view.ViewerRole, csrfToken, "", lang)
	page := layouts.Base("Household — HomeAdmin", csrfToken, email, isAdmin, lang, activePath)
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
			return middleware.Keyed(400, "household.already_has")
		}
		if errors.Is(err, services.ErrNameRequired) {
			return middleware.Keyed(400, "household.name_required")
		}
		return middleware.Keyed(500, "error.internal_server")
	}

	// Re-issue JWT with household_id and role=owner from the fresh DB user
	// (design D2) so the token carries the current household, role, email,
	// and site-admin flag (user.IsAdmin, slice 5).
	user, err := h.userRepo.FindByID(userID)
	if err != nil || user == nil {
		return middleware.Keyed(500, "error.internal_server")
	}
	token, err := services.CreateToken(user.ID, &hh.ID, user.Role, user.Email, user.Lang, user.IsAdmin, h.jwtSecret, h.jwtExpirationHours)
	if err != nil {
		return middleware.Keyed(500, "error.internal_server")
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
			return middleware.Keyed(400, "household.no_household")
		}
		if errors.Is(err, services.ErrNotAdmin) {
			return middleware.Keyed(403, "household.not_admin")
		}
		return middleware.Keyed(500, "error.internal_server")
	}

	// Re-fetch household to render HouseholdShow with the invite code.
	view, err := h.service.Show(userID)
	if err != nil || view == nil {
		return middleware.Keyed(500, "error.internal_server")
	}

	lang, _ := c.Locals("lang").(string)
	if lang == "" {
		lang = "en"
	}
	component := pages.HouseholdShow(view.Household, view.Members, view.ViewerRole, csrfToken, code, lang)
	activePath := c.Path()
	page := layouts.Base("Household — HomeAdmin", csrfToken, email, isAdmin, lang, activePath)
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
		return middleware.Keyed(400, "household.member_not_found")
	}

	role := c.FormValue("role")
	if err := h.service.SetMemberRole(userID, uint(targetID), role); err != nil {
		switch {
		case errors.Is(err, services.ErrNotOwner), errors.Is(err, services.ErrOwnerImmutable):
			return middleware.Keyed(403, "household.role_forbidden")
		case errors.Is(err, services.ErrSelfRoleChange):
			return middleware.Keyed(400, "household.self_role")
		case errors.Is(err, services.ErrNotMember):
			return middleware.Keyed(404, "household.member_not_found")
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
			return middleware.Keyed(400, "household.already_has")
		}
		if errors.Is(err, services.ErrInvalidCode) {
			return middleware.Keyed(400, "household.invalid_code")
		}
		if errors.Is(err, services.ErrExpiredCode) {
			return middleware.Keyed(400, "household.expired")
		}
		if errors.Is(err, services.ErrUsedCode) {
			return middleware.Keyed(400, "household.used")
		}
		return middleware.Keyed(500, "error.internal_server")
	}

	// Re-issue JWT with household_id and role=member from the fresh DB user
	// (design D2) so the token carries the current household, role, email,
	// and site-admin flag (user.IsAdmin, slice 5).
	user, err := h.userRepo.FindByID(userID)
	if err != nil || user == nil {
		return middleware.Keyed(500, "error.internal_server")
	}
	token, err := services.CreateToken(user.ID, &hh.ID, user.Role, user.Email, user.Lang, user.IsAdmin, h.jwtSecret, h.jwtExpirationHours)
	if err != nil {
		return middleware.Keyed(500, "error.internal_server")
	}

	SetJWTCookie(c, token)

	return c.Status(fiber.StatusFound).Redirect("/dashboard")
}

// RemoveMember handles POST /household/members/:id/remove — owner-only member
// removal. The target member is removed from the household (HouseholdID cleared).
func (h *HouseholdHandler) RemoveMember(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)

	targetID, err := c.ParamsInt("id")
	if err != nil || targetID <= 0 {
		return middleware.Keyed(400, "household.member_not_found")
	}

	if err := h.service.RemoveMember(userID, uint(targetID)); err != nil {
		switch {
		case errors.Is(err, services.ErrSelfRemoval):
			return middleware.Keyed(400, "household.self_removal")
		case errors.Is(err, services.ErrNotOwner), errors.Is(err, services.ErrOwnerImmutable):
			return middleware.Keyed(403, "household.role_forbidden")
		case errors.Is(err, services.ErrNotMember):
			return middleware.Keyed(404, "household.member_not_found")
		default:
			return middleware.Keyed(500, "error.internal_server")
		}
	}

	return c.Status(fiber.StatusFound).Redirect("/household")
}
