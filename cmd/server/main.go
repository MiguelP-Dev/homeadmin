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
	dbConn, err := database.ConnectWithDriver(cfg.DatabaseURL, cfg.DBDriver)
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

	householdRepo := repositories.NewHouseholdRepository(dbConn)
	// The service's inviteRepo surface (MarkUsed) lives on the same repository,
	// so householdRepo satisfies both arguments.
	householdService := services.NewHouseholdService(householdRepo, userRepo, householdRepo)
	householdHandler := handlers.NewHouseholdHandler(householdService, userRepo, cfg.JWTSecret, cfg.JWTExpirationHours)

	// 5. Create Fiber app with centralized error handler
	app := fiber.New(fiber.Config{
		ErrorHandler: middleware.ErrorHandler,
	})

	// 6. Middleware chain — order matters (spec §9)
	app.Use(logger.New())         // Position 1: request logging
	app.Use(cors.New(cors.Config{ // Position 2: CORS headers
		AllowOrigins:     strings.Join(cfg.CORSOrigins, ","),
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders:     "Content-Type,Authorization,X-CSRF-Token,HX-Request",
		AllowCredentials: true,
	}))

	// Position 3: Static file serving (spec §6.8)
	app.Static("/static", "./static", fiber.Static{
		MaxAge: 31536000, // 1 year cache — files are versioned by the build process
	})

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

	// Root redirect — token-aware: authenticated users go to /dashboard,
	// everyone else to /login.
	app.Get("/", rootRedirect(cfg.JWTSecret))

	// 8. Protected routes — RequireAuth middleware applied per-route.
	// NOTE: do NOT use app.Group("", RequireAuth): Fiber mounts middleware of an
	// empty-prefix group as a fallback for UNMATCHED paths too, which would turn
	// every 404 into a redirect to /login before the 404 is generated.
	// RequireHousehold runs after RequireAuth on household-mandatory routes
	// (design §2); /household Show and /household/join must stay reachable with
	// a nil household (spec: create/join require nil, invite 400s via handler).
	app.Get("/dashboard", middleware.RequireAuth(cfg.JWTSecret), middleware.RequireHousehold(), expenseHandler.Dashboard)

	// Expense routes (Phase 4 — PR #2)
	app.Get("/expenses", middleware.RequireAuth(cfg.JWTSecret), middleware.RequireHousehold(), expenseHandler.List)
	app.Post("/expenses", middleware.RequireAuth(cfg.JWTSecret), middleware.RequireHousehold(), expenseHandler.Create)
	app.Get("/expenses/new", middleware.RequireAuth(cfg.JWTSecret), middleware.RequireHousehold(), expenseHandler.ShowNew)
	app.Get("/expenses/:id/edit", middleware.RequireAuth(cfg.JWTSecret), middleware.RequireHousehold(), expenseHandler.ShowEdit)
	app.Post("/expenses/:id/update", middleware.RequireAuth(cfg.JWTSecret), middleware.RequireHousehold(), expenseHandler.Update)
	app.Post("/expenses/:id/delete", middleware.RequireAuth(cfg.JWTSecret), middleware.RequireHousehold(), expenseHandler.Delete)

	// Household routes
	app.Get("/household", middleware.RequireAuth(cfg.JWTSecret), householdHandler.Show)
	app.Post("/household", middleware.RequireAuth(cfg.JWTSecret), householdHandler.Create)
	app.Post("/household/invite", middleware.RequireAuth(cfg.JWTSecret), householdHandler.Invite)
	app.Post("/household/join", middleware.RequireAuth(cfg.JWTSecret), householdHandler.Join)

	// 9. Start server
	log.Printf("server starting on :%s (env: %s)", cfg.Port, cfg.Env)
	if err := app.Listen(fmt.Sprintf(":%s", cfg.Port)); err != nil {
		log.Fatalf("server failed to start: %v", err)
	}
}

// rootRedirect returns a handler that redirects authenticated users (valid
// "jwt" cookie) to /dashboard and unauthenticated users to /login.
func rootRedirect(jwtSecret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		cookie := c.Cookies("jwt")
		if cookie == "" {
			return c.Redirect("/login", fiber.StatusFound)
		}
		if _, err := services.ValidateToken(cookie, jwtSecret); err != nil {
			return c.Redirect("/login", fiber.StatusFound)
		}
		return c.Redirect("/dashboard", fiber.StatusFound)
	}
}

// csrfMiddleware wraps Fiber's CSRF middleware with form-based token lookup.
// KeyLookup "form:csrf" extracts the CSRF token from form field "csrf" on POST requests.
// No custom KeyGenerator — Fiber's default uses crypto/rand for secure random tokens.
func csrfMiddleware(_ string) fiber.Handler {
	return csrf.New(csrf.Config{
		KeyLookup:      "form:csrf",
		ContextKey:     "csrfToken", // matches handler reads via c.Locals("csrfToken")
		CookieName:     "csrf",
		CookieHTTPOnly: true,
		CookieSameSite: "Strict",
	})
}
