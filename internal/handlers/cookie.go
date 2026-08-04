package handlers

import "github.com/gofiber/fiber/v2"

// SetJWTCookie sets the JWT session cookie with the application's standard
// attributes (same as auth.go Login/Logout): name "jwt", HttpOnly, SameSite
// Strict, Path "/", and no Secure flag (dev runs over plain http).
func SetJWTCookie(c *fiber.Ctx, token string) {
	c.Cookie(&fiber.Cookie{
		Name:     "jwt",
		Value:    token,
		HTTPOnly: true,
		SameSite: "Strict",
		Path:     "/",
	})
}
