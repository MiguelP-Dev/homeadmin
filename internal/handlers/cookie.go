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

// ClearJWTCookie expires the JWT cookie using the same standard attributes as
// SetJWTCookie, plus MaxAge 0 so the browser deletes it immediately.
func ClearJWTCookie(c *fiber.Ctx) {
	c.Cookie(&fiber.Cookie{
		Name:     "jwt",
		Value:    "",
		MaxAge:   0,
		HTTPOnly: true,
		SameSite: "Strict",
		Path:     "/",
	})
}

// anonLangCookieMaxAge keeps the visitor's language choice for one year.
const anonLangCookieMaxAge = 31536000

// SetAnonLangCookie persists the language choice of a visitor who does not
// have an account yet. It reuses the JWT cookie's attribute conventions
// (HttpOnly, SameSite Strict, Path "/") so all cookies behave consistently.
func SetAnonLangCookie(c *fiber.Ctx, lang string) {
	c.Cookie(&fiber.Cookie{
		Name:     "lang",
		Value:    lang,
		MaxAge:   anonLangCookieMaxAge,
		HTTPOnly: true,
		SameSite: "Strict",
		Path:     "/",
	})
}

// ResolveAnonLang resolves the language for unauthenticated pages from the
// "lang" cookie, falling back to English when the cookie is absent or holds
// an unsupported value. Once a user logs in, the JWT lang claim takes over.
func ResolveAnonLang(c *fiber.Ctx) string {
	if lang := c.Cookies("lang"); lang == "en" || lang == "es" {
		return lang
	}
	return "en"
}
