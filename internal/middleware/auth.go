package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/homeadmin/internal/services"
)

// RequireAuth returns a Fiber middleware that validates JWT from the "jwt" cookie.
// On success, it sets c.Locals("userID"), c.Locals("householdID"), and c.Locals("role").
// On failure, it redirects to /login.
func RequireAuth(jwtSecret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		cookie := c.Cookies("jwt")
		if cookie == "" {
			return c.Redirect("/login", fiber.StatusFound)
		}

		claims, err := services.ValidateToken(cookie, jwtSecret)
		if err != nil {
			return c.Redirect("/login", fiber.StatusFound)
		}

		c.Locals("userID", claims.UserID)
		c.Locals("householdID", claims.HouseholdID)
		c.Locals("role", claims.Role)

		return c.Next()
	}
}
