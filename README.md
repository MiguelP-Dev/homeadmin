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

## License

[MIT](LICENSE)
