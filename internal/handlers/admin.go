package handlers

import (
	"github.com/a-h/templ"
	"github.com/gofiber/fiber/v2"
	"github.com/homeadmin/internal/middleware"
	"github.com/homeadmin/internal/services"
	"github.com/homeadmin/internal/templates/layouts"
	"github.com/homeadmin/internal/templates/pages"
)

// AdminHandler handles site-admin routes.
type AdminHandler struct {
	service *services.SiteAdminService
}

// NewAdminHandler creates a new AdminHandler.
func NewAdminHandler(service *services.SiteAdminService) *AdminHandler {
	return &AdminHandler{service: service}
}

// Show handles GET /admin — renders the read-only admin page with users
// and households tables (RF-11).
func (h *AdminHandler) Show(c *fiber.Ctx) error {
	users, err := h.service.ListUsers()
	if err != nil {
		return middleware.Keyed(500, "admin.load_failed")
	}

	households, err := h.service.ListHouseholds()
	if err != nil {
		return middleware.Keyed(500, "admin.load_failed")
	}

	email, _ := c.Locals("email").(string)
	isAdmin, _ := c.Locals("isAdmin").(bool)
	csrfToken, _ := c.Locals("csrfToken").(string)

	component := pages.Admin(users, households)
	page := layouts.Base("Admin — HomeAdmin", csrfToken, email, isAdmin)
	c.Type("html")
	ctx := templ.WithChildren(c.Context(), component)
	return page.Render(ctx, c.Response().BodyWriter())
}
