# E-Commerce System Design

## Overview

Mở rộng hệ thống go-fiber backend với các module ecommerce: Category, Shop, Product, và Order. Tuân theo kiến trúc clean architecture hiện có (handler → service → repository → model).

## Scope

- Category (2 cấp: parent-child)
- Shop (1-1 với User admin)
- Product (1-N Shop, N-N Category, có Variant)
- Order system (Order, OrderItem, Payment, Status History)
- Soft delete trên tất cả bảng chính
- Inventory tracking qua ProductVariant.Stock

## Quan hệ tổng quan

```
User (1) ──── (1) Shop
                 │
                 │ 1-N
                 ▼
Category (N) ── (N) Product ── (1-N) ProductVariant
    │                              │
    │ parent_id                    │
    ▼                              ▼
Category (con)              OrderItem (snapshot price)
                                │
Customer ──── Order ────────────┘
                │
                ├── OrderStatusHistory
                └── Payment (1-1)
```

---

## 1. Category

### Schema

| Field       | Type           | Constraint      | Mô tả                    |
|-------------|----------------|-----------------|---------------------------|
| id          | uuid           | PK              |                           |
| name        | varchar(255)   | NOT NULL        | Tên danh mục              |
| slug        | varchar(255)   | UNIQUE NOT NULL | URL-friendly              |
| description | text           |                 | Mô tả                    |
| image       | varchar(500)   |                 | Ảnh danh mục              |
| parent_id   | uuid           | FK → category   | null = cấp cha            |
| sort_order  | int            | DEFAULT 0       | Thứ tự hiển thị           |
| status      | varchar(20)    | DEFAULT active  | active/inactive           |
| deleted_at  | timestamp      | INDEX           | Soft delete               |
| created_at  | timestamp      |                 |                           |
| updated_at  | timestamp      |                 |                           |

### Business Rules

- Nếu `parent_id` != null → danh mục con. Không lồng sâu hơn 2 cấp.
- Slug được generate tự động từ name, đảm bảo unique.
- Không xóa category nếu còn product liên kết.

---

## 2. Shop

### Schema

| Field       | Type           | Constraint      | Mô tả                    |
|-------------|----------------|-----------------|---------------------------|
| id          | uuid           | PK              |                           |
| user_id     | uuid           | FK → user, UNIQUE | 1-1 với User (admin)    |
| name        | varchar(255)   | NOT NULL        | Tên shop                  |
| slug        | varchar(255)   | UNIQUE NOT NULL | URL-friendly              |
| description | text           |                 | Mô tả shop               |
| logo        | varchar(500)   |                 | Logo URL                  |
| address     | varchar(500)   |                 | Địa chỉ shop              |
| phone       | varchar(20)    |                 | SĐT shop                  |
| status      | varchar(20)    | DEFAULT active  | active/inactive/suspended |
| deleted_at  | timestamp      | INDEX           | Soft delete               |
| created_at  | timestamp      |                 |                           |
| updated_at  | timestamp      |                 |                           |

### Business Rules

- Mỗi User chỉ tạo được 1 Shop (unique constraint trên user_id).
- Shop có thể bị suspended bởi hệ thống.

---

## 3. Product

### Schema

| Field       | Type           | Constraint      | Mô tả                    |
|-------------|----------------|-----------------|---------------------------|
| id          | uuid           | PK              |                           |
| shop_id     | uuid           | FK → shop, INDEX| Thuộc shop nào            |
| name        | varchar(255)   | NOT NULL        | Tên sản phẩm              |
| slug        | varchar(255)   | UNIQUE NOT NULL | URL-friendly              |
| description | text           |                 | Mô tả sản phẩm           |
| price       | decimal(12,2)  | NOT NULL        | Giá gốc                   |
| status      | varchar(20)    | DEFAULT active  | active/draft/archived     |
| deleted_at  | timestamp      | INDEX           | Soft delete               |
| created_at  | timestamp      |                 |                           |
| updated_at  | timestamp      |                 |                           |

### ProductVariant

| Field       | Type           | Constraint      | Mô tả                    |
|-------------|----------------|-----------------|---------------------------|
| id          | uuid           | PK              |                           |
| product_id  | uuid           | FK → product    |                           |
| name        | varchar(255)   | NOT NULL        | "Đỏ / XL"                |
| sku         | varchar(100)   | UNIQUE          | Mã SKU                    |
| price       | decimal(12,2)  | NOT NULL        | Giá riêng (override)      |
| stock       | int            | NOT NULL, DEFAULT 0 | Tồn kho              |
| attributes  | jsonb          |                 | {"color":"red","size":"XL"}|
| deleted_at  | timestamp      | INDEX           | Soft delete               |
| created_at  | timestamp      |                 |                           |
| updated_at  | timestamp      |                 |                           |

### ProductImage

| Field       | Type           | Constraint      | Mô tả                    |
|-------------|----------------|-----------------|---------------------------|
| id          | uuid           | PK              |                           |
| product_id  | uuid           | FK → product    |                           |
| url         | varchar(500)   | NOT NULL        | Ảnh URL                   |
| sort_order  | int            | DEFAULT 0       | Thứ tự hiển thị           |

### ProductCategory (N-N join table)

| Field        | Type | Constraint        | Mô tả       |
|--------------|------|-------------------|--------------|
| product_id   | uuid | PK, FK → product  |              |
| category_id  | uuid | PK, FK → category |              |

### Business Rules

- Product phải có ít nhất 1 ProductVariant.
- Khi tạo OrderItem, lấy giá từ ProductVariant (nếu có) hoặc Product.Price.
- Stock giảm khi Order confirmed, tăng khi Order cancelled.

---

## 4. Order System

### Order

| Field             | Type           | Constraint        | Mô tả                    |
|-------------------|----------------|-------------------|---------------------------|
| id                | uuid           | PK                |                           |
| customer_id       | uuid           | FK → customer     | Người đặt                 |
| shop_id           | uuid           | FK → shop         | Shop bán                  |
| order_number      | varchar(50)    | UNIQUE NOT NULL   | ORD-YYYYMMDD-XXXX         |
| status            | varchar(20)    | DEFAULT pending   | pending/confirmed/shipping/delivered/cancelled |
| sub_total         | decimal(12,2)  | NOT NULL          | Tổng tiền hàng            |
| shipping_fee      | decimal(12,2)  | DEFAULT 0         | Phí ship                  |
| total_amount      | decimal(12,2)  | NOT NULL          | sub_total + shipping_fee  |
| shipping_address  | jsonb          | NOT NULL          | Snapshot địa chỉ giao     |
| note              | text           |                   | Ghi chú đơn hàng          |
| deleted_at        | timestamp      | INDEX             | Soft delete               |
| created_at        | timestamp      |                   |                           |
| updated_at        | timestamp      |                   |                           |

### OrderItem

| Field         | Type           | Constraint    | Mô tả                    |
|---------------|----------------|---------------|---------------------------|
| id            | uuid           | PK            |                           |
| order_id      | uuid           | FK → order    |                           |
| product_id    | uuid           | FK → product  |                           |
| variant_id    | uuid           | FK → variant  | Nullable                  |
| product_name  | varchar(255)   | NOT NULL      | Snapshot tên sản phẩm     |
| variant_name  | varchar(255)   |               | Snapshot tên variant       |
| price         | decimal(12,2)  | NOT NULL      | Snapshot giá              |
| quantity      | int            | NOT NULL      | Số lượng                   |
| total         | decimal(12,2)  | NOT NULL      | price * quantity           |

### OrderStatusHistory

| Field       | Type         | Constraint    | Mô tả                    |
|-------------|--------------|---------------|---------------------------|
| id          | uuid         | PK            |                           |
| order_id    | uuid         | FK → order    |                           |
| status      | varchar(20)  | NOT NULL      | Trạng thái tại thời điểm  |
| note        | text         |               | Ghi chú                   |
| created_at  | timestamp    |               |                           |

### Payment

| Field           | Type           | Constraint        | Mô tả                    |
|-----------------|----------------|-------------------|---------------------------|
| id              | uuid           | PK                |                           |
| order_id        | uuid           | FK → order, UNIQUE| 1-1 với Order             |
| method          | varchar(50)    | NOT NULL          | cod/bank_transfer/e_wallet|
| status          | varchar(20)    | DEFAULT pending   | pending/paid/failed/refunded|
| amount          | decimal(12,2)  | NOT NULL          | Số tiền                   |
| transaction_id  | varchar(255)   |                   | Mã giao dịch cổng TT      |
| paid_at         | timestamp      |                   | Thời điểm thanh toán      |
| deleted_at      | timestamp      | INDEX             | Soft delete               |
| created_at      | timestamp      |                   |                           |
| updated_at      | timestamp      |                   |                           |

### Luồng trạng thái Order

```
Customer tạo Order
       │
       ▼
    pending ──────┐
       │          │ (Customer/Admin hủy)
       ▼          ▼
   confirmed   cancelled
       │
       ▼
    shipping
       │
       ▼
   delivered
```

### Business Rules

- Tạo Order: snapshot giá từ ProductVariant/Product vào OrderItem, trừ stock.
- Confirm: admin xác nhận đơn.
- Shipping: admin đánh dấu đang giao.
- Delivered: customer xác nhận nhận hàng, Payment → paid (nếu COD).
- Cancel: hoàn stock, Payment → failed (nếu đã paid thì → refunded).
- Mỗi lần thay đổi status → ghi OrderStatusHistory.

---

## File Structure (tuân theo pattern hiện có)

```
internal/
├── models/
│   ├── category.go
│   ├── shop.go
│   ├── product.go
│   ├── order.go
│   └── payment.go
├── repositories/
│   ├── category_repo.go
│   ├── shop_repo.go
│   ├── product_repo.go
│   └── order_repo.go
├── services/
│   ├── category_service.go
│   ├── shop_service.go
│   ├── product_service.go
│   └── order_service.go
├── handlers/
│   ├── category_handler.go
│   ├── shop_handler.go
│   ├── product_handler.go
│   └── order_handler.go
```

## API Endpoints (dự kiến)

| Method | Endpoint                    | Handler               | Mô tả                    |
|--------|-----------------------------|-----------------------|---------------------------|
| POST   | /api/categories             | CreateCategory        | Tạo danh mục              |
| GET    | /api/categories             | ListCategories        | Danh sách danh mục        |
| GET    | /api/categories/:id         | GetCategory           | Chi tiết danh mục          |
| PUT    | /api/categories/:id         | UpdateCategory        | Cập nhật danh mục          |
| DELETE | /api/categories/:id         | DeleteCategory        | Xóa danh mục               |
| POST   | /api/shops                  | CreateShop            | Tạo shop (User admin)      |
| GET    | /api/shops/:id              | GetShop               | Chi tiết shop              |
| PUT    | /api/shops/:id              | UpdateShop            | Cập nhật shop              |
| POST   | /api/products               | CreateProduct         | Tạo sản phẩm              |
| GET    | /api/products               | ListProducts          | Danh sách sản phẩm        |
| GET    | /api/products/:id           | GetProduct            | Chi tiết sản phẩm          |
| PUT    | /api/products/:id           | UpdateProduct         | Cập nhật sản phẩm          |
| DELETE | /api/products/:id           | DeleteProduct         | Xóa sản phẩm               |
| POST   | /api/orders                 | CreateOrder           | Customer tạo đơn           |
| GET    | /api/orders                 | ListOrders            | Danh sách đơn (filter)     |
| GET    | /api/orders/:id             | GetOrder              | Chi tiết đơn               |
| PUT    | /api/orders/:id/status      | UpdateOrderStatus     | Cập nhật trạng thái        |
| POST   | /api/orders/:id/cancel      | CancelOrder           | Hủy đơn                    |
