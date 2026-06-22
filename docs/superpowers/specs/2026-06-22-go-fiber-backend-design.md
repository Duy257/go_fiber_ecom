# Go-Fiber Backend Design

## Overview

Backend service xây dựng bằng Go + Fiber framework, PostgreSQL database. Hệ thống gồm 2 loại người dùng:

- **Customer**: Người dùng cuối, đăng nhập bằng email hoặc phone_number
- **Admin User**: Người quản trị, đăng nhập bằng email hoặc phone_number, có phân quyền RBAC

## Architecture

### Approach: Monolithic Single-Binary

Mọi thứ trong 1 service duy nhất. Routes nhóm theo domain. Clean Architecture với 3 layers: Handler → Service → Repository.

### Project Structure

```
go-fiber/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── config/
│   │   └── config.go
│   ├── database/
│   │   └── database.go
│   ├── middleware/
│   │   ├── auth.go
│   │   └── rbac.go
│   ├── models/
│   │   ├── user.go
│   │   ├── customer.go
│   │   ├── role.go
│   │   └── permission.go
│   ├── handlers/
│   │   ├── auth.go
│   │   ├── customer.go
│   │   ├── user.go
│   │   ├── role.go
│   │   ├── permission.go
│   │   └── dashboard.go
│   ├── services/
│   │   ├── auth_service.go
│   │   ├── customer_service.go
│   │   ├── user_service.go
│   │   ├── role_service.go
│   │   └── dashboard_service.go
│   ├── repositories/
│   │   ├── user_repo.go
│   │   ├── customer_repo.go
│   │   ├── role_repo.go
│   │   └── permission_repo.go
│   └── utils/
│       ├── response.go
│       ├── validator.go
│       └── password.go
├── .env.example
├── go.mod
└── go.sum
```

## Database Schema

### `customers`

| Column | Type | Constraints |
|---|---|---|
| id | uuid | PK, auto-gen |
| email | varchar(255) | unique, nullable |
| phone_number | varchar(20) | unique, nullable |
| password | varchar(255) | not null |
| name | varchar(255) | |
| status | varchar(20) | default "active" |
| created_at | timestamp | auto |
| updated_at | timestamp | auto |

### `users` (Admin)

| Column | Type | Constraints |
|---|---|---|
| id | uuid | PK, auto-gen |
| email | varchar(255) | unique, nullable |
| phone_number | varchar(20) | unique, nullable |
| password | varchar(255) | not null |
| name | varchar(255) | |
| role_id | uuid | FK → roles |
| status | varchar(20) | default "active" |
| created_at | timestamp | auto |
| updated_at | timestamp | auto |

### `roles`

| Column | Type | Constraints |
|---|---|---|
| id | uuid | PK, auto-gen |
| name | varchar(100) | unique |
| description | text | |
| created_at | timestamp | auto |
| updated_at | timestamp | auto |

### `permissions`

| Column | Type | Constraints |
|---|---|---|
| id | uuid | PK, auto-gen |
| name | varchar(100) | unique |
| description | text | |
| created_at | timestamp | auto |

### `role_permissions`

| Column | Type | Constraints |
|---|---|---|
| role_id | uuid | FK → roles, composite PK |
| permission_id | uuid | FK → permissions, composite PK |

## Authentication

### JWT Configuration

- **Algorithm**: HS256
- **Access token TTL**: 15 phút
- **Refresh token TTL**: 7 ngày
- **Storage**: Stateless (không lưu DB)

### Token Payload

Access token:
```json
{
  "sub": "user-uuid",
  "role": "admin",
  "type": "access",
  "exp": 1234567890
}
```

Refresh token:
```json
{
  "sub": "user-uuid",
  "type": "refresh",
  "exp": 1234567890
}
```

### Login Flow

1. Client gửi `{ "login": "email_or_phone", "password": "xxx" }`
2. Backend tìm user/customer theo email HOẶC phone_number
3. So sánh password bằng bcrypt
4. Tạo access_token + refresh_token
5. Trả về `{ "access_token": "...", "refresh_token": "..." }`

### Refresh Flow

1. Client gửi `{ "refresh_token": "..." }`
2. Verify chữ ký + thời hạn
3. Tạo access_token mới
4. Trả về `{ "access_token": "..." }`

### Middleware Chain

```
Request → JWT Middleware → RBAC Middleware → Handler
            ↓                ↓
     Verify token      Check permission
     Attach user       (admin routes only)
     to context
```

## RBAC

Simple role-based access control:

- Mỗi admin user có 1 role
- Mỗi role có nhiều permissions
- Permission format: `resource:action` (e.g. `customer:read`, `role:write`)
- Middleware kiểm tra permission trước khi cho phép truy cập endpoint

### Default Roles

- **super_admin**: Tất cả permissions
- **editor**: customer:read, customer:write, dashboard:read
- **viewer**: customer:read, dashboard:read

### Default Permissions

```
customer:read, customer:write, customer:delete
user:read, user:write, user:delete
role:read, role:write, role:delete
permission:read, permission:write
dashboard:read
```

## API Endpoints

### Public

| Method | Path | Description |
|---|---|---|
| POST | /api/v1/auth/customer/login | Customer login |
| POST | /api/v1/auth/admin/login | Admin login |
| POST | /api/v1/auth/refresh | Refresh token |

### Customer (auth required)

| Method | Path | Description |
|---|---|---|
| GET | /api/v1/customer/profile | Xem profile |
| PUT | /api/v1/customer/profile | Cập nhật profile |

### Admin (auth + RBAC required)

| Method | Path | Permission |
|---|---|---|
| GET | /api/v1/admin/dashboard/stats | dashboard:read |
| GET | /api/v1/admin/customers | customer:read |
| GET | /api/v1/admin/customers/:id | customer:read |
| POST | /api/v1/admin/customers | customer:write |
| PUT | /api/v1/admin/customers/:id | customer:write |
| DELETE | /api/v1/admin/customers/:id | customer:delete |
| GET | /api/v1/admin/users | user:read |
| GET | /api/v1/admin/users/:id | user:read |
| POST | /api/v1/admin/users | user:write |
| PUT | /api/v1/admin/users/:id | user:write |
| DELETE | /api/v1/admin/users/:id | user:delete |
| GET | /api/v1/admin/roles | role:read |
| POST | /api/v1/admin/roles | role:write |
| PUT | /api/v1/admin/roles/:id | role:write |
| DELETE | /api/v1/admin/roles/:id | role:delete |
| GET | /api/v1/admin/permissions | permission:read |
| POST | /api/v1/admin/permissions | permission:write |

## Response Format

### Success

```json
{
  "success": true,
  "data": { ... },
  "message": "Operation successful"
}
```

### Success with Pagination

```json
{
  "success": true,
  "data": [ ... ],
  "pagination": {
    "page": 1,
    "limit": 10,
    "total": 100,
    "total_pages": 10
  }
}
```

### Error

```json
{
  "success": false,
  "error": {
    "code": "INVALID_CREDENTIALS",
    "message": "Email/phone or password is incorrect"
  }
}
```

### Error Codes

| Code | HTTP Status | Description |
|---|---|---|
| INVALID_CREDENTIALS | 401 | Sai thông tin đăng nhập |
| UNAUTHORIZED | 401 | Không có token / token hết hạn |
| FORBIDDEN | 403 | Không có quyền truy cập |
| NOT_FOUND | 404 | Resource không tồn tại |
| VALIDATION_ERROR | 400 | Dữ liệu không hợp lệ |
| DUPLICATE_ENTRY | 409 | Email/phone đã tồn tại |
| INTERNAL_ERROR | 500 | Lỗi server |

## Dependencies

```
github.com/gofiber/fiber/v2
gorm.io/gorm
gorm.io/driver/postgres
github.com/golang-jwt/jwt/v5
golang.org/x/crypto
github.com/google/uuid
github.com/joho/godotenv
```

## Environment Variables

```
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=go_fiber
JWT_SECRET=your-secret-key
JWT_ACCESS_TTL=15m
JWT_REFRESH_TTL=168h
SERVER_PORT=3000
```

## Seed Data

Khởi tạo lần đầu:

1. Tạo role `super_admin` với tất cả permissions
2. Tạo admin user đầu tiên (super_admin) từ env vars hoặc CLI flag
