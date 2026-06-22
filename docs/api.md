# API Documentation — Go Fiber

**Base URL:** `http://localhost:3000/api/v1`

---

## Authentication

### POST /auth/admin/login
Login as admin.

**Rate limit:** 5 requests/min per IP

**Request:**
```json
{
  "login": "admin@gmail.com",
  "password": "123123"
}
```

**Response 200:**
```json
{
  "success": true,
  "data": {
    "access_token": "eyJ...",
    "refresh_token": "eyJ..."
  },
  "message": "Login successful"
}
```

**Response 401:**
```json
{
  "success": false,
  "error": { "code": "INVALID_CREDENTIALS", "message": "invalid credentials" }
}
```

---

### POST /auth/customer/login
Login as customer.

**Rate limit:** 5 requests/min per IP

**Request:**
```json
{
  "login": "customer@example.com",
  "password": "123123"
}
```

**Response 200:**
```json
{
  "success": true,
  "data": {
    "access_token": "eyJ...",
    "refresh_token": "eyJ..."
  },
  "message": "Login successful"
}
```

---

### POST /auth/refresh
Refresh access token.

**Request:**
```json
{
  "refresh_token": "eyJ..."
}
```

**Response 200:**
```json
{
  "success": true,
  "data": {
    "access_token": "eyJ..."
  },
  "message": "Token refreshed"
}
```

---

## Customer Self-Service (JWT required)

**Headers:** `Authorization: Bearer <access_token>`

### GET /customer/profile
Get current customer profile.

**Response 200:**
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "name": "John",
    "email": "john@example.com",
    "phone_number": "0335909200",
    "status": "active",
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
}
```

---

### PUT /customer/profile
Update current customer profile.

**Request:**
```json
{
  "name": "John Updated",
  "email": "john_new@example.com",
  "phone_number": "0335909201",
  "status": "active"
}
```

**Response 200:**
```json
{
  "success": true,
  "data": { ... },
  "message": "Profile updated"
}
```

---

## Admin Endpoints (JWT required, RBAC enforced)

**Headers:** `Authorization: Bearer <access_token>`

### GET /admin/dashboard/stats
Permission: `dashboard:read`

**Response 200:**
```json
{
  "success": true,
  "data": {
    "total_customers": 42,
    "total_users": 5,
    "total_roles": 3,
    "active_customers": 38
  }
}
```

---

### GET /admin/customers
Permission: `customer:read`

**Query params:** `page=1&limit=10`

**Response 200:**
```json
{
  "success": true,
  "data": [ ... ],
  "pagination": {
    "page": 1,
    "limit": 10,
    "total": 42,
    "total_pages": 5
  }
}
```

---

### GET /admin/customers/:id
Permission: `customer:read`

**Response 200:** Single customer object.
**Response 404:** `{ "code": "NOT_FOUND", "message": "Customer not found" }`

---

### POST /admin/customers
Permission: `customer:write`

**Request:**
```json
{
  "name": "New Customer",
  "email": "new@example.com",
  "phone_number": "0335909200",
  "password": "password123"
}
```

**Response 201:**
```json
{
  "success": true,
  "data": { ... },
  "message": "Customer created"
}
```

**Response 409:**
```json
{
  "success": false,
  "error": { "code": "DUPLICATE_ENTRY", "message": "Email or phone already exists" }
}
```

---

### PUT /admin/customers/:id
Permission: `customer:write`

**Request (partial):**
```json
{
  "name": "Updated Name",
  "status": "inactive"
}
```

**Response 200:**
```json
{
  "success": true,
  "data": { ... },
  "message": "Customer updated"
}
```

---

### DELETE /admin/customers/:id
Permission: `customer:delete`

**Response 200:**
```json
{
  "success": true,
  "data": null,
  "message": "Customer deleted"
}
```

---

### GET /admin/users
Permission: `user:read`

**Query params:** `page=1&limit=10`

**Response 200:**
```json
{
  "success": true,
  "data": [...],
  "pagination": {
    "page": 1,
    "limit": 10,
    "total": 5,
    "total_pages": 1
  }
}
```

---

### GET /admin/users/:id
Permission: `user:read`

**Response 200:** Single user with role & permissions.

---

### POST /admin/users
Permission: `user:write`

**Request:**
```json
{
  "name": "Staff User",
  "email": "staff@example.com",
  "phone_number": "0335909200",
  "password": "password123",
  "role_id": "uuid-of-role"
}
```

**Response 200:**
```json
{
  "success": true,
  "data": { ... },
  "message": "User created"
}
```

---

### PUT /admin/users/:id
Permission: `user:write`

**Request (partial):**
```json
{
  "email": "newemail@example.com",
  "name": "Updated Staff",
  "role_id": "uuid-of-new-role",
  "status": "inactive"
}
```

**Response 200:**
```json
{
  "success": true,
  "data": { ... },
  "message": "User updated"
}
```

---

### DELETE /admin/users/:id
Permission: `user:delete`

**Response 200:**
```json
{
  "success": true,
  "data": null,
  "message": "User deleted"
}
```

---

### GET /admin/roles
Permission: `role:read`

**Response 200:**
```json
{
  "success": true,
  "data": [
    {
      "id": "uuid",
      "name": "super_admin",
      "description": "Full access",
      "permissions": [...],
      "created_at": "...",
      "updated_at": "..."
    }
  ]
}
```

---

### POST /admin/roles
Permission: `role:write`

**Request:**
```json
{
  "name": "moderator",
  "description": "Moderate customers",
  "permission_ids": ["uuid1", "uuid2"]
}
```

**Response 200:**
```json
{
  "success": true,
  "data": { ... },
  "message": "Role created"
}
```

---

### PUT /admin/roles/:id
Permission: `role:write`

**Request (partial):**
```json
{
  "name": "moderator_v2",
  "description": "Updated description",
  "permission_ids": ["uuid1", "uuid3"]
}
```

**Response 200:**
```json
{
  "success": true,
  "data": { ... },
  "message": "Role updated"
}
```

---

### DELETE /admin/roles/:id
Permission: `role:delete`

**Response 200:**
```json
{
  "success": true,
  "data": null,
  "message": "Role deleted"
}
```

---

### GET /admin/permissions
Permission: `permission:read`

**Response 200:**
```json
{
  "success": true,
  "data": [
    { "id": "uuid", "name": "customer:read", "description": "View customers", "created_at": "..." },
    ...
  ]
}
```

---

### POST /admin/permissions
Permission: `permission:write`

**Request:**
```json
{
  "name": "report:read",
  "description": "View reports"
}
```

**Response 200:**
```json
{
  "success": true,
  "data": { ... },
  "message": "Permission created"
}
```

---

## Error Response Format

All errors follow this shape:

```json
{
  "success": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable message"
  }
}
```

### Common error codes

| Status | Code                | Description                      |
|--------|---------------------|----------------------------------|
| 400    | `VALIDATION_ERROR`  | Invalid input / body             |
| 401    | `UNAUTHORIZED`      | Missing/invalid JWT              |
| 401    | `INVALID_CREDENTIALS`| Wrong login/password            |
| 403    | `FORBIDDEN`         | Insufficient permissions         |
| 404    | `NOT_FOUND`         | Resource not found               |
| 409    | `DUPLICATE_ENTRY`   | Unique constraint violation      |
| 500    | `INTERNAL_ERROR`    | Server error                     |

---

## Permissions System

| Permission        | Description               |
|-------------------|---------------------------|
| `customer:read`   | View customers            |
| `customer:write`  | Create/update customers   |
| `customer:delete` | Delete customers          |
| `user:read`       | View users                |
| `user:write`      | Create/update users       |
| `user:delete`     | Delete users              |
| `role:read`       | View roles                |
| `role:write`      | Create/update roles       |
| `role:delete`     | Delete roles              |
| `permission:read` | View permissions          |
| `permission:write`| Create permissions        |
| `dashboard:read`  | View dashboard            |

### Seed Roles

| Role          | Description            | Permissions                                         |
|---------------|------------------------|-----------------------------------------------------|
| `super_admin` | Full access            | All 12 permissions                                  |
| `editor`      | Edit customers         | `customer:read`, `customer:write`, `dashboard:read` |
| `viewer`      | Read-only access       | `customer:read`, `dashboard:read`                   |

---

## Route Summary

| Method   | Path                         | Auth   | Permission          |
|----------|------------------------------|--------|---------------------|
| POST     | `/auth/admin/login`          | —      | —                   |
| POST     | `/auth/customer/login`       | —      | —                   |
| POST     | `/auth/refresh`              | —      | —                   |
| GET      | `/customer/profile`          | JWT    | — (customer only)   |
| PUT      | `/customer/profile`          | JWT    | — (customer only)   |
| GET      | `/admin/dashboard/stats`     | JWT    | `dashboard:read`    |
| GET      | `/admin/customers`           | JWT    | `customer:read`     |
| GET      | `/admin/customers/:id`       | JWT    | `customer:read`     |
| POST     | `/admin/customers`           | JWT    | `customer:write`    |
| PUT      | `/admin/customers/:id`       | JWT    | `customer:write`    |
| DELETE   | `/admin/customers/:id`       | JWT    | `customer:delete`   |
| GET      | `/admin/users`               | JWT    | `user:read`         |
| GET      | `/admin/users/:id`           | JWT    | `user:read`         |
| POST     | `/admin/users`               | JWT    | `user:write`        |
| PUT      | `/admin/users/:id`           | JWT    | `user:write`        |
| DELETE   | `/admin/users/:id`           | JWT    | `user:delete`       |
| GET      | `/admin/roles`               | JWT    | `role:read`         |
| POST     | `/admin/roles`               | JWT    | `role:write`        |
| PUT      | `/admin/roles/:id`           | JWT    | `role:write`        |
| DELETE   | `/admin/roles/:id`           | JWT    | `role:delete`       |
| GET      | `/admin/permissions`         | JWT    | `permission:read`   |
| POST     | `/admin/permissions`         | JWT    | `permission:write`  |
