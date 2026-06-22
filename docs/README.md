# Go-Fiber API

Fiber web server + PostgreSQL + JWT auth + RBAC.

## Yêu cầu

- Go 1.26+
- PostgreSQL

## Cài đặt

```bash
# Copy env
cp .env.example .env
# Sửa .env theo DB của bạn (xem bên dưới)
```

## Cấu hình (.env)

| Biến | Mô tả | Mặc định |
|------|-------|----------|
| DB_HOST | Host PostgreSQL | localhost |
| DB_PORT | Port PostgreSQL | 5432 |
| DB_USER | User PostgreSQL | postgres |
| DB_PASSWORD | Password PostgreSQL | postgres |
| DB_NAME | Database name | go_fiber |
| JWT_SECRET | Secret key (≥16 ký tự) | — |
| JWT_ACCESS_TTL | Access token hết hạn | 15m |
| JWT_REFRESH_TTL | Refresh token hết hạn | 168h |
| SERVER_PORT | Cổng chạy server | 3000 |
| ADMIN_EMAIL | Email admin mặc định | admin@example.com |
| ADMIN_PASSWORD | Password admin mặc định | (bắt buộc đổi) |
| ADMIN_PHONE | SĐT admin | 0900000000 |

> **Quan trọng**: Đổi `JWT_SECRET` (≥16 ký tự) và `ADMIN_PASSWORD` trước khi chạy.

## Tạo database

```bash
createdb go_fiber
# Hoặc vào psql:
# CREATE DATABASE go_fiber;
```

## Chạy

```bash
go run cmd/server/main.go
```

Server chạy tại `http://localhost:3000` (hoặc port bạn cấu hình).

## Seed data

Lần chạy đầu, tự động tạo:
- **Roles**: super_admin, editor, viewer
- **Permissions**: 12 quyền (customer:read/write/delete, user:read/write/delete, role:read/write/delete, permission:read/write, dashboard:read)
- **Admin user** (.env → ADMIN_EMAIL / ADMIN_PASSWORD)

## API endpoints

### Public
| Method | Path | Mô tả |
|--------|------|-------|
| POST | /api/v1/auth/customer/login | Đăng nhập customer |
| POST | /api/v1/auth/admin/login | Đăng nhập admin |
| POST | /api/v1/auth/refresh | Refresh token |

### Customer (cần token)
| Method | Path | Mô tả |
|--------|------|-------|
| GET | /api/v1/customer/profile | Xem profile |
| PUT | /api/v1/customer/profile | Sửa profile |

### Admin (cần token + permission)
| Method | Path | Permission |
|--------|------|------------|
| GET | /api/v1/admin/dashboard/stats | dashboard:read |
| GET/POST | /api/v1/admin/customers | customer:read/write |
| PUT/DELETE | /api/v1/admin/customers/:id | customer:write/delete |
| GET/POST | /api/v1/admin/users | user:read/write |
| PUT/DELETE | /api/v1/admin/users/:id | user:write/delete |
| GET/POST | /api/v1/admin/roles | role:read/write |
| PUT/DELETE | /api/v1/admin/roles/:id | role:write/delete |
| GET/POST | /api/v1/admin/permissions | permission:read/write |

## Cấu trúc project

```
go-fiber/
├── cmd/server/main.go      # Entry point
├── internal/
│   ├── config/             # Config from .env
│   ├── database/           # DB connect + migrate
│   ├── models/             # GORM models
│   ├── repositories/       # DB queries
│   ├── services/           # Business logic
│   ├── handlers/           # HTTP handlers
│   ├── middleware/         # JWT auth + RBAC
│   └── utils/              # Validator, response, password
├── .env                    # Config (git-ignored)
├── .env.example            # Config mẫu
├── go.mod / go.sum
└── docs/                   # Docs
```
