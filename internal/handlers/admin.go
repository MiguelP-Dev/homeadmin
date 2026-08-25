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

// Show handles GET /admin — renders the read-only admin page with the site
// summary, users table, and per-household overview rows (RF-11).
func (h *AdminHandler) Show(c *fiber.Ctx) error {
	users, err := h.service.ListUsers()
	if err != nil {
		return middleware.Keyed(500, "admin.load_failed")
	}

	blocks, err := h.service.SiteAdminOverview()
	if err != nil {
		return middleware.Keyed(500, "admin.load_failed")
	}
	summary := services.BuildAdminSummary(blocks)

	email, _ := c.Locals("email").(string)
	isAdmin, _ := c.Locals("isAdmin").(bool)
	csrfToken, _ := c.Locals("csrfToken").(string)
	lang, _ := c.Locals("lang").(string)
	if lang == "" {
		lang = "en"
	}
	activePath := c.Path()

	component := pages.Admin(users, &summary, lang)
	page := layouts.Base(pageTitle(lang, "title.admin"), csrfToken, email, isAdmin, lang, activePath)
	c.Type("html")
	ctx := templ.WithChildren(c.Context(), component)
	return page.Render(ctx, c.Response().BodyWriter())
}
