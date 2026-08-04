package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

// TestSetJWTCookie verifies the shared JWT cookie helper sets the exact
// attributes main's auth handlers use (auth.go Login/Logout): name "jwt",
// HttpOnly, SameSite Strict, Path "/", and NO Secure flag (dev over http).
func TestSetJWTCookie(t *testing.T) {
	tests := []struct {
		name  string
		token string
	}{
		{name: "round-trips plain token", token: "first-token-value"},
		{name: "round-trips JWT-shaped token", token: "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxIn0.signature"},
		{name: "round-trips another token", token: "third-token-value"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Get("/probe", func(c *fiber.Ctx) error {
				SetJWTCookie(c, tt.token)
				return c.SendStatus(fiber.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/probe", nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test failed: %v", err)
			}
			defer resp.Body.Close()

			var jwt *http.Cookie
			for _, c := range resp.Cookies() {
				if c.Name == "jwt" {
					jwt = c
					break
				}
			}
			if jwt == nil {
				t.Fatal("expected a 'jwt' cookie in the response")
			}
			if jwt.Value != tt.token {
				t.Errorf("cookie value = %q, want %q", jwt.Value, tt.token)
			}
			if !jwt.HttpOnly {
				t.Error("expected jwt cookie to be HttpOnly")
			}
			if jwt.SameSite != http.SameSiteStrictMode {
				t.Errorf("cookie SameSite = %v, want %v", jwt.SameSite, http.SameSiteStrictMode)
			}
			if jwt.Path != "/" {
				t.Errorf("cookie Path = %q, want %q", jwt.Path, "/")
			}
			if jwt.Secure {
				t.Error("expected jwt cookie to NOT be Secure (dev runs over http)")
			}
		})
	}
}
