# HomeAdmin v1.0.0 — Software Design Document

## 1. Introduction

### 1.1 Purpose
This document defines the technical architecture, data models, API routes, middleware chain, security strategy, and development phases for **HomeAdmin v1** — a self-hosted household financial management application built with the GoTTH stack.

This spec is the single source of truth for development. Every implementation decision should trace back to this document.

### 1.2 Scope (v1)
HomeAdmin v1 delivers:
- **Authentication**: Email/password login with JWT (HttpOnly cookies)
- **Household Management**: Create/join households, invite members, RBAC (admin/member)
- **Expense Tracking**: Fixed and variable expenses with visibility controls (shared editable, shared read-only, private)
- **Dashboard Summary**: Monthly totals for income and expenses

**Explicitly deferred to v2+:**
- Income tracking model
- Savings goals and assets
- Debt tracking and amortization
- Category management (user-defined)
- CSV import
- Charts and analytics

### 1.3 Learning Goals
This project serves as a learning exercise for:
- GoTTH stack (Go + Templ + Tailwind + HTMX)
- Go fundamentals: interfaces, error handling, struct embedding
- GORM: models, relationships, migrations, query building
- Fiber: middleware chain, context, routing
- godotenv: configuration management
- Testing: interface-based mocking, table-driven tests
- Security: JWT, CSRF, bcrypt, CORS

---

## 2. Functional Requirements

### 2.1 Authentication & Session
- **Registration**: Email + password → bcrypt hash → store user → issue JWT
- **Login**: Validate credentials → issue JWT in HttpOnly, Secure, SameSite=Strict cookie
- **Logout**: Clear JWT cookie
- **Session**: JWT contains `user_id`, `household_id`, `role`; validated on every protected request
- **CSRF**: Fiber CSRF middleware injects token into all forms; HTMX requests include token in headers

### 2.2 Household Management
- **Create Household**: Admin creates a household, becomes its admin
- **Invite Member**: Admin generates invite (email-based or code-based for v1)
- **Join Household**: User accepts invite, links to household
- **Roles**: `admin` (manage members, full access), `member` (standard access)
- **One Household per User**: A user belongs to exactly one household (v1 constraint)

### 2.3 Expense Tracking
- **Create Expense**: Amount, description, category (fixed list), date, visibility
- **Edit Expense**: Creator can edit own expenses; shared editable expenses can be edited by any household member
- **Delete Expense**: Soft-delete with same permission logic as edit
- **Visibility States**:
  - `visible_editable`: All household members can see and edit
  - `visible_only`: All can see, only creator can edit/delete
  - `hidden_private`: Only creator can see/edit/delete

### 2.4 Dashboard
- **Monthly Summary**: Total expenses (fixed + variable) for current month
- **Category Breakdown**: Expenses grouped by category
- **Recent Activity**: Last 5 expenses added

---

## 3. Non-Functional Requirements

### 3.1 Deployment
- Docker multi-stage build → Alpine or Scratch base → single binary < 30MB
- Stateless container (no local disk persistence)
- Environment variables via `.env` file or cloud platform config
- Supabase PostgreSQL with SSL/TLS

### 3.2 Security
- All secrets in environment variables (zero hardcoding)
- CSRF on all state-mutating requests
- Password hashing with bcrypt (cost factor 12)
- JWT expiration: 24 hours
- Cookie: HttpOnly, Secure, SameSite=Strict
- Input validation on all user-facing forms

### 3.3 Performance
- Server-side rendering with Templ (compiled to Go functions)
- HTMX for dynamic updates (no full page reloads)
- GORM connection pooling (MaxOpenConns: 25, MaxIdleConns: 10)
- Sub-100ms response times for HTML pages

---

## 4. Technical Stack

| Layer | Technology | Version | Purpose |
|-------|------------|---------|---------|
| Language | Go | 1.22+ | Type safety, performance |
| Web Framework | Fiber | v2 | HTTP handling, middleware |
| ORM | GORM | v2 | Database, migrations |
| Templating | Templ | latest | Type-safe HTML templates |
| Client Interactivity | HTMX | v1.9+ | AJAX without JS |
| Styling | Tailwind CSS | v3+ | Utility-first CSS |
| Configuration | godotenv | latest | .env loading |
| Database | PostgreSQL | 15+ | Supabase-hosted |
| Testing | stdlib + mocks | - | Interface-based mocking |

---

## 5. Architecture

### 5.1 Layer Diagram

```
HTTP Request
    ↓
Fiber Router
    ↓
Middleware Chain (Logger → CORS → CSRF → Auth)
    ↓
Handler (extracts params, validates, calls service)
    ↓
Service (business logic, authorization checks)
    ↓
Repository (database queries via GORM)
    ↓
PostgreSQL (Supabase)
```

### 5.2 Directory Structure

```
homeadmin/
├── cmd/
│   └── server/
│       └── main.go              # Entry point, DI, server start
├── internal/
│   ├── config/
│   │   └── config.go            # Config struct, Load()
│   ├── database/
│   │   ├── models.go            # GORM models
│   │   ├── database.go          # DB connection, migrations
│   │   └── seed.go              # Fixed categories seed
│   ├── handlers/
│   │   ├── auth.go              # Login, register, logout
│   │   ├── household.go         # Create, invite, join
│   │   └── expense.go           # CRUD operations
│   ├── middleware/
│   │   ├── auth.go              # JWT validation, set context
│   │   └── csrf.go              # CSRF token injection
│   ├── services/
│   │   ├── auth.go              # Password hashing, JWT creation
│   │   ├── household.go         # Household business logic
│   │   └── expense.go           # Expense business logic, visibility
│   └── repositories/
│       ├── user.go              # User DB operations
│       ├── household.go         # Household DB operations
│       └── expense.go           # Expense DB operations
├── templates/
│   ├── layouts/
│   │   └── base.templ           # Base layout with head, nav
│   ├── pages/
│   │   ├── login.templ
│   │   ├── register.templ
│   │   ├── dashboard.templ
│   │   ├── expenses.templ
│   │   └── household.templ
│   └── components/
│       ├── expense_card.templ
│       ├── expense_form.templ
│       ├── nav.templ
│       └── toast.templ
├── static/
│   ├── css/
│   │   └── input.css            # Tailwind source
│   └── js/
│       └── htmx.min.js
├── go.mod
├── go.sum
├── Dockerfile
├── docker-compose.yml           # Local dev (optional)
├── tailwind.config.js
├── .env.example                 # Template for required env vars
└── Makefile                     # Common commands
```

---

## 6. Data Models

### 6.1 Models

```go
package database

import (
    "time"
    "gorm.io/gorm"
)

// Household represents a shared group (family, roommates)
type Household struct {
    ID        uint           `gorm:"primaryKey"`
    Name      string         `gorm:"size:100;not null"`
    CreatedAt time.Time
    UpdatedAt time.Time
    DeletedAt gorm.DeletedAt `gorm:"index"`
    Users     []User         `gorm:"foreignKey:HouseholdID"`
}

// User represents an authenticated user
type User struct {
    ID           uint           `gorm:"primaryKey"`
    Email        string         `gorm:"size:255;uniqueIndex;not null"`
    PasswordHash string         `gorm:"size:255;not null"`
    Role         string         `gorm:"size:20;default:'member'"` // 'admin' or 'member'
    HouseholdID  *uint          `gorm:"default:null"`
    Household    *Household     `gorm:"foreignKey:HouseholdID"`
    CreatedAt    time.Time
    UpdatedAt    time.Time
}

// VisibilityType controls who can see/edit an expense
type VisibilityType string

const (
    VisibleEditable VisibilityType = "visible_editable"
    VisibleOnly     VisibilityType = "visible_only"
    HiddenPrivate   VisibilityType = "hidden_private"
)

// Expense tracks fixed and variable payments
type Expense struct {
    ID          uint           `gorm:"primaryKey"`
    Amount      float64        `gorm:"type:decimal(10,2);not null"`
    Description string         `gorm:"size:255;not null"`
    Category    string         `gorm:"size:50;not null"` // Fixed list (see categories.go)
    IsFixed     bool           `gorm:"default:false;not null"`
    CreatedByID uint           `gorm:"not null"`
    CreatedBy   User           `gorm:"foreignKey:CreatedByID"`
    HouseholdID uint           `gorm:"not null"`
    Household   Household      `gorm:"foreignKey:HouseholdID"`
    Visibility  VisibilityType `gorm:"type:varchar(20);default:'visible_editable'"`
    Date        time.Time      `gorm:"not null"`
    CreatedAt   time.Time
    UpdatedAt   time.Time
    DeletedAt   gorm.DeletedAt `gorm:"index"`
}
```

### 6.2 Fixed Categories

```go
package database

// ExpenseCategories is the fixed list of allowed categories
var ExpenseCategories = []string{
    "Rent",
    "Utilities",
    "Groceries",
    "Dining Out",
    "Transportation",
    "Entertainment",
    "Subscriptions",
    "Insurance",
    "Household",
    "Personal Care",
    "Education",
    "Savings",
    "Other",
}

// IsValidCategory checks if a category is in the fixed list
func IsValidCategory(category string) bool {
    for _, c := range ExpenseCategories {
        if c == category {
            return true
        }
    }
    return false
}
```

---

## 7. Configuration

### 7.1 Environment Variables

```env
# .env.example

# Server
PORT=8080
ENV=development  # development | production

# Database (Supabase)
DATABASE_URL=postgres://postgres:[password]@db.[project-id].supabase.co:5432/postgres?sslmode=require

# JWT
JWT_SECRET=your-256-bit-secret-here
JWT_EXPIRATION_HOURS=24

# CORS (production origins, comma-separated)
CORS_ORIGINS=http://localhost:8080

# CSRF
CSRF_KEY=32-byte-random-key-here
```

### 7.2 Config Struct

```go
package config

import (
    "os"
    "strconv"
    "github.com/joho/godotenv"
)

type Config struct {
    Port              string
    Env               string
    DatabaseURL       string
    JWTSecret         string
    JWTExpirationHours int
    CORSOrigins       []string
    CSRFKey           string
}

func Load() (*Config, error) {
    godotenv.Load() // Ignore error if .env doesn't exist (cloud envs)

    expiration, _ := strconv.Atoi(getEnv("JWT_EXPIRATION_HOURS", "24"))

    return &Config{
        Port:              getEnv("PORT", "8080"),
        Env:               getEnv("ENV", "development"),
        DatabaseURL:       os.Getenv("DATABASE_URL"),
        JWTSecret:         os.Getenv("JWT_SECRET"),
        JWTExpirationHours: expiration,
        CORSOrigins:       strings.Split(getEnv("CORS_ORIGINS", "http://localhost:8080"), ","),
        CSRFKey:           os.Getenv("CSRF_KEY"),
    }, nil
}

func getEnv(key, fallback string) string {
    if v := os.Getenv(key); v != "" {
        return v
    }
    return fallback
}
```

---

## 8. Route Map

### 8.1 Public Routes

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| GET | `/` | Redirect | → `/login` or `/dashboard` based on auth |
| GET | `/login` | AuthHandler.ShowLogin | Login form |
| POST | `/login` | AuthHandler.Login | Process login |
| GET | `/register` | AuthHandler.ShowRegister | Registration form |
| POST | `/register` | AuthHandler.Register | Process registration |
| POST | `/logout` | AuthHandler.Logout | Clear session |

### 8.2 Protected Routes (require valid JWT)

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| GET | `/dashboard` | ExpenseHandler.Dashboard | Monthly summary |
| GET | `/expenses` | ExpenseHandler.List | List expenses (filtered by visibility) |
| GET | `/expenses/new` | ExpenseHandler.ShowNew | New expense form |
| POST | `/expenses` | ExpenseHandler.Create | Create expense |
| GET | `/expenses/:id/edit` | ExpenseHandler.ShowEdit | Edit expense form |
| PUT | `/expenses/:id` | ExpenseHandler.Update | Update expense |
| DELETE | `/expenses/:id` | ExpenseHandler.Delete | Soft-delete expense |
| GET | `/household` | HouseholdHandler.Show | Household info |
| POST | `/household` | HouseholdHandler.Create | Create household |
| POST | `/household/invite` | HouseholdHandler.Invite | Invite member |
| POST | `/household/join` | HouseholdHandler.Join | Join via invite |

---

## 9. Middleware Chain

```go
// Order matters — execute top to bottom
app.Use(logger.New())           // 1. Request logging
app.Use(cors.New(cors.Config{   // 2. CORS headers
    AllowOrigins:     strings.Join(config.CORSOrigins, ","),
    AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
    AllowCredentials: true,
}))
app.Use(csrf.New(csrf.Config{   // 3. CSRF token (for forms)
    KeyLookup:      "form:csrf",
    CookieName:     "csrf",
    CookieHTTPOnly: true,
    CookieSameSite: "Strict",
}))
```

### 9.1 Auth Middleware (route-level)

```go
// Protected route group
protected := app.Group("/")
protected.Use(middleware.RequireAuth(config.JWTSecret))

// middleware/auth.go
func RequireAuth(jwtSecret string) fiber.Handler {
    return func(c *fiber.Ctx) error {
        // 1. Get JWT from cookie
        token := c.Cookies("jwt")
        if token == "" {
            return c.Redirect("/login")
        }

        // 2. Validate and parse JWT
        claims, err := auth.ValidateToken(token, jwtSecret)
        if err != nil {
            return c.Redirect("/login")
        }

        // 3. Set user context for handlers
        c.Locals("userID", claims.UserID)
        c.Locals("householdID", claims.HouseholdID)
        c.Locals("role", claims.Role)

        return c.Next()
    }
}
```

---

## 10. Security Implementation

### 10.1 Password Hashing

```go
package services

import "golang.org/x/crypto/bcrypt"

const bcryptCost = 12

func HashPassword(password string) (string, error) {
    bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
    return string(bytes), err
}

func CheckPassword(password, hash string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
    return err == nil
}
```

### 10.2 JWT Structure

```go
type Claims struct {
    UserID      uint   `json:"user_id"`
    HouseholdID *uint  `json:"household_id"`
    Role        string `json:"role"`
    jwt.RegisteredClaims
}

// Token creation
func CreateToken(userID uint, householdID *uint, role, secret string, expirationHours int) (string, error) {
    claims := Claims{
        UserID:      userID,
        HouseholdID: householdID,
        Role:        role,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expirationHours) * time.Hour)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
        },
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(secret))
}
```

### 10.3 CSRF in Templ

```templ
templ ExpenseForm(csrfToken string) {
    <form hx-post="/expenses" hx-target="#expense-list" hx-swap="beforeend">
        <input type="hidden" name="csrf" value={ csrfToken } />
        <!-- form fields -->
    </form>
}
```

---

## 11. Testing Strategy

### 11.1 Interface-Based Mocking

Define interfaces for repositories, then mock them in tests:

```go
// repositories/user.go
type UserRepository interface {
    FindByEmail(email string) (*User, error)
    Save(user *User) error
    FindByID(id uint) (*User, error)
}

// Test mock
type MockUserRepo struct {
    FindByEmailFn func(email string) (*User, error)
    SaveFn        func(user *User) error
    FindByIDFn    func(id uint) (*User, error)
}

func (m *MockUserRepo) FindByEmail(email string) (*User, error) {
    return m.FindByEmailFn(email)
}

func (m *MockUserRepo) Save(user *User) error {
    return m.SaveFn(user)
}

func (m *MockUserRepo) FindByID(id uint) (*User, error) {
    return m.FindByIDFn(id)
}
```

### 11.2 Test Categories

| Category | Tool | Coverage Target |
|----------|------|-----------------|
| Unit (services) | stdlib testing + mocks | Business logic, validation |
| Unit (repositories) | GORM + SQLite in-memory | Query correctness |
| Integration | Testcontainers (optional) | Full stack with real DB |
| HTTP handlers | Fiber's `app.Test()` | Request/response flows |

### 11.3 Example Test

```go
func TestCreateExpense(t *testing.T) {
    mockRepo := &MockExpenseRepo{
        SaveFn: func(e *Expense) error {
            e.ID = 1 // Simulate DB save
            return nil
        },
    }

    svc := services.NewExpenseService(mockRepo)

    expense := &Expense{
        Amount:      50.00,
        Description: "Groceries",
        Category:    "Groceries",
        HouseholdID: 1,
        CreatedByID: 1,
    }

    err := svc.Create(expense)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    if expense.ID != 1 {
        t.Errorf("expected ID 1, got %d", expense.ID)
    }
}
```

---

## 12. Development Phases

### Phase 1: Foundation (Day 1)
- [ ] Project setup: `go mod init`, directory structure
- [ ] Config loading: `.env` + `config.go`
- [ ] Database connection: GORM + Supabase
- [ ] Models: Household, User, Expense
- [ ] Migrations: AutoMigrate + verification
- [ ] Fixed categories seed

### Phase 2: Authentication (Day 2)
- [ ] Auth service: `HashPassword`, `CheckPassword`, `CreateToken`, `ValidateToken`
- [ ] User repository: `FindByEmail`, `Save`
- [ ] Auth handlers: login, register, logout
- [ ] Auth middleware: JWT validation
- [ ] Templ components: login form, register form

### Phase 3: Household (Day 3)
- [ ] Household repository: `Create`, `FindByName`, `AddMember`
- [ ] Household service: create, invite, join logic
- [ ] Household handlers: create, show, invite, join
- [ ] Templ components: household page, invite form

### Phase 4: Expenses (Day 4-5)
- [ ] Expense repository: CRUD + visibility-filtered queries
- [ ] Expense service: create, update, delete with permission checks
- [ ] Expense handlers: full CRUD
- [ ] Templ components: expense card, expense form, expense list

### Phase 5: Dashboard (Day 6)
- [ ] Dashboard queries: monthly totals, category breakdown
- [ ] Dashboard handler: aggregate data
- [ ] Dashboard templ: summary cards, recent activity

### Phase 6: Polish & Deploy (Day 7)
- [ ] Error handling: centralized error handler
- [ ] Input validation: form validation middleware
- [ ] Docker: multi-stage Dockerfile
- [ ] Testing: unit tests for services, handler tests
- [ ] CORS configuration
- [ ] Final review

---

## 13. Open Questions

1. **Invite mechanism**: For v1, should invites be email-based (requires SMTP) or code-based (simpler, show code on screen)? → Recommend **code-based** for simplicity.
2. **Category enforcement**: Should the form validate categories server-side, or just trust the fixed list? → Recommend **server-side validation** always.
3. **Soft-delete visibility**: Should soft-deleted expenses appear in any view? → Recommend **no**, completely hidden.

---

## Appendix A: Go Dependencies

```go
require (
    github.com/gofiber/fiber/v2      v2.x.x
    github.com/gofiber/fiber/v2/middleware/csrf
    github.com/gofiber/fiber/v2/middleware/cors
    github.com/gofiber/fiber/v2/middleware/logger
    github.com/joho/godotenv           v1.x.x
    github.com/golang-jwt/jwt/v5       v5.x.x
    golang.org/x/crypto               v0.x.x
    gorm.io/gorm                       v1.x.x
    gorm.io/driver/postgres            v1.x.x
    github.com/a-h/templ               v0.x.x
)
```

---

*Document version: 1.0.0 | Last updated: 2025-01-27 | Status: Ready for Development*
