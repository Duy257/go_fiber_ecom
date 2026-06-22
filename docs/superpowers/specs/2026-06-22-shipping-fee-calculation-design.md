# Shipping Fee Calculation by Distance

**Date**: 2026-06-22
**Status**: Approved
**Scope**: Add lat/long to Shop, auto-calculate shipping fee based on distance using Haversine formula

## Overview

Tính phí ship tự động dựa trên khoảng cách giữa shop và địa chỉ giao hàng của khách hàng. Sử dụng Haversine formula (đường chim bay) với cấu hình global: base fee + per-km rate.

## Requirements

- Thêm `Latitude`, `Longitude` vào Shop model
- Client gửi tọa độ giao hàng (lat/long) khi tạo order
- Công thức: `base_fee + (per_km_rate × distance_km)`, làm tròn lên 1000 VND
- Global config: `base_fee`, `per_km_rate`, `max_distance_km`
- API preview phí ship trước khi đặt hàng
- Tự động tính phí ship khi tạo order (cho phép client override)
- Giới hạn max distance — reject nếu vượt quá

## Design Decisions

| Quyết định | Lựa chọn | Lý do |
|---|---|---|
| Approach tính khoảng cách | Haversine thuần trong Go | Đơn giản, không phụ thuộc外部, miễn phí, đủ chính xác cho e-commerce VN |
| Cấu hình phí | Global config | Dễ quản lý, tất cả shop dùng chung |
| Nguồn tọa độ khách | Client truyền lat/long | Frontend dùng Geolocation API, không cần backend geocode |
| Giới hạn khoảng cách | Global max_distance | Đơn giản, thống nhất cho tất cả shop |

## Model Changes

### Shop Model (`internal/models/shop.go`)

Thêm 2 field:

```go
type Shop struct {
    // ... existing fields ...
    Address   string  `gorm:"type:varchar(500)" json:"address,omitempty"`
    Latitude  float64 `gorm:"type:decimal(10,7)" json:"latitude,omitempty"`
    Longitude float64 `gorm:"type:decimal(10,7)" json:"longitude,omitempty"`
    Phone     string  `gorm:"type:varchar(20)" json:"phone,omitempty"`
    // ...
}
```

### Order Model (`internal/models/order.go`)

Thêm 3 field:

```go
type Order struct {
    // ... existing fields ...
    ShippingFee        float64                `gorm:"type:decimal(12,2);default:0" json:"shipping_fee"`
    ShippingAddress    map[string]interface{} `gorm:"type:jsonb;serializer:json;not null" json:"shipping_address"`
    ShippingLatitude   float64                `gorm:"type:decimal(10,7)" json:"shipping_latitude,omitempty"`
    ShippingLongitude  float64                `gorm:"type:decimal(10,7)" json:"shipping_longitude,omitempty"`
    ShippingDistanceKm float64                `gorm:"type:decimal(8,2)" json:"shipping_distance_km,omitempty"`
    // ...
}
```

### ShippingConfig Model (`internal/models/shipping_config.go`) — MỚI

```go
type ShippingConfig struct {
    ID            uint      `gorm:"primaryKey" json:"id"`
    BaseFee       float64   `gorm:"type:decimal(12,2);not null" json:"base_fee"`
    PerKmRate     float64   `gorm:"type:decimal(12,2);not null" json:"per_km_rate"`
    MaxDistanceKm float64   `gorm:"type:decimal(8,2);not null" json:"max_distance_km"`
    UpdatedAt     time.Time `json:"updated_at"`
}
```

**Seed data mặc định**: `base_fee=10000`, `per_km_rate=3000`, `max_distance_km=30`

## Utility: Haversine Distance (`internal/utils/haversine.go`)

```go
func HaversineDistance(lat1, lon1, lat2, lon2 float64) float64
```

- Input: 2 cặp tọa độ (lat/long)
- Output: khoảng cách km
- Earth radius = 6371 km

## Shipping Service (`internal/services/shipping_service.go`)

### Input/Output types

```go
type ShippingEstimateInput struct {
    ShopID            string  `json:"shop_id" validate:"required"`
    ShippingLatitude  float64 `json:"shipping_latitude" validate:"required"`
    ShippingLongitude float64 `json:"shipping_longitude" validate:"required"`
}

type ShippingEstimateResult struct {
    DistanceKm    float64 `json:"distance_km"`
    BaseFee       float64 `json:"base_fee"`
    PerKmFee      float64 `json:"per_km_fee"`
    TotalFee      float64 `json:"total_fee"`
    MaxDistanceKm float64 `json:"max_distance_km"`
}
```

### Calculate flow

1. Load global `ShippingConfig` từ DB
2. Lấy shop by ID → validate shop đã có lat/long
3. `distance = HaversineDistance(shop.Lat, shop.Long, input.Lat, input.Long)`
4. Nếu `distance > max_distance_km` → return error `OUTSIDE_DELIVERY_RANGE`
5. `total_fee = base_fee + (per_km_rate × distance)`
6. Làm tròn `total_fee` lên đến 1000 VND (ví dụ: 25300 → 26000)
7. Return `ShippingEstimateResult`

## API Endpoints

### Preview phí ship

```
POST /api/v1/shipping/estimate
Body: {
    "shop_id": "uuid",
    "shipping_latitude": 10.7769,
    "shipping_longitude": 106.7009
}
Response: {
    "distance_km": 5.2,
    "base_fee": 10000,
    "per_km_fee": 15600,
    "total_fee": 26000,
    "max_distance_km": 30
}
```

### Tạo order — tự tính phí ship

Modify `CreateOrderInput` trong `internal/services/order_service.go`:

```go
type CreateOrderInput struct {
    CustomerID         string                 `json:"customer_id" validate:"required"`
    ShopID             string                 `json:"shop_id" validate:"required"`
    Items              []CreateOrderItemInput `json:"items" validate:"required,min=1"`
    ShippingFee        *float64               `json:"shipping_fee"`         // optional override
    ShippingAddress    map[string]interface{} `json:"shipping_address" validate:"required"`
    ShippingLatitude   float64                `json:"shipping_latitude" validate:"required"`
    ShippingLongitude  float64                `json:"shipping_longitude" validate:"required"`
    Note               string                 `json:"note"`
    PaymentMethod      string                 `json:"payment_method" validate:"required,oneof=cod bank_transfer e_wallet"`
}
```

Logic trong `OrderService.Create()`:
1. Inject `ShippingService` vào `OrderService`
2. Gọi `ShippingService.Calculate(shopID, lat, long)` để tính phí
3. Nếu `input.ShippingFee != nil` → dùng giá trị client truyền (override)
4. Nếu không → dùng phí tự tính
5. Lưu `ShippingLatitude`, `ShippingLongitude`, `ShippingDistanceKm` vào order

### Admin quản lý shipping config

Yêu cầu authentication + admin role (sử dụng middleware hiện có).

```
GET  /api/v1/admin/shipping-config       → lấy config hiện tại
PUT  /api/v1/admin/shipping-config       → cập nhật config
Body: {
    "base_fee": 10000,
    "per_km_rate": 3000,
    "max_distance_km": 30
}
```

### Shop update tọa độ

Modify `UpdateShopInput` trong `internal/services/shop_service.go`:

```go
type UpdateShopInput struct {
    // ... existing fields ...
    Latitude  *float64 `json:"latitude"`
    Longitude *float64 `json:"longitude"`
}
```

Modify `CreateShopInput`:

```go
type CreateShopInput struct {
    // ... existing fields ...
    Latitude  float64 `json:"latitude"`
    Longitude float64 `json:"longitude"`
}
```

## Validation Rules

| Field | Rule |
|---|---|
| Latitude | -90 đến 90 |
| Longitude | -180 đến 180 |
| ShippingLatitude | required khi tạo order |
| ShippingLongitude | required khi tạo order |
| Shop.Lat/Long | phải được set trước khi tính phí ship |

## Error Handling

| Trường hợp | HTTP Status | Error Code |
|---|---|---|
| Shop chưa set lat/long | 400 | `SHOP_LOCATION_NOT_SET` |
| Ngoài phạm vi giao hàng | 400 | `OUTSIDE_DELIVERY_RANGE` |
| Shipping config chưa setup | 500 | `SHIPPING_CONFIG_NOT_FOUND` |
| Lat/long không hợp lệ | 400 | `VALIDATION_ERROR` |

## File Changes Summary

### MỚI
- `internal/models/shipping_config.go` — ShippingConfig model
- `internal/repositories/shipping_config_repo.go` — CRUD shipping config
- `internal/services/shipping_service.go` — tính phí ship
- `internal/handlers/shipping_handler.go` — API endpoints
- `internal/utils/haversine.go` — Haversine distance function

### THAY ĐỔI
- `internal/models/shop.go` — thêm Latitude, Longitude
- `internal/models/order.go` — thêm ShippingLatitude, ShippingLongitude, ShippingDistanceKm
- `internal/repositories/shop_repo.go` — update Create/Update
- `internal/services/shop_service.go` — update CreateShopInput, UpdateShopInput
- `internal/services/order_service.go` — inject ShippingService, modify Create()
- `internal/handlers/shop_handler.go` — update input binding
- `cmd/main.go` hoặc nơi setup routes — register shipping routes

### Migration

GORM AutoMigrate sẽ tự động:
- Thêm cột `latitude`, `longitude` vào bảng `shops`
- Thêm cột `shipping_latitude`, `shipping_longitude`, `shipping_distance_km` vào bảng `orders`
- Tạo bảng mới `shipping_configs`

### Seed Data

Sau khi AutoMigrate, trong `internal/database/database.go` thêm logic seed `shipping_configs`:
- Kiểm tra bảng có rỗng không (`SELECT count(*) FROM shipping_configs`)
- Nếu rỗng → insert 1 dòng mặc định: `base_fee=10000`, `per_km_rate=3000`, `max_distance_km=30`
- Dùng `FirstOrCreate` để idempotent
