package handlers

import (
	"strings"

	"github.com/a-h/templ"
	"github.com/gofiber/fiber/v2"
	"github.com/homeadmin/internal/database"
	"github.com/homeadmin/internal/middleware"
	"github.com/homeadmin/internal/repositories"
	"github.com/homeadmin/internal/services"
	"github.com/homeadmin/internal/templates/layouts"
	"github.com/homeadmin/internal/templates/pages"
)

// AuthHandler holds dependencies for authentication HTTP handlers.
type AuthHandler struct {
	UserRepo    repositories.UserRepository
	AuthService authServiceProvider
	JWTSecret   string
}

// authServiceProvider defines the auth service functions needed by handlers.
// This decouples handlers from the concrete services package.
type authServiceProvider interface {
	HashPassword(password string) (string, error)
	CheckPassword(password, hash string) bool
	CreateToken(userID uint, householdID *uint, role, email, lang string, isAdmin bool, secret string, expHours int) (string, error)
}

// authServiceAdapter wraps the package-level services functions to satisfy authServiceProvider.
type authServiceAdapter struct{}

func (a *authServiceAdapter) HashPassword(password string) (string, error) {
	return services.HashPassword(password)
}

func (a *authServiceAdapter) CheckPassword(password, hash string) bool {
	return services.CheckPassword(password, hash)
}

func (a *authServiceAdapter) CreateToken(userID uint, householdID *uint, role, email, lang string, isAdmin bool, secret string, expHours int) (string, error) {
	return services.CreateToken(userID, householdID, role, email, lang, isAdmin, secret, expHours)
}

// NewAuthHandler creates a new AuthHandler with real service dependencies.
func NewAuthHandler(repo repositories.UserRepository, jwtSecret string) *AuthHandler {
	return &AuthHandler{
		UserRepo:    repo,
		AuthService: &authServiceAdapter{},
		JWTSecret:   jwtSecret,
	}
}

// renderPage renders a templ component wrapped in the base layout.
func (h *AuthHandler) renderPage(c *fiber.Ctx, title, csrfToken, email string, isAdmin bool, content templ.Component) error {
	c.Type("html")
	lang, _ := c.Locals("lang").(string)
	if lang == "" {
		lang = "en"
	}
	activePath := c.Path()
	base := layouts.Base(title, csrfToken, email, isAdmin, lang, activePath)
	ctx := templ.WithChildren(c.Context(), content)
	return base.Render(ctx, c.Response().BodyWriter())
}

// ShowLogin renders the login form via templ templates.
func (h *AuthHandler) ShowLogin(c *fiber.Ctx) error {
	csrfToken, _ := c.Locals("csrfToken").(string)
	return h.renderPage(c, "Login", csrfToken, "", false, pages.Login(csrfToken, ""))
}

// ShowRegister renders the registration form via templ templates.
func (h *AuthHandler) ShowRegister(c *fiber.Ctx) error {
	csrfToken, _ := c.Locals("csrfToken").(string)
	return h.renderPage(c, "Register", csrfToken, "", false, pages.Register(csrfToken, ""))
}

// Login validates credentials, sets JWT cookie, and redirects by household.
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	email := c.FormValue("email")
	password := c.FormValue("password")

	user, err := h.UserRepo.FindByEmail(email)
	if err != nil {
		return middleware.Keyed(500, "error.internal_server")
	}

	if user == nil || !h.AuthService.CheckPassword(password, user.PasswordHash) {
		csrfToken, _ := c.Locals("csrfToken").(string)
		return h.renderPage(c, "Login", csrfToken, "", false, pages.Login(csrfToken, "Invalid credentials"))
	}

	token, err := h.AuthService.CreateToken(user.ID, user.HouseholdID, user.Role, user.Email, user.Lang, user.IsAdmin, h.JWTSecret, 24)
	if err != nil {
		return middleware.Keyed(500, "error.internal_server")
	}

	SetJWTCookie(c, token)

	if user.HouseholdID == nil {
		return c.Redirect("/household", fiber.StatusFound)
	}
	return c.Redirect("/dashboard", fiber.StatusFound)
}

const minPasswordLength = 8

// Register creates a new user, sets JWT cookie, and redirects to /household
// (new users never belong to a household yet).
func (h *AuthHandler) Register(c *fiber.Ctx) error {
	email := c.FormValue("email")
	password := c.FormValue("password")
	name := c.FormValue("name")

	// Validate inputs before touching the database.
	if err := middleware.Validate(
		middleware.ValidateRequired(email, "email"),
		middleware.ValidateEmailFormat(email, "email"),
		middleware.ValidateRequired(password, "password"),
		middleware.ValidateRequired(name, "name"),
		middleware.ValidateMinLength(password, "password", minPasswordLength),
	); err != nil {
		return err
	}

	// Check for duplicate email
	existing, err := h.UserRepo.FindByEmail(email)
	if err != nil {
		return middleware.Keyed(500, "error.internal_server")
	}
	if existing != nil {
		csrfToken, _ := c.Locals("csrfToken").(string)
		return h.renderPage(c, "Register", csrfToken, "", false, pages.Register(csrfToken, "Email already registered"))
	}

	hash, err := h.AuthService.HashPassword(password)
	if err != nil {
		return middleware.Keyed(500, "error.internal_server")
	}

	user := &database.User{
		Email:        email,
		PasswordHash: hash,
		Role:         "member",
	}
	if err := h.UserRepo.Create(user); err != nil {
		return middleware.Keyed(500, "error.internal_server")
	}

	token, err := h.AuthService.CreateToken(user.ID, user.HouseholdID, user.Role, user.Email, user.Lang, user.IsAdmin, h.JWTSecret, 24)
	if err != nil {
		return middleware.Keyed(500, "error.internal_server")
	}

	SetJWTCookie(c, token)

	// New users have no household — send them to create/join (mirrors Login).
	if user.HouseholdID == nil {
		return c.Redirect("/household", fiber.StatusFound)
	}
	return c.Redirect("/dashboard", fiber.StatusFound)
}

// Logout clears the JWT cookie and redirects to /login.
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	ClearJWTCookie(c)
	return c.Redirect("/login", fiber.StatusFound)
}

// LangSwitch updates the user's preferred language and re-issues the JWT
// with the new lang claim. It redirects back to the Referer path when safe
// (starts with "/" and not "//"), otherwise to /dashboard.
func (h *AuthHandler) LangSwitch(c *fiber.Ctx) error {
	lang := c.FormValue("lang")
	if lang != "en" && lang != "es" {
		lang = "en" // fallback
	}

	// Validate JWT from cookie
	cookie := c.Cookies("jwt")
	if cookie == "" {
		return c.Redirect("/login", fiber.StatusFound)
	}

	claims, err := services.ValidateToken(cookie, h.JWTSecret)
	if err != nil {
		return c.Redirect("/login", fiber.StatusFound)
	}

	// Fetch user from DB
	user, err := h.UserRepo.FindByID(claims.UserID)
	if err != nil || user == nil {
		return c.Redirect("/login", fiber.StatusFound)
	}

	// Update lang
	user.Lang = lang
	if err := h.UserRepo.Update(user); err != nil {
		return middleware.Keyed(500, "error.internal_server")
	}

	// Re-issue token with updated lang
	token, err := h.AuthService.CreateToken(
		user.ID, user.HouseholdID, user.Role, user.Email, user.Lang, user.IsAdmin,
		h.JWTSecret, 24,
	)
	if err != nil {
		return middleware.Keyed(500, "error.internal_server")
	}
	SetJWTCookie(c, token)

	// Redirect to Referer path if safe
	referer := c.Get("Referer")
	redirectPath := "/dashboard"
	if len(referer) > 0 {
		// Extract path from referer URL
		if idx := strings.Index(referer, "://"); idx > 0 {
			pathStart := idx + 3
			if slashIdx := strings.Index(referer[pathStart:], "/"); slashIdx >= 0 {
				referer = referer[pathStart+slashIdx:]
			} else {
				referer = "/"
			}
		}
		if strings.HasPrefix(referer, "/") && !strings.HasPrefix(referer, "//") {
			redirectPath = referer
		}
	}

	return c.Redirect(redirectPath, fiber.StatusFound)
}
