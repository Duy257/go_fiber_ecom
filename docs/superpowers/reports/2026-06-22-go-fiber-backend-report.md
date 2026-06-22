# Go-Fiber Backend Implementation Report

> **Date:** 2026-06-22
> **Plan:** [2026-06-22-go-fiber-backend.md](../plans/2026-06-22-go-fiber-backend.md)
> **Spec:** [2026-06-22-go-fiber-backend-design.md](../specs/2026-06-22-go-fiber-backend-design.md)

---

## Tổng quan

Xây dựng backend Go-Fiber với **Clean Architecture** (Handler → Service → Repository), JWT auth 2 role (admin + customer), RBAC phân quyền, CRUD Customer/User/Role/Permission, PostgreSQL với GORM.

---

## Cấu trúc project

```
go-fiber/
├── cmd/server/main.go          # Entry point, routes, seed data
├── internal/
│   ├── config/config.go        # Env-based config loader
│   ├── database/database.go    # Postgres connect + auto-migrate
│   ├── middleware/
│   │   ├── auth.go             # JWT Bearer token verification
│   │   └── rbac.go             # Permission-based access control
│   ├── models/
│   │   ├── customer.go         # Customer model (email/phone login)
│   │   ├── user.go             # Admin user model (with role FK)
│   │   ├── role.go             # Role model (many2many permissions)
│   │   └── permission.go       # Permission model
│   ├── repositories/
│   │   ├── customer_repo.go    # Customer CRUD + pagination
│   │   ├── user_repo.go        # User CRUD + preload role/permissions
│   │   ├── role_repo.go        # Role CRUD + preload permissions
│   │   └── permission_repo.go  # Permission lookup by IDs
│   ├── services/
│   │   ├── auth_service.go     # Login (admin/customer), JWT token pair, refresh
│   │   ├── customer_service.go # Customer business logic
│   │   ├── user_service.go     # User business logic
│   │   ├── role_service.go     # Role+permission assignment logic
│   │   └── dashboard_service.go# Stats aggregation
│   ├── handlers/
│   │   ├── auth_handler.go     # POST /auth/admin/login, /auth/customer/login, /auth/refresh
│   │   ├── customer_handler.go # CRUD + self-profile endpoints
│   │   ├── user_handler.go     # Admin user CRUD
│   │   ├── role_handler.go     # Role CRUD
│   │   ├── permission_handler.go # List permissions
│   │   └── dashboard_handler.go  # GET /admin/dashboard/stats
│   └── utils/
│       ├── response.go         # Standard JSON response format
│       ├── password.go         # bcrypt hash/verify
│       └── validator.go        # Struct validation + email/phone check
├── .env.example
├── .gitignore
├── go.mod
└── go.sum
```

---

## API Routes

### Public
| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/auth/customer/login` | Customer login (email or phone + password) |
| POST | `/api/v1/auth/admin/login` | Admin login |
| POST | `/api/v1/auth/refresh` | Refresh access token |

### Customer (self-service, cần JWT)
| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/customer/profile` | Lấy profile của customer đang đăng nhập |
| PUT | `/api/v1/customer/profile` | Cập nhật profile |

### Admin (cần JWT + RBAC)
| Method | Path | Permission |
|--------|------|------------|
| GET | `/api/v1/admin/dashboard/stats` | `dashboard:read` |
| GET | `/api/v1/admin/customers` | `customer:read` |
| GET | `/api/v1/admin/customers/:id` | `customer:read` |
| POST | `/api/v1/admin/customers` | `customer:write` |
| PUT | `/api/v1/admin/customers/:id` | `customer:write` |
| DELETE | `/api/v1/admin/customers/:id` | `customer:delete` |
| GET | `/api/v1/admin/users` | `user:read` |
| GET | `/api/v1/admin/users/:id` | `user:read` |
| POST | `/api/v1/admin/users` | `user:write` |
| PUT | `/api/v1/admin/users/:id` | `user:write` |
| DELETE | `/api/v1/admin/users/:id` | `user:delete` |
| GET | `/api/v1/admin/roles` | `role:read` |
| POST | `/api/v1/admin/roles` | `role:write` |
| PUT | `/api/v1/admin/roles/:id` | `role:write` |
| DELETE | `/api/v1/admin/roles/:id` | `role:delete` |
| GET | `/api/v1/admin/permissions` | `permission:read` |

---

## Seed Data

Khi chạy lần đầu, tự động seed:

- **12 permissions**: `customer:read/write/delete`, `user:read/write/delete`, `role:read/write/delete`, `permission:read/write`, `dashboard:read`
- **3 roles**:
  - `super_admin` — full permissions
  - `editor` — `customer:read`, `customer:write`, `dashboard:read`
  - `viewer` — `customer:read`, `dashboard:read`
- **1 admin user**: email/password từ biến môi trường (`ADMIN_EMAIL`, `ADMIN_PASSWORD`), role `super_admin`

---

## Tech Stack

| Component | Library |
|-----------|---------|
| Web framework | `github.com/gofiber/fiber/v2` v2.52 |
| ORM | `gorm.io/gorm` v1.31 + `gorm.io/driver/postgres` v1.6 |
| JWT | `github.com/golang-jwt/jwt/v5` v5.3 |
| Password | `golang.org/x/crypto` (bcrypt) |
| Validation | `github.com/go-playground/validator/v10` v10.30 |
| UUID | `github.com/google/uuid` v1.6 |
| Env | `github.com/joho/godotenv` v1.5 |

---

## Git Log

```
4450026 chore: final cleanup and verification
691618a feat: add main entry point with routes and seed data
a54047e feat: add handlers (auth, customer, user, role, permission, dashboard)
6728b74 feat: add JWT auth and RBAC middleware
5689fe1 feat: add services (auth, customer, user, role, dashboard)
2db72ec feat: add database connection, migration, and repositories
00cb428 feat: add models (customer, user, role, permission)
af211dc feat: add config loader and utils (response, password, validator)
eb82649 chore: initialize project with dependencies
```

---

## Hướng dẫn chạy

```bash
# 1. Copy env
cp .env.example .env
# Sửa .env với thông tin PostgreSQL của bạn

# 2. Run
go run ./cmd/server/

# Output:
# Server starting on port 3000
# Seed data created successfully  (lần chạy đầu)
```
