# HomeAdmin

Self-hosted household financial management built with the GoTTH stack.

## What it does

Manage shared expenses within households with fine-grained visibility controls:

- **Expense tracking** — fixed and variable expenses with categories
- **Visibility rules** — shared editable, shared read-only, or private
- **Household management** — create households, invite members, role-based access
- **Authentication** — email/password with JWT, CSRF protection

## Tech stack

| Layer | Technology |
|-------|------------|
| Language | Go 1.22+ |
| Web framework | Fiber v2 |
| ORM | GORM v2 |
| Database | PostgreSQL (SQLite for tests) |
| Auth | JWT (HttpOnly cookies) + CSRF |
| Templating | Templ + HTMX |
| Styling | Tailwind CSS |

## Project structure

```
homeadmin/
├── cmd/server/          # Entry point, DI, routes
├── internal/
│   ├── config/          # Environment config
│   ├── database/        # Models, migrations, seed
│   ├── handlers/        # HTTP handlers (Fiber)
│   ├── middleware/       # Auth, CSRF
│   ├── repositories/    # Data access (GORM)
│   └── services/        # Business logic
├── docs/spec.md         # Full specification
└── go.mod
```

## Getting started

```bash
# Clone
git clone https://github.com/MiguelP-Dev/homeadmin.git
cd homeadmin

# Install dependencies
go mod tidy

# Set up environment
cp .env.example .env
# Edit .env with your database URL and secrets

# Run migrations and start
go run cmd/server/main.go
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

## Development phases

- [x] Phase 1 — Foundation (config, models, migrations)
- [x] Phase 2 — Authentication (JWT, register, login)
- [x] Phase 3 — Household management (create, invite, join)
- [x] Phase 4 — Expenses (CRUD with visibility controls)
- [ ] Phase 5 — Dashboard summary
- [ ] Phase 6 — Polish and deployment

## License

[MIT](LICENSE)
