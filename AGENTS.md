# AGENTS.md

## Project Overview

Go REST API built with **Fiber v2** web framework, **PostgreSQL** database via **GORM** ORM, **JWT** authentication, and **RBAC** (Role-Based Access Control). Implements an e-commerce backend with customers, products, orders, shops, categories, payments, and shipping.

**Key technologies:** Go 1.26, Fiber v2, GORM, PostgreSQL, golang-jwt/jwt/v5, go-playground/validator, robfig/cron

## Architecture

Clean architecture with 4 layers:

```
cmd/server/main.go          → Entry point, route registration, dependency wiring
internal/
├── config/config.go        → Env loading (godotenv), Config struct
├── database/database.go    → PostgreSQL connection (GORM), AutoMigrate
├── models/                 → GORM models (UUID primary keys, soft deletes)
├── repositories/           → Data access layer (DB queries)
├── services/               → Business logic layer
├── handlers/               → HTTP handlers (Fiber context)
├── middleware/auth.go      → JWT authentication middleware
├── middleware/rbac.go       → Permission-based authorization
├── cron/                   → Scheduled tasks (order auto-completion)
└── utils/                  → Response helpers, validation, password hashing, slug, haversine
```

**Dependency flow:** `main.go` → handler → service → repository → GORM → PostgreSQL

## Setup Commands

```bash
# 1. Copy environment file
cp .env.example .env

# 2. Edit .env — change JWT_SECRET (≥16 chars) and ADMIN_PASSWORD (must not be default)

# 3. Create PostgreSQL database
createdb go_fiber

# 4. Run the server (auto-migrates DB and seeds initial data)
go run cmd/server/main.go
```

Server starts at `http://localhost:3000` (configurable via `SERVER_PORT`).

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DB_HOST` | PostgreSQL host | `localhost` |
| `DB_PORT` | PostgreSQL port | `5432` |
| `DB_USER` | PostgreSQL user | `postgres` |
| `DB_PASSWORD` | PostgreSQL password | `postgres` |
| `DB_NAME` | Database name | `go_fiber` |
| `JWT_SECRET` | JWT signing key (≥16 chars, required) | — |
| `JWT_ACCESS_TTL` | Access token lifetime | `15m` |
| `JWT_REFRESH_TTL` | Refresh token lifetime | `168h` |
| `SERVER_PORT` | HTTP server port | `3000` |
| `ADMIN_EMAIL` | Seed admin email | `admin@example.com` |
| `ADMIN_PASSWORD` | Seed admin password (must change) | — |
| `ADMIN_PHONE` | Seed admin phone | `0900000000` |

## Development Workflow

```bash
# Run server
go run cmd/server/main.go

# Build binary
go build -o server ./cmd/server

# Run with Docker
docker compose up --build
```

## Testing Instructions

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests for a specific package
go test ./internal/services/...
go test ./internal/repositories/...
go test ./internal/models/...
go test ./internal/cron/...

# Run a specific test by name
go test -run TestAutoCompleteDeliveredOrdersBefore ./internal/services/...

# Run tests with race detector
go test -race ./...

# Check test coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out
```

**Testing patterns:**
- Tests use **SQLite in-memory** databases for isolation (via `glebarez/sqlite`)
- Test files are co-located with source: `order_repo.go` → `order_repo_test.go`
- Helper functions create test DBs with `newXxxTestDB(t)` pattern
- Test data created via `createTestXxx(t, db, ...)` helpers
- Cleanup uses `t.Cleanup()` to remove temp SQLite files

**Existing test files:**
- `internal/models/order_test.go` — Model constants and fields
- `internal/repositories/order_repo_test.go` — Repository queries
- `internal/services/order_service_test.go` — Business logic
- `internal/cron/cron_test.go` — Cron schedule validation

## Code Style

- **Language:** Go, follow standard Go conventions (`gofmt`, `go vet`)
- **No unused imports** — Go compiler enforces this
- **Error handling:** Always check errors, return via `utils.Error(c, statusCode, errorCode, message)`
- **Validation:** Use `go-playground/validator` struct tags, validate with `utils.Validate(input)`
- **Response format:** Use `utils.Success()` and `utils.Error()` helpers for consistent JSON responses
- **Naming:**
  - Exported functions/types: PascalCase
  - Unexported: camelCase
  - Package names: lowercase, single word
  - Files: snake_case (`auth_handler.go`, `order_repo.go`)
- **Struct tags:** `json:"field_name,omitempty"` for JSON, `gorm:"..."` for DB, `validate:"required"` for validation
- **UUID primary keys:** All models use `uuid.UUID` with `gen_random_uuid()` default
- **Soft deletes:** Models with `DeletedAt *time.Time` use GORM soft delete

### Response Format

All API responses follow this structure:

```go
// Success
utils.Success(c, data, "message")
// Returns: { "success": true, "data": ..., "message": "..." }

// Success with pagination
utils.SuccessWithPagination(c, data, page, limit, total)
// Returns: { "success": true, "data": ..., "pagination": { "page": 1, "limit": 10, "total": 100, "total_pages": 10 } }

// Error
utils.Error(c, 400, "VALIDATION_ERROR", "message")
// Returns: { "success": false, "error": { "code": "ERROR_CODE", "message": "..." } }
```

### Error Codes

| Status | Code | Usage |
|--------|------|-------|
| 400 | `VALIDATION_ERROR` | Invalid input/body |
| 401 | `UNAUTHORIZED` | Missing/invalid JWT |
| 401 | `INVALID_CREDENTIALS` | Wrong login/password |
| 403 | `FORBIDDEN` | Insufficient permissions |
| 404 | `NOT_FOUND` | Resource not found |
| 409 | `DUPLICATE_ENTRY` | Unique constraint violation |
| 500 | `INTERNAL_ERROR` | Server error |

## Adding a New Feature

Follow this pattern when adding a new resource (e.g., `Product`):

1. **Model** — `internal/models/product.go`: Define GORM struct with UUID, timestamps, soft delete
2. **Repository** — `internal/repositories/product_repo.go`: DB queries via GORM
3. **Service** — `internal/services/product_service.go`: Business logic, validation
4. **Handler** — `internal/handlers/product_handler.go`: Parse request, call service, return response
5. **Routes** — `cmd/server/main.go`: Wire dependencies, register routes under `/api/v1/`
6. **Tests** — Add `*_test.go` files at each layer using SQLite in-memory DB

## Seed Data

On first run, the app automatically seeds:
- **Roles:** `super_admin` (all permissions), `editor` (customer CRUD + dashboard), `viewer` (read-only)
- **Permissions:** 25 permissions covering customer, user, role, permission, dashboard, category, shop, product, order, shipping_config
- **Admin user:** Created from `ADMIN_EMAIL` / `ADMIN_PASSWORD` env vars
- **Shipping config:** Default base fee, per-km rate, max distance

## Docker

```bash
# Build and run
docker compose up --build

# The Dockerfile uses multi-stage build:
# Stage 1: Go build (golang:1.26-alpine)
# Stage 2: Runtime (alpine:3.21, non-root user)
```

The app container connects to PostgreSQL on the host via `host.docker.internal`.

## API Documentation

Full API docs with request/response examples: `docs/api.md`

API base URL: `http://localhost:3000/api/v1`

## Common Gotchas

- `JWT_SECRET` must be ≥16 characters and not the default value — app will crash otherwise
- `ADMIN_PASSWORD` must be changed from `admin123` — app will crash otherwise
- Database migrations run automatically on startup via `db.AutoMigrate()`
- Seed data only runs if no roles exist in the database
- Auth endpoints have rate limiting: 5 requests/min per IP
- Tests require no external database — they use SQLite in-memory
