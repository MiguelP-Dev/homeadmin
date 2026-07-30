package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/csrf"
	"github.com/gofiber/fiber/v2/middleware/logger"

	"github.com/homeadmin/internal/config"
	"github.com/homeadmin/internal/database"
	"github.com/homeadmin/internal/handlers"
	"github.com/homeadmin/internal/middleware"
	"github.com/homeadmin/internal/repositories"
	"github.com/homeadmin/internal/services"
)

func main() {
	// 1. Load configuration from environment
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// 2. Connect to database
	dbConn, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	// 3. Run migrations
	if err := database.Migrate(dbConn); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	// 4. Dependency injection
	userRepo := repositories.NewUserRepository(dbConn)
	expenseRepo := repositories.NewExpenseRepository(dbConn)
	authHandler := handlers.NewAuthHandler(userRepo, cfg.JWTSecret)
	expenseService := services.NewExpenseService(expenseRepo)
	expenseHandler := handlers.NewExpenseHandler(expenseService)

	// 5. Create Fiber app with centralized error handler
	app := fiber.New(fiber.Config{
		ErrorHandler: middleware.ErrorHandler,
	})

	// 6. Middleware chain — order matters (spec §9)
	app.Use(logger.New()) // Position 1: request logging
	app.Use(cors.New(cors.Config{ // Position 2: CORS headers
		AllowOrigins:     strings.Join(cfg.CORSOrigins, ","),
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders:     "Content-Type,Authorization,X-CSRF-Token,HX-Request",
		AllowCredentials: true,
	}))

	// Position 3: Static file serving (spec §6.8)
	app.Static("/static", "./static")

	// Position 4: CSRF protection (spec §2.1)
	csrfKey := cfg.CSRFKey
	if csrfKey == "" {
		csrfKey = "csrf-dev-fallback"
	}
	app.Use(csrfMiddleware(csrfKey))

	// 7. Public auth routes (no auth required)
	app.Get("/login", authHandler.ShowLogin)
	app.Post("/login", authHandler.Login)
	app.Get("/register", authHandler.ShowRegister)
	app.Post("/register", authHandler.Register)
	app.Post("/logout", authHandler.Logout)

	// Root redirect — unauthenticated goes to /login
	app.Get("/", func(c *fiber.Ctx) error {
		return c.Redirect("/login", fiber.StatusFound)
	})

	// 8. Protected routes — RequireAuth middleware group
	protected := app.Group("", middleware.RequireAuth(cfg.JWTSecret))
	protected.Get("/dashboard", expenseHandler.Dashboard)

	// Expense routes (Phase 4 — PR #2)
	protected.Get("/expenses", expenseHandler.List)
	protected.Post("/expenses", expenseHandler.Create)
	protected.Put("/expenses/:id", expenseHandler.Update)
	protected.Delete("/expenses/:id", expenseHandler.Delete)

	// 9. Start server
	log.Printf("server starting on :%s (env: %s)", cfg.Port, cfg.Env)
	if err := app.Listen(fmt.Sprintf(":%s", cfg.Port)); err != nil {
		log.Fatalf("server failed to start: %v", err)
	}
}

// csrfMiddleware wraps Fiber's CSRF middleware with form-based token lookup.
// KeyLookup "form:csrf" extracts the CSRF token from form field "csrf" on POST requests.
// No custom KeyGenerator — Fiber's default uses crypto/rand for secure random tokens.
func csrfMiddleware(_ string) fiber.Handler {
	return csrf.New(csrf.Config{
		KeyLookup:      "form:csrf",
		CookieName:     "csrf",
		CookieHTTPOnly: true,
		CookieSameSite: "Strict",
	})
}
