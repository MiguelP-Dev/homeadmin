package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/homeadmin/internal/database"
	"github.com/homeadmin/internal/repositories"
	"github.com/homeadmin/internal/services"
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

// ShowLogin renders the login form (standalone HTML, no templ in Phase 2).
func (h *AuthHandler) ShowLogin(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).SendString(`<!DOCTYPE html>
<html>
<head><title>Login</title></head>
<body>
<h1>Login</h1>
<form method="POST" action="/login">
  <label for="email">Email:</label>
  <input type="email" id="email" name="email" required>
  <label for="password">Password:</label>
  <input type="password" id="password" name="password" required>
  <button type="submit">Login</button>
</form>
<p><a href="/register">Register</a></p>
</body>
</html>`)
}

// ShowRegister renders the registration form (standalone HTML, no templ in Phase 2).
func (h *AuthHandler) ShowRegister(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).SendString(`<!DOCTYPE html>
<html>
<head><title>Register</title></head>
<body>
<h1>Register</h1>
<form method="POST" action="/register">
  <label for="email">Email:</label>
  <input type="email" id="email" name="email" required>
  <label for="password">Password:</label>
  <input type="password" id="password" name="password" required minlength="8">
  <button type="submit">Register</button>
</form>
<p><a href="/login">Already have an account? Login</a></p>
</body>
</html>`)
}

// Login validates credentials, sets JWT cookie, and redirects by household.
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	email := c.FormValue("email")
	password := c.FormValue("password")

	user, err := h.UserRepo.FindByEmail(email)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Internal server error")
	}
	if user == nil || !h.AuthService.CheckPassword(password, user.PasswordHash) {
		return c.Status(fiber.StatusOK).SendString(loginErrorHTML)
	}

	token, err := h.AuthService.CreateToken(user.ID, user.HouseholdID, user.Role, h.JWTSecret, 24)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Internal server error")
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

const loginErrorHTML = `<!DOCTYPE html>
<html>
<head><title>Login</title></head>
<body>
<h1>Login</h1>
<p style="color:red">Invalid credentials</p>
<form method="POST" action="/login">
  <label for="email">Email:</label>
  <input type="email" id="email" name="email" required>
  <label for="password">Password:</label>
  <input type="password" id="password" name="password" required>
  <button type="submit">Login</button>
</form>
<p><a href="/register">Register</a></p>
</body>
</html>`

const minPasswordLength = 8

// Register creates a new user, sets JWT cookie, and redirects to /dashboard.
func (h *AuthHandler) Register(c *fiber.Ctx) error {
	email := c.FormValue("email")
	password := c.FormValue("password")

	if email == "" || password == "" {
		return c.Status(fiber.StatusOK).SendString(registerMissingFieldsHTML)
	}

	if len(password) < minPasswordLength {
		return c.Status(fiber.StatusOK).SendString(registerWeakPasswordHTML)
	}

	// Check for duplicate email
	existing, err := h.UserRepo.FindByEmail(email)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Internal server error")
	}
	if existing != nil {
		return c.Status(fiber.StatusOK).SendString(registerDuplicateEmailHTML)
	}

	hash, err := h.AuthService.HashPassword(password)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Internal server error")
	}

	user := &database.User{
		Email:        email,
		PasswordHash: hash,
		Role:         "member",
	}
	if err := h.UserRepo.Create(user); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Internal server error")
	}

	token, err := h.AuthService.CreateToken(user.ID, user.HouseholdID, user.Role, h.JWTSecret, 24)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Internal server error")
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

const registerDuplicateEmailHTML = `<!DOCTYPE html>
<html>
<head><title>Register</title></head>
<body>
<h1>Register</h1>
<p style="color:red">Email already registered</p>
<form method="POST" action="/register">
  <label for="email">Email:</label>
  <input type="email" id="email" name="email" required>
  <label for="password">Password:</label>
  <input type="password" id="password" name="password" required minlength="8">
  <button type="submit">Register</button>
</form>
<p><a href="/login">Already have an account? Login</a></p>
</body>
</html>`

const registerWeakPasswordHTML = `<!DOCTYPE html>
<html>
<head><title>Register</title></head>
<body>
<h1>Register</h1>
<p style="color:red">Password must be at least 8 characters</p>
<form method="POST" action="/register">
  <label for="email">Email:</label>
  <input type="email" id="email" name="email" required>
  <label for="password">Password:</label>
  <input type="password" id="password" name="password" required minlength="8">
  <button type="submit">Register</button>
</form>
<p><a href="/login">Already have an account? Login</a></p>
</body>
</html>`

const registerMissingFieldsHTML = `<!DOCTYPE html>
<html>
<head><title>Register</title></head>
<body>
<h1>Register</h1>
<p style="color:red">Email and password are required</p>
<form method="POST" action="/register">
  <label for="email">Email:</label>
  <input type="email" id="email" name="email" required>
  <label for="password">Password:</label>
  <input type="password" id="password" name="password" required minlength="8">
  <button type="submit">Register</button>
</form>
<p><a href="/login">Already have an account? Login</a></p>
</body>
</html>`
