# HomeAdmin

Self-hosted household financial management built with the GoTTH stack.

## What it does

Manage shared expenses within households with fine-grained visibility controls:

- **Expense tracking** — fixed and variable expenses with categories
- **Visibility rules** — shared editable, shared read-only, or private
- **Dashboard** — monthly summaries with category breakdown and recent activity
- **Household management** — create households, invite members, role-based access
- **Authentication** — email/password with JWT, CSRF protection

## Tech stack

| Layer | Technology |
|-------|------------|
| Language | Go 1.26+ |
| Web framework | Fiber v2 |
| ORM | GORM v2 |
| Database | PostgreSQL (production) / SQLite (dev/test) |
| Auth | JWT (HttpOnly cookies) + CSRF |
| Templating | Templ + HTMX |
| Styling | Tailwind CSS |
| Runtime | Docker + docker-compose |

## Project structure

```
homeadmin/
├── cmd/server/              # Entry point, DI, routes
├── internal/
│   ├── config/              # Environment config
│   ├── database/            # Models, migrations, seed
│   ├── handlers/            # HTTP handlers (Fiber)
│   ├── middleware/           # Auth, CSRF, error handling, validation
│   ├── repositories/        # Data access (GORM)
│   ├── services/            # Business logic
│   └── templates/           # Templ components
│       ├── layouts/         # Base layout shell
│       ├── pages/           # Page templates (login, register, dashboard, expenses)
│       └── components/      # Reusable components (nav, expense_card, toast)
├── static/
│   ├── css/input.css        # Tailwind source
│   └── js/htmx.min.js       # Vendored HTMX
├── docs/spec.md             # Full specification
├── Dockerfile               # Multi-stage production build
├── docker-compose.yml       # App + PostgreSQL local dev
├── Makefile                 # Build, test, coverage, docker targets
├── tailwind.config.js       # Tailwind content paths
├── .env.example
└── go.mod
```

## Getting started

### Local dev (SQLite)

```bash
# Clone
git clone https://github.com/MiguelP-Dev/homeadmin.git
cd homeadmin

# Install dependencies
go mod tidy

# Generate templ code
make templ

# Set up environment
cp .env.example .env
# Edit .env with your secrets

# Run the app (SQLite by default, zero config)
go run cmd/server/main.go
```

### Docker (PostgreSQL)

```bash
# Copy and configure environment
cp .env.example .env
# Set DB_DRIVER=postgres and DATABASE_URL for your setup

# Build and start
make docker-build
make docker-up
```

## Common commands

```bash
# Run all tests
make test

# Coverage report (opens HTML)
make coverage

# Lint
make lint

# Regenerate templ templates after changes
make templ

# Build Docker image
make docker-build

# Start/stop Docker services
make docker-up
make docker-stop
```

## Testing

```bash
# Run all tests
go test ./...

# With coverage
go test ./... -cover

# Verbose
go test ./... -v
```

Current coverage:

| Package | Coverage |
|---------|----------|
| config | 100% |
| services | 94.4% |
| database | 90% |
| middleware | 86.7% |
| repositories | 85.9% |
| handlers | 84.5% |

## Development phases

- [x] Phase 1 — Foundation (config, models, migrations)
- [x] Phase 2 — Authentication (JWT, register, login)
- [x] Phase 3 — Household management (create, invite, join)
- [x] Phase 4 — Expenses (CRUD with visibility controls)
- [x] Phase 5 — Dashboard summary
- [x] Phase 6 — Polish and deployment (CSRF fix, error handling, validation, templ migration, Docker, CI targets)

## Open follow-ups

Tracked items that do not block current functionality but are pending
attention. Captured from review and design findings of the household
onboarding chain (PR1–PR5, merged to main 2026-08-07).

1. **CSRF round-trip test** — no integration test sends a request with a valid
   CSRF token through a real handler; household E2E POST flows run with CSRF
   disabled (`csrfKey` empty) while production mounts CSRF unconditionally.
2. **JWT claim assertion after Join (E2E)** — the integration suite checks the
   re-issued JWT is non-empty after Create but never re-reads its claims after
   Join. Handler-level unit tests already decode claims; add an E2E assert.
3. **Nav renders unauthenticated for logged-in users** — `RequireAuth`
   (`internal/middleware/auth.go`) sets only `userID`/`householdID`/`role`
   locals, never `email`; handlers that read `c.Locals("email")` for the Nav
   username get `""`, so `components.Nav` shows Login/Register links for
   authenticated users. Fix requires either adding `Email` to
   `Claims`+`CreateToken` (signature ripple across auth/household handlers and
   every `CreateToken` test call) or giving `RequireAuth` a user-repo
   dependency (ripple at all mount sites).
4. **JWT expiration hardcoded** — `internal/services/auth.go` hardcodes 24h
   expiry despite `cfg.JWTExpirationHours`; refactor to read the config value.
5. **Secure cookie flag** — `SetJWTCookie` does not set `Secure` for HTTPS
   production; matches current dev behavior, needs enabling for production.
6. **gofmt debt (pre-existing)** — `internal/database/models.go`,
   `internal/repositories/expense_test.go`, `internal/services/expense_test.go`
   were already non-gofmt-clean at the chain base commit.

## License

[MIT](LICENSE)
