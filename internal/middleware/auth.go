package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/homeadmin/internal/services"
)

// RequireAuth returns a Fiber middleware that validates JWT from the "jwt" cookie.
// On success, it sets c.Locals("userID"), c.Locals("householdID"), c.Locals("role"),
// c.Locals("email"), and c.Locals("isAdmin").
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
		c.Locals("email", claims.Email)
		c.Locals("isAdmin", claims.IsAdmin)

		return c.Next()
	}
}

// RequireHousehold returns a Fiber middleware that requires a non-nil
// householdID in Locals. RequireAuth stores claims.HouseholdID, which is
// *uint; a nil or absent pointer means the user has no household, so the
// request is redirected to /household (create/join). A typed-nil *uint
// (null household_id claim) must NOT pass. Intended to run AFTER RequireAuth.
func RequireHousehold() fiber.Handler {
	return func(c *fiber.Ctx) error {
		householdID, ok := c.Locals("householdID").(*uint)
		if !ok || householdID == nil {
			return c.Redirect("/household", fiber.StatusFound)
		}
		return c.Next()
	}
}
