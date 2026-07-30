package handlers

import (
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
	CreateToken(userID uint, householdID *uint, role, secret string, expHours int) (string, error)
}

// authServiceAdapter wraps the package-level services functions to satisfy authServiceProvider.
type authServiceAdapter struct{}

func (a *authServiceAdapter) HashPassword(password string) (string, error) {
	return services.HashPassword(password)
}

func (a *authServiceAdapter) CheckPassword(password, hash string) bool {
	return services.CheckPassword(password, hash)
}

func (a *authServiceAdapter) CreateToken(userID uint, householdID *uint, role, secret string, expHours int) (string, error) {
	return services.CreateToken(userID, householdID, role, secret, expHours)
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
func (h *AuthHandler) renderPage(c *fiber.Ctx, title, csrfToken, username string, content templ.Component) error {
	base := layouts.Base(title, csrfToken, username)
	ctx := templ.WithChildren(c.Context(), content)
	return base.Render(ctx, c.Response().BodyWriter())
}

// ShowLogin renders the login form via templ templates.
func (h *AuthHandler) ShowLogin(c *fiber.Ctx) error {
	csrfToken, _ := c.Locals("csrfToken").(string)
	return h.renderPage(c, "Login", csrfToken, "", pages.Login(csrfToken, ""))
}

// ShowRegister renders the registration form via templ templates.
func (h *AuthHandler) ShowRegister(c *fiber.Ctx) error {
	csrfToken, _ := c.Locals("csrfToken").(string)
	return h.renderPage(c, "Register", csrfToken, "", pages.Register(csrfToken, ""))
}

// Login validates credentials, sets JWT cookie, and redirects by household.
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	email := c.FormValue("email")
	password := c.FormValue("password")

	user, err := h.UserRepo.FindByEmail(email)
	if err != nil {
		return middleware.Internal("internal server error")
	}

	if user == nil || !h.AuthService.CheckPassword(password, user.PasswordHash) {
		csrfToken, _ := c.Locals("csrfToken").(string)
		return h.renderPage(c, "Login", csrfToken, "", pages.Login(csrfToken, "Invalid credentials"))
	}

	token, err := h.AuthService.CreateToken(user.ID, user.HouseholdID, user.Role, h.JWTSecret, 24)
	if err != nil {
		return middleware.Internal("internal server error")
	}

	c.Cookie(&fiber.Cookie{
		Name:     "jwt",
		Value:    token,
		HTTPOnly: true,
		SameSite: "Strict",
		Path:     "/",
	})

	if user.HouseholdID == nil {
		return c.Redirect("/household", fiber.StatusFound)
	}
	return c.Redirect("/dashboard", fiber.StatusFound)
}

const minPasswordLength = 8

// Register creates a new user, sets JWT cookie, and redirects to /dashboard.
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
		return middleware.Internal("internal server error")
	}
	if existing != nil {
		csrfToken, _ := c.Locals("csrfToken").(string)
		return h.renderPage(c, "Register", csrfToken, "", pages.Register(csrfToken, "Email already registered"))
	}

	hash, err := h.AuthService.HashPassword(password)
	if err != nil {
		return middleware.Internal("internal server error")
	}

	user := &database.User{
		Email:        email,
		PasswordHash: hash,
		Role:         "member",
	}
	if err := h.UserRepo.Create(user); err != nil {
		return middleware.Internal("internal server error")
	}

	token, err := h.AuthService.CreateToken(user.ID, user.HouseholdID, user.Role, h.JWTSecret, 24)
	if err != nil {
		return middleware.Internal("internal server error")
	}

	c.Cookie(&fiber.Cookie{
		Name:     "jwt",
		Value:    token,
		HTTPOnly: true,
		SameSite: "Strict",
		Path:     "/",
	})

	return c.Redirect("/dashboard", fiber.StatusFound)
}

// Logout clears the JWT cookie and redirects to /login.
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	c.Cookie(&fiber.Cookie{
		Name:     "jwt",
		Value:    "",
		MaxAge:   0,
		HTTPOnly: true,
		SameSite: "Strict",
		Path:     "/",
	})
	return c.Redirect("/login", fiber.StatusFound)
}
