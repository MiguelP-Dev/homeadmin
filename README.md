# HomeAdmin

Self-hosted household financial management built with the GoTTH stack.

## What it does

Manage shared expenses within households with fine-grained visibility controls:

- **Expense tracking** — fixed and variable expenses with categories, fully server-rendered HTML
- **Visibility rules** — shared editable, shared read-only, or private (owner-only)
- **Dashboard** — monthly summaries with category breakdown and recent activity
- **Household management** — create households, invite members, role-based access (owner/admin/member), remove members
- **Site administration** — true site-admin experience: global views of every household's transactions grouped by household with member details across dashboard/expenses/savings, plus a site summary panel at `/admin`
- **Internationalization** — English/Spanish everywhere, including login/register; logged-in users get a per-user preference persisted in JWT, visitors get a language cookie; translated UI and currency/date formatting
- **Authentication** — email/password with JWT, CSRF protection
- **Theming** — dark mode toggle with localStorage persistence
- **Navigation** — responsive drawer (hamburger on mobile), language switcher, aria-current page indicator

## Tech stack

| Layer | Technology |
|-------|------------|
| Language | Go 1.26+ |
| Web framework | Fiber v2 |
| ORM | GORM v2 |
| Database | PostgreSQL (production) / SQLite (dev/test) |
| Auth | JWT (HttpOnly cookies) + CSRF |
| Templating | Templ + HTMX |
| Styling | Tailwind CSS (compiled to `static/css/output.css`, embedded into the binary at build time) |
| Runtime | Docker + docker-compose |

## Project structure

```
homeadmin/
├── cmd/server/              # Entry point, DI, routes
├── cmd/promote/              # CLI: promote user to site admin
├── internal/
│   ├── config/              # Environment config
│   ├── database/            # Models, migrations, seed
│   ├── handlers/            # HTTP handlers (Fiber)
│   ├── middleware/           # Auth, CSRF, error handling, validation
│   ├── repositories/        # Data access (GORM)
│   ├── services/            # Business logic
│   └── templates/           # Templ components
│       ├── layouts/         # Base layout shell
│       ├── pages/           # Page templates (login, register, dashboard, expenses, admin)
│       └── components/      # Reusable components (nav, expense_card, toast)
├── static/
│   ├── css/input.css        # Tailwind source
│   ├── css/output.css       # Compiled CSS (committed)
│   └── js/htmx.min.js       # Vendored HTMX
├── docs/spec.md             # Full specification
├── Dockerfile               # Multi-stage production build
├── docker-compose.yml       # App + PostgreSQL local dev
├── Makefile                 # Build, test, coverage, docker targets
├── tailwind.config.js       # Tailwind content paths + darkMode: 'class'
├── package.json             # Tailwind build scripts
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

# Build CSS
make css

# Set up environment
cp .env.example .env
# Edit .env with your secrets

# Run the app (SQLite by default, zero config)
go run cmd/server/main.go

# Or build a local binary instead (output: bin/homeadmin)
make build
./bin/homeadmin
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

# Build CSS (Tailwind)
make css

# Watch CSS during development
make css-watch

# Build the binary (outputs bin/homeadmin, not committed)
make build

# Run the app
make run

# Build Docker image
make docker-build

# Start/stop Docker services
make docker-up
make docker-stop
```

### Static assets and binaries

- Static assets are **embedded into the binary** at build time with `go:embed`
  (root [`assets.go`](assets.go), serving the `static/` tree under `/static`).
  No external `./static` directory is required next to the binary at runtime —
  but rebuild the binary after changing CSS or vendored JS so the embed picks
  up the new files.
- `bin/` is local build output produced by `make build` as `bin/homeadmin`.
  It is intentionally **not committed** (gitignored); remove it with
  `make clean`.

## Site administration

The first registered user automatically becomes a site admin (first-user admin privilege).

Promote a user to site admin manually:

```bash
go run cmd/promote/main.go -email user@example.com
```

Site admins get:

- **Global views** — `/dashboard`, `/expenses` and `/savings` render every household's
  operations grouped by household, with owner email per transaction. Site admins do not
  need their own household to use any section.
- **Summary panel** — `/admin` shows site-wide totals (households, users, income,
  transactions) plus a per-household breakdown with links into each section.

## Internationalization

- **Languages**: English (default) and Spanish
- **Per-user preference**: stored in database, persisted in JWT
- **Visitors**: language choice kept in a 1-year cookie; the EN/ES switcher is always visible in the nav, including login and register. Once logged in, the account preference takes over
- **Locale-aware display**: currency (`$1.234,56` in ES), dates (`2 de enero de 2006`), categories, and visibility labels
- **Translated errors**: form validation, auth, and household error messages

### Key routes

| Method | Path | Description |
|--------|------|-------------|
| POST | `/settings/lang` | Switch language — authenticated: persists on account + JWT re-issue; anonymous: sets a `lang` cookie (CSRF-protected) |
| POST | `/household/members/:id/role` | Promote/demote member (owner only) |
| POST | `/household/members/:id/remove` | Remove member (owner only) |

## Role model

| Role | Scope | Privileges |
|------|-------|------------|
| Owner | Household | Full control: invite, change roles, remove members, view/edit all expenses |
| Admin | Household | Invite members, view/edit shared expenses |
| Member | Household | View shared expenses, add own expenses |
| Site admin | Global | Full read access to all households' data across dashboard/expenses/savings plus the `/admin` summary panel; first registered user gets this automatically |

### Expense visibility

| Visibility | Who can view | Who can edit |
|------------|--------------|--------------|
| `visible_editable` | All members | All members |
| `visible_only` | All members | Creator only |
| `hidden_private` | Owner only | Owner only |

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
- [x] Phase 6 — UI Polish & Admin Roles (HTML-first, roles, site-admin, Tailwind CSS, dark mode)
- [x] Phase 7 — Polish and deployment (CSRF fix, error handling, validation, templ migration, Docker, CI targets)

## Open follow-ups

Tracked items that do not block current functionality but are pending
attention. Captured from review and design findings of the household
onboarding chain (PR1–PR5, merged to main 2026-08-07).

1. **CSRF round-trip test coverage** — a full CSRF handshake integration test now
    exists for `POST /settings/lang` (valid-token success + token-less 403);
    household E2E POST flows still run with CSRF disabled (`csrfKey` empty) while
    production mounts CSRF unconditionally — extend the handshake pattern there.
2. **JWT claim assertion after Join (E2E)** — the integration suite checks the
    re-issued JWT is non-empty after Create but never re-reads its claims after
    Join. Handler-level unit tests already decode claims; add an E2E assert.
3. **gofmt debt (pre-existing)** — `internal/database/models.go`,
    `internal/repositories/expense_test.go`, `internal/services/expense_test.go`
    were already non-gofmt-clean at the chain base commit.

## License

[MIT](LICENSE)
